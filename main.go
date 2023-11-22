package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/pelletier/go-toml/v2"
)

// Post type enum
type PostType int

const (
	SpecialPost PostType = iota
	BlogPost
	ProjectPost
	HomepagePost
	BlogHomePost
)

// Structs
type FooterData struct {
	Github   string
	Linkedin string
}

type SiteConfig struct {
	Name        string
	Username    string
	Description string
	ProfilePic  string
	Footer      FooterData
}

type MarkdownFile struct {
	FileName string
	FileText string
}

type ProjectFolder struct {
	SourceDir      string
	DestinationDir string
	ProjectType    PostType
	MarkdownFiles  []MarkdownFile
}

type Frontmatter struct {
	FileName    string
	Title       string
	Date        string
	Description string
	SortDate    time.Time
}

type FrontmatterList []Frontmatter

func (f FrontmatterList) Len() int           { return len(f) }
func (f FrontmatterList) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f FrontmatterList) Less(i, j int) bool { return f[i].SortDate.After(f[j].SortDate) }

// Global variables
var blog_name string
var pfp_path string
var footer string
var blogposts_data FrontmatterList
var projects_data FrontmatterList

// Constants
const SiteBasePath = "site/"
const SiteBlogPath = "site/blog/"
const BlogPathForLinks = "/site/blog/"
const SiteProjectPath = "site/projects/"
const ProjectPathForLinks = "/site/projects/"
const GitHubFooter = "<a href=\"%s\"><img src=\"/images/base/github-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"
const LinkedInFooter = "<a href=\"%s\"><img src=\"/images/base/linkedin-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"
const HtmlTemplate = `<!DOCTYPE html>
<head>
    <title> %s </title>
	<link rel="stylesheet" href="/style/style.css">
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.3.1/styles/default.min.css">
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
	<link href="https://fonts.googleapis.com/css2?family=Ubuntu:ital,wght@0,300;0,400;0,500;0,700;1,300;1,400;1,500;1,700&display=swap" rel="stylesheet"> 
	<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.3.1/highlight.min.js"></script>
</head>
<body>
	<div class="sidebar">
		%s
	</div>
	<div class="content">
		%s
	</div>
	<script>
		hljs.highlightAll();
	</script>
</body>
`
const HtmlBlogpostTemplate = `<a href="%s" class="index-post-title">%s</a>
<div class="index-post-date">%s</div>
<p class="index-post-desc">%s</p>
`

func assemble_sidebar() string {
	sidebar := ""
	// Profile pic
	sidebar += fmt.Sprintf("<img class=\"profile-pic\" src=\" %s \">", pfp_path)
	// Poster's name
	sidebar += fmt.Sprintf("<h1> %s </h1>", blog_name)
	return sidebar
}

func assemble_webpage(page_title string, content string) string {
	return fmt.Sprintf(HtmlTemplate, page_title, assemble_sidebar(), content)
}

func process_md_file(md_file MarkdownFile) (string, Frontmatter) {
	// First find the frontmatter data
	md_data := strings.Replace(md_file.FileText, "\n", "", -1)
	frontmatter_data := regexp.MustCompile(`===(.*)===`).FindStringSubmatch(md_data)[1]

	// Then parse it
	title_loc := strings.Index(frontmatter_data, "title")
	date_loc := strings.Index(frontmatter_data, "date")
	description_loc := strings.Index(frontmatter_data, "description")

	frontmatter_obj := Frontmatter{
		FileName:    md_file.FileName,
		Title:       frontmatter_data[title_loc+len("title:")+1 : date_loc],
		Date:        frontmatter_data[date_loc+len("date:")+1 : description_loc],
		Description: frontmatter_data[description_loc+len("description:"):],
	}

	var err error
	frontmatter_obj.SortDate, err = time.Parse("01-02-2006", frontmatter_obj.Date)
	if err != nil {
		fmt.Println("Error parsing date:", err)
	}

	end_ptr := strings.Index(strings.Replace(md_file.FileText, "===", "xxx", 1), "===")

	return md_file.FileText[end_ptr+len("==="):], frontmatter_obj
}

func generate_footer(cfg SiteConfig) {
	// Look for a GitHub and LinkedIn link
	if cfg.Footer.Github != "" {
		github_logo := GitHubFooter
		footer += fmt.Sprintf(github_logo, cfg.Footer.Github)
		footer += "\n"
	}
	if cfg.Footer.Linkedin != "" {
		linkedin_logo := LinkedInFooter
		footer += fmt.Sprintf(linkedin_logo, cfg.Footer.Linkedin)
	}
}

func load_blog_pages(source_dir string, dest_dir string, post_type PostType) ProjectFolder {
	var project_folder ProjectFolder

	dir, err := os.Open(source_dir)
	if err != nil {
		fmt.Println("Error opening directory:", err)
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(0)
	if err != nil {
		fmt.Println("Error reading directory:", err)
	}

	for _, fileInfo := range fileInfos {
		// Check if it's a regular file
		if fileInfo.Mode().IsRegular() {
			filePath := filepath.Join(source_dir, fileInfo.Name())

			// Read and print the file content
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
			}

			project_folder.MarkdownFiles = append(project_folder.MarkdownFiles, MarkdownFile{strings.Replace(fileInfo.Name(), "md", "html", -1), string(fileData)})
		}
	}

	project_folder.SourceDir = source_dir
	project_folder.DestinationDir = dest_dir
	project_folder.ProjectType = post_type

	return project_folder
}

func md_to_html(md_file MarkdownFile, p_type PostType) string {
	md_str, md_frontmatter := process_md_file(md_file)

	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(md_str))

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	if p_type == BlogPost {
		blogposts_data = append(blogposts_data, md_frontmatter)
	} else if p_type == ProjectPost {
		projects_data = append(projects_data, md_frontmatter)
	}

	return assemble_webpage(blog_name+" · "+md_frontmatter.Title, string(markdown.Render(doc, renderer)))
}

func publish_folder(p_folder ProjectFolder) {
	for _, fileData := range p_folder.MarkdownFiles {
		html_file := md_to_html(fileData, p_folder.ProjectType)
		err := os.WriteFile(filepath.Join(p_folder.DestinationDir, fileData.FileName), []byte(html_file), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
			return
		}
	}

}

func generate_blog_homepage() {
	sort.Sort(blogposts_data)
	html_data := ""
	for x, blogpost := range blogposts_data {
		blogpost_html := fmt.Sprintf(HtmlBlogpostTemplate, path.Join(BlogPathForLinks, blogpost.FileName), blogpost.Title, blogpost.Date, blogpost.Description)
		if x < len(blogposts_data)-1 {
			blogpost_html += "<hr>"
		}
		html_data += blogpost_html
	}
	err := os.WriteFile("site/blog.html", []byte(assemble_webpage(blog_name+" · Blog", html_data)), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
		return
	}
}

func generate_projects_homepage() {
	sort.Sort(projects_data)
	html_data := ""
	for x, project := range projects_data {
		blogpost_html := fmt.Sprintf(HtmlBlogpostTemplate, path.Join(ProjectPathForLinks, project.FileName), project.Title, project.Date, project.Description)
		if x < len(projects_data)-1 {
			blogpost_html += "<hr>"
		}
		html_data += blogpost_html
	}
	err := os.WriteFile("index.html", []byte(assemble_webpage(blog_name+" · Homepage", html_data)), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
		return
	}
}

func main() {
	fmt.Println("Building site...")

	// Get config.toml
	config_txt, err := os.ReadFile("config.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Load the config.toml
	var cfg SiteConfig
	err = toml.Unmarshal([]byte(config_txt), &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing config: %v\n", err)
		os.Exit(1)
	}

	// Get the blog name
	blog_name = cfg.Name
	pfp_path = cfg.ProfilePic

	// Generate a footer
	generate_footer(cfg)

	// Then iterate through and convert the posts to HTML
	// == First convert the about and resume pages
	{
		special_pages := load_blog_pages("posts/special/", SiteBasePath, SpecialPost)
		publish_folder(special_pages)
	}
	{
		blogpost_pages := load_blog_pages("posts/blog", SiteBlogPath, BlogPost)
		publish_folder(blogpost_pages)
		generate_blog_homepage()
	}
	{
		projects_pages := load_blog_pages("posts/projects", SiteProjectPath, ProjectPost)
		publish_folder(projects_pages)
		generate_projects_homepage()
	}
}

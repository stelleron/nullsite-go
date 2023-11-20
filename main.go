package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
)

// Global variables
var blog_name string
var footer string

// Constants
const GitHubFooter = "<a href=\"%s\"><img src=\"/images/base/github-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"
const LinkedInFooter = "<a href=\"%s\"><img src=\"/images/base/linkedin-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"
const HtmlTemplate = `<!DOCTYPE html>
<head>
    <title> %s </title>
</head>
<body>
<div id="header">
<h1>%s</h1>
</div>
<div id="content">
%s
</div>
<div id="footer">
%s
</div>
</body>
`

type FooterData struct {
	Github   string
	Linkedin string
}

type SiteConfig struct {
	Name   string
	Footer FooterData
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
	Title       string
	Date        string
	Description string
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
		Title:       frontmatter_data[title_loc+len("title:")+1 : date_loc],
		Date:        frontmatter_data[date_loc+len("date:")+1 : description_loc],
		Description: frontmatter_data[description_loc+len("description:"):],
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

	return fmt.Sprintf(HtmlTemplate, md_frontmatter.Title, blog_name, markdown.Render(doc, renderer), footer)
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
	blog_name = fmt.Sprintf("%s's Blog", cfg.Name)

	// Generate a footer
	generate_footer(cfg)

	// Then iterate through and convert the posts to HTML
	// == First convert the about and resume pages
	{
		special_pages := load_blog_pages("posts/special/", "site/", SpecialPost)
		publish_folder(special_pages)
	}
	{
		blogpost_pages := load_blog_pages("posts/blog", "site/blog", BlogPost)
		publish_folder(blogpost_pages)
	}
}

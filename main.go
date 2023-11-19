package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"
)

var blog_name string
var footer string
var about_path string
var resume_path string

const GitHubFooter = "<a href=\"%s\"><img src=\"/images/base/github-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"
const LinkedInFooter = "<a href=\"%s\"><img src=\"/images/base/linkedin-mark.svg\" class=\"icon\" width=\"32\" height=\"32\"></a>"

type FooterData struct {
	Github   string
	Linkedin string
}

type SiteConfig struct {
	Name   string
	About  string
	Resume string
	Footer FooterData
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

func generate_special_pages(cfg SiteConfig) {

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

	// Get the blog name
	blog_name = fmt.Sprintf("%s's Blog", cfg.Name)

	// Generate a footer
	generate_footer(cfg)
	fmt.Println(footer)

	// Then iterate through and convert the posts into HTML
	dir, err := os.Open("posts")
	if err != nil {
		fmt.Println("Error opening directory:", err)
		return
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(0)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	var md_files []fs.FileInfo

	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() {
			md_files = append(md_files, fileInfo)
		}
	}

	fmt.Println("Converting special pages...")
	generate_special_pages(cfg)
}

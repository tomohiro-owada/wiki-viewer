package main

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Page holds all data needed to render a wiki page.
type Page struct {
	Title      string
	Body       template.HTML
	Sidebar    template.HTML
	Footer     template.HTML
	HasSidebar bool
	HasFooter  bool
	WikiName   string // empty for single-wiki mode
	HomeURL    string // "{base}/wiki/Home" or "{base}/{wiki}/wiki/Home"
	BasePath   string // e.g. "/docs" or ""
	IndexURL   string // "{base}/" for multi-wiki mode
}

// WikiEntry represents a wiki (subdirectory) in multi-wiki mode.
type WikiEntry struct {
	Name    string
	Display string
	URL     string
}

// IndexPage holds data for the multi-wiki index page.
type IndexPage struct {
	BasePath string
	Wikis    []WikiEntry
}

var (
	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

// expandWikiLinks converts [[Page Name]] to HTML links before markdown rendering.
// prefix includes basePath, e.g. "/docs" or "/docs/my-wiki".
func expandWikiLinks(src []byte, prefix string) []byte {
	return wikiLinkRe.ReplaceAllFunc(src, func(match []byte) []byte {
		inner := wikiLinkRe.FindSubmatch(match)[1]
		name := string(inner)

		// Support [[display|Page Name]] syntax
		display := name
		target := name
		if idx := strings.Index(name, "|"); idx >= 0 {
			display = name[:idx]
			target = name[idx+1:]
		}

		href := strings.ReplaceAll(target, " ", "-")
		return []byte(`<a href="` + prefix + `/wiki/` + href + `">` + display + `</a>`)
	})
}

// renderMarkdown converts markdown bytes to HTML.
func renderMarkdown(src []byte, linkPrefix string) (template.HTML, error) {
	src = expandWikiLinks(src, linkPrefix)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// loadPage reads a wiki page from disk and returns a fully populated Page.
func loadPage(dir, name, wikiName, basePath string) (*Page, error) {
	raw, err := os.ReadFile(filepath.Join(dir, name+".md"))
	if err != nil {
		return nil, err
	}

	linkPrefix := basePath
	homeURL := basePath + "/wiki/Home"
	if wikiName != "" {
		linkPrefix = basePath + "/" + wikiName
		homeURL = basePath + "/" + wikiName + "/wiki/Home"
	}

	body, err := renderMarkdown(raw, linkPrefix)
	if err != nil {
		return nil, err
	}

	title := strings.ReplaceAll(name, "-", " ")

	page := &Page{
		Title:    title,
		Body:     body,
		WikiName: wikiName,
		HomeURL:  homeURL,
		BasePath: basePath,
		IndexURL: basePath + "/",
	}

	if sidebar, err := loadSpecial(dir, "_Sidebar", linkPrefix); err == nil {
		page.Sidebar = sidebar
		page.HasSidebar = true
	}

	if footer, err := loadSpecial(dir, "_Footer", linkPrefix); err == nil {
		page.Footer = footer
		page.HasFooter = true
	}

	return page, nil
}

// loadSpecial reads and renders a special wiki file (_Sidebar, _Footer).
func loadSpecial(dir, name, linkPrefix string) (template.HTML, error) {
	raw, err := os.ReadFile(filepath.Join(dir, name+".md"))
	if err != nil {
		return "", err
	}
	return renderMarkdown(raw, linkPrefix)
}

// listPages returns all wiki page names (excluding special files) sorted alphabetically.
func listPages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var pages []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		if strings.HasPrefix(base, "_") {
			continue
		}
		pages = append(pages, base)
	}
	sort.Strings(pages)
	return pages
}

// listWikis returns all subdirectories that contain at least one .md file.
func listWikis(rootDir, basePath string) []WikiEntry {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil
	}
	var wikis []WikiEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		subEntries, err := os.ReadDir(filepath.Join(rootDir, name))
		if err != nil {
			continue
		}
		hasMD := false
		for _, se := range subEntries {
			if !se.IsDir() && strings.HasSuffix(se.Name(), ".md") {
				hasMD = true
				break
			}
		}
		if hasMD {
			wikis = append(wikis, WikiEntry{
				Name:    name,
				Display: strings.ReplaceAll(name, "-", " "),
				URL:     basePath + "/" + name + "/wiki/Home",
			})
		}
	}
	sort.Slice(wikis, func(i, j int) bool {
		return wikis[i].Name < wikis[j].Name
	})
	return wikis
}

// detectMode checks whether the directory contains .md files directly (single-wiki)
// or only subdirectories with .md files (multi-wiki).
func detectMode(dir string) (isMulti bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return false
		}
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			subEntries, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, se := range subEntries {
				if !se.IsDir() && strings.HasSuffix(se.Name(), ".md") {
					return true
				}
			}
		}
	}
	return false
}

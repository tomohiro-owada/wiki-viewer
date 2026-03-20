package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PagesInput struct {
	Wiki string `json:"wiki,omitempty" jsonschema:"wiki name (optional: lists wikis when omitted in multi-wiki mode)"`
}

type PageEntry struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

type PagesOutput struct {
	Mode  string      `json:"mode"`
	Wikis []string    `json:"wikis,omitempty"`
	Pages []PageEntry `json:"pages,omitempty"`
}

type DetailInput struct {
	Wiki string `json:"wiki,omitempty" jsonschema:"wiki name (required in multi-wiki mode)"`
	Page string `json:"page" jsonschema:"page name (e.g. Home, Getting-Started)"`
}

type DetailOutput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func setupMCPServer(dir string, multi bool) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "wiki-viewer",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pages",
		Description: "List wiki pages. In multi-wiki mode without a wiki parameter, lists available wikis. With a wiki parameter (or in single-wiki mode), lists pages in that wiki.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PagesInput) (*mcp.CallToolResult, PagesOutput, error) {
		if multi {
			if input.Wiki == "" {
				wikis := listWikis(dir, "")
				names := make([]string, len(wikis))
				for i, w := range wikis {
					names[i] = w.Name
				}
				return nil, PagesOutput{Mode: "multi", Wikis: names}, nil
			}
			if !validName.MatchString(input.Wiki) {
				return nil, PagesOutput{}, fmt.Errorf("invalid wiki name: %s", input.Wiki)
			}
			wikiDir := filepath.Join(dir, input.Wiki)
			pages := listPages(wikiDir)
			entries := toPageEntries(pages)
			return nil, PagesOutput{Mode: "multi", Pages: entries}, nil
		}
		pages := listPages(dir)
		entries := toPageEntries(pages)
		return nil, PagesOutput{Mode: "single", Pages: entries}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detail",
		Description: "Get the raw Markdown content of a wiki page.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DetailInput) (*mcp.CallToolResult, DetailOutput, error) {
		if input.Page == "" {
			return nil, DetailOutput{}, fmt.Errorf("page parameter is required")
		}
		if !validName.MatchString(input.Page) {
			return nil, DetailOutput{}, fmt.Errorf("invalid page name: %s", input.Page)
		}

		targetDir := dir
		if multi {
			if input.Wiki == "" {
				return nil, DetailOutput{}, fmt.Errorf("wiki parameter is required in multi-wiki mode")
			}
			if !validName.MatchString(input.Wiki) {
				return nil, DetailOutput{}, fmt.Errorf("invalid wiki name: %s", input.Wiki)
			}
			targetDir = filepath.Join(dir, input.Wiki)
		}

		raw, err := os.ReadFile(filepath.Join(targetDir, input.Page+".md"))
		if err != nil {
			return nil, DetailOutput{}, fmt.Errorf("page not found: %s", input.Page)
		}

		title := strings.ReplaceAll(input.Page, "-", " ")
		return nil, DetailOutput{Title: title, Content: string(raw)}, nil
	})

	return server
}

func toPageEntries(pages []string) []PageEntry {
	entries := make([]PageEntry, len(pages))
	for i, p := range pages {
		entries[i] = PageEntry{
			Name:    p,
			Display: strings.ReplaceAll(p, "-", " "),
		}
	}
	return entries
}

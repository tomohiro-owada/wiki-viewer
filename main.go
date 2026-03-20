package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func main() {
	dir := flag.String("dir", ".", "directory containing wiki .md files")
	port := flag.Int("port", 8080, "HTTP port to listen on")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	multi := detectMode(absDir)

	if multi {
		log.Printf("multi-wiki mode: serving wikis from subdirectories of %s", absDir)
		setupMultiWiki(mux, absDir)
	} else {
		log.Printf("single-wiki mode: serving %s", absDir)
		setupSingleWiki(mux, absDir)
	}

	// MCP server on /mcp
	mcpServer := setupMCPServer(absDir, multi)
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return mcpServer },
		nil,
	)
	mux.Handle("/mcp", mcpHandler)
	log.Printf("MCP endpoint: http://localhost:%d/mcp", *port)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func setupSingleWiki(mux *http.ServeMux, dir string) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/wiki/Home", http.StatusFound)
	})

	mux.HandleFunc("GET /wiki/{page}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("page")
		if !validName.MatchString(name) {
			http.NotFound(w, r)
			return
		}

		page, err := loadPage(dir, name, "")
		if err != nil {
			render404(w, dir, name, "", "/wiki/Home")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pageTmpl.Execute(w, page)
	})
}

func setupMultiWiki(mux *http.ServeMux, rootDir string) {
	// Index page listing all wikis
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		wikis := listWikis(rootDir)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTmpl.Execute(w, wikis)
	})

	// Redirect /{wiki}/ to /{wiki}/wiki/Home
	mux.HandleFunc("GET /{wiki}/{$}", func(w http.ResponseWriter, r *http.Request) {
		wikiName := r.PathValue("wiki")
		if !validName.MatchString(wikiName) {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/"+wikiName+"/wiki/Home", http.StatusFound)
	})

	// Serve wiki pages
	mux.HandleFunc("GET /{wiki}/wiki/{page}", func(w http.ResponseWriter, r *http.Request) {
		wikiName := r.PathValue("wiki")
		pageName := r.PathValue("page")

		if !validName.MatchString(wikiName) || !validName.MatchString(pageName) {
			http.NotFound(w, r)
			return
		}

		wikiDir := filepath.Join(rootDir, wikiName)
		page, err := loadPage(wikiDir, pageName, wikiName)
		if err != nil {
			render404(w, wikiDir, pageName, wikiName, "/"+wikiName+"/wiki/Home")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pageTmpl.Execute(w, page)
	})
}

func render404(w http.ResponseWriter, dir, name, wikiName, homeURL string) {
	title := strings.ReplaceAll(name, "-", " ")

	linkPrefix := ""
	if wikiName != "" {
		linkPrefix = "/" + wikiName
	}

	pages := listPages(dir)
	var links []string
	for _, p := range pages {
		display := strings.ReplaceAll(p, "-", " ")
		links = append(links, fmt.Sprintf(`<li><a href="%s/wiki/%s">%s</a></li>`, linkPrefix, p, display))
	}

	var pageListHTML string
	if len(links) > 0 {
		pageListHTML = `<div class="page-list"><h3>Available pages:</h3><ul>` +
			strings.Join(links, "") + `</ul></div>`
	}

	body := fmt.Sprintf(`<div class="not-found">
<h2>Page not found</h2>
<p>The page "%s" does not exist. Create <code>%s.md</code> to add it.</p>
%s
</div>`, title, name, pageListHTML)

	page := &Page{
		Title:    "Not Found",
		Body:     template.HTML(body),
		WikiName: wikiName,
		HomeURL:  homeURL,
	}

	if sidebar, err := loadSpecial(dir, "_Sidebar", linkPrefix); err == nil {
		page.Sidebar = sidebar
		page.HasSidebar = true
	}
	if footer, err := loadSpecial(dir, "_Footer", linkPrefix); err == nil {
		page.Footer = footer
		page.HasFooter = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	pageTmpl.Execute(w, page)
}

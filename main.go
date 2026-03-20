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
	base := flag.String("base", "", "base path prefix (e.g. /docs) for reverse proxy setups")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}

	// Normalize base path: ensure leading slash, no trailing slash
	basePath := strings.TrimRight(*base, "/")
	if basePath != "" && !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	// Inner mux handles routes without base prefix
	innerMux := http.NewServeMux()
	multi := detectMode(absDir)

	if multi {
		log.Printf("multi-wiki mode: serving wikis from subdirectories of %s", absDir)
		setupMultiWiki(innerMux, absDir, basePath)
	} else {
		log.Printf("single-wiki mode: serving %s", absDir)
		setupSingleWiki(innerMux, absDir, basePath)
	}

	// MCP server
	mcpServer := setupMCPServer(absDir, multi)
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return mcpServer },
		nil,
	)
	innerMux.Handle("/mcp", mcpHandler)

	// Wrap with StripPrefix if base path is set
	outerMux := http.NewServeMux()
	if basePath != "" {
		outerMux.Handle(basePath+"/", http.StripPrefix(basePath, innerMux))
		log.Printf("base path: %s", basePath)
		log.Printf("MCP endpoint: http://localhost:%d%s/mcp", *port, basePath)
	} else {
		outerMux = innerMux
		log.Printf("MCP endpoint: http://localhost:%d/mcp", *port)
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("listening on http://localhost%s%s", addr, basePath)
	log.Fatal(http.ListenAndServe(addr, outerMux))
}

func setupSingleWiki(mux *http.ServeMux, dir, basePath string) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, basePath+"/wiki/Home", http.StatusFound)
	})

	mux.HandleFunc("GET /wiki/{page}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("page")
		if !validName.MatchString(name) {
			http.NotFound(w, r)
			return
		}

		page, err := loadPage(dir, name, "", basePath)
		if err != nil {
			render404(w, dir, name, "", basePath)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pageTmpl.Execute(w, page)
	})
}

func setupMultiWiki(mux *http.ServeMux, rootDir, basePath string) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		wikis := listWikis(rootDir, basePath)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTmpl.Execute(w, IndexPage{BasePath: basePath, Wikis: wikis})
	})

	mux.HandleFunc("GET /{wiki}/{$}", func(w http.ResponseWriter, r *http.Request) {
		wikiName := r.PathValue("wiki")
		if !validName.MatchString(wikiName) {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, basePath+"/"+wikiName+"/wiki/Home", http.StatusFound)
	})

	mux.HandleFunc("GET /{wiki}/wiki/{page}", func(w http.ResponseWriter, r *http.Request) {
		wikiName := r.PathValue("wiki")
		pageName := r.PathValue("page")

		if !validName.MatchString(wikiName) || !validName.MatchString(pageName) {
			http.NotFound(w, r)
			return
		}

		wikiDir := filepath.Join(rootDir, wikiName)
		page, err := loadPage(wikiDir, pageName, wikiName, basePath)
		if err != nil {
			render404(w, wikiDir, pageName, wikiName, basePath)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		pageTmpl.Execute(w, page)
	})
}

func render404(w http.ResponseWriter, dir, name, wikiName, basePath string) {
	title := strings.ReplaceAll(name, "-", " ")

	linkPrefix := basePath
	homeURL := basePath + "/wiki/Home"
	if wikiName != "" {
		linkPrefix = basePath + "/" + wikiName
		homeURL = basePath + "/" + wikiName + "/wiki/Home"
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	pageTmpl.Execute(w, page)
}

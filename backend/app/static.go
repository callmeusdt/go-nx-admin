package app

import (
	_ "embed"
	"html"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go-nx-admin/internal/config"
)

const (
	defaultSiteTitle = "我的nxadmin"
	faviconBuiltin   = "/favicon.svg"
	faviconCustom    = "/_site/favicon"
)

//go:embed assets/favicon.svg
var builtinFavicon []byte

var (
	titleTag   = regexp.MustCompile(`(?is)<title[^>]*>.*?</title>`)
	iconLink   = regexp.MustCompile(`(?is)<link[^>]+rel=["'](?:shortcut )?icon["'][^>]*>`)
	headCloser = regexp.MustCompile(`(?i)</head>`)
)

func siteConfig() config.SiteConfig {
	if config.AppConfig != nil {
		return config.AppConfig.Site
	}
	return config.SiteConfig{}
}

func remoteFavicon(v string) bool {
	l := strings.ToLower(v)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "data:")
}

// renderIndex injects the configured title and favicon into the SPA shell.
func renderIndex(src []byte, site config.SiteConfig) []byte {
	title := strings.TrimSpace(site.Title)
	if title == "" {
		title = defaultSiteTitle
	}
	href := faviconBuiltin
	if f := strings.TrimSpace(site.Favicon); f != "" {
		href = faviconCustom
		if remoteFavicon(f) {
			href = f
		}
	}
	out := string(src)
	tag := "<title>" + html.EscapeString(title) + "</title>"
	link := `<link rel="icon" href="` + html.EscapeString(href) + `" />`
	out = iconLink.ReplaceAllString(out, "")
	if titleTag.MatchString(out) {
		out = titleTag.ReplaceAllLiteralString(out, tag)
	} else {
		link = tag + link
	}
	if headCloser.MatchString(out) {
		out = headCloser.ReplaceAllLiteralString(out, "  "+link+"\n  </head>")
	} else {
		out = link + out
	}
	return []byte(out)
}

func serveIndex(w http.ResponseWriter, r *http.Request, files fs.FS) {
	src, err := fs.ReadFile(files, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(renderIndex(src, siteConfig()))
	}
}

// serveFavicon answers the built-in icon and a locally configured one.
func serveFavicon(w http.ResponseWriter, r *http.Request, clean string, files fs.FS) bool {
	switch clean {
	case faviconBuiltin:
		if _, err := fs.Stat(files, "favicon.svg"); err == nil {
			return false
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		if r.Method != http.MethodHead {
			_, _ = w.Write(builtinFavicon)
		}
		return true
	case faviconCustom:
		f := strings.TrimSpace(siteConfig().Favicon)
		if f == "" || remoteFavicon(f) {
			http.NotFound(w, r)
			return true
		}
		file, err := os.Open(filepath.Clean(f))
		if err != nil {
			http.NotFound(w, r)
			return true
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return true
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, filepath.Base(f), info.ModTime().Truncate(time.Second), file)
		return true
	}
	return false
}

// StaticHandler serves a compiled SPA without rewriting API or asset misses to HTML.
// Only GET/HEAD navigation requests accepting HTML receive the index fallback.
// The index page carries the title and favicon from the site config.
func StaticHandler(files fs.FS) http.Handler {
	server := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clean := path.Clean("/" + r.URL.Path)
		if clean == "/api" || strings.HasPrefix(clean, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(`{"message":"not found"}`))
			}
			return
		}
		if serveFavicon(w, r, clean, files) {
			return
		}
		name := strings.TrimPrefix(clean, "/")
		if name == "" || name == "index.html" {
			serveIndex(w, r, files)
			return
		}
		info, err := fs.Stat(files, name)
		if err == nil && !info.IsDir() {
			server.ServeHTTP(w, r)
			return
		}
		if path.Ext(clean) != "" || !strings.Contains(r.Header.Get("Accept"), "text/html") || strings.HasPrefix(clean, "/assets/") {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, files)
	})
}

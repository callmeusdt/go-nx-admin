package app

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// StaticHandler serves a compiled SPA without rewriting API or asset misses to HTML.
// Only GET/HEAD navigation requests accepting HTML receive the index fallback.
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
		name := strings.TrimPrefix(clean, "/")
		if name == "" {
			name = "index.html"
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
		request := r.Clone(r.Context())
		request.URL.Path = "/"
		server.ServeHTTP(w, request)
	})
}

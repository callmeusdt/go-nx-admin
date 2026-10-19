package app

import (
	"go-nx-admin/internal/config"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStaticHandler(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte("<html>BIM</html>")}, "assets/app.js": {Data: []byte("export {}")}}
	h := StaticHandler(files)
	for _, tc := range []struct {
		method, path, accept string
		status               int
		contains             string
	}{
		{"GET", "/", "text/html", 200, "BIM"},
		{"GET", "/bim/users", "text/html", 200, "BIM"},
		{"GET", "/assets/app.js", "*/*", 200, "export"},
		{"GET", "/assets/missing.js", "text/html", 404, "404"},
		{"GET", "/api/v1/missing", "text/html", 404, "not found"},
		{"GET", "/config.yaml", "text/html", 404, "404"},
		{"GET", "/missing", "application/json", 404, "404"},
		{"POST", "/bim/users", "text/html", 405, "method"},
		{"HEAD", "/bim/users", "text/html", 200, ""},
	} {
		t.Run(tc.method+tc.path+tc.accept, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Accept", tc.accept)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD returned body")
			}
		})
	}
}

func TestStaticHandlerSiteBranding(t *testing.T) {
	prev := config.AppConfig
	defer func() { config.AppConfig = prev }()
	dir := t.TempDir()
	icon := filepath.Join(dir, "icon.png")
	if err := os.WriteFile(icon, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{"index.html": {Data: []byte(`<html><head><title>NX Admin</title><link rel="icon" href="/x.ico"></head><body></body></html>`)}}
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		StaticHandler(files).ServeHTTP(w, r)
		return w
	}

	config.AppConfig = &config.Config{}
	body := get("/deep/link").Body.String()
	if !strings.Contains(body, "<title>信息管理系统</title>") || !strings.Contains(body, `href="/favicon.svg"`) || strings.Contains(body, "x.ico") {
		t.Fatalf("fallback: %s", body)
	}
	if w := get("/favicon.svg"); w.Code != 200 || !strings.Contains(w.Body.String(), "<svg") {
		t.Fatalf("builtin icon: %d", w.Code)
	}

	config.AppConfig = &config.Config{Site: config.SiteConfig{Title: `A<b>&"`, Favicon: "https://cdn.example/i.png"}}
	body = get("/").Body.String()
	if !strings.Contains(body, "<title>A&lt;b&gt;&amp;&#34;</title>") || !strings.Contains(body, `href="https://cdn.example/i.png"`) {
		t.Fatalf("remote: %s", body)
	}

	config.AppConfig = &config.Config{Site: config.SiteConfig{Favicon: icon}}
	if body = get("/").Body.String(); !strings.Contains(body, `href="/_site/favicon"`) {
		t.Fatalf("local: %s", body)
	}
	if w := get("/_site/favicon"); w.Code != 200 || w.Body.String() != "PNGDATA" {
		t.Fatalf("local icon: %d %s", w.Code, w.Body.String())
	}
	config.AppConfig = &config.Config{}
	if w := get("/_site/favicon"); w.Code != 404 {
		t.Fatalf("unset local icon: %d", w.Code)
	}
}

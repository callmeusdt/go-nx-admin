package app

import (
	"net/http/httptest"
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

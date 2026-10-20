package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-nx-admin/internal/models"
	"go-nx-admin/internal/storage"
)

type mediaHarness struct {
	*bearerHarness
	token string
}

func newMediaHarness(t *testing.T, mutate func(*RuntimeOptions)) *mediaHarness {
	return newMediaHarnessSeeded(t, mutate, nil)
}

func newMediaHarnessSeeded(t *testing.T, mutate func(*RuntimeOptions), seed func(*RuntimeOptions)) *mediaHarness {
	t.Helper()
	b := newBearerHarnessSeeded(t, func(o *RuntimeOptions) {
		o.DisableMedia = false
		o.Config.Upload.MaxSize = 1 << 20
		o.Config.Upload.Dir = t.TempDir()
		if mutate != nil {
			mutate(o)
		}
	}, seed)
	access, _ := b.login()
	return &mediaHarness{bearerHarness: b, token: access}
}

func (h *mediaHarness) upload(name, content, mime string) (int, map[string]any) {
	h.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="` + name + `"`},
		"Content-Type":        {mime},
	})
	io.WriteString(part, content)
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/media/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+h.token)
	res, err := h.r.App.Test(req, 5000)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	var data map[string]any
	json.NewDecoder(res.Body).Decode(&data)
	return res.StatusCode, data
}

func (h *mediaHarness) list() []map[string]any {
	h.t.Helper()
	_, data := h.do("GET", "/api/v1/media", "", h.token)
	var out []map[string]any
	for _, v := range data["data"].([]any) {
		out = append(out, v.(map[string]any))
	}
	return out
}

func TestMediaLocalUploadListDelete(t *testing.T) {
	var dir string
	h := newMediaHarness(t, func(o *RuntimeOptions) { dir = o.Config.Upload.Dir })

	code, up := h.upload("pic.PNG", pngHeader+"png-bytes", "image/png")
	if code != 200 {
		t.Fatalf("upload=%d %v", code, up)
	}
	url := up["url"].(string)
	if !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("url=%q", url)
	}
	items := h.list()
	if len(items) != 1 || items[0]["storage"] != "local" || items[0]["type"] != "image" {
		t.Fatalf("list=%v", items)
	}
	// The object really is under the configured directory.
	key := strings.TrimPrefix(url, "/")
	if b, err := os.ReadFile(filepath.Join(dir, key)); err != nil || string(b) != pngHeader+"png-bytes" {
		t.Fatalf("file on disk: %q %v", b, err)
	}

	id := int(items[0]["id"].(float64))
	if c, _ := h.do("DELETE", "/api/v1/media/"+itoa(id), "", h.token); c != 200 {
		t.Fatalf("delete=%d", c)
	}
	if _, err := os.Stat(filepath.Join(dir, key)); !os.IsNotExist(err) {
		t.Fatal("file must be removed from storage on delete")
	}
	if len(h.list()) != 0 {
		t.Fatal("record must be removed")
	}
}

// pngHeader is a real PNG signature so content sniffing sees an image.
const pngHeader = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

// The URL returned by the API must actually be served, with hardening headers,
// and anything that is not a plain image/media type must download.
func TestUploadedFileIsServedAtItsReturnedURL(t *testing.T) {
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.PublicUploads = true })
	code, up := h.upload("pic.png", pngHeader, "image/png")
	if code != 200 {
		t.Fatalf("upload=%d", code)
	}
	res, err := h.r.App.Test(httptest.NewRequest("GET", up["url"].(string), nil), 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("returned URL not served: %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if res.Header.Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(res.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("missing hardening headers: %v", res.Header)
	}

	code, txt := h.upload("note.txt", "hello", "text/plain")
	if code != 200 {
		t.Fatalf("upload txt=%d", code)
	}
	res2, err := h.r.App.Test(httptest.NewRequest("GET", txt["url"].(string), nil), 5000)
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	if res2.Header.Get("Content-Disposition") != "attachment" {
		t.Fatalf("non-media file must download, got %q", res2.Header.Get("Content-Disposition"))
	}
}

func TestMediaRejectsActiveAndMismatchedContent(t *testing.T) {
	h := newMediaHarness(t, nil)
	cases := map[string]struct{ name, content, mime string }{
		"html":              {"x.html", "<html><script>alert(1)</script></html>", "text/html"},
		"svg":               {"x.svg", "<svg xmlns='http://www.w3.org/2000/svg'><script/></svg>", "image/svg+xml"},
		"html as txt":       {"x.txt", "<!DOCTYPE html><script>alert(1)</script>", "text/plain"},
		"html as png":       {"x.png", "<html><script>alert(1)</script></html>", "image/png"},
		"script as png":     {"x.png", "plain text pretending to be an image", "image/png"},
		"no extension":      {"readme", "hello", "text/plain"},
		"double ext":        {"x.png.html", pngHeader, "image/png"},
		"unlisted":          {"x.exe", "MZ", "application/octet-stream"},
		"upper-case active": {"X.HTML", "<html></html>", "text/html"},
	}
	for name, tc := range cases {
		if code, res := h.upload(tc.name, tc.content, tc.mime); code != 400 {
			t.Fatalf("%s: upload accepted (%d %v)", name, code, res)
		}
	}
	if len(h.list()) != 0 {
		t.Fatal("rejected uploads left records")
	}
	// The client-supplied MIME is not trusted: a real PNG labelled as text is
	// stored as an image.
	if code, _ := h.upload("ok.png", pngHeader, "text/plain"); code != 200 {
		t.Fatalf("valid png rejected: %d", code)
	}
	if items := h.list(); len(items) != 1 || items[0]["type"] != "image" || items[0]["mime_type"] != "image/png" {
		t.Fatalf("type/mime must come from the content: %v", items)
	}
}

func TestMediaConfiguredAllowListIsHonouredButNeverActive(t *testing.T) {
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Config.Upload.AllowTypes = []string{"txt", "html"} })
	if code, _ := h.upload("a.txt", "hello", "text/plain"); code != 200 {
		t.Fatalf("listed type rejected: %d", code)
	}
	if code, _ := h.upload("a.png", pngHeader, "image/png"); code != 400 {
		t.Fatalf("unlisted type accepted: %d", code)
	}
	if code, _ := h.upload("a.html", "<html></html>", "text/html"); code != 400 {
		t.Fatalf("active content accepted although listed: %d", code)
	}
}

func TestMediaRejectsOversizedUpload(t *testing.T) {
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Config.Upload.MaxSize = 4 })
	if code, _ := h.upload("big.txt", "too large", "text/plain"); code != 400 {
		t.Fatalf("oversize upload = %d", code)
	}
	if len(h.list()) != 0 {
		t.Fatal("rejected upload left a record")
	}
}

// Legacy rows predate the storage column; they must stay deletable. The old
// handler removed `media.Path` relative to the working directory, which never
// matched where files were written - the new one resolves it in the store.
func TestMediaLegacyRecordWithoutStorageName(t *testing.T) {
	var dir string
	h := newMediaHarnessSeeded(t, func(o *RuntimeOptions) { dir = o.Config.Upload.Dir }, func(o *RuntimeOptions) {
		key := "uploads/2025/01/01/legacy.txt"
		os.MkdirAll(filepath.Join(dir, filepath.Dir(key)), 0o755)
		os.WriteFile(filepath.Join(dir, key), []byte("old"), 0o644)
		o.DB.Create(&models.Media{OriginalName: "legacy.txt", FileName: "legacy.txt", Path: key, URL: "/" + key, Type: "file"})
	})
	items := h.list()
	if len(items) != 1 || items[0]["url"] != "/uploads/2025/01/01/legacy.txt" {
		t.Fatalf("legacy list = %v", items)
	}
	if c, _ := h.do("DELETE", "/api/v1/media/"+itoa(int(items[0]["id"].(float64))), "", h.token); c != 200 {
		t.Fatalf("delete legacy = %d", c)
	}
	if _, err := os.Stat(filepath.Join(dir, "uploads/2025/01/01/legacy.txt")); !os.IsNotExist(err) {
		t.Fatal("legacy file must actually be deleted")
	}
}

func TestMediaCustomStorageAndSwitchKeepsOldFilesReachable(t *testing.T) {
	var dir string
	mem := &memStorage{objs: map[string][]byte{}}
	h := newMediaHarnessSeeded(t, func(o *RuntimeOptions) {
		dir = o.Config.Upload.Dir
		// The deployment has switched to a custom driver...
		o.Storage = mem
	}, func(o *RuntimeOptions) {
		// ...but a file written while the local driver was active still exists.
		key := "uploads/2025/02/02/before.txt"
		os.MkdirAll(filepath.Join(dir, filepath.Dir(key)), 0o755)
		os.WriteFile(filepath.Join(dir, key), []byte("before"), 0o644)
		o.DB.Create(&models.Media{OriginalName: "before.txt", FileName: "before.txt", Path: key, URL: "/" + key, Storage: "local", Type: "file"})
	})

	code, up := h.upload("after.txt", "after", "text/plain")
	if code != 200 || len(mem.objs) != 1 {
		t.Fatalf("upload to custom storage: %d objs=%d", code, len(mem.objs))
	}
	if !strings.HasPrefix(up["url"].(string), "mem://") {
		t.Fatalf("url must come from the driver, got %v", up["url"])
	}

	// Both records are deletable, each through the driver that holds it.
	for _, it := range h.list() {
		if c, _ := h.do("DELETE", "/api/v1/media/"+itoa(int(it["id"].(float64))), "", h.token); c != 200 {
			t.Fatalf("delete %v = %d", it["storage"], c)
		}
	}
	if len(mem.objs) != 0 {
		t.Fatal("custom storage object not deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "uploads/2025/02/02/before.txt")); !os.IsNotExist(err) {
		t.Fatal("pre-switch local file not deleted")
	}
}

func TestMediaDeleteKeepsRecordWhenStorageFails(t *testing.T) {
	mem := &memStorage{objs: map[string][]byte{}}
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Storage = mem })
	h.upload("a.txt", "a", "text/plain")
	items := h.list()

	mem.failDelete = true
	if c, _ := h.do("DELETE", "/api/v1/media/"+itoa(int(items[0]["id"].(float64))), "", h.token); c != 500 {
		t.Fatalf("delete with failing storage = %d", c)
	}
	if len(h.list()) != 1 {
		t.Fatal("record was dropped although the file still exists - it would be orphaned")
	}
	mem.failDelete = false
	if c, _ := h.do("DELETE", "/api/v1/media/"+itoa(int(items[0]["id"].(float64))), "", h.token); c != 200 {
		t.Fatalf("retry = %d", c)
	}
}

func TestMediaUploadFailureLeavesNoRecord(t *testing.T) {
	mem := &memStorage{objs: map[string][]byte{}, failPut: true}
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Storage = mem })
	if code, _ := h.upload("a.txt", "a", "text/plain"); code != 500 {
		t.Fatalf("upload with failing storage = %d", code)
	}
	if len(h.list()) != 0 {
		t.Fatal("failed upload left a record")
	}
}

func TestRuntimeRestoresStorageOnClose(t *testing.T) {
	before := storage.Default()
	opts := runtimeOptions(t)
	opts.DisableMedia = false
	opts.Config.Upload.Dir = t.TempDir()
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	if err := opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`).Error; err != nil {
		t.Fatal(err)
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	if storage.Default() == before {
		t.Fatal("runtime did not install storage")
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if storage.Default() != before {
		t.Fatal("storage not restored on close")
	}
}

func TestRuntimeRejectsBadStorageConfig(t *testing.T) {
	opts := runtimeOptions(t)
	opts.DisableMedia = false
	opts.Config.Upload.Dir = t.TempDir()
	opts.Config.Upload.Driver = "s3" // no endpoint/bucket
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	before := storage.Default()
	if _, err := NewRuntime(opts); err == nil {
		t.Fatal("invalid storage config accepted")
	}
	if storage.Default() != before {
		t.Fatal("failed construction leaked storage")
	}
	// The one-runtime guard must be released so construction can be retried.
	opts.Config.Upload.Driver = ""
	if err := opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`).Error; err != nil {
		t.Fatal(err)
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatalf("retry after failed construction: %v", err)
	}
	r.Close(context.Background())
}

func TestMediaFailedRecordInsertRemovesUploadedObject(t *testing.T) {
	mem := &memStorage{objs: map[string][]byte{}}
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Storage = mem })
	// Dropping the table makes the insert fail after the object was written.
	if err := h.db.Migrator().DropTable(&models.Media{}); err != nil {
		t.Fatal(err)
	}
	if code, _ := h.upload("a.txt", "a", "text/plain"); code != 500 {
		t.Fatalf("upload with failing insert = %d", code)
	}
	if len(mem.objs) != 0 {
		t.Fatalf("object orphaned in storage after a failed insert: %v", mem.objs)
	}
}

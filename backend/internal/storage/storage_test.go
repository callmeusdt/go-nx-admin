package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-nx-admin/internal/config"
)

// contract is run against every driver so they stay interchangeable.
func contract(t *testing.T, s Storage) {
	t.Helper()
	ctx := context.Background()
	key := "uploads/2026/01/02/hello.txt"

	if _, err := s.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("open missing: want ErrNotFound, got %v", err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("deleting a missing object must succeed: %v", err)
	}

	body := []byte("hello storage")
	if err := s.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain"); err != nil {
		t.Fatal(err)
	}
	rc, err := s.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, body) {
		t.Fatalf("round trip: %q", got)
	}

	// Overwrite replaces the content.
	if err := s.Put(ctx, key, strings.NewReader("v2"), 2, "text/plain"); err != nil {
		t.Fatal(err)
	}
	rc, _ = s.Open(ctx, key)
	got, _ = io.ReadAll(rc)
	rc.Close()
	if string(got) != "v2" {
		t.Fatalf("overwrite: %q", got)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("open after delete: %v", err)
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("second delete must be idempotent: %v", err)
	}
}

func TestLocalContract(t *testing.T) {
	l, err := NewLocal(t.TempDir(), "/uploads")
	if err != nil {
		t.Fatal(err)
	}
	contract(t, l)
}

func TestLocalRejectsEscapingKeys(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside-secret.txt")
	os.WriteFile(outside, []byte("secret"), 0o644)
	defer os.Remove(outside)

	l, _ := NewLocal(root, "")
	ctx := context.Background()
	for _, key := range []string{"", "/", ".", "..", "../outside-secret.txt", "a/../../outside-secret.txt", "a\\..\\x", "a\x00b"} {
		if err := l.Put(ctx, key, strings.NewReader("x"), 1, ""); err == nil {
			t.Errorf("Put(%q) must fail", key)
		}
		if _, err := l.Open(ctx, key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("Open(%q) must be rejected as invalid, got %v", key, err)
		}
		if err := l.Delete(ctx, key); err == nil {
			t.Errorf("Delete(%q) must fail", key)
		}
	}
	if b, _ := os.ReadFile(outside); string(b) != "secret" {
		t.Fatal("a traversal key modified a file outside the root")
	}
}

func TestLocalNormalisesKeysInsideRoot(t *testing.T) {
	root := t.TempDir()
	l, _ := NewLocal(root, "")
	ctx := context.Background()
	// Leading and repeated slashes and "." segments normalise inside the root.
	if err := l.Put(ctx, "/a//./c.txt", strings.NewReader("x"), 1, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a", "c.txt")); err != nil {
		t.Fatalf("file not placed at normalised path: %v", err)
	}
}

func TestLocalPutLeavesNoTempFilesAndDoesNotExposePartialWrites(t *testing.T) {
	root := t.TempDir()
	l, _ := NewLocal(root, "")
	ctx := context.Background()

	boom := errors.New("read failed")
	err := l.Put(ctx, "dir/f.bin", io.MultiReader(strings.NewReader("partial"), &failReader{boom}), -1, "")
	if !errors.Is(err, boom) {
		t.Fatalf("want read error, got %v", err)
	}
	if _, err := l.Open(ctx, "dir/f.bin"); !errors.Is(err, ErrNotFound) {
		t.Fatal("a failed upload must not leave a visible file")
	}
	entries, _ := os.ReadDir(filepath.Join(root, "dir"))
	if len(entries) != 0 {
		t.Fatalf("temp file leaked: %v", entries)
	}
}

type failReader struct{ err error }

func (f *failReader) Read([]byte) (int, error) { return 0, f.err }

func TestLocalURL(t *testing.T) {
	ctx := context.Background()
	pub, _ := NewLocal(t.TempDir(), "/uploads/")
	if u, _ := pub.URL(ctx, "uploads/a.png", time.Minute); u != "/uploads/uploads/a.png" {
		t.Fatalf("public url = %q", u)
	}
	priv, _ := NewLocal(t.TempDir(), "")
	if u, err := priv.URL(ctx, "uploads/a.png", time.Minute); err != nil || u != "" {
		t.Fatalf("private storage must not expose a url: %q %v", u, err)
	}
	if _, err := pub.URL(ctx, "../x", time.Minute); err == nil {
		t.Fatal("invalid key must be rejected")
	}
}

func TestRegistryRoutesRecordsToTheirDriver(t *testing.T) {
	local, _ := NewLocal(t.TempDir(), "")
	remote := &fakeNamed{name: "s3"}
	reg := NewRegistry(remote, local)

	if reg.Active().Name() != "s3" {
		t.Fatal("active driver wrong")
	}
	if s, err := reg.For("local"); err != nil || s.Name() != "local" {
		t.Fatalf("local lookup: %v %v", s, err)
	}
	if s, err := reg.For(""); err != nil || s.Name() != "local" {
		t.Fatalf("legacy empty name must resolve to local: %v %v", s, err)
	}
	if _, err := reg.For("oss"); err == nil {
		t.Fatal("unconfigured driver must error")
	}
}

type fakeNamed struct {
	Storage
	name string
}

func (f *fakeNamed) Name() string { return f.name }

func TestNewFromConfig(t *testing.T) {
	dir := t.TempDir()
	reg, err := NewFromConfig(config.UploadConfig{Dir: dir}, true)
	if err != nil || reg.Active().Name() != "local" {
		t.Fatalf("default driver: %v %v", reg, err)
	}
	if _, err := NewFromConfig(config.UploadConfig{Dir: dir, Driver: "bogus"}, false); err == nil {
		t.Fatal("unknown driver must fail")
	}
	if _, err := NewFromConfig(config.UploadConfig{Dir: dir, Driver: "s3"}, false); err == nil {
		t.Fatal("s3 without endpoint/bucket must fail")
	}
	reg, err = NewFromConfig(config.UploadConfig{Dir: dir, Driver: "s3", S3: config.S3Config{Endpoint: "127.0.0.1:1", Bucket: "b"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if reg.Active().Name() != "s3" {
		t.Fatal("s3 should be active")
	}
	if s, err := reg.For("local"); err != nil || s.Name() != "local" {
		t.Fatal("local must stay reachable after switching to s3")
	}
}

func TestS3KeyPrefixAndValidation(t *testing.T) {
	s, err := NewS3(S3Options{Endpoint: "127.0.0.1:1", Bucket: "b", Prefix: "/tenant/", PublicBaseURL: "https://cdn.example.com/"})
	if err != nil {
		t.Fatal(err)
	}
	k, err := s.objectKey("uploads/a.png")
	if err != nil || k != "tenant/uploads/a.png" {
		t.Fatalf("prefixed key = %q %v", k, err)
	}
	for _, bad := range []string{"", "/", "..", "/../x", "a/../b", "a\\b", "a\x00"} {
		if _, err := s.objectKey(bad); err == nil {
			t.Errorf("objectKey(%q) must fail", bad)
		}
	}
	if u, _ := s.URL(context.Background(), "uploads/a.png", 0); u != "https://cdn.example.com/tenant/uploads/a.png" {
		t.Fatalf("public url = %q", u)
	}
}

// TestS3 runs the shared contract against a real S3-compatible server.
// Example: NX_TEST_S3_ENDPOINT=127.0.0.1:9000 NX_TEST_S3_ACCESS_KEY=... NX_TEST_S3_SECRET_KEY=... NX_TEST_S3_BUCKET=...
func TestS3(t *testing.T) {
	endpoint := os.Getenv("NX_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("NX_TEST_S3_ENDPOINT not set")
	}
	s, err := NewS3(S3Options{
		Endpoint: endpoint, Bucket: os.Getenv("NX_TEST_S3_BUCKET"),
		AccessKey: os.Getenv("NX_TEST_S3_ACCESS_KEY"), SecretKey: os.Getenv("NX_TEST_S3_SECRET_KEY"),
		PathStyle: true, Prefix: "nxtest",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract(t, s)

	ctx := context.Background()
	if err := s.Put(ctx, "u/p.txt", strings.NewReader("presign"), 7, "text/plain"); err != nil {
		t.Fatal(err)
	}
	defer s.Delete(ctx, "u/p.txt")
	u, err := s.URL(ctx, "u/p.txt", time.Minute)
	if err != nil || !strings.Contains(u, "X-Amz-Signature") {
		t.Fatalf("expected a presigned url, got %q %v", u, err)
	}
}

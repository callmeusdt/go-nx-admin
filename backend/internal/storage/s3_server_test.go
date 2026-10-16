package storage

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

// newFakeS3 starts an in-process S3 server so the driver is exercised over the
// real wire protocol (SigV4 signing, XML errors) without external services.
func newFakeS3(t *testing.T, prefix string) *S3 {
	t.Helper()
	backend := s3mem.New()
	if err := backend.CreateBucket("media"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(srv.Close)

	s, err := NewS3(S3Options{
		Endpoint:  strings.TrimPrefix(srv.URL, "http://"),
		Bucket:    "media",
		AccessKey: "test", SecretKey: "testsecret",
		PathStyle: true,
		Prefix:    prefix,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestS3ContractOverWire(t *testing.T) {
	contract(t, newFakeS3(t, ""))
}

func TestS3ContractWithPrefixOverWire(t *testing.T) {
	contract(t, newFakeS3(t, "tenant-a"))
}

func TestS3PrefixIsolatesTenants(t *testing.T) {
	a := newFakeS3(t, "a")
	ctx := context.Background()
	if err := a.Put(ctx, "f.txt", strings.NewReader("A"), 1, "text/plain"); err != nil {
		t.Fatal(err)
	}
	// Same bucket, different prefix: must not see the other tenant's object.
	b := *a
	b.opts.Prefix = "b"
	if _, err := b.Open(ctx, "f.txt"); err == nil {
		t.Fatal("prefix b must not read prefix a's object")
	}
}

func TestS3PresignedURL(t *testing.T) {
	s := newFakeS3(t, "")
	ctx := context.Background()
	if err := s.Put(ctx, "u/p.txt", strings.NewReader("presign"), 7, "text/plain"); err != nil {
		t.Fatal(err)
	}
	u, err := s.URL(ctx, "u/p.txt", time.Minute)
	if err != nil || !strings.Contains(u, "X-Amz-Signature") || !strings.Contains(u, "X-Amz-Expires=60") {
		t.Fatalf("expected a 60s presigned url, got %q %v", u, err)
	}
}

func TestS3UnknownBucketSurfacesError(t *testing.T) {
	s := newFakeS3(t, "")
	s.opts.Bucket = "missing"
	if err := s.Put(context.Background(), "x.txt", strings.NewReader("x"), 1, ""); err == nil {
		t.Fatal("writing to a missing bucket must fail")
	}
}

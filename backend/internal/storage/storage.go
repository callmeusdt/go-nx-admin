package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

var ErrNotFound = errors.New("storage: object not found")

// Storage abstracts where uploaded files live. Keys are slash-separated,
// relative and never begin with "/" or contain "..".
type Storage interface {
	// Name identifies the driver and is persisted with each record, so files
	// written by a previous driver stay reachable after the config changes.
	Name() string
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete must succeed when the object is already gone.
	Delete(ctx context.Context, key string) error
	// URL returns a browser-reachable address, or "" if the object is only
	// served through an authenticated route.
	URL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// cleanKey validates a key and returns its normalised form. Keys that try to
// climb out with ".." are rejected rather than silently folded into the root,
// because "../x" and "x" would otherwise alias the same object.
func cleanKey(key string) (string, error) {
	if key == "" || strings.Contains(key, "\\") || strings.ContainsRune(key, 0) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == ".." {
			return "", fmt.Errorf("storage: invalid key %q", key)
		}
	}
	clean := strings.TrimPrefix(path.Clean("/"+key), "/")
	if clean == "" {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return clean, nil
}

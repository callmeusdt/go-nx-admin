package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Local struct {
	root string
	// baseURL is the public prefix (e.g. "/uploads"); empty means the files are
	// not publicly served.
	baseURL string
}

func NewLocal(root, baseURL string) (*Local, error) {
	if root == "" {
		return nil, errors.New("storage: local root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	// The directory is created on first Put so that constructing a driver never
	// touches the filesystem.
	return &Local{root: abs, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (l *Local) Name() string { return "local" }

// resolve maps a key to a path inside root. cleanKey already rejects "..", the
// Rel check is a second line of defence against anything that still escapes.
func (l *Local) resolve(key string) (string, error) {
	clean, err := cleanKey(key)
	if err != nil {
		return "", err
	}
	full := filepath.Join(l.root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(l.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return full, nil
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	full, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	// Write to a temp file and rename so readers never observe a partial upload.
	tmp, err := os.CreateTemp(filepath.Dir(full), ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, full); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	full, err := l.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (l *Local) Delete(_ context.Context, key string) error {
	full, err := l.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) URL(_ context.Context, key string, _ time.Duration) (string, error) {
	if _, err := l.resolve(key); err != nil {
		return "", err
	}
	if l.baseURL == "" {
		return "", nil
	}
	clean, _ := cleanKey(key)
	return l.baseURL + "/" + clean, nil
}

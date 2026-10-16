package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"time"

	"go-nx-admin/internal/storage"
)

func itoa(n int) string { return strconv.Itoa(n) }

// memStorage is a minimal custom Storage driver, as an embedding project would write.
type memStorage struct {
	objs       map[string][]byte
	failPut    bool
	failDelete bool
}

func (m *memStorage) Name() string { return "mem" }
func (m *memStorage) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	if m.failPut {
		return errors.New("put failed")
	}
	b, _ := io.ReadAll(r)
	m.objs[key] = b
	return nil
}
func (m *memStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := m.objs[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (m *memStorage) Delete(_ context.Context, key string) error {
	if m.failDelete {
		return errors.New("delete failed")
	}
	delete(m.objs, key)
	return nil
}
func (m *memStorage) URL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "mem://" + key, nil
}

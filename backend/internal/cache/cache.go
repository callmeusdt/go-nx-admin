package cache

import (
	"context"
	"errors"
	"time"
)

var ErrMiss = errors.New("cache: miss")

// Cache is a small key/value contract shared by rate limiting, captcha and
// session lookups. Implementations must be safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	// GetDel atomically reads and removes a key, enabling one-shot tokens.
	GetDel(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, keys ...string) error
	// Incr increments an integer counter. ttl is applied only when the key is
	// created, so the counter keeps a fixed window.
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Close() error
}

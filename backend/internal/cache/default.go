package cache

import (
	"fmt"
	"sync"

	"go-nx-admin/internal/config"
)

var (
	mu      sync.RWMutex
	current Cache
)

// Default returns the process-wide cache, lazily creating an in-memory one so
// callers never see nil even before Init (tests, tools).
func Default() Cache {
	mu.RLock()
	c := current
	mu.RUnlock()
	if c != nil {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if current == nil {
		current = NewMemory()
	}
	return current
}

// Set installs a custom implementation and returns the previous one.
func Set(c Cache) Cache {
	mu.Lock()
	defer mu.Unlock()
	prev := current
	current = c
	return prev
}

// New builds a cache from configuration. The memory driver is the default.
func New(cfg config.CacheConfig) (Cache, error) {
	switch cfg.Driver {
	case "", "memory":
		return NewMemory(), nil
	case "redis":
		if cfg.Redis.Addr == "" {
			return nil, fmt.Errorf("cache: redis.addr is required")
		}
		return NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, cfg.Redis.Prefix)
	default:
		return nil, fmt.Errorf("cache: unknown driver %q", cfg.Driver)
	}
}

// Init builds the configured cache and installs it as the default.
func Init(cfg config.CacheConfig) error {
	c, err := New(cfg)
	if err != nil {
		return err
	}
	if prev := Set(c); prev != nil {
		prev.Close()
	}
	return nil
}

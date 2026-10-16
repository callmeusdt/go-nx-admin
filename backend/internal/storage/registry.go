package storage

import (
	"fmt"
	"sync"

	"go-nx-admin/internal/config"
)

// Registry holds the driver new uploads go to plus any drivers needed to reach
// files written earlier, looked up by the name stored on each record.
type Registry struct {
	mu      sync.RWMutex
	active  Storage
	drivers map[string]Storage
}

func NewRegistry(active Storage, others ...Storage) *Registry {
	r := &Registry{active: active, drivers: map[string]Storage{active.Name(): active}}
	for _, s := range others {
		if _, exists := r.drivers[s.Name()]; !exists {
			r.drivers[s.Name()] = s
		}
	}
	return r
}

// Active is where new uploads are written.
func (r *Registry) Active() Storage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

// For returns the driver that holds a record. Records created before drivers
// were tracked have an empty name and are local files.
func (r *Registry) For(name string) (Storage, error) {
	if name == "" {
		name = "local"
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.drivers[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("storage: driver %q is not configured", name)
}

// NewFromConfig builds the active driver. The local driver is always kept so
// files uploaded before a switch remain readable and deletable.
func NewFromConfig(cfg config.UploadConfig, publicLocal bool) (*Registry, error) {
	base := ""
	if publicLocal {
		base = "/uploads"
	}
	local, err := NewLocal(cfg.Dir, base)
	if err != nil {
		return nil, err
	}
	switch cfg.Driver {
	case "", "local":
		return NewRegistry(local), nil
	case "s3":
		s3, err := NewS3(S3Options{
			Endpoint: cfg.S3.Endpoint, Region: cfg.S3.Region, Bucket: cfg.S3.Bucket,
			AccessKey: cfg.S3.AccessKey, SecretKey: cfg.S3.SecretKey, UseSSL: cfg.S3.UseSSL,
			PathStyle: cfg.S3.PathStyle, Prefix: cfg.S3.Prefix, PublicBaseURL: cfg.S3.PublicBaseURL,
		})
		if err != nil {
			return nil, err
		}
		return NewRegistry(s3, local), nil
	default:
		return nil, fmt.Errorf("storage: unknown driver %q", cfg.Driver)
	}
}

var (
	defaultMu sync.RWMutex
	current   *Registry
)

// Default returns the process-wide registry, or nil before Init/Set.
func Default() *Registry {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return current
}

// Set installs a registry and returns the previous one.
func Set(r *Registry) *Registry {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	prev := current
	current = r
	return prev
}

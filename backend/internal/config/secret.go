package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DefaultJWTSecretFile is where a generated signing key is kept, relative to
// the working directory.
const DefaultJWTSecretFile = "data/jwt.secret"

// IsWeakJWTSecret reports whether v is a value that shipped in earlier
// templates and is therefore public.
func IsWeakJWTSecret(v string) bool { return knownWeakSecrets[v] }

// ResolveJWTSecret fills JWT.Secret when it is unset. An explicit value (config
// or NX_JWT_SECRET) always wins. Otherwise the key persisted in the secret
// file is used, and generated on first start so later restarts keep sessions
// valid. A published default is treated as unset, so upgrading a deployment
// that copied an old template keeps starting.
func (c *Config) ResolveJWTSecret() error {
	if c.JWT.Secret != "" && !IsWeakJWTSecret(c.JWT.Secret) {
		return nil
	}
	if IsWeakJWTSecret(c.JWT.Secret) {
		log.Printf("config: jwt.secret is a published default and is ignored; using the key in the secret file")
	}
	path := c.JWT.SecretFile
	if path == "" {
		path = DefaultJWTSecretFile
	}
	secret, err := LoadOrCreateSecretFile(path)
	if err != nil {
		return err
	}
	c.JWT.Secret = secret
	return nil
}

// LoadOrCreateSecretFile returns the signing key stored at path, creating it
// (mode 0600, 32 random bytes) when it does not exist. An existing file that
// holds an unusable value is an error and is never overwritten.
func LoadOrCreateSecretFile(path string) (string, error) {
	for attempt := 0; attempt < 2; attempt++ {
		data, err := os.ReadFile(path)
		if err == nil {
			s := strings.TrimSpace(string(data))
			if len(s) < MinJWTSecretLen || IsWeakJWTSecret(s) {
				return "", fmt.Errorf("config: %s holds an unusable secret (need >= %d bytes); fix or delete it to regenerate", path, MinJWTSecretLen)
			}
			return s, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("config: read %s: %w", path, err)
		}
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		secret := hex.EncodeToString(buf)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", fmt.Errorf("config: create %s: %w", filepath.Dir(path), err)
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue // another process won the race; read its key
		}
		if err != nil {
			return "", fmt.Errorf("config: create %s: %w", path, err)
		}
		_, werr := f.WriteString(secret + "\n")
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(path)
			return "", fmt.Errorf("config: write %s: %w", path, werr)
		}
		log.Printf("config: generated a JWT signing key at %s (keep it; deleting it signs everyone out)", path)
		return secret, nil
	}
	return "", fmt.Errorf("config: could not read or create %s", path)
}

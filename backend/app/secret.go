package app

import "go-nx-admin/internal/config"

// EnsureJWTSecret returns the JWT signing key stored at path, creating it
// (mode 0600, 32 random bytes) on first use. NewRuntime never writes files, so
// a project that wants the same "generate once, keep" behaviour as Run calls
// this before building its Config.
func EnsureJWTSecret(path string) (string, error) {
	if path == "" {
		path = config.DefaultJWTSecretFile
	}
	return config.LoadOrCreateSecretFile(path)
}

package app

import (
	"path/filepath"
	"testing"

	"go-nx-admin/internal/config"
)

func TestRequiredDirsCoverEverythingRunWrites(t *testing.T) {
	has := func(dirs []string, want string) bool {
		for _, d := range dirs {
			if filepath.Clean(d) == filepath.Clean(want) {
				return true
			}
		}
		return false
	}
	cfg := &config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: "state/db/app.db?_pragma=busy_timeout(5000)"},
		Upload: config.UploadConfig{Dir: "./files"}, JWT: config.JWTConfig{SecretFile: "keys/jwt.secret"}}
	dirs := RequiredDirs(cfg)
	for _, want := range []string{"data", "logs", "files", "state/db", "keys"} {
		if !has(dirs, want) {
			t.Fatalf("%q missing from %v", want, dirs)
		}
	}
	// A DSN with a "file:" prefix is still a path.
	cfg.Database.DSN = "file:other/x.db"
	if !has(RequiredDirs(cfg), "other") {
		t.Fatalf("file: DSN parent missing: %v", RequiredDirs(cfg))
	}
	// Server databases have no local directory to create.
	cfg.Database = config.DatabaseConfig{Driver: "postgres", DSN: "host=db user=x"}
	if has(RequiredDirs(cfg), "host=db user=x") {
		t.Fatal("non-sqlite DSN treated as a path")
	}
}

// The signing key's directory may not exist on a new deployment.
func TestJWTSecretCreatesMissingDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does", "not", "exist", "jwt.secret")
	first, err := EnsureJWTSecret(path)
	if err != nil {
		t.Fatalf("missing parent directory must be created: %v", err)
	}
	if second, _ := EnsureJWTSecret(path); second != first {
		t.Fatal("second call produced a different key")
	}
}

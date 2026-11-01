package app

import (
	"context"
	"io"
	"testing"

	"gorm.io/gorm"
)

// The pinned connection remains read-only while runtime construction loads
// Casbin policy. A hidden migration, seed, or API discovery write must fail.
func TestRuntimeMySQLReadOnlyStartup(t *testing.T) {
	db := bootstrapTarget(t, "mysql")
	if err := MigrateSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Connection(func(conn *gorm.DB) error {
		if err := conn.Exec("SET SESSION TRANSACTION READ ONLY").Error; err != nil {
			return err
		}
		defer conn.Exec("SET SESSION TRANSACTION READ WRITE")
		// Verify this connection really rejects writes before relying on it.
		if err := conn.Session(&gorm.Session{NewDB: true}).Exec("INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','probe','.*','.*')").Error; err == nil {
			t.Fatal("read-only session accepted a write")
		}
		var cfg Config
		cfg.Server.Host, cfg.Server.Port = "127.0.0.1", 19876
		cfg.JWT.Secret = "mysql-runtime-synthetic-test-key-123456"
		cfg.CORS.Origins = "https://admin.example.test"
		r, err := NewRuntime(RuntimeOptions{Config: cfg, DB: conn, LogOutput: io.Discard, DisableMedia: true})
		if err != nil {
			return err
		}
		if err := r.Close(context.Background()); err != nil {
			return err
		}
		var users, policies int64
		if err := conn.Table("admin_users").Count(&users).Error; err != nil {
			return err
		}
		if err := conn.Table("admin_casbin_rule").Count(&policies).Error; err != nil {
			return err
		}
		if users != 0 || policies != 0 {
			t.Fatalf("runtime wrote seed data: users=%d policies=%d", users, policies)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

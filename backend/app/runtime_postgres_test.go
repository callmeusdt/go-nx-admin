package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// A read-only transaction makes any hidden DDL, policy seed or API discovery
// fail in PostgreSQL, including operations that a row-count assertion misses.
func TestRuntimePostgresReadOnlyStartup(t *testing.T) {
	dsn := os.Getenv("NX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NX_TEST_POSTGRES_DSN not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	schema := fmt.Sprintf("bim_runtime_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateSchema(db); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("SET TRANSACTION READ ONLY").Error; err != nil {
		t.Fatal(err)
	}
	var cfg Config
	cfg.Server.Host, cfg.Server.Port = "127.0.0.1", 19876
	cfg.JWT.Secret = "postgres-runtime-synthetic-test-key-12345"
	cfg.CORS.Origins = "https://admin.example.test"
	r, err := NewRuntime(RuntimeOptions{Config: cfg, DB: tx, LogOutput: io.Discard, DisableMedia: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := tx.Table("admin_users").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("unexpected default administrator")
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
}

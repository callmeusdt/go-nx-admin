package app

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
)

func TestSQLiteDSNDefaultsRespectOperatorChoices(t *testing.T) {
	cases := map[string]string{
		"data/nx.db":                         "data/nx.db?_busy_timeout=5000&_journal_mode=WAL",
		"file:data/nx.db?cache=shared":       "file:data/nx.db?cache=shared&_busy_timeout=5000&_journal_mode=WAL",
		"data/nx.db?_busy_timeout=100":       "data/nx.db?_busy_timeout=100&_journal_mode=WAL",
		"data/nx.db?_journal_mode=DELETE":    "data/nx.db?_journal_mode=DELETE&_busy_timeout=5000",
		"data/nx.db?_pragma=busy_timeout(1)": "data/nx.db?_pragma=busy_timeout(1)",
		":memory:":                           ":memory:",
		"file::memory:?cache=shared":         "file::memory:?cache=shared",
	}
	for in, want := range cases {
		if got := sqliteDSN(in); got != want {
			t.Errorf("sqliteDSN(%q) = %q, want %q", in, got, want)
		}
	}
}

// Concurrent writes must queue on the busy timeout rather than fail with
// "database is locked".
func TestOpenDatabaseSQLiteHandlesConcurrentWrites(t *testing.T) {
	cfg := &config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "pool.db")}}
	db, err := OpenDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if got := pool.Stats().MaxOpenConnections; got < 2 || got > 16 {
		t.Fatalf("sqlite MaxOpenConnections = %d, want a small pool (a single connection deadlocks libraries that nest handles)", got)
	}
	var mode string
	db.Raw("PRAGMA journal_mode").Scan(&mode)
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- db.Create(&models.LoginLog{Username: "u", Status: "success"}).Error
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent write failed: %v", err)
		}
	}
}

// Startup runs Casbin's SavePolicy, which nests a transaction inside the outer
// handle. It must complete on the pool OpenDatabase builds (a single SQLite
// connection would hang here), so the whole startup sequence is exercised.
func TestStartupSequenceCompletesOnSQLitePool(t *testing.T) {
	cfg := &config.Config{Database: config.DatabaseConfig{Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "start.db")}}
	db, err := OpenDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	done := make(chan error, 1)
	go func() {
		if err := models.AutoMigrate(db); err != nil {
			done <- err
			return
		}
		if err := models.Seed(db); err != nil {
			done <- err
			return
		}
		_, err := middleware.InitCasbin(db)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("startup sequence hung (pool too small for nested handles?)")
	}
}

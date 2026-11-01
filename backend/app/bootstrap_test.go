package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// bootstrapTarget opens an isolated, empty database per dialect. PostgreSQL and
// MySQL/MariaDB run only when their DSN is configured; SQLite always runs.
func bootstrapTarget(t *testing.T, dialect string) *gorm.DB {
	t.Helper()
	switch dialect {
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bootstrap.db")+"?_busy_timeout=5000"), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		p, _ := db.DB()
		t.Cleanup(func() { p.Close() })
		return db
	case "mysql":
		dsn := os.Getenv("NX_TEST_MYSQL_DSN")
		if dsn == "" {
			t.Skip("NX_TEST_MYSQL_DSN required")
		}
		cfg, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			t.Fatal(err)
		}
		admin, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("nx_bootstrap_%d", time.Now().UnixNano())
		if err := admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4").Error; err != nil {
			t.Fatal(err)
		}
		ap, _ := admin.DB()
		t.Cleanup(func() { admin.Exec("DROP DATABASE " + name); ap.Close() })
		cfg.DBName = name
		db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		p, _ := db.DB()
		t.Cleanup(func() { p.Close() })
		return db
	default:
		dsn := os.Getenv("NX_TEST_DATABASE_DSN")
		if dsn == "" {
			t.Skip("NX_TEST_DATABASE_DSN required")
		}
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		pool, _ := db.DB()
		schema := fmt.Sprintf("nx_bootstrap_%d", time.Now().UnixNano())
		if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Exec("DROP SCHEMA " + schema + " CASCADE"); pool.Close() })
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := u.Query()
		query.Set("search_path", schema)
		u.RawQuery = query.Encode()
		target, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		tp, _ := target.DB()
		t.Cleanup(func() { tp.Close() })
		return target
	}
}

func TestBootstrapAdminConcurrentAndNonDestructive(t *testing.T) {
	for _, dialect := range []string{"postgres", "mysql", "sqlite"} {
		t.Run(dialect, func(t *testing.T) { testBootstrapAdmin(t, bootstrapTarget(t, dialect)) })
	}
}

func testBootstrapAdmin(t *testing.T, target *gorm.DB) {
	t.Helper()
	if err := MigrateSchema(target); err != nil {
		t.Fatal(err)
	}
	if err := BootstrapAdmin(target, "bad user", "bootstrap-test-password"); err == nil {
		t.Fatal("invalid account accepted")
	}
	var group sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			results <- BootstrapAdmin(target, "operator", "bootstrap-test-password")
		}()
	}
	group.Wait()
	close(results)
	succeeded, rejected := 0, 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrAdminAlreadyInitialized) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", succeeded, rejected)
	}
	var users []models.User
	if err := target.Find(&users).Error; err != nil || len(users) != 1 {
		t.Fatalf("unexpected users: %d error=%v", len(users), err)
	}
	if users[0].Username != "operator" || users[0].MFAEnabled || bcrypt.CompareHashAndPassword([]byte(users[0].Password), []byte("bootstrap-test-password")) != nil {
		t.Fatal("bootstrap account/password/MFA incorrect")
	}
	if !errors.Is(BootstrapAdmin(target, "replacement", "different-test-password"), ErrAdminAlreadyInitialized) {
		t.Fatal("repeat bootstrap was not rejected")
	}
	var current models.User
	target.First(&current, users[0].ID)
	if current.Password != users[0].Password || current.Username != users[0].Username {
		t.Fatal("repeat bootstrap changed credentials")
	}
}

func TestBootstrapAdminMySQLRejectsNonEmptyAndRollsBack(t *testing.T) {
	target := bootstrapTarget(t, "mysql")
	if err := MigrateSchema(target); err != nil {
		t.Fatal(err)
	}
	if err := target.Exec("INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','existing','.*','.*')").Error; err != nil {
		t.Fatal(err)
	}
	if !errors.Is(BootstrapAdmin(target, "operator", "bootstrap-test-password"), ErrAdminAlreadyInitialized) {
		t.Fatal("nonempty policy table did not block bootstrap")
	}
	if err := target.Exec("DELETE FROM admin_casbin_rule").Error; err != nil {
		t.Fatal(err)
	}
	if err := target.Exec("CREATE TRIGGER nx_u02_reject_policy BEFORE INSERT ON admin_casbin_rule FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'u02 injected failure'").Error; err != nil {
		t.Fatal(err)
	}
	if err := BootstrapAdmin(target, "operator", "bootstrap-test-password"); err == nil || !strings.Contains(err.Error(), "u02 injected failure") {
		t.Fatalf("injected policy failure was not returned: %v", err)
	}
	for _, table := range bootstrapTables {
		var count int64
		if err := target.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("failed bootstrap left data in %s: count=%d err=%v", table, count, err)
		}
	}
	if err := target.Exec("DROP TRIGGER nx_u02_reject_policy").Error; err != nil {
		t.Fatal(err)
	}
	if err := BootstrapAdmin(target, "operator", "bootstrap-test-password"); err != nil {
		t.Fatalf("retry after rollback failed: %v", err)
	}
}

func TestBootstrapAdminMySQLLockTimeoutAndDisconnect(t *testing.T) {
	target := bootstrapTarget(t, "mysql")
	if err := MigrateSchema(target); err != nil {
		t.Fatal(err)
	}
	lockPool, err := sql.Open("mysql", os.Getenv("NX_TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	defer lockPool.Close()
	lockConn, err := lockPool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lockConn.Close()
	var locked int
	if err := lockConn.QueryRowContext(context.Background(), "SELECT GET_LOCK('nx_bootstrap_admin', 0)").Scan(&locked); err != nil || locked != 1 {
		t.Fatalf("could not hold bootstrap lock: locked=%d err=%v", locked, err)
	}
	if err := BootstrapAdmin(target, "operator", "bootstrap-test-password"); err == nil || !strings.Contains(err.Error(), "bootstrap lock timeout") {
		t.Fatalf("expected lock timeout, got %v", err)
	}
	// Closing the owning pool and connection disconnects the lock holder; no
	// server-wide KILL privilege is required of the test account.
	if err := lockPool.Close(); err != nil {
		t.Fatal(err)
	}
	if err := lockConn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := BootstrapAdmin(target, "operator", "bootstrap-test-password"); err != nil {
		t.Fatalf("bootstrap after lock holder disconnect failed: %v", err)
	}
}

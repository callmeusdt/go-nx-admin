package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestBootstrapAdminConcurrentAndNonDestructive(t *testing.T) {
	dsn := os.Getenv("NX_TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("NX_TEST_DATABASE_DSN required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	schema := fmt.Sprintf("nx_bootstrap_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
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
	defer tp.Close()
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

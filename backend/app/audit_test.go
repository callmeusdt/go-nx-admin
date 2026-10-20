package app

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"gorm.io/gorm"
)

func TestAuditPersistenceCompletesBeforeResponse(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql"} {
		t.Run(dialect, func(t *testing.T) {
			db := bootstrapTarget(t, dialect)
			if err := MigrateSchema(db); err != nil {
				t.Fatal(err)
			}
			enforcer, err := middleware.LoadCasbin(db)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := enforcer.AddPolicy("audit-fixture", "^/api/v1/audit-probe$", "^POST$"); err != nil {
				t.Fatal(err)
			}
			entered, release, written := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			if err := db.Callback().Create().Before("gorm:create").Register("audit-test-block", func(tx *gorm.DB) {
				if tx.Statement.Table != "admin_audit_logs" {
					return
				}
				close(entered)
				select {
				case <-release:
				case <-tx.Statement.Context.Done():
					tx.AddError(tx.Statement.Context.Err())
				}
			}); err != nil {
				t.Fatal(err)
			}
			if err := db.Callback().Create().After("gorm:create").Register("audit-test-written", func(tx *gorm.DB) {
				if tx.Statement.Table == "admin_audit_logs" {
					close(written)
				}
			}); err != nil {
				t.Fatal(err)
			}
			web := fiber.New()
			web.Use(func(c *fiber.Ctx) error {
				c.Locals("role", "audit-fixture")
				c.Locals("username", "fixture-operator")
				return c.Next()
			})
			web.Use(middleware.CasbinMiddleware(enforcer, db))
			web.Post("/api/v1/audit-probe", func(c *fiber.Ctx) error { return c.SendStatus(204) })
			done := make(chan error, 1)
			go func() {
				response, err := web.Test(httptest.NewRequest("POST", "/api/v1/audit-probe", nil), -1)
				if response != nil {
					response.Body.Close()
				}
				done <- err
			}()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("audit insert not reached")
			}
			returned := false
			select {
			case err := <-done:
				returned = true
				if err != nil {
					t.Error(err)
				}
				t.Error("request returned while its audit write was still blocked")
			case <-time.After(100 * time.Millisecond):
			}
			unblock()
			if !returned {
				select {
				case err := <-done:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("request did not finish")
				}
			}
			select {
			case <-written:
			case <-time.After(3 * time.Second):
				t.Fatal("audit insert did not finish")
			}
			var count int64
			if err := db.Model(&models.AuditLog{}).Where("operator=? AND path=?", "fixture-operator", "/api/v1/audit-probe").Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("audit rows=%d err=%v", count, err)
			}
		})
	}
}

package app

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/handlers"
	"go-nx-admin/internal/models"
)

// Every statement the base issues by hand must work on all three databases.
// SQLite always runs; MySQL/MariaDB and PostgreSQL run when their DSN is set.
func TestSQLDialectCompatibility(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) { testSQLDialect(t, bootstrapTarget(t, dialect)) })
	}
}

func testSQLDialect(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := MigrateSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := models.Seed(db); err != nil {
		t.Fatal(err)
	}
	// A second seed takes the "already seeded" path, which re-reads configs.
	if err := models.Seed(db); err != nil {
		t.Fatal(err)
	}
	var seeded int64
	db.Model(&models.SystemConfig{}).Count(&seeded)
	if seeded != 4 {
		t.Fatalf("system config seeds = %d, want 4 (repeat seed must not duplicate or lose rows)", seeded)
	}

	api := fiber.New()
	api.Get("/api/v1/apis", handlers.ListApis(db))
	api.Get("/api/v1/system-configs", handlers.ListSystemConfigs(db))
	api.Put("/api/v1/system-configs", handlers.UpdateSystemConfigs(db))
	api.Get("/api/v1/login-logs", handlers.ListLoginLogs(db))
	api.Get("/api/v1/audit-logs", handlers.ListAuditLogs(db))
	api.Get("/api/v1/users", handlers.ListUsers(db))
	var systemRole, limitedRole models.Role
	db.Where(&models.Role{Slug: "admin"}).First(&systemRole)
	limitedRole = models.Role{Name: "Limited", Slug: "limited"}
	db.Create(&limitedRole)
	var firstMenu models.Menu
	db.Order("id asc").First(&firstMenu)
	db.Create(&models.RoleMenu{RoleID: limitedRole.ID, MenuID: firstMenu.ID})
	asRole := func(id uint) fiber.Handler {
		return func(c *fiber.Ctx) error { c.Locals("role_id", id); return c.Next() }
	}
	api.Get("/api/v1/menus/system", asRole(systemRole.ID), handlers.MyMenus(db))
	api.Get("/api/v1/menus/limited", asRole(limitedRole.ID), handlers.MyMenus(db))

	// A stale endpoint registered earlier must be pruned, live ones kept.
	// An endpoint discovered earlier is pruned; one an operator added by hand
	// is not, even though no route backs it.
	db.Create(&models.Api{Group: "old", Name: "gone", Path: "/api/v1/gone", Method: "GET", Source: models.ApiSourceAuto})
	db.Create(&models.Api{Group: "old", Name: "manual", Path: "/api/v1/manual", Method: "GET"})
	models.AutoDiscoverAPIs(api, db)
	var gone, manual int64
	db.Model(&models.Api{}).Where("path = ?", "/api/v1/gone").Count(&gone)
	db.Model(&models.Api{}).Where("path = ?", "/api/v1/manual").Count(&manual)
	if gone != 0 {
		t.Fatal("stale API row was not removed by auto-discovery")
	}
	if manual != 1 {
		t.Fatal("a hand-made API row was removed by auto-discovery")
	}
	var kept int64
	db.Model(&models.Api{}).Where("path = ? AND method = ?", "/api/v1/apis", "GET").Count(&kept)
	if kept != 1 {
		t.Fatalf("live API row count = %d, want 1", kept)
	}

	call := func(method, target, body string) (int, string) {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := api.Test(req, 10000)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}

	for target, wantMin := range map[string]int{"/api/v1/menus/system": 2, "/api/v1/menus/limited": 1} {
		code, body := call("GET", target, "")
		var out struct {
			Data []json.RawMessage `json:"data"`
		}
		if code != 200 || json.Unmarshal([]byte(body), &out) != nil || len(out.Data) < wantMin {
			t.Fatalf("%s: %d %s", target, code, body)
		}
	}
	if _, body := call("GET", "/api/v1/menus/limited", ""); strings.Count(body, `"permission"`) > 3 {
		t.Fatalf("a role with one granted menu must not see the whole tree: %s", body)
	}

	if code, body := call("GET", "/api/v1/apis", ""); code != 200 || !strings.Contains(body, "/api/v1/apis") {
		t.Fatalf("list apis: %d %s", code, body)
	}

	if code, body := call("PUT", "/api/v1/system-configs", `{"site_name":"Renamed"}`); code != 200 {
		t.Fatalf("update configs: %d %s", code, body)
	}
	var site models.SystemConfig
	if err := db.Where(&models.SystemConfig{Key: "site_name"}).First(&site).Error; err != nil || site.Value != "Renamed" {
		t.Fatalf("system config was not updated: %q err=%v", site.Value, err)
	}
	if code, body := call("PUT", "/api/v1/system-configs", `{"no_such_key":"x"}`); code != 400 {
		t.Fatalf("unknown config key must be reported, got %d %s", code, body)
	}

	db.Create(&models.LoginLog{Username: "Alice", IP: "10.0.0.1", Status: "success", CreatedAt: time.Now()})
	db.Create(&models.LoginLog{Username: "bob", IP: "10.0.0.2", Status: "success", CreatedAt: time.Now()})
	db.Create(&models.AuditLog{Operator: "Alice", Path: "/api/v1/Users", Method: "POST", IP: "10.0.0.1", CreatedAt: time.Now()})
	for _, tc := range []struct {
		target string
		want   int
	}{
		{"/api/v1/login-logs?username=alice", 1}, // case-insensitive on every database
		{"/api/v1/login-logs?username=ALI", 1},
		{"/api/v1/login-logs?ip=10.0.0.", 2},
		{"/api/v1/login-logs?ip=10_0", 0},      // "_" is a literal, not a wildcard
		{"/api/v1/login-logs?username=%25", 0}, // "%" is a literal, not a wildcard
		{"/api/v1/audit-logs?path=/api/v1/users", 1},
		{"/api/v1/audit-logs?operator=ALICE", 1},
		{"/api/v1/users?username=ADM", 1},
	} {
		code, body := call("GET", tc.target, "")
		if code != 200 {
			t.Fatalf("%s: %d %s", tc.target, code, body)
		}
		var out struct {
			Data  []json.RawMessage `json:"data"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal([]byte(body), &out); err != nil {
			t.Fatalf("%s: %v (%s)", tc.target, err, body)
		}
		if len(out.Data) != tc.want {
			t.Fatalf("%s: got %d rows, want %d (%s)", tc.target, len(out.Data), tc.want, body)
		}
	}
}

// The helper exported to downstream projects must behave identically on every
// database, against a table the base knows nothing about.
type dialectNote struct {
	ID    uint `gorm:"primaryKey"`
	Title string
}

func TestContainsFoldIsPortableForDownstreamModels(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := bootstrapTarget(t, dialect)
			if err := db.AutoMigrate(&dialectNote{}); err != nil {
				t.Fatal(err)
			}
			for _, title := range []string{"Quarterly Report", "100% done", "a_b", "axb", "O'Brien"} {
				db.Create(&dialectNote{Title: title})
			}
			count := func(term string) int {
				var n int64
				if err := ContainsFold(db.Model(&dialectNote{}), "title", term).Count(&n).Error; err != nil {
					t.Fatalf("%q: %v", term, err)
				}
				return int(n)
			}
			for term, want := range map[string]int{
				"REPORT": 1, "report": 1, "quart": 1,
				"%":   1, // only the row that really contains a percent sign
				"_":   1, // "a_b" but not "axb"
				"a_b": 1, "o'brien": 1, "!": 0, "": 5,
			} {
				if got := count(term); got != want {
					t.Errorf("ContainsFold(%q) matched %d rows, want %d", term, got, want)
				}
			}
		})
	}
}

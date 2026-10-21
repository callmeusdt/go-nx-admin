package app

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/models"
)

// leakMarkers are fragments of driver and SQL text that must never reach a
// client. Any of them in a response body means a raw database error was echoed.
var leakMarkers = []string{
	"sqlite", "SQLSTATE", "constraint", "UNIQUE", "Duplicate entry", "duplicate key",
	"no such table", "no such column", "syntax error", "admin_users", "admin_roles",
	"admin_menus", "admin_apis", "admin_", "FOREIGN KEY", "SELECT ", "INSERT ", "UPDATE ",
}

func assertNoLeak(t *testing.T, what string, code int, res map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(res)
	for _, m := range leakMarkers {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(m)) {
			t.Errorf("%s (status %d) leaked %q: %s", what, code, m, raw)
			return
		}
	}
}

// Duplicate keys are an expected user error: the client gets a stable 409 with
// a short message, not the driver's constraint text.
func TestDuplicateKeysAreReportedWithoutDriverText(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var roleID uint
	h.db.Raw("SELECT id FROM admin_roles LIMIT 1").Scan(&roleID)

	create := body(map[string]any{"username": "dupe-user", "password": "a-good-password1", "role_id": roleID})
	if c, res := h.do("POST", "/api/v1/users", create, access); c != 200 {
		t.Fatalf("first create = %d %v", c, res)
	}
	c, res := h.do("POST", "/api/v1/users", create, access)
	if c != 409 {
		t.Fatalf("duplicate username = %d %v, want 409", c, res)
	}
	assertNoLeak(t, "duplicate user", c, res)

	role := body(map[string]any{"name": "Dupe", "slug": "dupe-role"})
	h.do("POST", "/api/v1/roles", role, access)
	c, res = h.do("POST", "/api/v1/roles", role, access)
	if c != 409 {
		t.Fatalf("duplicate role slug = %d %v, want 409", c, res)
	}
	assertNoLeak(t, "duplicate role", c, res)
}

// A broken database must produce a generic 500 on every list and write path.
func TestDatabaseFailuresDoNotLeakThroughHandlers(t *testing.T) {
	// Media routes only exist when media is enabled; without it they would
	// answer 404 and the test would not reach the handlers it claims to cover.
	m := newMediaHarness(t, nil)
	h, access := m.bearerHarness, m.token
	var roleID uint
	h.db.Raw("SELECT id FROM admin_roles LIMIT 1").Scan(&roleID)
	for _, table := range []string{"admin_apis", "admin_menus", "admin_login_logs", "admin_audit_logs", "system_configs", "admin_user_ip_whitelists", "admin_role_menus", "media"} {
		if err := h.db.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct{ method, path, payload string }{
		{"GET", "/api/v1/apis", ""},
		{"GET", "/api/v1/menus", ""},
		{"GET", "/api/v1/login-logs", ""},
		{"GET", "/api/v1/audit-logs", ""},
		{"GET", "/api/v1/system-configs", ""},
		{"PUT", "/api/v1/system-configs", `{"site_name":"x"}`},
		{"GET", "/api/v1/roles/" + itoa(int(roleID)) + "/menus", ""},
		{"PUT", "/api/v1/roles/" + itoa(int(roleID)) + "/menus", `{"menu_ids":[1]}`},
		{"POST", "/api/v1/apis", `{"group":"g","name":"n","path":"/p","method":"GET"}`},
		{"POST", "/api/v1/menus", `{"name":"m"}`},
		{"DELETE", "/api/v1/menus/1", ""},
		{"DELETE", "/api/v1/apis", `{"ids":[1]}`},
		{"DELETE", "/api/v1/media", `{"ids":[1]}`},
		{"GET", "/api/v1/auth/menus", ""},
		{"GET", "/api/v1/media", ""},
		{"GET", "/api/v1/dashboard/stats", ""},
	}
	for _, tc := range cases {
		code, res := h.do(tc.method, tc.path, tc.payload, access)
		if code == 401 || code == 403 || code == 404 {
			t.Fatalf("%s %s: request never reached the handler (%d %v)", tc.method, tc.path, code, res)
		}
		if code < 400 {
			t.Fatalf("%s %s: expected a failure with the table gone, got %d %v", tc.method, tc.path, code, res)
		}
		assertNoLeak(t, tc.method+" "+tc.path, code, res)
	}
}

// Duplicate-key reporting must hold on every database, not only SQLite: each
// driver words the violation differently and none may reach the client.
func TestDuplicateKeyIsStableAcrossDatabases(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			h := newBearerHarnessSeeded(t, func(o *RuntimeOptions) { o.DB = bootstrapTarget(t, dialect) }, nil)
			access, _ := h.login()
			role := body(map[string]any{"name": "Dupe", "slug": "dupe-" + dialect})
			if c, res := h.do("POST", "/api/v1/roles", role, access); c != 200 {
				t.Fatalf("first create = %d %v", c, res)
			}
			c, res := h.do("POST", "/api/v1/roles", role, access)
			if c != 409 {
				t.Fatalf("duplicate slug = %d %v, want 409", c, res)
			}
			assertNoLeak(t, "duplicate role on "+dialect, c, res)
		})
	}
}

// The client gets a generic 500, but the operator must still be able to find
// the cause: it is written to the log.
func TestInternalErrorCauseIsLogged(t *testing.T) {
	var logged strings.Builder
	prev := log.Writer()
	log.SetOutput(&logged)
	defer log.SetOutput(prev)

	h := newBearerHarness(t, nil)
	access, _ := h.login()
	if err := h.db.Exec("DROP TABLE admin_apis").Error; err != nil {
		t.Fatal(err)
	}
	c, res := h.do("GET", "/api/v1/apis", "", access)
	if c != 500 || res["message"] != "internal error" {
		t.Fatalf("got %d %v", c, res)
	}
	if !strings.Contains(logged.String(), "admin_apis") {
		t.Fatalf("cause missing from the log: %q", logged.String())
	}
}

// A failing handler must leave a correct audit row on every database: the
// status the client saw, and an error text that fits the column (MySQL and
// PostgreSQL reject the whole row when it does not).
func TestAuditRowSurvivesLongErrorsOnEveryDatabase(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			h := newBearerHarnessSeeded(t, func(o *RuntimeOptions) {
				o.DB = bootstrapTarget(t, dialect)
				o.Extensions.ExtraRoutes = func(api fiber.Router, _ *gorm.DB, _ *casbin.Enforcer) {
					api.Get("/zz-long", func(c *fiber.Ctx) error { return errors.New(strings.Repeat("错x", 400)) })
					api.Get("/zz-teapot", func(c *fiber.Ctx) error { return fiber.NewError(418, "short") })
				}
			}, nil)
			access, _ := h.login()
			for path, want := range map[string]int{"/api/v1/zz-long": 500, "/api/v1/zz-teapot": 418} {
				if c, _ := h.do("GET", path, "", access); c != want {
					t.Fatalf("%s response = %d, want %d", path, c, want)
				}
				var rec models.AuditLog
				for i := 0; i < 50 && rec.ID == 0; i++ {
					// Find, not First: an absent row is expected while the async writer runs.
					h.db.Where("path = ?", path).Limit(1).Find(&rec)
					if rec.ID == 0 {
						time.Sleep(50 * time.Millisecond)
					}
				}
				if rec.ID == 0 {
					t.Fatalf("%s: audit row was lost", path)
				}
				if rec.Status != want {
					t.Errorf("%s: audit status %d, want %d", path, rec.Status, want)
				}
				if len(rec.Error) > 255 || !utf8.ValidString(rec.Error) {
					t.Errorf("%s: stored error is not bounded valid text (len %d)", path, len(rec.Error))
				}
			}
		})
	}
}

func TestMediaListClampsPagination(t *testing.T) {
	h := newMediaHarness(t, nil)
	for i := 0; i < 3; i++ {
		h.db.Create(&models.Media{OriginalName: "f.txt", FileName: "f.txt", Path: "uploads/f" + itoa(i), URL: "/uploads/f", Storage: "local", Type: "file"})
	}
	for _, q := range []string{"page_size=-5", "page_size=0", "page_size=999999", "page=-3", "page=0&page_size=abc", "page=99999999999999999999"} {
		c, res := h.do("GET", "/api/v1/media?"+q, "", h.token)
		if c != 200 {
			t.Fatalf("%s: %d %v", q, c, res)
		}
		size, _ := res["page_size"].(float64)
		page, _ := res["page"].(float64)
		if size < 1 || size > 200 || page < 1 {
			t.Errorf("%s: page=%v page_size=%v escaped the limits", q, page, size)
		}
	}
	_, res := h.do("GET", "/api/v1/media?page_size=2&page=2", "", h.token)
	if items, _ := res["data"].([]any); len(items) != 1 || res["total"].(float64) != 3 {
		t.Fatalf("page 2 of 3 rows with size 2 = %v", res)
	}
}

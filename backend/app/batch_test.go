package app

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func ids(v any) []int {
	out := []int{}
	for _, x := range v.([]any) {
		out = append(out, int(x.(float64)))
	}
	return out
}

func reasons(v any) map[int]string {
	out := map[int]string{}
	for _, x := range v.([]any) {
		m := x.(map[string]any)
		out[int(m["id"].(float64))] = m["reason"].(string)
	}
	return out
}

func body(v any) string { b, _ := json.Marshal(v); return string(b) }

func makeUsers(t *testing.T, db *gorm.DB, roleID uint, names ...string) []uint {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("x-password-1"), bcrypt.MinCost)
	var out []uint
	for _, n := range names {
		u := models.User{Username: n, Password: string(hash), RoleID: roleID, Status: 1}
		if err := db.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		out = append(out, u.ID)
	}
	return out
}

func TestBatchDeleteUsersProtectsSelfAndReportsPerID(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)
	others := makeUsers(t, h.db, me.RoleID, "u1", "u2")

	code, res := h.do("DELETE", "/api/v1/users", body(map[string]any{"ids": []uint{others[0], me.ID, others[1], 99999, others[0]}}), access)
	if code != 200 {
		t.Fatalf("batch delete = %d %v", code, res)
	}
	if got := ids(res["deleted"]); len(got) != 2 {
		t.Fatalf("deleted = %v (duplicates must be collapsed)", got)
	}
	r := reasons(res["failed"])
	if r[int(me.ID)] != "cannot delete current user" || r[99999] != "not found" {
		t.Fatalf("failed = %v", r)
	}
	// The caller survives and the others are gone.
	var n int64
	h.db.Model(&models.User{}).Where("id IN ?", others).Count(&n)
	if n != 0 {
		t.Fatalf("%d targeted users still exist", n)
	}
	if c, _ := h.do("GET", "/api/v1/auth/me", "", access); c != 200 {
		t.Fatalf("current user lost the session: %d", c)
	}
}

func TestBatchDeleteUsersEndsTheirSessions(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)
	victim := makeUsers(t, h.db, me.RoleID, "victim")[0]
	h.db.Create(&models.OnlineUser{UserID: victim, Username: "victim", Token: "victim-token"})

	h.do("DELETE", "/api/v1/users", body(map[string]any{"ids": []uint{victim}}), access)
	var n int64
	h.db.Model(&models.OnlineUser{}).Where("user_id = ?", victim).Count(&n)
	if n != 0 {
		t.Fatal("deleted user kept a live session row")
	}
}

func TestBatchValidation(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	big := make([]uint, 201)
	for i := range big {
		big[i] = uint(i + 1)
	}
	for name, payload := range map[string]string{
		"empty":     `{"ids":[]}`,
		"missing":   `{}`,
		"zero only": `{"ids":[0,0]}`,
		"too many":  body(map[string]any{"ids": big}),
		"malformed": `{"ids":`,
	} {
		for _, path := range []string{"/api/v1/users", "/api/v1/roles", "/api/v1/apis"} {
			if c, _ := h.do("DELETE", path, payload, access); c != 400 {
				t.Errorf("%s %s = %d, want 400", name, path, c)
			}
		}
	}
}

func TestBatchUserStatus(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)
	us := makeUsers(t, h.db, me.RoleID, "s1", "s2")
	h.db.Create(&models.OnlineUser{UserID: us[0], Username: "s1", Token: "s1-token"})

	code, res := h.do("PUT", "/api/v1/users/status", body(map[string]any{"ids": []uint{us[0], us[1], me.ID}, "status": 0}), access)
	if code != 200 {
		t.Fatalf("status = %d %v", code, res)
	}
	if got := ids(res["updated"]); len(got) != 2 {
		t.Fatalf("updated = %v", got)
	}
	if reasons(res["failed"])[int(me.ID)] != "cannot disable current user" {
		t.Fatalf("self-disable not blocked: %v", res["failed"])
	}
	var disabled int64
	h.db.Model(&models.User{}).Where("id IN ? AND status = 0", us).Count(&disabled)
	if disabled != 2 {
		t.Fatalf("disabled = %d", disabled)
	}
	var sessions int64
	h.db.Model(&models.OnlineUser{}).Where("user_id = ?", us[0]).Count(&sessions)
	if sessions != 0 {
		t.Fatal("disabling a user must end their sessions")
	}

	// Re-enabling the caller is fine; invalid status is rejected.
	if c, _ := h.do("PUT", "/api/v1/users/status", body(map[string]any{"ids": []uint{us[0]}, "status": 1}), access); c != 200 {
		t.Fatalf("enable = %d", c)
	}
	for _, bad := range []string{`{"ids":[1],"status":2}`, `{"ids":[1]}`} {
		if c, _ := h.do("PUT", "/api/v1/users/status", bad, access); c != 400 {
			t.Fatalf("payload %s = %d, want 400", bad, c)
		}
	}
}

func TestBatchDeleteRolesSkipsSystemAndInUse(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var admin models.Role
	h.db.Where("slug = ?", "admin").First(&admin)
	h.db.Model(&admin).Update("is_system", true)
	free := models.Role{Name: "Free", Slug: "free"}
	used := models.Role{Name: "Used", Slug: "used"}
	h.db.Create(&free)
	h.db.Create(&used)
	makeUsers(t, h.db, used.ID, "holder")

	code, res := h.do("DELETE", "/api/v1/roles", body(map[string]any{"ids": []uint{free.ID, used.ID, admin.ID}}), access)
	if code != 200 {
		t.Fatalf("= %d %v", code, res)
	}
	if got := ids(res["deleted"]); len(got) != 1 || got[0] != int(free.ID) {
		t.Fatalf("deleted = %v", got)
	}
	r := reasons(res["failed"])
	if r[int(admin.ID)] != "system role cannot be deleted" || r[int(used.ID)] != "role is assigned to users" {
		t.Fatalf("failed = %v", r)
	}
}

func TestBatchDeleteMediaDeletesObjectsAndKeepsFailures(t *testing.T) {
	mem := &memStorage{objs: map[string][]byte{}}
	h := newMediaHarness(t, func(o *RuntimeOptions) { o.Storage = mem })
	for _, n := range []string{"a", "b", "c"} {
		h.upload(n+".txt", n, "text/plain")
	}
	items := h.list()
	all := []uint{}
	for _, it := range items {
		all = append(all, uint(it["id"].(float64)))
	}
	if len(mem.objs) != 3 {
		t.Fatalf("setup objs=%d", len(mem.objs))
	}

	mem.failDelete = true
	code, res := h.do("DELETE", "/api/v1/media", body(map[string]any{"ids": all}), h.token)
	if code != 200 || len(ids(res["deleted"])) != 0 || len(ids2(res["failed"])) != 3 {
		t.Fatalf("with failing storage: %d %v", code, res)
	}
	if len(h.list()) != 3 {
		t.Fatal("records dropped although objects still exist (would be orphaned)")
	}

	mem.failDelete = false
	code, res = h.do("DELETE", "/api/v1/media", body(map[string]any{"ids": all}), h.token)
	if code != 200 || len(ids(res["deleted"])) != 3 || len(mem.objs) != 0 || len(h.list()) != 0 {
		t.Fatalf("retry: %d %v objs=%d", code, res, len(mem.objs))
	}
}

func ids2(v any) []any { return v.([]any) }

func TestBatchRoutesDoNotShadowSingleRoutes(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)
	one := makeUsers(t, h.db, me.RoleID, "single")[0]

	// DELETE /users/:id keeps working and returns the legacy shape.
	code, res := h.do("DELETE", "/api/v1/users/"+itoa(int(one)), "", access)
	if code != 200 || res["message"] != "ok" || res["deleted"] != nil {
		t.Fatalf("single delete broke: %d %v", code, res)
	}
	// PUT /users/status must not be captured by PUT /users/:id.
	code, res = h.do("PUT", "/api/v1/users/status", body(map[string]any{"ids": []uint{me.ID}, "status": 1}), access)
	if code != 200 || res["updated"] == nil {
		t.Fatalf("status route captured by :id: %d %v", code, res)
	}
}

func newFileHarness(t *testing.T) *bearerHarness {
	t.Helper()
	return newBearerHarness(t, func(o *RuntimeOptions) {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p.db")), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		o.DB = db
	})
}

func loginAs(t *testing.T, h *bearerHarness, slug string, keys []string, admin string) string {
	t.Helper()
	role := models.Role{Name: slug, Slug: slug}
	h.db.Create(&role)
	hash, _ := bcrypt.GenerateFromPassword([]byte("limited-pass-1"), bcrypt.MinCost)
	h.db.Create(&models.User{Username: slug, Password: string(hash), RoleID: role.ID, Status: 1})
	if c, _ := h.do("PUT", "/api/v1/roles/"+slug+"/permissions", body(map[string]any{"keys": keys}), admin); c != 200 {
		t.Fatalf("configuring role = %d", c)
	}
	_, login := h.do("POST", "/api/v1/auth/login", `{"username":"`+slug+`","password":"limited-pass-1"}`, "")
	return login["token"].(string)
}

// Regression: permissions saved from the role editor were stored as
// (method, path) while the Casbin model matches (path, method), so a granted
// permission never authorised anything.
func TestSavedPermissionsActuallyAuthorise(t *testing.T) {
	h := newFileHarness(t)
	admin, _ := h.login()
	tok := loginAs(t, h, "lister", []string{"users:list"}, admin)

	if c, _ := h.do("GET", "/api/v1/users", "", tok); c != 200 {
		t.Fatalf("granted users:list but GET /users = %d", c)
	}
	if c, _ := h.do("DELETE", "/api/v1/users/9999", "", tok); c != 403 {
		t.Fatalf("ungranted delete = %d, want 403", c)
	}
	if c, _ := h.do("GET", "/api/v1/roles", "", tok); c != 403 {
		t.Fatalf("ungranted roles list = %d, want 403", c)
	}
	_, perms := h.do("GET", "/api/v1/roles/lister/permissions", "", admin)
	if got := perms["granted"].([]any); len(got) != 1 || got[0] != "users:list" {
		t.Fatalf("granted keys round trip = %v", got)
	}
}

func TestLegacyReversedPoliciesAreRepairedOnStart(t *testing.T) {
	var db *gorm.DB
	h := newBearerHarnessSeeded(t, func(o *RuntimeOptions) {
		d, _ := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
		o.DB = d
		db = d
	}, func(o *RuntimeOptions) {
		// Rows exactly as the old editor wrote them: (role, method, path).
		o.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','legacy','GET','/api/v1/users')`)
		// A correct row, and a wildcard method row, must be left alone.
		o.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','legacy','/api/v1/roles','GET')`)
		// Upgrading runs the schema migration again, which repairs the rows.
		if err := MigrateSchema(o.DB); err != nil {
			t.Fatal(err)
		}
	})
	_ = h
	var rows []struct{ V1, V2 string }
	db.Table("admin_casbin_rule").Where("v0 = ?", "legacy").Order("id").Find(&rows)
	if len(rows) != 2 || rows[0].V1 != "/api/v1/users" || rows[0].V2 != "GET" || rows[1].V1 != "/api/v1/roles" || rows[1].V2 != "GET" {
		t.Fatalf("rows after repair = %v", rows)
	}
}

func TestBatchAuthorizationFollowsPermissionKeys(t *testing.T) {
	h := newFileHarness(t)
	admin, _ := h.login()
	var adminRole models.Role
	h.db.Where("slug = ?", "admin").First(&adminRole)
	victims := makeUsers(t, h.db, adminRole.ID, "v1", "v2")

	tok := loginAs(t, h, "limited", []string{"users:delete"}, admin)
	if c, _ := h.do("PUT", "/api/v1/users/status", body(map[string]any{"ids": victims, "status": 0}), tok); c != 403 {
		t.Fatalf("users:delete role reached batch status (users:edit): %d", c)
	}
	if c, _ := h.do("DELETE", "/api/v1/users", body(map[string]any{"ids": victims}), tok); c != 200 {
		t.Fatalf("users:delete role denied batch delete: %d", c)
	}

	// Editing alone must not permit deleting.
	editor := loginAs(t, h, "editonly", []string{"users:edit"}, admin)
	if c, _ := h.do("DELETE", "/api/v1/users", body(map[string]any{"ids": []uint{adminRole.ID}}), editor); c != 403 {
		t.Fatalf("users:edit role reached batch delete: %d", c)
	}
}

package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"

	"go-nx-admin/internal/models"
)

func me(h *bearerHarness) models.User {
	var u models.User
	h.db.Where("username = ?", "refresher").First(&u)
	return u
}

// A disabled account must be indistinguishable from a wrong password, and the
// failure of an unknown user must count against the rate limit like any other.
func TestLoginDoesNotRevealWhichUsernamesExist(t *testing.T) {
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Config.Login.RateLimit.Enabled = true
		o.Config.Login.RateLimit.MaxAttempts = 3
		o.Config.Login.RateLimit.WindowMinutes = 5
		o.Config.Login.RateLimit.LockMinutes = 5
	})
	h.db.Model(&models.User{}).Where("username = ?", "refresher").Update("status", 0)

	c1, r1 := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"refresh-test-password"}`, "")
	c2, r2 := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"wrong-password-1"}`, "")
	c3, r3 := h.do("POST", "/api/v1/auth/login", `{"username":"nobody","password":"wrong-password-1"}`, "")
	if c1 != 401 || c2 != 401 || c3 != 401 {
		t.Fatalf("statuses = %d %d %d, want all 401", c1, c2, c3)
	}
	if fmt.Sprint(r1) != fmt.Sprint(r2) || fmt.Sprint(r2) != fmt.Sprint(r3) {
		t.Fatalf("responses differ: %v | %v | %v", r1, r2, r3)
	}
	// Three failures against unknown/disabled accounts have used the budget.
	if c, _ := h.do("POST", "/api/v1/auth/login", `{"username":"nobody","password":"x"}`, ""); c != 429 {
		t.Fatalf("4th attempt = %d, want 429 (unknown users must be rate limited)", c)
	}
}

func TestMFACodeGuessingIsLimitedPerUser(t *testing.T) {
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Config.Login.RateLimit.Enabled = true
		o.Config.Login.RateLimit.MaxAttempts = 3
		o.Config.Login.RateLimit.WindowMinutes = 5
		o.Config.Login.RateLimit.LockMinutes = 5
	})
	access, _ := h.login()
	_, setup := h.do("POST", "/api/v1/auth/mfa/setup", `{}`, access)
	secret := setup["secret"].(string)
	code, _ := totp.GenerateCode(secret, time.Now())
	if c, _ := h.do("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, access); c != 200 {
		t.Fatalf("enable = %d", c)
	}
	_, login := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"refresh-test-password"}`, "")
	mfa := login["mfa_token"].(string)
	wrong := `{"code":"000000"}`
	for i := 0; i < 3; i++ {
		if c, _ := h.do("POST", "/api/v1/auth/mfa/verify", wrong, mfa); c != 401 {
			t.Fatalf("guess %d = %d, want 401", i, c)
		}
	}
	// Locked: even the correct code is refused until the lock expires.
	good, _ := totp.GenerateCode(secret, time.Now())
	if c, _ := h.do("POST", "/api/v1/auth/mfa/verify", `{"code":"`+good+`"}`, mfa); c != 429 {
		t.Fatalf("after 3 failures = %d, want 429", c)
	}
}

func TestRecoveryCodeIsSingleUse(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	_, setup := h.do("POST", "/api/v1/auth/mfa/setup", `{}`, access)
	code, _ := totp.GenerateCode(setup["secret"].(string), time.Now())
	_, enabled := h.do("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, access)
	recovery := enabled["recovery_codes"].([]any)[0].(string)

	verify := func() int {
		_, login := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"refresh-test-password"}`, "")
		c, _ := h.do("POST", "/api/v1/auth/mfa/verify", `{"recovery_code":"`+recovery+`"}`, login["mfa_token"].(string))
		return c
	}
	if c := verify(); c != 200 {
		t.Fatalf("first use = %d", c)
	}
	if c := verify(); c != 401 {
		t.Fatalf("second use = %d, want 401", c)
	}
}

func TestDeleteUserEndsSessionsAndGuardsTargets(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	self := me(h)
	other := makeUsers(t, h.db, self.RoleID, "victim")[0]
	h.db.Create(&models.OnlineUser{UserID: other, Username: "victim", Token: "tok-victim"})

	if c, _ := h.do("DELETE", "/api/v1/users/"+itoa(int(self.ID)), "", access); c != 400 {
		t.Fatalf("delete self = %d", c)
	}
	if c, _ := h.do("DELETE", "/api/v1/users/999999", "", access); c != 404 {
		t.Fatalf("delete missing = %d, want 404", c)
	}
	if c, _ := h.do("DELETE", "/api/v1/users/"+itoa(int(other)), "", access); c != 200 {
		t.Fatalf("delete other = %d", c)
	}
	var sessions int64
	h.db.Model(&models.OnlineUser{}).Where("user_id = ?", other).Count(&sessions)
	if sessions != 0 {
		t.Fatal("deleted user kept a live session")
	}
}

func TestUpdateUserWithoutRoleKeepsRole(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	self := me(h)
	target := makeUsers(t, h.db, self.RoleID, "target")[0]
	if c, res := h.do("PUT", "/api/v1/users/"+itoa(int(target)), `{"status":0}`, access); c != 200 {
		t.Fatalf("update = %d %v", c, res)
	}
	var got models.User
	h.db.First(&got, target)
	if got.RoleID != self.RoleID || got.Status != 0 {
		t.Fatalf("role=%d status=%d, want role kept and status 0", got.RoleID, got.Status)
	}
	if c, _ := h.do("PUT", "/api/v1/users/"+itoa(int(target)), `{"role_id":0}`, access); c != 400 {
		t.Fatalf("role_id 0 = %d, want 400", c)
	}
}

func TestDeletingRoleRevokesItsPolicyAndGrants(t *testing.T) {
	var enforcer *casbin.Enforcer
	h := newBearerHarnessSeeded(t, func(o *RuntimeOptions) {
		o.Extensions.ExtraRoutes = func(_ fiber.Router, _ *gorm.DB, e *casbin.Enforcer) { enforcer = e }
	}, nil)
	access, _ := h.login()
	role := models.Role{Name: "Temp", Slug: "temp-role"}
	h.db.Create(&role)
	h.db.Create(&models.RoleMenu{RoleID: role.ID, MenuID: 1})
	if c, res := h.do("PUT", "/api/v1/roles/temp-role/permissions", `{"keys":["users:list"]}`, access); c != 200 {
		t.Fatalf("grant = %d %v", c, res)
	}
	if len(enforcer.GetFilteredPolicy(0, "temp-role")) == 0 {
		t.Fatal("setup: policy not stored")
	}
	holder := makeUsers(t, h.db, role.ID, "holder")[0]
	if c, _ := h.do("DELETE", "/api/v1/roles/"+itoa(int(role.ID)), "", access); c != 409 {
		t.Fatalf("delete in-use role = %d, want 409", c)
	}
	h.db.Delete(&models.User{}, holder)

	if c, _ := h.do("DELETE", "/api/v1/roles/"+itoa(int(role.ID)), "", access); c != 200 {
		t.Fatalf("delete role = %d", c)
	}
	if n := len(enforcer.GetFilteredPolicy(0, "temp-role")); n != 0 {
		t.Fatalf("%d policies survived the role", n)
	}
	var grants int64
	h.db.Model(&models.RoleMenu{}).Where("role_id = ?", role.ID).Count(&grants)
	if grants != 0 {
		t.Fatal("menu grants survived the role")
	}
}

func TestIPWhitelistRulesAreScopedToTheirUserAndSupportCIDR(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	self := me(h)
	others := makeUsers(t, h.db, self.RoleID, "a", "b")
	a, b := itoa(int(others[0])), itoa(int(others[1]))

	c, res := h.do("POST", "/api/v1/users/"+a+"/ip-whitelists", `{"ip":"10.0.0.0/8","enabled":false}`, access)
	if c != 200 {
		t.Fatalf("create cidr = %d %v", c, res)
	}
	rule := res["data"].(map[string]any)
	wid := itoa(int(rule["id"].(float64)))
	if rule["enabled"] != false {
		t.Fatalf("enabled=false was lost: %v", rule)
	}
	if c, _ := h.do("POST", "/api/v1/users/"+a+"/ip-whitelists", `{"ip":"not-an-ip"}`, access); c != 400 {
		t.Fatalf("invalid ip = %d", c)
	}
	// The rule belongs to user a: addressing it through user b must fail.
	if c, _ := h.do("PUT", "/api/v1/users/"+b+"/ip-whitelists/"+wid, `{"remark":"hijack"}`, access); c != 404 {
		t.Fatalf("cross-user update = %d, want 404", c)
	}
	if c, _ := h.do("DELETE", "/api/v1/users/"+b+"/ip-whitelists/"+wid, "", access); c != 404 {
		t.Fatalf("cross-user delete = %d, want 404", c)
	}
	if c, _ := h.do("PUT", "/api/v1/users/"+a+"/ip-whitelists/"+wid, `{"remark":"mine"}`, access); c != 200 {
		t.Fatalf("own update = %d", c)
	}
	var got models.UserIPWhitelist
	h.db.First(&got, wid)
	if got.Enabled || got.Remark != "mine" || got.IP != "10.0.0.0/8" {
		t.Fatalf("partial update changed other fields: %+v", got)
	}
}

func TestMenuTreeRejectsCyclesAndDeleteCleansGrants(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	mk := func(parent *uint) uint {
		m := models.Menu{Name: "m", ParentID: parent}
		h.db.Create(&m)
		return m.ID
	}
	root := mk(nil)
	child := mk(&root)
	grand := mk(&child)

	for _, tc := range []struct {
		id     uint
		parent uint
	}{{root, grand}, {root, root}, {child, grand}} {
		body := fmt.Sprintf(`{"name":"m","parent_id":%d}`, tc.parent)
		if c, _ := h.do("PUT", "/api/v1/menus/"+itoa(int(tc.id)), body, access); c != 400 {
			t.Fatalf("move %d under %d = %d, want 400", tc.id, tc.parent, c)
		}
	}
	if c, _ := h.do("POST", "/api/v1/menus", `{"name":"x","parent_id":987654}`, access); c != 400 {
		t.Fatalf("missing parent = %d, want 400", c)
	}

	// A corrupt cycle already in the table must not hang the delete.
	h.db.Model(&models.Menu{}).Where("id = ?", root).Update("parent_id", grand)
	h.db.Create(&models.RoleMenu{RoleID: 99, MenuID: grand})
	done := make(chan int, 1)
	go func() { c, _ := h.do("DELETE", "/api/v1/menus/"+itoa(int(root)), "", access); done <- c }()
	select {
	case c := <-done:
		if c != 200 {
			t.Fatalf("delete = %d", c)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("delete never returned on a cyclic tree")
	}
	var left, grants int64
	h.db.Model(&models.Menu{}).Where("id IN ?", []uint{root, child, grand}).Count(&left)
	h.db.Model(&models.RoleMenu{}).Where("menu_id = ?", grand).Count(&grants)
	if left != 0 || grants != 0 {
		t.Fatalf("menus left=%d grants left=%d", left, grants)
	}
}

func TestAutoDiscoverSkipsHeadOptionsAndIsIdempotent(t *testing.T) {
	h := newBearerHarness(t, nil)
	models.AutoDiscoverAPIs(h.r.App, h.db)
	var methods []string
	h.db.Model(&models.Api{}).Distinct().Pluck("method", &methods)
	for _, m := range methods {
		if m == "HEAD" || m == "OPTIONS" {
			t.Fatalf("%s endpoints were registered", m)
		}
	}
	var before, after int64
	h.db.Model(&models.Api{}).Count(&before)
	models.AutoDiscoverAPIs(h.r.App, h.db)
	h.db.Model(&models.Api{}).Count(&after)
	if before != after {
		t.Fatalf("second discovery changed the row count: %d -> %d", before, after)
	}
}

func TestSystemMenusKeepOperatorChanges(t *testing.T) {
	h := newBearerHarness(t, nil)
	h.db.Create(&models.Menu{Name: "后台管理", Permission: "backend:manage"})
	if err := models.Seed(h.db); err != nil {
		t.Fatal(err)
	}
	h.db.Model(&models.Menu{}).Where("path = ?", "/login-logs").Updates(map[string]any{"name": "My Logs", "sort": 42})
	h.db.Where("path = ?", "/media").Delete(&models.Menu{})
	if err := models.Seed(h.db); err != nil {
		t.Fatal(err)
	}
	var m models.Menu
	h.db.Where("path = ?", "/login-logs").First(&m)
	if m.Name != "My Logs" || m.Sort != 42 {
		t.Fatalf("restart overwrote operator edits: %+v", m)
	}
	var missing int64
	h.db.Model(&models.Menu{}).Where("path = ?", "/media").Count(&missing)
	if missing != 1 {
		t.Fatal("a missing system menu must be recreated")
	}
}

func TestExtraModelsMayReferenceBaseTablesOnAFreshDatabase(t *testing.T) {
	type Note struct {
		ID     uint `gorm:"primaryKey"`
		UserID uint
		User   models.User `gorm:"foreignKey:UserID"`
	}
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			db := bootstrapTarget(t, dialect)
			if err := models.AutoMigrate(db, &Note{}); err != nil {
				t.Fatal(err)
			}
			if !db.Migrator().HasTable(&Note{}) {
				t.Fatal("extra table not created")
			}
			if err := models.AutoMigrate(db, &Note{}); err != nil {
				t.Fatalf("second migrate: %v", err)
			}
		})
	}
}

func TestUploadBodyLimitFollowsConfiguredMaxSize(t *testing.T) {
	small := bodyLimit(RuntimeOptions{})
	opts := RuntimeOptions{}
	opts.Config.Upload.MaxSize = 20 << 20
	if got := bodyLimit(opts); got <= 20<<20 || got < small {
		t.Fatalf("limit %d must exceed the 20 MiB upload size", got)
	}
	opts.DisableMedia = true
	if bodyLimit(opts) != small {
		t.Fatal("no upload route: keep the default limit")
	}
}

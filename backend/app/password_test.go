package app

import (
	"context"
	"encoding/json"
	"go-nx-admin/internal/handlers"
	"go-nx-admin/internal/middleware"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const goodNewPassword = "a-much-better-pass-1"

func passwordBody(old, next string) string {
	return body(map[string]string{"old_password": old, "new_password": next})
}

func TestChangePasswordRequiresOldAndFollowsPolicy(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, refresh := h.login()
	old := "refresh-test-password"

	for name, tc := range map[string]struct{ body, code string }{
		"wrong old":      {passwordBody("nope-nope-nope-1", goodNewPassword), "OLD_PASSWORD_INCORRECT"},
		"too short":      {passwordBody(old, "short-1"), "PASSWORD_POLICY"},
		"too long":       {passwordBody(old, string(make([]byte, 33))), "PASSWORD_POLICY"},
		"same as old":    {passwordBody(old, old), "PASSWORD_UNCHANGED"},
		"missing fields": {`{}`, "PASSWORD_POLICY"},
	} {
		code, res := h.do("POST", "/api/v1/auth/password", tc.body, access)
		if code != 400 || res["code"] != tc.code {
			t.Fatalf("%s: %d %v want %s", name, code, res, tc.code)
		}
	}

	if code, res := h.do("POST", "/api/v1/auth/password", passwordBody(old, goodNewPassword), access); code != 200 {
		t.Fatalf("change: %d %v", code, res)
	}
	// Only the password changes: this and other sessions keep working.
	if c, _ := h.do("GET", "/api/v1/auth/me", "", access); c != 200 {
		t.Fatalf("current session ended by password change: %d", c)
	}
	if c, _ := h.refresh(refresh); c != 200 {
		t.Fatalf("refresh token invalidated by password change: %d", c)
	}
	if c, _ := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"`+old+`"}`, ""); c != 401 {
		t.Fatalf("old password still works: %d", c)
	}
	if c, _ := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"`+goodNewPassword+`"}`, ""); c != 200 {
		t.Fatalf("new password rejected: %d", c)
	}
}

func TestChangePasswordNeedsMFACodeWhenEnabled(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	_, setup := h.do("POST", "/api/v1/auth/mfa/setup", `{}`, access)
	secret := setup["secret"].(string)
	code, _ := totp.GenerateCode(secret, time.Now())
	if c, _ := h.do("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, access); c != 200 {
		t.Fatalf("enable=%d", c)
	}
	old := "refresh-test-password"
	if c, res := h.do("POST", "/api/v1/auth/password", passwordBody(old, goodNewPassword), access); c != 400 || res["code"] != "MFA_CODE_INVALID" {
		t.Fatalf("no code: %d %v", c, res)
	}
	valid, _ := totp.GenerateCode(secret, time.Now())
	withCode := body(map[string]string{"old_password": old, "new_password": goodNewPassword, "code": valid})
	if c, res := h.do("POST", "/api/v1/auth/password", withCode, access); c != 200 {
		t.Fatalf("with code: %d %v", c, res)
	}
}

func TestChangePasswordIsRateLimited(t *testing.T) {
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Config.Login.RateLimit.Enabled = true
		o.Config.Login.RateLimit.MaxAttempts = 3
		o.Config.Login.RateLimit.WindowMinutes = 5
		o.Config.Login.RateLimit.LockMinutes = 5
	})
	access, _ := h.login()
	for i := 0; i < 3; i++ {
		h.do("POST", "/api/v1/auth/password", passwordBody("wrong-wrong-wrong-1", goodNewPassword), access)
	}
	if c, res := h.do("POST", "/api/v1/auth/password", passwordBody("refresh-test-password", goodNewPassword), access); c != 429 {
		t.Fatalf("expected lock after repeated failures, got %d %v", c, res)
	}
}

func TestChangePasswordNeedsNoRolePermission(t *testing.T) {
	h := newBearerHarness(t, nil)
	var admin models.User
	h.db.Where("username = ?", "refresher").First(&admin)
	role := models.Role{Name: "Plain", Slug: "plain"}
	h.db.Create(&role)
	hash, _ := bcrypt.GenerateFromPassword([]byte("plain-user-pass-1"), bcrypt.MinCost)
	h.db.Create(&models.User{Username: "plain", Password: string(hash), RoleID: role.ID, Status: 1})
	_, login := h.do("POST", "/api/v1/auth/login", `{"username":"plain","password":"plain-user-pass-1"}`, "")
	token := login["token"].(string)
	// The role has no policies at all, so a regular route is forbidden...
	if c, _ := h.do("GET", "/api/v1/users", "", token); c != 403 {
		t.Fatalf("expected 403 for unauthorised route, got %d", c)
	}
	// ...but every signed-in user can still change their own password.
	if c, res := h.do("POST", "/api/v1/auth/password", passwordBody("plain-user-pass-1", goodNewPassword), token); c != 200 {
		t.Fatalf("plain role change: %d %v", c, res)
	}
}

func TestMustChangePasswordConfinesAccountUntilChanged(t *testing.T) {
	h := newBearerHarness(t, nil)
	h.db.Model(&models.User{}).Where("username = ?", "refresher").Update("must_change_password", true)
	access, _ := h.login()

	_, me := h.do("GET", "/api/v1/auth/me", "", access)
	if me["must_change_password"] != true {
		t.Fatalf("/me must expose the flag: %v", me)
	}
	for _, path := range []string{"/api/v1/users", "/api/v1/roles", "/api/v1/auth/menus", "/api/v1/dashboard/stats"} {
		if c, res := h.do("GET", path, "", access); c != 403 || res["code"] != "PASSWORD_CHANGE_REQUIRED" {
			t.Fatalf("%s reachable while flagged: %d %v", path, c, res)
		}
	}
	if c, _ := h.do("POST", "/api/v1/users", `{"username":"x"}`, access); c != 403 {
		t.Fatalf("mutation reachable while flagged: %d", c)
	}
	if c, res := h.do("POST", "/api/v1/auth/password", passwordBody("refresh-test-password", goodNewPassword), access); c != 200 {
		t.Fatalf("change while flagged: %d %v", c, res)
	}
	var u models.User
	h.db.Where("username = ?", "refresher").First(&u)
	if u.MustChangePassword {
		t.Fatal("flag must clear after the password is changed")
	}
	if c, _ := h.do("GET", "/api/v1/users", "", access); c != 200 {
		t.Fatalf("still confined after change: %d", c)
	}
}

func TestMustChangePasswordStaysSetWhenChangeFails(t *testing.T) {
	h := newBearerHarness(t, nil)
	h.db.Model(&models.User{}).Where("username = ?", "refresher").Update("must_change_password", true)
	access, _ := h.login()
	h.do("POST", "/api/v1/auth/password", passwordBody("wrong-wrong-wrong-1", goodNewPassword), access)
	h.do("POST", "/api/v1/auth/password", passwordBody("refresh-test-password", "short"), access)
	var u models.User
	h.db.Where("username = ?", "refresher").First(&u)
	if !u.MustChangePassword {
		t.Fatal("flag cleared by a failed change")
	}
}

func TestAdminCanFlagUsersAndPolicyAppliesToCreateAndEdit(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)

	if c, res := h.do("POST", "/api/v1/users", body(map[string]any{"username": "weak", "password": "short", "role_id": me.RoleID}), access); c != 400 {
		t.Fatalf("weak password accepted on create: %d %v", c, res)
	}
	c, res := h.do("POST", "/api/v1/users", body(map[string]any{"username": "flagged", "password": "created-pass-123", "role_id": me.RoleID, "must_change_password": true}), access)
	if c != 200 {
		t.Fatalf("create: %d %v", c, res)
	}
	id := uint(res["data"].(map[string]any)["id"].(float64))
	var u models.User
	h.db.First(&u, id)
	if !u.MustChangePassword {
		t.Fatal("create must persist the flag")
	}
	// Editing without the field keeps the flag; an explicit false clears it.
	if c, _ := h.do("PUT", "/api/v1/users/"+itoa(int(id)), body(map[string]any{"username": "flagged", "role_id": me.RoleID}), access); c != 200 {
		t.Fatal("edit failed")
	}
	h.db.First(&u, id)
	if !u.MustChangePassword {
		t.Fatal("edit without the field must not clear the flag")
	}
	if c, _ := h.do("PUT", "/api/v1/users/"+itoa(int(id)), body(map[string]any{"username": "flagged", "role_id": me.RoleID, "password": "tiny"}), access); c != 400 {
		t.Fatalf("weak reset accepted: %d", c)
	}
	if c, _ := h.do("PUT", "/api/v1/users/"+itoa(int(id)), body(map[string]any{"username": "flagged", "role_id": me.RoleID, "must_change_password": false}), access); c != 200 {
		t.Fatal("edit failed")
	}
	h.db.First(&u, id)
	if u.MustChangePassword {
		t.Fatal("explicit false must clear the flag")
	}
}

func TestSeedAdminIsNotForcedToChangePassword(t *testing.T) {
	opts := runtimeOptions(t)
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	if err := models.Seed(opts.DB); err != nil {
		t.Fatal(err)
	}
	var admin models.User
	opts.DB.Where("username = ?", "admin").First(&admin)
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("qwer1234")) != nil {
		t.Fatal("seed admin must keep the documented default password")
	}
	if admin.MustChangePassword {
		t.Fatal("seed admin must not be forced to change password; operators opt in themselves")
	}
	// Re-running the seed on an existing deployment must not flag it either.
	if err := models.Seed(opts.DB); err != nil {
		t.Fatal(err)
	}
	opts.DB.First(&admin, admin.ID)
	if admin.MustChangePassword {
		t.Fatal("a repeated seed must not flag the default admin")
	}
}

func TestPasswordPolicyIsEightToThirtyTwoBytes(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)

	cases := []struct {
		name string
		pw   string
		ok   bool
	}{
		{"7 bytes", "1234567", false},
		{"8 bytes", "12345678", true},
		{"32 bytes", strings.Repeat("a", 32), true},
		{"33 bytes", strings.Repeat("a", 33), false},
		{"30 bytes multibyte", strings.Repeat("\u4e2d", 10), true},
		{"33 bytes multibyte", strings.Repeat("\u4e2d", 11), false},
	}
	for i, tc := range cases {
		c, res := h.do("POST", "/api/v1/users", body(map[string]any{"username": "bound" + itoa(i), "password": tc.pw, "role_id": me.RoleID}), access)
		if tc.ok && c != 200 || !tc.ok && c != 400 {
			t.Fatalf("%s: got %d %v", tc.name, c, res)
		}
	}
}

func TestCookieModeForcedPasswordChange(t *testing.T) {
	opts := runtimeOptions(t)
	opts.Config.Session.Cookie = true
	opts.Config.Session.Origin = opts.Config.CORS.Origins
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	role := models.Role{Name: "System", Slug: "admin"}
	opts.DB.Create(&role)
	hash, _ := bcrypt.GenerateFromPassword([]byte("strict-test-password"), bcrypt.MinCost)
	secret := "JBSWY3DPEHPK3PXP"
	user := models.User{Username: "strict", Password: string(hash), RoleID: role.ID, Status: 1,
		MFAEnabled: true, MFASecret: secret, MustChangePassword: true}
	if err := opts.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`)
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(context.Background())

	cookies := map[string]*http.Cookie{}
	origin := opts.Config.Session.Origin
	call := func(method, path, payload string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		if c := cookies[middleware.CSRFCookie]; c != nil {
			req.Header.Set("X-CSRF-Token", c.Value)
		}
		res, err := r.App.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		for _, c := range res.Cookies() {
			cookies[c.Name] = c
		}
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	call("POST", "/api/v1/auth/login", `{"username":"strict","password":"strict-test-password"}`)
	code, _ := totp.GenerateCode(secret, time.Now())
	if c, res := call("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`); c != 200 {
		t.Fatalf("verify: %d %v", c, res)
	}
	if c, res := call("GET", "/api/v1/users", ""); c != 403 || res["code"] != "PASSWORD_CHANGE_REQUIRED" {
		t.Fatalf("cookie session reached users while flagged: %d %v", c, res)
	}
	if _, me := call("GET", "/api/v1/auth/me", ""); me["must_change_password"] != true {
		t.Fatalf("me: %v", me)
	}
	fresh, _ := totp.GenerateCode(secret, time.Now())
	payload := body(map[string]string{"old_password": "strict-test-password", "new_password": goodNewPassword, "code": fresh})
	if c, res := call("POST", "/api/v1/auth/password", payload); c != 200 {
		t.Fatalf("change: %d %v", c, res)
	}
	if c, _ := call("GET", "/api/v1/users", ""); c != 200 {
		t.Fatalf("still confined after change: %d", c)
	}
}

// app.Test uses an in-memory connection with no real peer, so this only proves
// the configured header is wired through the login path; the trust boundary
// itself (who may supply the header) is covered by middleware.TestClientIP.
func TestLoginUsesConfiguredClientIPHeaderWiring(t *testing.T) {
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Config.Server.ClientIPHeader = "X-Real-IP"
		o.Config.Server.TrustedProxies = []string{"0.0.0.0/0"}
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"refresher","password":"refresh-test-password"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "198.51.100.23")
	res, err := h.r.App.Test(req, 5000)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("login: %v", err)
	}
	res.Body.Close()
	var s models.OnlineUser
	h.db.Order("id desc").First(&s)
	if s.IP != "198.51.100.23" {
		t.Fatalf("online session IP = %q, want the proxied client address", s.IP)
	}
	var log models.LoginLog
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && h.db.Order("id desc").First(&log).Error != nil {
		time.Sleep(20 * time.Millisecond)
	}
	if log.IP != "198.51.100.23" {
		t.Fatalf("login log IP = %q", log.IP)
	}
}

func TestUserListExposesMustChangePassword(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	var me models.User
	h.db.Where("username = ?", "refresher").First(&me)
	h.db.Model(&models.User{}).Where("id = ?", me.ID).Update("must_change_password", false)
	ids := makeUsers(t, h.db, me.RoleID, "pending")
	h.db.Model(&models.User{}).Where("id = ?", ids[0]).Update("must_change_password", true)
	_, res := h.do("GET", "/api/v1/users", "", access)
	flag := map[string]any{}
	for _, raw := range res["data"].([]any) {
		u := raw.(map[string]any)
		v, ok := u["must_change_password"]
		if !ok {
			t.Fatalf("list item for %v lacks must_change_password", u["username"])
		}
		flag[u["username"].(string)] = v
	}
	if flag["pending"] != true || flag["refresher"] != false {
		t.Fatalf("flags = %v", flag)
	}
}

func TestDashboardStatsMergeAndIsolation(t *testing.T) {
	var mode string
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Extensions.DashboardStats = func(db *gorm.DB) map[string]any {
			switch mode {
			case "panic":
				panic("boom")
			case "override":
				return map[string]any{"total_users": 999, "open_orders": 7}
			}
			return map[string]any{"open_orders": 7}
		}
	})
	access, _ := h.login()
	get := func() map[string]any {
		code, res := h.do("GET", "/api/v1/dashboard/stats", "", access)
		if code != 200 {
			t.Fatalf("stats=%d", code)
		}
		return res
	}
	res := get()
	if res["open_orders"] != float64(7) || res["total_users"] != float64(1) || res["online_users"] == nil {
		t.Fatalf("extra figures must be added beside the built-in ones: %v", res)
	}
	if _, ok := res["today_api_calls"]; ok {
		t.Fatal("the dead today_api_calls figure must be gone")
	}
	mode = "override"
	if res = get(); res["total_users"] != float64(999) {
		t.Fatalf("a provider key must override the built-in value: %v", res)
	}
	mode = "panic"
	if res = get(); res["total_users"] != float64(1) {
		t.Fatalf("a panicking provider must not break the built-in figures: %v", res)
	}
}

func TestDashboardProviderRestoredAfterClose(t *testing.T) {
	before := handlers.ExtraDashboardStats
	h := newBearerHarness(t, func(o *RuntimeOptions) {
		o.Extensions.DashboardStats = func(*gorm.DB) map[string]any { return nil }
	})
	if handlers.ExtraDashboardStats == nil {
		t.Fatal("provider not installed")
	}
	if err := h.r.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if (before == nil) != (handlers.ExtraDashboardStats == nil) {
		t.Fatal("provider leaked past Close")
	}
}

func TestMyPermissionsReportsOnlyTheCallersRole(t *testing.T) {
	h := newBearerHarnessSeeded(t, nil, func(o *RuntimeOptions) {
		role := models.Role{Name: "Viewer", Slug: "viewer"}
		o.DB.Create(&role)
		hash, _ := bcrypt.GenerateFromPassword([]byte("viewer-user-pass-1"), bcrypt.MinCost)
		o.DB.Create(&models.User{Username: "viewer", Password: string(hash), RoleID: role.ID, Status: 1})
		// Stored rows are (role, path, method).
		o.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','viewer','/api/v1/dashboard/stats','GET')`)
	})
	adminToken, _ := h.login()
	if _, res := h.do("GET", "/api/v1/auth/permissions", "", adminToken); res["all"] != true {
		t.Fatalf("admin (.* .*) must report all: %v", res)
	}

	_, login := h.do("POST", "/api/v1/auth/login", `{"username":"viewer","password":"viewer-user-pass-1"}`, "")
	viewer := login["token"].(string)
	code, res := h.do("GET", "/api/v1/auth/permissions", "", viewer)
	if code != 200 || res["all"] != false {
		t.Fatalf("viewer: %d %v", code, res)
	}
	granted := map[string]bool{}
	for _, k := range res["granted"].([]any) {
		granted[k.(string)] = true
	}
	if !granted["dashboard:view"] || granted["users:list"] {
		t.Fatalf("viewer must hold dashboard:view only, got %v", granted)
	}
}

package app

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestLegacyBearerLoginMFAAndLogout(t *testing.T) {
	opts := runtimeOptions(t)
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	role := models.Role{Name: "Admin", Slug: "admin"}
	if err := opts.DB.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("legacy-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "legacy", Password: string(hash), RoleID: role.ID, Status: 1}
	if err := opts.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`).Error; err != nil {
		t.Fatal(err)
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(context.Background())
	request := func(method, path, body, token string, want int) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := r.App.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want=%d", method, path, res.StatusCode, want)
		}
		if len(res.Cookies()) != 0 {
			t.Fatal("legacy login emitted browser-session cookies")
		}
		var data map[string]any
		if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	login := `{"username":"legacy","password":"legacy-test-password"}`
	data := request("POST", "/api/v1/auth/login", login, "", 200)
	token := data["token"].(string)
	request("POST", "/api/v1/auth/verify-password", `{"password":"legacy-test-password"}`, token, 200)
	setup := request("POST", "/api/v1/auth/mfa/setup", `{}`, token, 200)
	code, err := totp.GenerateCode(setup["secret"].(string), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, token, 200)
	data = request("POST", "/api/v1/auth/login", login, "", 200)
	if data["mfa_required"] != true || data["token"] != nil {
		t.Fatal("legacy MFA login contract changed")
	}
	verified := request("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`, data["mfa_token"].(string), 200)
	if verified["token"] == nil {
		t.Fatal("legacy MFA response omitted bearer token")
	}
	full := verified["token"].(string)
	request("GET", "/api/v1/auth/me", "", full, 200)
	request("POST", "/api/v1/auth/logout", "", full, 200)
	request("GET", "/api/v1/auth/me", "", full, 401)
}

type bearerHarness struct {
	t  *testing.T
	r  *Runtime
	db *gorm.DB
}

func newBearerHarness(t *testing.T, mutate func(*RuntimeOptions)) *bearerHarness {
	return newBearerHarnessSeeded(t, mutate, nil)
}

// newBearerHarnessSeeded additionally runs seed after the schema exists but
// before the runtime starts, for tests that need pre-existing rows.
func newBearerHarnessSeeded(t *testing.T, mutate func(*RuntimeOptions), seed func(*RuntimeOptions)) *bearerHarness {
	t.Helper()
	opts := runtimeOptions(t)
	if mutate != nil {
		mutate(&opts)
	}
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	if seed != nil {
		seed(&opts)
	}
	role := models.Role{Name: "Admin", Slug: "admin"}
	if err := opts.DB.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("refresh-test-password"), bcrypt.MinCost)
	if err := opts.DB.Create(&models.User{Username: "refresher", Password: string(hash), RoleID: role.ID, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`).Error; err != nil {
		t.Fatal(err)
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return &bearerHarness{t: t, r: r, db: opts.DB}
}

func (h *bearerHarness) do(method, path, body, token string) (int, map[string]any) {
	h.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := h.r.App.Test(req, 5000)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	var data map[string]any
	json.NewDecoder(res.Body).Decode(&data)
	return res.StatusCode, data
}

func (h *bearerHarness) login() (access, refresh string) {
	h.t.Helper()
	code, data := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"refresh-test-password"}`, "")
	if code != 200 {
		h.t.Fatalf("login=%d", code)
	}
	access, _ = data["token"].(string)
	refresh, _ = data["refresh_token"].(string)
	if access == "" || len(refresh) != 64 {
		h.t.Fatalf("login must return access and 64-char refresh token: %v", data)
	}
	return
}

func (h *bearerHarness) refresh(token string) (int, map[string]any) {
	body, _ := json.Marshal(map[string]string{"refresh_token": token})
	return h.do("POST", "/api/v1/auth/refresh", string(body), "")
}

func TestRefreshRotatesAndInvalidatesOldAccessToken(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, refresh := h.login()

	code, data := h.refresh(refresh)
	if code != 200 {
		t.Fatalf("refresh=%d", code)
	}
	newAccess, newRefresh := data["token"].(string), data["refresh_token"].(string)
	if newAccess == access || newRefresh == refresh {
		t.Fatal("refresh must rotate both tokens")
	}
	if c, _ := h.do("GET", "/api/v1/auth/me", "", newAccess); c != 200 {
		t.Fatalf("new access token rejected: %d", c)
	}
	// The session row was updated in place, so the old access token is gone.
	if c, _ := h.do("GET", "/api/v1/auth/me", "", access); c != 401 {
		t.Fatalf("old access token still valid: %d", c)
	}
}

func TestRefreshKeepsSingleOnlineSession(t *testing.T) {
	var db *gorm.DB
	h := newBearerHarness(t, func(o *RuntimeOptions) { db = o.DB })
	_, refresh := h.login()
	h.refresh(refresh)
	var n int64
	db.Model(&models.OnlineUser{}).Count(&n)
	if n != 1 {
		t.Fatalf("refresh must update the session in place, found %d rows", n)
	}
}

func TestRefreshReplayBeyondGraceRevokesSession(t *testing.T) {
	var db *gorm.DB
	h := newBearerHarness(t, func(o *RuntimeOptions) { db = o.DB })
	_, refresh := h.login()

	code, data := h.refresh(refresh)
	if code != 200 {
		t.Fatalf("first refresh=%d", code)
	}
	latest := data["token"].(string)

	// Racing tab inside the grace window: rejected, but the session survives.
	if c, _ := h.refresh(refresh); c != 401 {
		t.Fatalf("reused token must be rejected, got %d", c)
	}
	if c, _ := h.do("GET", "/api/v1/auth/me", "", latest); c != 200 {
		t.Fatalf("grace-window reuse killed the session: %d", c)
	}

	// Replay after the grace window means theft: the whole session is revoked.
	db.Model(&models.OnlineUser{}).Where("1 = 1").Update("updated_at", time.Now().Add(-time.Minute))
	if c, _ := h.refresh(refresh); c != 401 {
		t.Fatalf("replayed token accepted: %d", c)
	}
	if c, _ := h.do("GET", "/api/v1/auth/me", "", latest); c != 401 {
		t.Fatalf("session survived a replay: %d", c)
	}
}

func TestRefreshRejectsExpiredAndUnknownTokens(t *testing.T) {
	var db *gorm.DB
	h := newBearerHarness(t, func(o *RuntimeOptions) { db = o.DB })
	_, refresh := h.login()

	for _, bad := range []string{"", "short", strings.Repeat("a", 64)} {
		if c, _ := h.refresh(bad); c != 401 {
			t.Fatalf("token %q accepted: %d", bad, c)
		}
	}
	db.Model(&models.OnlineUser{}).Where("1 = 1").Update("refresh_expires", time.Now().Add(-time.Minute))
	if c, _ := h.refresh(refresh); c != 401 {
		t.Fatalf("expired refresh token accepted: %d", c)
	}
}

func TestRefreshRejectsDisabledUserAndLogoutRevokes(t *testing.T) {
	var db *gorm.DB
	h := newBearerHarness(t, func(o *RuntimeOptions) { db = o.DB })
	access, refresh := h.login()

	h.do("POST", "/api/v1/auth/logout", "", access)
	if c, _ := h.refresh(refresh); c != 401 {
		t.Fatalf("refresh after logout accepted: %d", c)
	}

	_, refresh = h.login()
	db.Model(&models.User{}).Where("username = ?", "refresher").Update("status", 0)
	if c, _ := h.refresh(refresh); c != 401 {
		t.Fatalf("disabled user refreshed: %d", c)
	}
}

func TestRefreshTokenIsNotAnAccessToken(t *testing.T) {
	h := newBearerHarness(t, nil)
	_, refresh := h.login()
	if c, _ := h.do("GET", "/api/v1/auth/me", "", refresh); c != 401 {
		t.Fatalf("refresh token worked as access token: %d", c)
	}
}

func TestMFAInterimTokenCannotAccessAPI(t *testing.T) {
	h := newBearerHarness(t, nil)
	access, _ := h.login()
	_, setup := h.do("POST", "/api/v1/auth/mfa/setup", `{}`, access)
	code, _ := totp.GenerateCode(setup["secret"].(string), time.Now())
	h.do("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, access)

	_, data := h.do("POST", "/api/v1/auth/login", `{"username":"refresher","password":"refresh-test-password"}`, "")
	mfa := data["mfa_token"].(string)
	if data["refresh_token"] != nil || data["token"] != nil {
		t.Fatal("MFA step must not hand out session tokens")
	}
	if c, _ := h.do("GET", "/api/v1/auth/me", "", mfa); c != 401 {
		t.Fatalf("interim MFA token reached the API: %d", c)
	}
	// /mfa/* skips the online-session lookup, so the purpose check is the only
	// thing stopping a password-only token from managing MFA.
	for _, path := range []string{"/api/v1/auth/mfa/setup", "/api/v1/auth/mfa/disable", "/api/v1/auth/mfa/recovery-codes", "/api/v1/auth/mfa/enable"} {
		if c, _ := h.do("POST", path, `{"code":"`+code+`","password":"refresh-test-password"}`, mfa); c != 401 {
			t.Fatalf("interim MFA token reached %s: %d", path, c)
		}
	}
	// An ordinary access token must not be accepted where an MFA token is required.
	if c, _ := h.do("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`","mfa_token":"`+access+`"}`, access); c != 401 {
		t.Fatalf("access token accepted as MFA token: %d", c)
	}
	c, verified := h.do("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`, mfa)
	if c != 200 || len(verified["refresh_token"].(string)) != 64 {
		t.Fatalf("MFA verify must return a refresh token: %d %v", c, verified)
	}
}

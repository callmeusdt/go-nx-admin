package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStrictCookieEnrollmentAndMFA(t *testing.T) {
	opts := runtimeOptions(t)
	if dsn := os.Getenv("NX_TEST_DATABASE_DSN"); dsn != "" {
		admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		pool, err := admin.DB()
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()
		schema := fmt.Sprintf("nx_browser_%d", time.Now().UnixNano())
		if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
			t.Fatal(err)
		}
		defer admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		p, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		defer p.Close()
		opts.DB = db
	}
	opts.Config.Session.Cookie = true
	opts.Config.Session.Origin = opts.Config.CORS.Origins
	if err := MigrateSchema(opts.DB); err != nil {
		t.Fatal(err)
	}
	role := models.Role{Name: "System", Slug: "admin"}
	if err := opts.DB.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("strict-test-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "strict", Password: string(hash), RoleID: role.ID, Status: 1}
	if err := opts.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := opts.DB.Exec(`INSERT INTO admin_casbin_rule(ptype,v0,v1,v2) VALUES('p','admin','.*','.*')`).Error; err != nil {
		t.Fatal(err)
	}
	if os.Getenv("NX_BROWSER_QA_ADDR") != "" {
		// Browser QA serves built assets through its HTTPS proxy. Leave the test
		// control routes reachable rather than hiding them behind SPA fallback.
		opts.Extensions.EmbedFS = nil
	}
	r, err := NewRuntime(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(context.Background())
	if address := os.Getenv("NX_BROWSER_QA_ADDR"); address != "" {
		stopped := make(chan struct{}, 1)
		r.App.Post("/qa/age-reauth", func(c *fiber.Ctx) error {
			if err := opts.DB.Model(&models.BrowserSession{}).Where("user_id = ?", user.ID).Update("reauthenticated_at", time.Now().Add(-6*time.Minute)).Error; err != nil {
				return err
			}
			return c.SendStatus(200)
		})
		r.App.Post("/qa/stop", func(c *fiber.Ctx) error {
			select {
			case stopped <- struct{}{}:
			default:
			}
			return c.SendStatus(200)
		})
		go func() { _ = r.App.Listen(address) }()
		deadline := time.NewTimer(5 * time.Minute)
		defer deadline.Stop()
		select {
		case <-deadline.C:
		case <-stopped:
		}
		return
	}
	cookies := map[string]*http.Cookie{}
	request := func(method, path, body, origin string, csrf bool, want int) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		if csrf && cookies[middleware.CSRFCookie] != nil {
			req.Header.Set("X-CSRF-Token", cookies[middleware.CSRFCookie].Value)
		}
		res, err := r.App.Test(req, 5000)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s=%d want=%d %s", method, path, res.StatusCode, want, data)
		}
		for _, cookie := range res.Cookies() {
			if !cookie.Secure || cookie.Domain != "" || cookie.Path != "/" || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatal("unsafe cookie attributes")
			}
			if cookie.Name == middleware.SessionCookie && !cookie.HttpOnly {
				t.Fatal("session exposed to script")
			}
			cookies[cookie.Name] = cookie
		}
		var result map[string]any
		_ = json.Unmarshal(data, &result)
		return result
	}
	origin := opts.Config.Session.Origin
	login := `{"username":"strict","password":"strict-test-password"}`
	request("POST", "/api/v1/auth/login", login, "https://evil.test", false, 403)
	result := request("POST", "/api/v1/auth/login", login, origin, false, 200)
	if result["token"] != nil || result["enrollment_required"] != true {
		t.Fatal("browser login exposed token or skipped enrollment")
	}
	request("GET", "/api/v1/users", "", origin, false, 403)
	request("POST", "/api/v1/bim/login", "{}", origin, true, 403)
	request("POST", "/api/v1/auth/mfa/setup", "{}", origin, false, 403)
	setup := request("POST", "/api/v1/auth/mfa/setup", "{}", origin, true, 200)
	secret := setup["secret"].(string)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result = request("POST", "/api/v1/auth/mfa/enable", `{"code":"`+code+`"}`, origin, true, 200)
	codes := result["recovery_codes"].([]any)
	if len(codes) != 8 {
		t.Fatal("missing recovery codes")
	}
	request("GET", "/api/v1/users", "", origin, false, 200)
	request("GET", "/api/v1/unknown", "", origin, false, 404)
	request("GET", "/api/v1/media", "", origin, false, 404)
	request("GET", "/uploads/private.txt", "", origin, false, 404)
	request("POST", "/api/v1/auth/mfa/disable", "{}", origin, true, 403)
	request("POST", "/api/v1/auth/mfa/setup", "{}", origin, true, 403)
	request("POST", "/api/v1/auth/logout", "{}", origin, true, 200)
	request("GET", "/api/v1/auth/me", "", origin, false, 401)
	result = request("POST", "/api/v1/auth/login", login, origin, false, 200)
	if result["mfa_required"] != true || result["mfa_token"] != nil {
		t.Fatal("pending token exposed")
	}
	request("POST", "/api/v1/auth/mfa/setup", "{}", origin, true, 403)
	request("POST", "/api/v1/auth/mfa/recovery-codes", "{}", origin, true, 403)
	request("GET", "/api/v1/users", "", origin, false, 403)
	oldCookie := cookies[middleware.SessionCookie].Value
	request("POST", "/api/v1/auth/mfa/verify", `{"recovery_code":"`+codes[0].(string)+`"}`, origin, true, 200)
	if oldCookie == cookies[middleware.SessionCookie].Value {
		t.Fatal("MFA did not rotate credential")
	}
	request("GET", "/api/v1/auth/me", "", origin, false, 200)
	if err := opts.DB.Model(&models.BrowserSession{}).Where("user_id = ?", user.ID).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/v1/auth/me", "", origin, false, 401)
	request("POST", "/api/v1/auth/login", login, origin, false, 200)
	request("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`, origin, true, 200)
	if err := opts.DB.Model(&models.BrowserSession{}).Where("user_id = ?", user.ID).Update("reauthenticated_at", time.Now().Add(-6*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/v1/menus", `{}`, origin, true, 403)
	request("POST", "/api/v1/auth/unlock", `{"password":"strict-test-password"}`, origin, true, 200)
	request("POST", "/api/v1/menus", `{}`, origin, true, 403)
	request("POST", "/api/v1/auth/verify-password", `{"password":"wrong","code":"`+code+`"}`, origin, true, 401)
	request("POST", "/api/v1/auth/verify-password", `{"password":"strict-test-password","code":"`+code+`"}`, origin, true, 200)
	request("POST", "/api/v1/menus", `{"name":"Strict QA","path":"/strict-qa"}`, origin, false, 403)
	request("POST", "/api/v1/menus", `{"name":"Strict QA","path":"/strict-qa"}`, "https://evil.test", true, 403)
	request("POST", "/api/v1/menus", `{"name":"Strict QA","path":"/strict-qa"}`, origin, true, 200)
	var current models.BrowserSession
	if err := opts.DB.Where("token_hash = ?", middleware.Digest(cookies[middleware.SessionCookie].Value)).First(&current).Error; err != nil {
		t.Fatal(err)
	}
	request("DELETE", fmt.Sprintf("/api/v1/online-users/%d", current.ID), "", origin, true, 400)
	other := current
	other.ID = 0
	other.TokenHash = middleware.Digest(strings.Repeat("a", 64))
	if err := opts.DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	request("DELETE", fmt.Sprintf("/api/v1/online-users/%d", other.ID), "", origin, true, 200)
	savedCookie := cookies[middleware.SessionCookie]
	cookies[middleware.SessionCookie] = &http.Cookie{Name: middleware.SessionCookie, Value: strings.Repeat("a", 64)}
	request("GET", "/api/v1/auth/me", "", origin, false, 401)
	cookies[middleware.SessionCookie] = savedCookie
	request("POST", "/api/v1/auth/logout", "{}", origin, true, 200)
	request("POST", "/api/v1/auth/login", login, origin, false, 200)
	request("POST", "/api/v1/auth/mfa/verify", `{"recovery_code":"`+codes[0].(string)+`"}`, origin, true, 401)
	for i := 0; i < 4; i++ {
		request("POST", "/api/v1/auth/mfa/verify", `{"code":"bad"}`, origin, true, 401)
	}
	request("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`, origin, true, 401)
	request("POST", "/api/v1/auth/login", login, origin, false, 200)
	if err := opts.DB.Model(&models.BrowserSession{}).Where("user_id = ?", user.ID).Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/v1/auth/mfa/verify", `{"code":"`+code+`"}`, origin, true, 401)
}

func TestCookieRuntimeRejectsInvalidOrigins(t *testing.T) {
	for _, origin := range []string{"", "http://admin.example.test", "https://evil.test", "https://admin.example.test/", "https://admin.example.test?query=1"} {
		t.Run(origin, func(t *testing.T) {
			opts := runtimeOptions(t)
			opts.Config.Session.Cookie = true
			opts.Config.Session.Origin = origin
			if _, err := NewRuntime(opts); err == nil || !strings.Contains(err.Error(), "exact HTTPS origin") {
				t.Fatalf("expected origin validation, got %v", err)
			}
		})
	}
}

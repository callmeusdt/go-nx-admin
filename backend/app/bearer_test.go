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

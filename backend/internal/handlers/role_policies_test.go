package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type policyFailureAdapter struct{ fail string }

func (a *policyFailureAdapter) LoadPolicy(model.Model) error { return nil }
func (a *policyFailureAdapter) SavePolicy(model.Model) error {
	if a.fail == "save" {
		return errors.New("private database detail")
	}
	return nil
}
func (a *policyFailureAdapter) AddPolicy(string, string, []string) error {
	if a.fail == "add" {
		return errors.New("private database detail")
	}
	return nil
}
func (a *policyFailureAdapter) RemovePolicy(string, string, []string) error { return nil }
func (a *policyFailureAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
	if a.fail == "remove" {
		return errors.New("private database detail")
	}
	return nil
}

func newPolicyTestEnforcer(t *testing.T) (*casbin.Enforcer, *policyFailureAdapter) {
	t.Helper()
	m, err := model.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && regexMatch(r.obj, p.obj) && regexMatch(r.act, p.act)`)
	if err != nil {
		t.Fatal(err)
	}
	a := &policyFailureAdapter{}
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddPolicy("reader", "^/existing$", "^GET$"); err != nil {
		t.Fatal(err)
	}
	return e, a
}

func newPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "roles.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Role{}); err != nil {
		t.Fatal(err)
	}
	for _, r := range []models.Role{{Name: "Reader", Slug: "reader"}, {Name: "Root", Slug: "root", IsSystem: true}} {
		if err := db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func putKeys(t *testing.T, e *casbin.Enforcer, slug string, keys []string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"keys": keys})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Put("/roles/:slug/permissions", SaveRolePermissions(e, newPolicyTestDB(t)))
	req := httptest.NewRequest("PUT", "/roles/"+slug+"/permissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(data)
}

func TestSaveRolePermissionsRejectsInvalidBeforeRemovingExisting(t *testing.T) {
	for _, tc := range []struct {
		slug string
		keys []string
		want int
	}{
		{"reader", []string{"nope:nope"}, 400},
		{"reader", []string{"users:list", "nope:nope"}, 400},
		{"root", []string{"users:list"}, 403},
		{"ghost", []string{"users:list"}, 404},
	} {
		e, _ := newPolicyTestEnforcer(t)
		code, _ := putKeys(t, e, tc.slug, tc.keys)
		if code != tc.want {
			t.Errorf("slug=%s keys=%v status=%d want %d", tc.slug, tc.keys, code, tc.want)
		}
		if !e.HasPolicy("reader", "^/existing$", "^GET$") {
			t.Error("rejected replacement removed existing grant")
		}
	}
}

func TestSaveRolePermissionsReportsStorageFailure(t *testing.T) {
	e, adapter := newPolicyTestEnforcer(t)
	adapter.fail = "replace"
	code, body := putKeys(t, e, "reader", []string{"users:list"})
	if code != 500 {
		t.Errorf("failed storage status=%d", code)
	}
	if strings.Contains(body, "private database detail") {
		t.Error("storage details leaked")
	}
	if !e.HasPolicy("reader", "^/existing$", "^GET$") || len(e.GetFilteredPolicy(0, "reader")) != 1 {
		t.Error("failed atomic replacement changed live authorization")
	}
}

func TestSaveRolePermissionsReplacesValidPolicies(t *testing.T) {
	e, _ := newPolicyTestEnforcer(t)
	code, _ := putKeys(t, e, "reader", []string{"users:list"})
	expanded := models.ExpandPermission("users:list")
	if code != 200 || len(expanded) == 0 || e.HasPolicy("reader", "^/existing$", "^GET$") {
		t.Fatal("valid replacement failed")
	}
	for _, p := range expanded {
		if !e.HasPolicy("reader", p[1], p[0]) {
			t.Errorf("expanded policy %v not stored as (path, method)", p)
		}
	}
}

func (a *policyFailureAdapter) UpdatePolicy(string, string, []string, []string) error {
	return errors.New("unused")
}
func (a *policyFailureAdapter) UpdatePolicies(string, string, [][]string, [][]string) error {
	return errors.New("unused")
}
func (a *policyFailureAdapter) UpdateFilteredPolicies(string, string, [][]string, int, ...string) ([][]string, error) {
	if a.fail == "replace" {
		return nil, errors.New("private database detail")
	}
	return [][]string{{"reader", "^/existing$", "^GET$"}}, nil
}
func TestSaveRolePermissionsCanClearGrants(t *testing.T) {
	e, _ := newPolicyTestEnforcer(t)
	code, _ := putKeys(t, e, "reader", []string{})
	if code != 200 || len(e.GetFilteredPolicy(0, "reader")) != 0 {
		t.Fatal("explicit empty replacement failed")
	}
}

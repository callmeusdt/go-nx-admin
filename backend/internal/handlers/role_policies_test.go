package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gofiber/fiber/v2"
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

func putPolicies(t *testing.T, e *casbin.Enforcer, policies [][]string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"policies": policies})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Put("/roles/:slug/permissions", SaveRolePolicies(e))
	req := httptest.NewRequest("PUT", "/roles/reader/permissions", bytes.NewReader(body))
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

func TestSaveRolePoliciesRejectsInvalidBeforeRemovingExisting(t *testing.T) {
	for _, policies := range [][][]string{
		{{strings.Repeat("x", 101), "GET"}}, {{"/x", strings.Repeat("x", 101)}},
		{{"/x"}}, {{"/x", "GET", "extra"}}, {{"", "GET"}}, {{"/x", ""}},
		{{"[", "GET"}}, {{"/x", "["}},
	} {
		e, _ := newPolicyTestEnforcer(t)
		code, _ := putPolicies(t, e, policies)
		if code != 400 {
			t.Errorf("invalid policy status=%d", code)
		}
		if !e.HasPolicy("reader", "^/existing$", "^GET$") {
			t.Error("invalid replacement removed existing grant")
		}
	}
}

func TestSaveRolePoliciesReportsStorageFailure(t *testing.T) {
	e, adapter := newPolicyTestEnforcer(t)
	adapter.fail = "replace"
	code, body := putPolicies(t, e, [][]string{{"^/next$", "^GET$"}})
	if code != 500 {
		t.Errorf("failed storage status=%d", code)
	}
	if strings.Contains(body, "private database detail") {
		t.Error("storage details leaked")
	}
	if !e.HasPolicy("reader", "^/existing$", "^GET$") || e.HasPolicy("reader", "^/next$", "^GET$") {
		t.Error("failed atomic replacement changed live authorization")
	}
}

func TestSaveRolePoliciesReplacesValidPolicies(t *testing.T) {
	e, _ := newPolicyTestEnforcer(t)
	code, _ := putPolicies(t, e, [][]string{{"^/next$", "^GET$"}})
	if code != 200 || !e.HasPolicy("reader", "^/next$", "^GET$") || e.HasPolicy("reader", "^/existing$", "^GET$") {
		t.Fatal("valid replacement failed")
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
func TestSaveRolePoliciesCanClearGrants(t *testing.T) {
	e, _ := newPolicyTestEnforcer(t)
	code, _ := putPolicies(t, e, [][]string{})
	if code != 200 || len(e.GetFilteredPolicy(0, "reader")) != 0 {
		t.Fatal("explicit empty replacement failed")
	}
}

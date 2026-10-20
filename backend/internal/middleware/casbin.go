package middleware

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/models"
	"gorm.io/gorm"
)

const rbacModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && anchoredMatch(r.obj, p.obj) && anchoredMatch(r.act, p.act)
`

var (
	anchoredMu    sync.RWMutex
	anchoredCache = map[string]*regexp.Regexp{}
)

// anchoredMatch reports whether the whole of key matches the policy pattern.
// Casbin's built-in regexMatch is unanchored, so "/api/v1/roles" also matched
// "/api/v1/roles/x/permissions". A pattern that does not compile matches
// nothing instead of panicking the request.
func anchoredMatch(key, pattern string) bool {
	anchoredMu.RLock()
	re, ok := anchoredCache[pattern]
	anchoredMu.RUnlock()
	if !ok {
		compiled, err := regexp.Compile(`^(?:` + pattern + `)$`)
		if err != nil {
			compiled = nil
		}
		anchoredMu.Lock()
		if len(anchoredCache) > 4096 {
			anchoredCache = map[string]*regexp.Regexp{}
		}
		anchoredCache[pattern] = compiled
		anchoredMu.Unlock()
		re = compiled
	}
	return re != nil && re.MatchString(key)
}

// newEnforcer builds the enforcer without loading policies: the matcher
// function has to be registered before any policy is evaluated, and Casbin
// resets its function table during initialisation.
func newEnforcer(m model.Model, a persist.Adapter) (*casbin.Enforcer, error) {
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}
	e.AddFunction("anchoredMatch", func(args ...interface{}) (interface{}, error) {
		key, _ := args[0].(string)
		pattern, _ := args[1].(string)
		return anchoredMatch(key, pattern), nil
	})
	e.SetAdapter(a)
	return e, nil
}

// NormalizeLegacyPolicies repairs rules saved as (role, method, path) by older
// versions of the permission editor, which could never match a request. It is
// idempotent and only touches rows whose method column holds an HTTP verb and
// whose path column looks like a path.
func NormalizeLegacyPolicies(db *gorm.DB) error {
	const table = "admin_casbin_rule"
	if !db.Migrator().HasTable(table) {
		return nil
	}
	var rows []struct {
		ID uint
		V1 string
		V2 string
	}
	if err := db.Table(table).Select("id, v1, v2").
		Where("ptype = ? AND v1 IN ?", "p", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}).
		Find(&rows).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			if !strings.HasPrefix(r.V2, "/") {
				continue
			}
			if err := tx.Table(table).Where("id = ?", r.ID).
				Updates(map[string]any{"v1": r.V2, "v2": r.V1}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpgradeWildcardPolicies rewrites role policies saved with the old ".*"
// wildcard (which matched across path segments, and unanchored) to the
// single-segment form. The admin ".*" grant is left alone. Idempotent.
func UpgradeWildcardPolicies(db *gorm.DB) error {
	const table = "admin_casbin_rule"
	if !db.Migrator().HasTable(table) {
		return nil
	}
	var rows []struct {
		ID uint
		V1 string
	}
	if err := db.Table(table).Select("id, v1").
		Where("ptype = ? AND v1 LIKE ?", "p", "/api/%").Find(&rows).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			next := models.LegacyWildcardPolicy(r.V1)
			if next == r.V1 {
				continue
			}
			if err := tx.Table(table).Where("id = ?", r.ID).Update("v1", next).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func InitCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
	if err := NormalizeLegacyPolicies(db); err != nil {
		return nil, err
	}
	if err := UpgradeWildcardPolicies(db); err != nil {
		return nil, err
	}
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return nil, err
	}
	a, err := gormadapter.NewAdapterByDBUseTableName(db, "admin", "casbin_rule")
	if err != nil {
		return nil, err
	}
	e, err := newEnforcer(m, a)
	if err != nil {
		return nil, err
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, err
	}

	if !e.HasPolicy("admin", ".*", ".*") {
		e.AddPolicy("admin", ".*", ".*")
	}

	if !e.HasPolicy("editor", "/api/v1/auth/me", "GET") {
		e.AddPolicy("editor", "/api/v1/auth/me", "GET")
		e.AddPolicy("editor", "/api/v1/auth/menus", "GET")
		e.AddPolicy("editor", "/api/v1/dashboard", "GET")
		e.AddPolicy("editor", "/api/v1/users", "GET")
	}

	if err := e.SavePolicy(); err != nil {
		return nil, err
	}
	return e, nil
}

// LoadCasbin loads existing policies without schema changes or seed writes.
func LoadCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return nil, err
	}
	connection := db.Session(&gorm.Session{})
	gormadapter.TurnOffAutoMigrate(connection)
	a, err := gormadapter.NewAdapterByDBUseTableName(connection, "admin", "casbin_rule")
	if err != nil {
		return nil, err
	}
	e, err := newEnforcer(m, a)
	if err != nil {
		return nil, err
	}
	if err := e.LoadPolicy(); err != nil {
		return nil, err
	}
	return e, nil
}

func CasbinMiddleware(e *casbin.Enforcer, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		if permissionExempt(method, path) {
			return c.Next()
		}

		role, ok := c.Locals("role").(string)
		if !ok || role == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
		}

		allowed, err := e.Enforce(role, path, method)
		if err != nil {
			log.Printf("casbin enforce error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "enforce error"})
		}
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "permission denied"})
		}

		err = c.Next()

		username, _ := c.Locals("username").(string)
		ip := ClientIP(c)
		status := c.Response().StatusCode()
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		module := "api"
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 3 {
			module = parts[2]
		}
		// Fiber strings may alias pooled request buffers. Own every string before
		// the handler returns so the asynchronous writer cannot observe reuse.
		record := models.AuditLog{
			Operator: strings.Clone(username),
			Title:    method + " " + path,
			Module:   strings.Clone(module),
			Path:     strings.Clone(path),
			Method:   strings.Clone(method),
			IP:       strings.Clone(ip),
			Status:   status,
			Duration: time.Since(start).Microseconds(),
			Error:    strings.Clone(errMsg),
		}
		go func() {
			if err := db.Create(&record).Error; err != nil {
				log.Printf("audit log error: %v", err)
			}
		}()

		return err
	}
}

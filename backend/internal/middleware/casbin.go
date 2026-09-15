package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
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
m = g(r.sub, p.sub) && regexMatch(r.obj, p.obj) && regexMatch(r.act, p.act)
`

func InitCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(rbacModel)
	if err != nil {
		return nil, err
	}
	a, err := gormadapter.NewAdapterByDBUseTableName(db, "admin", "casbin_rule")
	if err != nil {
		return nil, err
	}
	e, err := casbin.NewEnforcer(m, a)
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
		e.AddPolicy("editor", "/api/v1/auth/profile", "PUT")
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
	return casbin.NewEnforcer(m, a)
}

func CasbinMiddleware(e *casbin.Enforcer, db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		if CookieMode() {
			switch method + " " + path {
			case "POST /api/v1/auth/login", "GET /api/v1/auth/captcha", "POST /api/v1/auth/logout", "GET /api/v1/auth/me", "POST /api/v1/auth/verify-password", "POST /api/v1/auth/unlock", "POST /api/v1/auth/mfa/verify", "POST /api/v1/auth/mfa/setup", "POST /api/v1/auth/mfa/enable", "POST /api/v1/auth/mfa/disable", "POST /api/v1/auth/mfa/recovery-codes":
				return c.Next()
			}
		} else if !strings.HasPrefix(path, "/api/") ||
			strings.HasSuffix(path, "/login") ||
			strings.HasSuffix(path, "/captcha") ||
			strings.HasSuffix(path, "/verify-password") ||
			strings.Contains(path, "/mfa/") {
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
		ip := c.IP()
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

package middleware

import "github.com/gofiber/fiber/v2"

// Routes are identified by exact "METHOD /path". Suffix or substring matching
// let unrelated routes (for example DELETE /users/:id/mfa/) inherit the
// exemptions of the auth endpoints.
type routeSet map[string]bool

func routeKey(method, path string) string { return method + " " + path }

func (s routeSet) has(method, path string) bool { return s[routeKey(method, path)] }

// tokenlessRoutes are reachable before any credential exists.
var tokenlessRoutes = routeSet{
	"POST /api/v1/auth/login":   true,
	"GET /api/v1/auth/captcha":  true,
	"POST /api/v1/auth/refresh": true,
}

const mfaVerifyRoute = "POST /api/v1/auth/mfa/verify"

// selfServiceRoutes only read or change the caller's own account, so any
// signed-in user may call them without a role permission. Relying on a role
// permission for these used to work only by accident of the unanchored
// matcher and left roles without the exact permission with no sidebar menu.
var selfServiceRoutes = routeSet{
	"GET /api/v1/auth/me":                  true,
	"GET /api/v1/auth/menus":               true,
	"GET /api/v1/auth/my-ip":               true,
	"GET /api/v1/auth/permissions":         true,
	"POST /api/v1/auth/logout":             true,
	"POST /api/v1/auth/unlock":             true,
	"POST /api/v1/auth/verify-password":    true,
	"POST /api/v1/auth/password":           true,
	"POST /api/v1/auth/mfa/verify":         true,
	"POST /api/v1/auth/mfa/setup":          true,
	"POST /api/v1/auth/mfa/enable":         true,
	"POST /api/v1/auth/mfa/disable":        true,
	"POST /api/v1/auth/mfa/recovery-codes": true,
}

func permissionExempt(method, path string) bool {
	return tokenlessRoutes.has(method, path) || selfServiceRoutes.has(method, path)
}

func isTokenless(c *fiber.Ctx) bool { return tokenlessRoutes.has(c.Method(), c.Path()) }

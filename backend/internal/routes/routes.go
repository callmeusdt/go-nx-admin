package routes

import (
	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/handlers"
	"go-nx-admin/internal/middleware"
	"gorm.io/gorm"
)

func Register(app *fiber.App, db *gorm.DB, enforcer *casbin.Enforcer, extraRoutes func(api fiber.Router, db *gorm.DB, enforcer *casbin.Enforcer)) {
	api := app.Group("/api/v1", middleware.JWTAuth(db), middleware.CasbinMiddleware(enforcer, db))

	api.Post("/auth/login", handlers.Login(db))
	api.Post("/auth/logout", handlers.Logout(db))
	api.Post("/auth/verify-password", handlers.VerifyPassword(db))
	api.Get("/auth/me", handlers.Me(db))
	api.Put("/auth/profile", handlers.UpdateProfile(db))
	api.Get("/auth/menus", handlers.MyMenus(db))
	api.Get("/auth/my-ip", handlers.MyIP)
	api.Get("/auth/captcha", handlers.GenerateCaptcha)
	api.Post("/auth/mfa/setup", handlers.MFASetup(db))
	api.Post("/auth/mfa/enable", handlers.MFAEnable(db))
	api.Post("/auth/mfa/disable", handlers.MFADisable(db))
	api.Post("/auth/mfa/recovery-codes", handlers.MFARecoveryCodes(db))
	api.Post("/auth/mfa/verify", handlers.MFAVerify(db))

	api.Get("/users", handlers.ListUsers(db))
	api.Post("/users", handlers.CreateUser(db))
	api.Put("/users/:id", handlers.UpdateUser(db))
	api.Delete("/users/:id", handlers.DeleteUser(db))
	api.Delete("/users/:id/mfa", handlers.ClearUserMFA(db))
	api.Get("/users/:id/ip-whitelists", handlers.ListUserIPWhitelists(db))
	api.Post("/users/:id/ip-whitelists", handlers.CreateUserIPWhitelist(db))
	api.Put("/users/:id/ip-whitelists/:wid", handlers.UpdateUserIPWhitelist(db))
	api.Delete("/users/:id/ip-whitelists/:wid", handlers.DeleteUserIPWhitelist(db))

	api.Get("/roles", handlers.ListRoles(db))
	api.Post("/roles", handlers.CreateRole(db))
	api.Put("/roles/:id", handlers.UpdateRole(db))
	api.Delete("/roles/:id", handlers.DeleteRole(db))
	api.Get("/roles/:id/menus", handlers.RoleMenuIDs(db))
	api.Put("/roles/:id/menus", handlers.SaveRoleMenus(db))
	api.Get("/roles/:slug/policies", handlers.ListRolePolicies(enforcer))
	api.Put("/roles/:slug/policies", handlers.SaveRolePolicies(enforcer))
	api.Get("/roles/:slug/permissions", handlers.ListRolePermissions(enforcer))
	api.Put("/roles/:slug/permissions", handlers.SaveRolePolicies(enforcer))

	api.Get("/menus", handlers.ListMenus(db))
	api.Post("/menus", handlers.CreateMenu(db))
	api.Put("/menus/:id", handlers.UpdateMenu(db))
	api.Delete("/menus/:id", handlers.DeleteMenu(db))

	api.Get("/apis", handlers.ListApis(db))
	api.Post("/apis", handlers.CreateApi(db))
	api.Put("/apis/:id", handlers.UpdateApi(db))
	api.Delete("/apis/:id", handlers.DeleteApi(db))

	api.Get("/audit-logs", handlers.ListAuditLogs(db))
	api.Get("/login-logs", handlers.ListLoginLogs(db))
	api.Get("/online-users", handlers.ListOnlineUsers(db))
	api.Delete("/online-users/:id", handlers.KickOnlineUser(db))

	api.Get("/dashboard/stats", handlers.DashboardStats(db))

	api.Get("/media", handlers.ListMedia(db))
	api.Post("/media/upload", handlers.UploadMedia(db))
	api.Delete("/media/:id", handlers.DeleteMedia(db))

	api.Get("/system-configs", handlers.ListSystemConfigs(db))
	api.Put("/system-configs", handlers.UpdateSystemConfigs(db))

	if extraRoutes != nil {
		extraRoutes(api, db, enforcer)
	}
}

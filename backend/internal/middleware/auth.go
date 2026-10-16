package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/models"
	"gorm.io/gorm"
)

func JWTAuth(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if CookieMode() {
			return BrowserAuth(db)(c)
		}
		if strings.HasSuffix(c.Path(), "/login") || strings.HasSuffix(c.Path(), "/captcha") || strings.HasSuffix(c.Path(), "/auth/refresh") {
			return c.Next()
		}

		tokenStr := c.Get("Authorization")
		if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "missing or invalid token",
			})
		}

		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid or expired token",
			})
		}

		isMfaPath := strings.Contains(c.Path(), "/mfa/")
		// An interim MFA token proves only the password step. It must never act
		// as an access token, and the only thing it may reach is MFA verify.
		if claims.Purpose == PurposeMFA && !strings.HasSuffix(c.Path(), "/mfa/verify") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "invalid or expired token"})
		}

		var count int64
		if !isMfaPath {
			if err := db.Model(&models.OnlineUser{}).Where("token = ?", tokenStr).Count(&count).Error; err != nil || count == 0 {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "session expired"})
			}
		}

		if !isMfaPath {
			if blocked, err := mustChangeBlocked(db, claims.UserID, c.Method(), c.Path()); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "internal error"})
			} else if blocked {
				return passwordChangeRequired(c)
			}
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("role", claims.Role)
		c.Locals("role_id", claims.RoleID)
		c.Locals("token", tokenStr)
		return c.Next()
	}
}

// passwordChangePaths is everything an account flagged must_change_password may
// call: enough to learn it is flagged, change the password, lock or sign out.
var passwordChangePaths = map[string]bool{
	"GET /api/v1/auth/me":               true,
	"POST /api/v1/auth/password":        true,
	"POST /api/v1/auth/logout":          true,
	"POST /api/v1/auth/refresh":         true,
	"POST /api/v1/auth/verify-password": true,
	"POST /api/v1/auth/unlock":          true,
}

func passwordChangeRequired(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"code": "PASSWORD_CHANGE_REQUIRED", "message": "PASSWORD_CHANGE_REQUIRED"})
}

func mustChangeBlocked(db *gorm.DB, userID uint, method, path string) (bool, error) {
	if passwordChangePaths[method+" "+path] {
		return false, nil
	}
	var user models.User
	if err := db.Select("id", "must_change_password").First(&user, userID).Error; err != nil {
		return false, err
	}
	return user.MustChangePassword, nil
}

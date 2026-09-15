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
		if strings.HasSuffix(c.Path(), "/login") || strings.HasSuffix(c.Path(), "/captcha") {
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

		var count int64
		isMfaPath := strings.Contains(c.Path(), "/mfa/")
		if !isMfaPath {
			if err := db.Model(&models.OnlineUser{}).Where("token = ?", tokenStr).Count(&count).Error; err != nil || count == 0 {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "session expired"})
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

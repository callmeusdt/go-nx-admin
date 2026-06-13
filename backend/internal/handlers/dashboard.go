package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/models"
)

func DashboardStats(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var totalUsers, onlineUsers, totalRoles, todayLogins int64
		db.Model(&models.User{}).Count(&totalUsers)
		db.Model(&models.OnlineUser{}).Count(&onlineUsers)
		db.Model(&models.Role{}).Count(&totalRoles)
		today := time.Now().Format("2006-01-02")
		db.Model(&models.LoginLog{}).Where("created_at >= ?", today).Count(&todayLogins)
		return c.JSON(fiber.Map{
			"total_users":    totalUsers,
			"online_users":   onlineUsers,
			"total_roles":    totalRoles,
			"today_logins":   todayLogins,
			"today_api_calls": 0,
		})
	}
}

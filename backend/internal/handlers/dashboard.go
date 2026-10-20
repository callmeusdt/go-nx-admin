package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/models"
)

// DashboardStatsProvider lets a downstream project add or override dashboard
// figures. Its keys are merged over the built-in ones, so returning
// "total_users" replaces the stock value.
type DashboardStatsProvider func(db *gorm.DB) map[string]any

// ExtraDashboardStats is set by the app layer; nil means built-in figures only.
var ExtraDashboardStats DashboardStatsProvider

func DashboardStats(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var totalUsers, onlineUsers, totalRoles, todayLogins int64
		today := time.Now().Format("2006-01-02")
		for _, q := range []struct {
			query *gorm.DB
			into  *int64
		}{
			{db.Model(&models.User{}), &totalUsers},
			{db.Model(&models.OnlineUser{}), &onlineUsers},
			{db.Model(&models.Role{}), &totalRoles},
			{db.Model(&models.LoginLog{}).Where("created_at >= ?", today), &todayLogins},
		} {
			if err := q.query.Count(q.into).Error; err != nil {
				return dbError(c, err)
			}
		}
		stats := fiber.Map{
			"total_users":  totalUsers,
			"online_users": onlineUsers,
			"total_roles":  totalRoles,
			"today_logins": todayLogins,
		}
		if provider := ExtraDashboardStats; provider != nil {
			mergeDashboardStats(stats, provider, db)
		}
		return c.JSON(stats)
	}
}

// mergeDashboardStats isolates a faulty extension: a panic there must not take
// the whole dashboard (and the built-in figures) down with it.
func mergeDashboardStats(stats fiber.Map, provider DashboardStatsProvider, db *gorm.DB) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("dashboard stats provider panicked: %v", r)
		}
	}()
	for k, v := range provider(db) {
		stats[k] = v
	}
}

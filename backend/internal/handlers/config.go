package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/models"
)

func ListSystemConfigs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var items []models.SystemConfig
		db.Order("sort asc").Find(&items)

		grouped := make(map[string][]models.SystemConfig)
		for _, item := range items {
			grouped[item.Group] = append(grouped[item.Group], item)
		}
		return c.JSON(fiber.Map{"data": grouped})
	}
}

func UpdateSystemConfigs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body map[string]string
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		for key, value := range body {
			db.Model(&models.SystemConfig{}).Where("key = ?", key).Update("value", value)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

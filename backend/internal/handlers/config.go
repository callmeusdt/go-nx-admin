package handlers

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"go-nx-admin/internal/models"
)

func ListSystemConfigs(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var items []models.SystemConfig
		if err := db.Order("sort asc").Find(&items).Error; err != nil {
			return dbError(c, err)
		}

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
		var items []models.SystemConfig
		if err := db.Select("key").Find(&items).Error; err != nil {
			return dbError(c, err)
		}
		known := make(map[string]bool, len(items))
		for _, item := range items {
			known[item.Key] = true
		}
		for key := range body {
			if !known[key] {
				return c.Status(400).JSON(fiber.Map{"message": "unknown config key: " + key})
			}
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			for key, value := range body {
				// "key" is reserved in MySQL, so the column is quoted by GORM
				// instead of being written into a SQL string.
				cond := clause.Eq{Column: clause.Column{Name: "key"}, Value: key}
				if err := tx.Model(&models.SystemConfig{}).Where(cond).Update("value", value).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return dbError(c, err)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

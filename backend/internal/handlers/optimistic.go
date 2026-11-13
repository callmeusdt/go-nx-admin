package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ErrVersionConflict is returned when a write was based on a stale resource
// representation.  The response code is deliberately stable: clients must
// not match the database or driver error text.
var ErrVersionConflict = errors.New("resource version conflict")

const VersionConflictCode = "RESOURCE_VERSION_CONFLICT"

func versionConflict(c *fiber.Ctx) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"code":    VersionConflictCode,
		"message": "resource has changed",
	})
}

// updateVersioned performs a compare-and-swap update. GORM's RowsAffected is
// the only result used for the decision, so the check is atomic at the
// database level and works on SQLite, MySQL/MariaDB, and PostgreSQL.
func updateVersioned(db *gorm.DB, model any, id uint, version uint, values map[string]any) error {
	values["version"] = gorm.Expr("version + ?", 1)
	query := db.Model(model).Where("id = ?", id)
	if version != 0 {
		query = query.Where("version = ?", version)
	}
	result := query.Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func deleteVersioned(db *gorm.DB, model any, id uint, version uint) error {
	query := db.Where("id = ?", id)
	if version != 0 {
		query = query.Where("version = ?", version)
	}
	result := query.Delete(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return nil
}

func requestVersion(c *fiber.Ctx) (uint, error) {
	var req struct {
		Version uint `json:"version"`
	}
	if len(c.Body()) == 0 {
		return 0, nil
	}
	if err := c.BodyParser(&req); err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	return req.Version, nil
}

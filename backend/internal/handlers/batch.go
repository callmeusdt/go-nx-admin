package handlers

import (
	"errors"
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"go-nx-admin/internal/storage"
)

// maxBatch bounds a single request so one call cannot lock a table or run long.
const maxBatch = 200

type batchFailure struct {
	ID     uint   `json:"id"`
	Reason string `json:"reason"`
}

// batchResult reports each id independently. A batch is deliberately not
// all-or-nothing: protected rows (yourself, system roles) must not block the
// rest, and the caller needs to know exactly which ones were skipped.
type batchResult struct {
	Deleted []uint         `json:"deleted"`
	Failed  []batchFailure `json:"failed"`
}

func (r *batchResult) fail(id uint, reason string) {
	r.Failed = append(r.Failed, batchFailure{ID: id, Reason: reason})
}

// normaliseIDs validates and de-duplicates an id list.
func normaliseIDs(in []uint) ([]uint, error) {
	seen := make(map[uint]bool, len(in))
	out := make([]uint, 0, len(in))
	for _, id := range in {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, errors.New("ids is required")
	}
	if len(out) > maxBatch {
		return nil, fmt.Errorf("at most %d ids per request", maxBatch)
	}
	return out, nil
}

// parseBatchIDs reads {"ids": [...]} from the request body.
func parseBatchIDs(c *fiber.Ctx) ([]uint, error) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return nil, errors.New("invalid request")
	}
	return normaliseIDs(req.IDs)
}

func batchBadRequest(c *fiber.Ctx, err error) error {
	return c.Status(400).JSON(fiber.Map{"message": err.Error()})
}

// BatchDeleteUsers: DELETE /users {ids}
func BatchDeleteUsers(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := parseBatchIDs(c)
		if err != nil {
			return batchBadRequest(c, err)
		}
		current, _ := c.Locals("user_id").(uint)

		var existing []models.User
		if err := db.Select("id").Where("id IN ?", ids).Find(&existing).Error; err != nil {
			return dbError(c, err)
		}
		found := make(map[uint]bool, len(existing))
		for _, u := range existing {
			found[u.ID] = true
		}

		res := batchResult{Deleted: []uint{}, Failed: []batchFailure{}}
		for _, id := range ids {
			switch {
			case id == current:
				res.fail(id, "cannot delete current user")
			case !found[id]:
				res.fail(id, "not found")
			default:
				if err := deleteUserCascade(db, id); err != nil {
					res.fail(id, "delete failed")
				} else {
					res.Deleted = append(res.Deleted, id)
				}
			}
		}
		return c.JSON(res)
	}
}

// deleteUserCascade removes the user and ends their sessions in one transaction,
// so a deleted account can not keep a live token.
func deleteUserCascade(db *gorm.DB, id uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&models.OnlineUser{}).Error; err != nil {
			return err
		}
		if middleware.CookieMode() {
			if err := tx.Where("user_id = ?", id).Delete(&models.BrowserSession{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&models.User{}, id).Error
	})
}

// BatchUpdateUserStatus: PUT /users/status {ids, status}
func BatchUpdateUserStatus(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			IDs    []uint `json:"ids"`
			Status *int   `json:"status"`
		}
		if err := c.BodyParser(&req); err != nil {
			return batchBadRequest(c, errors.New("invalid request"))
		}
		if req.Status == nil || (*req.Status != 0 && *req.Status != 1) {
			return batchBadRequest(c, errors.New("status must be 0 or 1"))
		}
		ids, err := normaliseIDs(req.IDs)
		if err != nil {
			return batchBadRequest(c, err)
		}
		current, _ := c.Locals("user_id").(uint)

		var existing []models.User
		if err := db.Select("id").Where("id IN ?", ids).Find(&existing).Error; err != nil {
			return dbError(c, err)
		}
		found := make(map[uint]bool, len(existing))
		for _, u := range existing {
			found[u.ID] = true
		}

		type result struct {
			Updated []uint         `json:"updated"`
			Failed  []batchFailure `json:"failed"`
		}
		res := result{Updated: []uint{}, Failed: []batchFailure{}}
		for _, id := range ids {
			switch {
			case id == current && *req.Status == 0:
				res.Failed = append(res.Failed, batchFailure{ID: id, Reason: "cannot disable current user"})
			case !found[id]:
				res.Failed = append(res.Failed, batchFailure{ID: id, Reason: "not found"})
			default:
				err := db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Model(&models.User{}).Where("id = ?", id).Update("status", *req.Status).Error; err != nil {
						return err
					}
					if *req.Status == 0 {
						// A disabled account must lose its live sessions at once.
						if err := tx.Where("user_id = ?", id).Delete(&models.OnlineUser{}).Error; err != nil {
							return err
						}
						if middleware.CookieMode() {
							return tx.Where("user_id = ?", id).Delete(&models.BrowserSession{}).Error
						}
					}
					return nil
				})
				if err != nil {
					res.Failed = append(res.Failed, batchFailure{ID: id, Reason: "update failed"})
				} else {
					res.Updated = append(res.Updated, id)
				}
			}
		}
		return c.JSON(res)
	}
}

// BatchDeleteRoles: DELETE /roles {ids}
func BatchDeleteRoles(db *gorm.DB, e *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := parseBatchIDs(c)
		if err != nil {
			return batchBadRequest(c, err)
		}
		var roles []models.Role
		if err := db.Where("id IN ?", ids).Find(&roles).Error; err != nil {
			return dbError(c, err)
		}
		byID := make(map[uint]models.Role, len(roles))
		for _, r := range roles {
			byID[r.ID] = r
		}

		res := batchResult{Deleted: []uint{}, Failed: []batchFailure{}}
		for _, id := range ids {
			role, ok := byID[id]
			switch {
			case !ok:
				res.fail(id, "not found")
			case role.IsSystem:
				res.fail(id, "system role cannot be deleted")
			default:
				if err := deleteRole(db, e, role); errors.Is(err, errRoleInUse) {
					res.fail(id, "role is assigned to users")
				} else if err != nil {
					res.fail(id, "delete failed")
				} else {
					res.Deleted = append(res.Deleted, id)
				}
			}
		}
		return c.JSON(res)
	}
}

// BatchDeleteApis: DELETE /apis {ids}
func BatchDeleteApis(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ids, err := parseBatchIDs(c)
		if err != nil {
			return batchBadRequest(c, err)
		}
		var existing []models.Api
		if err := db.Select("id").Where("id IN ?", ids).Find(&existing).Error; err != nil {
			return dbError(c, err)
		}
		found := make(map[uint]bool, len(existing))
		for _, a := range existing {
			found[a.ID] = true
		}
		res := batchResult{Deleted: []uint{}, Failed: []batchFailure{}}
		for _, id := range ids {
			if !found[id] {
				res.fail(id, "not found")
			} else if err := db.Delete(&models.Api{}, id).Error; err != nil {
				res.fail(id, "delete failed")
			} else {
				res.Deleted = append(res.Deleted, id)
			}
		}
		return c.JSON(res)
	}
}

// BatchDeleteMedia: DELETE /media {ids}
// The stored object goes first: if the driver fails the record is kept, so no
// file is ever orphaned and the id can simply be retried.
func BatchDeleteMedia(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reg, err := mediaStorage()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "storage unavailable"})
		}
		ids, err := parseBatchIDs(c)
		if err != nil {
			return batchBadRequest(c, err)
		}
		var items []models.Media
		if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
			return dbError(c, err)
		}
		byID := make(map[uint]models.Media, len(items))
		for _, m := range items {
			byID[m.ID] = m
		}

		res := batchResult{Deleted: []uint{}, Failed: []batchFailure{}}
		for _, id := range ids {
			m, ok := byID[id]
			if !ok {
				res.fail(id, "not found")
				continue
			}
			drv, err := reg.For(m.Storage)
			if err != nil {
				res.fail(id, "storage driver unavailable")
				continue
			}
			if err := drv.Delete(c.UserContext(), m.Path); err != nil && !errors.Is(err, storage.ErrNotFound) {
				res.fail(id, "delete file error")
				continue
			}
			if err := db.Delete(&m).Error; err != nil {
				res.fail(id, "delete failed")
				continue
			}
			res.Deleted = append(res.Deleted, id)
		}
		return c.JSON(res)
	}
}

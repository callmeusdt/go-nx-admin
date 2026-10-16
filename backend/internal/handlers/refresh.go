package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
)

// refreshGrace is how long a just-rotated refresh token is still accepted.
// Two browser tabs commonly refresh at nearly the same moment; without a grace
// window the loser would look like a replay and the whole session would die.
const refreshGrace = 10 * time.Second

func newRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)
	return token, hashRefresh(token), nil
}

func hashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// createOnlineSession issues an access/refresh pair and records the session.
func createOnlineSession(db *gorm.DB, user models.User, ip, userAgent string) (access, refresh string, err error) {
	access, err = middleware.GenerateToken(user.ID, user.Username, user.Role.Slug, user.RoleID)
	if err != nil {
		return "", "", err
	}
	refresh, hash, err := newRefreshToken()
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	err = db.Create(&models.OnlineUser{
		UserID:         user.ID,
		Username:       user.Username,
		Role:           user.Role.Slug,
		Token:          access,
		RefreshHash:    hash,
		RefreshExpires: now.Add(config.AppConfig.JWT.RefreshTTL()),
		IP:             ip,
		UserAgent:      userAgent,
		LoginAt:        now,
	}).Error
	return access, refresh, err
}

func unauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "invalid refresh token"})
}

// RefreshToken exchanges a refresh token for a fresh access/refresh pair.
// It is a public route: the refresh token itself is the credential.
func RefreshToken(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.BodyParser(&req); err != nil || len(req.RefreshToken) != 64 {
			return unauthorized(c)
		}
		presented := hashRefresh(req.RefreshToken)

		var session models.OnlineUser
		err := db.Where("refresh_hash = ? OR prev_refresh_hash = ?", presented, presented).First(&session).Error
		if err != nil {
			return unauthorized(c)
		}
		now := time.Now()
		if now.After(session.RefreshExpires) {
			db.Delete(&session)
			return unauthorized(c)
		}

		if session.RefreshHash != presented {
			// Presented an already-rotated token. Within the grace window this
			// is a racing tab; afterwards it is a replay, so end the session.
			if now.Sub(session.UpdatedAt) > refreshGrace {
				db.Delete(&session)
			}
			return unauthorized(c)
		}

		var user models.User
		if err := db.Preload("Role").First(&user, session.UserID).Error; err != nil || user.Status != 1 {
			db.Delete(&session)
			return unauthorized(c)
		}

		access, err := middleware.GenerateToken(user.ID, user.Username, user.Role.Slug, user.RoleID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "token generation failed"})
		}
		refresh, hash, err := newRefreshToken()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "token generation failed"})
		}

		// Conditional update: only one concurrent request can win the rotation.
		res := db.Model(&models.OnlineUser{}).
			Where("id = ? AND refresh_hash = ?", session.ID, presented).
			Updates(map[string]any{
				"token":             access,
				"role":              user.Role.Slug,
				"refresh_hash":      hash,
				"prev_refresh_hash": presented,
				"refresh_expires":   now.Add(config.AppConfig.JWT.RefreshTTL()),
				"updated_at":        now,
			})
		if res.Error != nil {
			return c.Status(500).JSON(fiber.Map{"message": "refresh failed"})
		}
		if res.RowsAffected == 0 {
			return unauthorized(c)
		}

		return c.JSON(fiber.Map{
			"token":         access,
			"refresh_token": refresh,
			"user": fiber.Map{
				"id":        user.ID,
				"username":  user.Username,
				"role":      user.Role.Slug,
				"role_name": user.Role.Name,
			},
		})
	}
}

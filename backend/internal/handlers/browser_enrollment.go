package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

func browserEnrollment(c *fiber.Ctx, db *gorm.DB, enable bool) error {
	session := c.Locals("browser_session").(models.BrowserSession)
	var in struct {
		Code string `json:"code"`
	}
	if enable && c.BodyParser(&in) != nil {
		return c.SendStatus(400)
	}
	var result fiber.Map
	invalid := false
	err := db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, session.UserID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&session, session.ID).Error; err != nil {
			return err
		}
		if user.Status != 1 || user.MFAEnabled || session.Purpose != "full" || time.Now().After(session.ExpiresAt) {
			invalid = true
			return nil
		}
		if !enable {
			key, err := totp.Generate(totp.GenerateOpts{Issuer: "NX Admin", AccountName: user.Username})
			if err != nil {
				return err
			}
			if err := tx.Model(&user).Update("mfa_secret", key.Secret()).Error; err != nil {
				return err
			}
			result = fiber.Map{"secret": key.Secret(), "url": key.URL()}
			return nil
		}
		if session.Attempts >= 5 || !totp.Validate(in.Code, user.MFASecret) {
			invalid = true
			return tx.Model(&session).Update("attempts", gorm.Expr("attempts + 1")).Error
		}
		codes := generateRecoveryCodes()
		hashes := make([]string, len(codes))
		for i, code := range codes {
			hashes[i] = middleware.Digest(code)
		}
		if err := tx.Model(&user).Updates(map[string]any{"mfa_enabled": true, "mfa_recovery_codes": strings.Join(hashes, ",")}).Error; err != nil {
			return err
		}
		// Other password-only sessions must not gain access after enrollment.
		if err := tx.Where("user_id = ? AND id <> ?", user.ID, session.ID).Delete(&models.BrowserSession{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&session).Updates(map[string]any{"reauthenticated_at": time.Now(), "attempts": 0}).Error; err != nil {
			return err
		}
		result = fiber.Map{"message": "ok", "recovery_codes": codes}
		return nil
	})
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"code": "ENROLLMENT_FAILED"})
	}
	if invalid {
		return c.Status(403).JSON(fiber.Map{"code": "ENROLLMENT_REJECTED"})
	}
	return c.JSON(result)
}

func browserRecoveryCodes(c *fiber.Ctx, db *gorm.DB) error {
	// Require fresh password and TOTP before rotating recovery credentials.
	if err := browserReauthenticate(c, db); err != nil {
		return err
	}
	if c.Response().StatusCode() != 200 {
		return nil
	}
	session := c.Locals("browser_session").(models.BrowserSession)
	codes := generateRecoveryCodes()
	hashes := make([]string, len(codes))
	for i, code := range codes {
		hashes[i] = middleware.Digest(code)
	}
	if err := db.Model(&models.User{}).Where("id = ? AND mfa_enabled = ?", session.UserID, true).Update("mfa_recovery_codes", strings.Join(hashes, ",")).Error; err != nil {
		return err
	}
	return c.JSON(fiber.Map{"recovery_codes": codes})
}

package handlers

import (
	"crypto/subtle"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

func browserLogin(c *fiber.Ctx, db *gorm.DB, user models.User) error {
	start := time.Now()
	purpose := "full"
	ttl := 8 * time.Hour
	if user.MFAEnabled {
		purpose = "mfa"
		ttl = 5 * time.Minute
	}
	token, err := middleware.RandomSecret()
	if err != nil {
		return err
	}
	csrf, err := middleware.RandomSecret()
	if err != nil {
		return err
	}
	session := models.BrowserSession{UserID: user.ID, TokenHash: middleware.Digest(token), CSRFHash: middleware.Digest(csrf), Purpose: purpose, ExpiresAt: time.Now().Add(ttl), IP: middleware.BrowserIP(c), UserAgent: c.Get("User-Agent")}
	err = db.Transaction(func(tx *gorm.DB) error {
		var current models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, user.ID).Error; err != nil {
			return err
		}
		if current.Status != 1 || current.Password != user.Password || current.MFAEnabled != user.MFAEnabled {
			return errors.New("account changed during login")
		}
		if err := tx.Where("user_id = ? AND purpose = ?", user.ID, "mfa").Delete(&models.BrowserSession{}).Error; err != nil {
			return err
		}
		return tx.Create(&session).Error
	})
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"code": "LOGIN_FAILED"})
	}
	middleware.SetSessionCookies(c, token, csrf, session.ExpiresAt)
	recordLoginLog(db, user.Username, c, "mfa_required", "browser MFA verification or enrollment required", start)
	return c.JSON(fiber.Map{"mfa_required": purpose == "mfa", "enrollment_required": !user.MFAEnabled, "user": fiber.Map{"id": user.ID, "username": user.Username, "role": user.Role.Slug}})
}

func browserMFAVerify(c *fiber.Ctx, db *gorm.DB) error {
	start := time.Now()
	session, ok := c.Locals("browser_session").(models.BrowserSession)
	if !ok || session.Purpose != "mfa" {
		return c.Status(403).JSON(fiber.Map{"code": "MFA_REQUIRED"})
	}
	var in struct {
		Code     string `json:"code"`
		Recovery string `json:"recovery_code"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.SendStatus(400)
	}
	token, err := middleware.RandomSecret()
	if err != nil {
		return err
	}
	csrf, err := middleware.RandomSecret()
	if err != nil {
		return err
	}
	valid := false
	var user models.User
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Role").First(&user, session.UserID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&session, session.ID).Error; err != nil {
			return err
		}
		if session.Purpose != "mfa" || time.Now().After(session.ExpiresAt) || session.Attempts >= 5 || !user.MFAEnabled || user.Status != 1 {
			return nil
		}
		valid = totp.Validate(in.Code, user.MFASecret)
		if in.Recovery != "" {
			valid = false
			kept := []string{}
			hash := middleware.Digest(in.Recovery)
			for _, saved := range strings.Split(user.MFARecoveryCodes, ",") {
				if saved != "" && subtle.ConstantTimeCompare([]byte(saved), []byte(hash)) == 1 {
					valid = true
				} else {
					kept = append(kept, saved)
				}
			}
			if valid {
				if err := tx.Model(&user).Update("mfa_recovery_codes", strings.Join(kept, ",")).Error; err != nil {
					return err
				}
			}
		}
		if !valid {
			return tx.Model(&session).Update("attempts", gorm.Expr("attempts + 1")).Error
		}
		now := time.Now()
		session.TokenHash = middleware.Digest(token)
		session.CSRFHash = middleware.Digest(csrf)
		session.Purpose = "full"
		session.ExpiresAt = now.Add(8 * time.Hour)
		session.ReauthenticatedAt = &now
		return tx.Save(&session).Error
	})
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"code": "MFA_INVALID"})
	}
	if !valid {
		recordLoginLog(db, user.Username, c, "mfa_fail", "browser MFA verification failed", start)
		return c.Status(401).JSON(fiber.Map{"code": "MFA_INVALID"})
	}
	middleware.SetSessionCookies(c, token, csrf, session.ExpiresAt)
	recordLoginLog(db, user.Username, c, "success", "browser login with MFA", start)
	return c.JSON(fiber.Map{"user": fiber.Map{"id": user.ID, "username": user.Username, "role": user.Role.Slug}})
}

func browserLogout(c *fiber.Ctx, db *gorm.DB) error {
	session := c.Locals("browser_session").(models.BrowserSession)
	if err := db.Delete(&session).Error; err != nil {
		return err
	}
	middleware.ClearSessionCookies(c)
	return c.JSON(fiber.Map{"message": "ok"})
}

func browserReauthenticate(c *fiber.Ctx, db *gorm.DB) error {
	var in struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.BodyParser(&in); err != nil {
		return c.SendStatus(400)
	}
	session := c.Locals("browser_session").(models.BrowserSession)
	allowed, _ := middleware.CheckRateLimit("reauth:" + session.TokenHash)
	if !allowed {
		return c.SendStatus(429)
	}
	valid := false
	err := db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, session.UserID).Error; err != nil {
			return err
		}
		if user.Status != 1 || !user.MFAEnabled || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)) != nil || !totp.Validate(in.Code, user.MFASecret) {
			return nil
		}
		result := tx.Model(&models.BrowserSession{}).Where("id = ? AND purpose = ? AND expires_at > ?", session.ID, "full", time.Now()).Update("reauthenticated_at", time.Now())
		valid = result.RowsAffected == 1
		return result.Error
	})
	if err != nil {
		return err
	}
	if !valid {
		middleware.RecordFailedAttempt("reauth:" + session.TokenHash)
		return c.Status(401).JSON(fiber.Map{"code": "REAUTH_FAILED"})
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

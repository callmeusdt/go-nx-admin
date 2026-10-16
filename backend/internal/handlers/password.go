package handlers

import (
	"errors"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 32
)

var errPasswordPolicy = errors.New("password must be 8-32 bytes")

// checkPasswordPolicy is shared by every path that sets a password for others
// or for oneself, so the rule cannot drift between them.
func checkPasswordPolicy(password string) error {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen || !utf8.ValidString(password) {
		return errPasswordPolicy
	}
	return nil
}

func passwordErr(c *fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(fiber.Map{"code": code, "message": code})
}

// ChangePassword lets the signed-in user replace their own password. It needs
// the current password (and a TOTP code when MFA is on), shares the login rate
// limit, and deliberately leaves every session untouched: ending other
// sessions is a separate, explicit action.
func ChangePassword(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(uint)
		var in struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
			Code        string `json:"code"`
		}
		if err := c.BodyParser(&in); err != nil {
			return passwordErr(c, 400, "INVALID_REQUEST")
		}
		if err := checkPasswordPolicy(in.NewPassword); err != nil {
			return passwordErr(c, 400, "PASSWORD_POLICY")
		}
		if in.NewPassword == in.OldPassword {
			return passwordErr(c, 400, "PASSWORD_UNCHANGED")
		}

		key := "pwd:" + strconv.FormatUint(uint64(userID), 10)
		if allowed, _ := middleware.CheckRateLimit(key); !allowed {
			return passwordErr(c, 429, "TOO_MANY_ATTEMPTS")
		}

		status, code := 0, ""
		err := db.Transaction(func(tx *gorm.DB) error {
			var user models.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
				status, code = 404, "NOT_FOUND"
				return nil
			}
			if user.Status != 1 {
				status, code = 403, "ACCOUNT_DISABLED"
				return nil
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.OldPassword)) != nil {
				middleware.RecordFailedAttempt(key)
				status, code = 400, "OLD_PASSWORD_INCORRECT"
				return nil
			}
			if user.MFAEnabled && !totp.Validate(in.Code, user.MFASecret) {
				middleware.RecordFailedAttempt(key)
				status, code = 400, "MFA_CODE_INVALID"
				return nil
			}
			hashed, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			return tx.Model(&user).Updates(map[string]any{
				"password":             string(hashed),
				"must_change_password": false,
				"updated_at":           time.Now(),
			}).Error
		})
		if err != nil {
			return passwordErr(c, 500, "INTERNAL_ERROR")
		}
		if code != "" {
			return passwordErr(c, status, code)
		}
		middleware.ClearRateLimit(key)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"github.com/gofiber/fiber/v2"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/models"
	"gorm.io/gorm"
	"net"
	"strings"
	"time"
)

const SessionCookie = "__Host-nx-session"
const CSRFCookie = "__Host-nx-csrf"

func CookieMode() bool { return config.AppConfig != nil && config.AppConfig.Session.Cookie }
func Digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func RandomSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func BrowserIP(c *fiber.Ctx) string {
	peer := c.Context().RemoteIP()
	if config.AppConfig.Session.TrustLoopbackProxy && peer.IsLoopback() {
		if ip := net.ParseIP(c.Get("X-BIM-Client-IP")); ip != nil {
			return ip.String()
		}
	}
	return peer.String()
}
func SetSessionCookies(c *fiber.Ctx, token, csrf string, expires time.Time) {
	for name, value := range map[string]string{SessionCookie: token, CSRFCookie: csrf} {
		c.Cookie(&fiber.Cookie{Name: name, Value: value, Path: "/", Secure: true, HTTPOnly: name == SessionCookie, SameSite: "Strict", Expires: expires})
	}
}
func ClearSessionCookies(c *fiber.Ctx) { SetSessionCookies(c, "", "", time.Unix(1, 0)) }

// BrowserAuth has an exact allowlist. A pending session can only verify MFA,
// never configure it, disable it, recover codes, or invoke business endpoints.
func BrowserAuth(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path, method := c.Path(), c.Method()
		mutation := method != "GET" && method != "HEAD" && method != "OPTIONS"
		if mutation && c.Get("Origin") != config.AppConfig.Session.Origin {
			return browserError(c, 403, "ORIGIN_REJECTED")
		}
		if path == "/api/v1/auth/login" && method == "POST" || path == "/api/v1/auth/captcha" && method == "GET" {
			return c.Next()
		}
		credential := c.Cookies(SessionCookie)
		if len(credential) != 64 {
			return browserError(c, 401, "UNAUTHENTICATED")
		}
		var session models.BrowserSession
		if err := db.Where("token_hash = ? AND expires_at > ?", Digest(credential), time.Now()).First(&session).Error; err != nil {
			return browserError(c, 401, "UNAUTHENTICATED")
		}
		if mutation {
			csrf := c.Get("X-CSRF-Token")
			if csrf == "" || subtle.ConstantTimeCompare([]byte(Digest(csrf)), []byte(session.CSRFHash)) != 1 {
				return browserError(c, 403, "CSRF_REJECTED")
			}
		}
		if session.Purpose == "mfa" {
			if path != "/api/v1/auth/mfa/verify" || method != "POST" {
				return browserError(c, 403, "MFA_REQUIRED")
			}
		} else if session.Purpose != "full" {
			return browserError(c, 401, "UNAUTHENTICATED")
		}
		var user models.User
		if err := db.Preload("Role").First(&user, session.UserID).Error; err != nil || user.Status != 1 {
			return browserError(c, 401, "UNAUTHENTICATED")
		}
		if session.Purpose == "full" && !user.MFAEnabled {
			allowed := path == "/api/v1/auth/me" && method == "GET" || method == "POST" && (path == "/api/v1/auth/mfa/setup" || path == "/api/v1/auth/mfa/enable" || path == "/api/v1/auth/logout")
			if !allowed {
				return browserError(c, 403, "MFA_ENROLLMENT_REQUIRED")
			}
		}
		if mutation && session.Purpose == "full" && user.MFAEnabled && !strings.HasPrefix(path, "/api/v1/auth/") {
			if session.ReauthenticatedAt == nil || time.Since(*session.ReauthenticatedAt) > 5*time.Minute {
				return browserError(c, 403, "REAUTH_REQUIRED")
			}
		}
		c.Locals("browser_session", session)
		c.Locals("user_id", user.ID)
		c.Locals("username", user.Username)
		c.Locals("role", user.Role.Slug)
		c.Locals("role_id", user.RoleID)
		c.Locals("token", session.TokenHash)
		return c.Next()
	}
}
func browserError(c *fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(fiber.Map{"code": code, "message": code})
}

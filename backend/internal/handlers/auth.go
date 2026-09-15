package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"go-nx-admin/internal/config"
	"go-nx-admin/internal/middleware"
	"go-nx-admin/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Login(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		ip := clientIP(c)

		// rate limit check (before parsing body)
		allowed, remain := middleware.CheckRateLimit(ip)
		if !allowed {
			return c.Status(429).JSON(fiber.Map{
				"message": fmt.Sprintf("登录尝试过于频繁，请%d分钟后再试", remain),
			})
		}

		var req struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			Fingerprint string `json:"fingerprint"`
			CaptchaID   string `json:"captcha_id"`
			SlideX      int    `json:"slide_x"`
		}
		if err := c.BodyParser(&req); err != nil {
			recordLoginLog(db, req.Username, c, "fail", "invalid request", start)
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}

		fp := req.Fingerprint

		// captcha fingerprint cooldown check
		if config.AppConfig.Captcha.Enabled && fp != "" {
			if cd := CheckCaptchaCooldown(fp); cd > 0 {
				return c.JSON(fiber.Map{
					"captcha_cooldown": cd,
					"message":          "验证过于频繁，请稍后再试",
				})
			}
		}

		// captcha: if enabled and no captcha_id, check if fingerprint is flagged
		if config.AppConfig.Captcha.Enabled && req.CaptchaID == "" {
			if fp != "" && captchaRequiredFp(fp) {
				// fingerprint already flagged — force captcha
				return sendCaptchaRequired(c)
			}
			// first attempt: check password first, captcha only on failure
			goto checkPassword
		}

		// captcha verification (captcha_id present)
		if config.AppConfig.Captcha.Enabled {
			if !VerifyCaptcha(req.CaptchaID, req.SlideX) {
				cd := RecordCaptchaFail(fp)
				return c.Status(400).JSON(fiber.Map{
					"message":          "验证码校验失败",
					"captcha_cooldown": cd,
				})
			}
		}

	checkPassword:
		var user models.User
		if err := db.Where("username = ?", req.Username).Preload("Role").First(&user).Error; err != nil {
			recordLoginLog(db, req.Username, c, "fail", "invalid credentials", start)
			return c.Status(401).JSON(fiber.Map{"message": "invalid credentials"})
		}
		if user.Status != 1 {
			recordLoginLog(db, req.Username, c, "fail", "account disabled", start)
			return c.Status(403).JSON(fiber.Map{"message": "account disabled"})
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			middleware.RecordFailedAttempt(ip)
			recordLoginLog(db, req.Username, c, "fail", "invalid credentials", start)
			FlagFp(fp)
			if config.AppConfig.Captcha.Enabled && req.CaptchaID == "" {
				return sendCaptchaRequired(c)
			}
			return c.Status(401).JSON(fiber.Map{"message": "invalid credentials"})
		}

		middleware.ClearRateLimit(ip)

		if allowed, reason := CheckLoginIPWhitelist(db, user.ID, ip); !allowed {
			recordLoginLog(db, req.Username, c, "fail", reason, start)
			return c.Status(403).JSON(fiber.Map{"message": reason})
		}
		if middleware.CookieMode() {
			return browserLogin(c, db, user)
		}

		token, err := middleware.GenerateToken(user.ID, user.Username, user.Role.Slug, user.RoleID)
		if err != nil {
			recordLoginLog(db, req.Username, c, "fail", "token generation failed", start)
			return c.Status(500).JSON(fiber.Map{"message": "token generation failed"})
		}

		if user.MFAEnabled {
			recordLoginLog(db, req.Username, c, "mfa_required", "MFA verification required", start)
			return c.JSON(fiber.Map{
				"mfa_required": true,
				"mfa_token":    token,
				"user": fiber.Map{
					"id":       user.ID,
					"username": user.Username,
					"mfa":      true,
				},
			})
		}

		recordLoginLog(db, req.Username, c, "success", "login success", start)
		db.Create(&models.OnlineUser{
			UserID:    user.ID,
			Username:  user.Username,
			Role:      user.Role.Slug,
			Token:     token,
			IP:        c.IP(),
			UserAgent: c.Get("User-Agent"),
			LoginAt:   time.Now(),
		})

		return c.JSON(fiber.Map{
			"token": token,
			"user": fiber.Map{
				"id":        user.ID,
				"username":  user.Username,
				"role":      user.Role.Slug,
				"role_name": user.Role.Name,
			},
		})
	}
}

func Logout(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserLogout(c, db)
		}
		token, _ := c.Locals("token").(string)
		if token != "" {
			db.Delete(&models.OnlineUser{}, "token = ?", token)
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func MFASetup(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserEnrollment(c, db, false)
		}
		userID := c.Locals("user_id").(uint)
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "NX Admin",
			AccountName: user.Username,
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "generate failed"})
		}
		user.MFASecret = key.Secret()
		if err := db.Save(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"secret": key.Secret(), "url": key.URL()})
	}
}

func MFAEnable(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserEnrollment(c, db, true)
		}
		userID := c.Locals("user_id").(uint)
		var req struct {
			Code string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		if user.MFASecret == "" {
			return c.Status(400).JSON(fiber.Map{"message": "please setup MFA first"})
		}
		if !totp.Validate(req.Code, user.MFASecret) {
			return c.Status(400).JSON(fiber.Map{"message": "invalid code"})
		}
		user.MFAEnabled = true
		codes := generateRecoveryCodes()
		hashedCodes := make([]string, len(codes))
		for i, c := range codes {
			h := sha256.Sum256([]byte(c))
			hashedCodes[i] = hex.EncodeToString(h[:])
		}
		user.MFARecoveryCodes = strings.Join(hashedCodes, ",")
		if err := db.Save(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok", "recovery_codes": codes})
	}
}

func MFADisable(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return c.Status(403).JSON(fiber.Map{"code": "MFA_REQUIRED"})
		}
		userID := c.Locals("user_id").(uint)
		var req struct {
			Password string `json:"password"`
			Code     string `json:"code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "password incorrect"})
		}
		if !user.MFAEnabled {
			return c.Status(400).JSON(fiber.Map{"message": "MFA not enabled"})
		}
		if !totp.Validate(req.Code, user.MFASecret) {
			return c.Status(400).JSON(fiber.Map{"message": "invalid code"})
		}
		user.MFAEnabled = false
		user.MFASecret = ""
		user.MFARecoveryCodes = ""
		if err := db.Save(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func MFARecoveryCodes(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserRecoveryCodes(c, db)
		}
		userID := c.Locals("user_id").(uint)
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		if !user.MFAEnabled {
			return c.Status(400).JSON(fiber.Map{"message": "MFA not enabled"})
		}
		codes := generateRecoveryCodes()
		hashedCodes := make([]string, len(codes))
		for i, c := range codes {
			h := sha256.Sum256([]byte(c))
			hashedCodes[i] = hex.EncodeToString(h[:])
		}
		user.MFARecoveryCodes = strings.Join(hashedCodes, ",")
		if err := db.Save(&user).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"recovery_codes": codes})
	}
}

func MFAVerify(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserMFAVerify(c, db)
		}
		start := time.Now()
		var req struct {
			MfaToken     string `json:"mfa_token"`
			Code         string `json:"code"`
			RecoveryCode string `json:"recovery_code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if req.MfaToken == "" {
			req.MfaToken, _ = c.Locals("token").(string)
		}
		claims, err := middleware.ParseToken(req.MfaToken)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"message": "invalid or expired mfa token"})
		}
		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"message": "user not found"})
		}
		if !user.MFAEnabled {
			return c.Status(400).JSON(fiber.Map{"message": "MFA not enabled"})
		}

		verified := false
		if req.RecoveryCode != "" {
			hash := sha256.Sum256([]byte(req.RecoveryCode))
			codeHex := hex.EncodeToString(hash[:])
			for _, saved := range strings.Split(user.MFARecoveryCodes, ",") {
				if saved == codeHex {
					verified = true
					hashedCodes := make([]string, 0)
					for _, sc := range strings.Split(user.MFARecoveryCodes, ",") {
						if sc != codeHex {
							hashedCodes = append(hashedCodes, sc)
						}
					}
					user.MFARecoveryCodes = strings.Join(hashedCodes, ",")
					db.Save(&user)
					break
				}
			}
		} else {
			verified = totp.Validate(req.Code, user.MFASecret)
		}

		if !verified {
			recordLoginLog(db, user.Username, c, "mfa_fail", "mfa verification failed", start)
			return c.Status(401).JSON(fiber.Map{"message": "invalid mfa code"})
		}

		username := user.Username
		if err := db.Preload("Role").First(&user, user.ID).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "reload user failed"})
		}
		token, err := middleware.GenerateToken(user.ID, user.Username, user.Role.Slug, user.RoleID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "token generation failed"})
		}
		recordLoginLog(db, username, c, "success", "login success with mfa", start)
		db.Create(&models.OnlineUser{
			UserID:    user.ID,
			Username:  user.Username,
			Role:      user.Role.Slug,
			Token:     token,
			IP:        clientIP(c),
			UserAgent: c.Get("User-Agent"),
			LoginAt:   time.Now(),
		})
		return c.JSON(fiber.Map{
			"token": token,
			"user": fiber.Map{
				"id":        user.ID,
				"username":  user.Username,
				"role":      user.Role.Slug,
				"role_name": user.Role.Name,
			},
		})
	}
}

func generateRecoveryCodes() []string {
	codes := make([]string, 8)
	for i := 0; i < 8; i++ {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			panic("recovery entropy unavailable")
		}
		codes[i] = strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
	}
	return codes
}

func MFAGenerateMfaToken(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)
		var user models.User
		if err := db.Preload("Role").First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		token, err := middleware.GenerateToken(user.ID, user.Username, user.Role.Slug, user.RoleID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "token failed"})
		}
		return c.JSON(fiber.Map{"mfa_token": token, "user": fiber.Map{
			"id":        user.ID,
			"username":  user.Username,
			"role":      user.Role.Slug,
			"role_name": user.Role.Name,
		}})
	}
}

func recordLoginLog(db *gorm.DB, username string, c *fiber.Ctx, status string, message string, start time.Time) {
	ip := clientIP(c)
	ua := c.Get("User-Agent")
	duration := time.Since(start).Microseconds()
	go db.Create(&models.LoginLog{
		Username:  strings.Clone(username),
		IP:        strings.Clone(ip),
		Geo:       resolveIPGeo(ip),
		UserAgent: strings.Clone(ua),
		Status:    strings.Clone(status),
		Message:   strings.Clone(message),
		Duration:  duration,
	})
}

func MyIP(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ip": clientIP(c)})
}

func clientIP(c *fiber.Ctx) string {
	if middleware.CookieMode() {
		return middleware.BrowserIP(c)
	}
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"} {
		value := strings.TrimSpace(c.Get(header))
		if value == "" {
			continue
		}
		ip := strings.TrimSpace(strings.Split(value, ",")[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	return c.IP()
}

func resolveIPGeo(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsUnspecified() {
		return ""
	}

	client := http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get("http://ip-api.com/json/" + url.PathEscape(ip) + "?fields=status,country,regionName,city,query")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var data struct {
		Status     string `json:"status"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Status != "success" {
		return ""
	}

	geo := data.Country
	if data.RegionName != "" {
		geo += " " + data.RegionName
	}
	if data.City != "" {
		geo += " " + data.City
	}
	if geo == "" {
		return ""
	}
	return geo
}

func Me(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)
		var user models.User
		if err := db.Preload("Role").First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		return c.JSON(fiber.Map{
			"id":          user.ID,
			"username":    user.Username,
			"role":        user.Role.Slug,
			"role_name":   user.Role.Name,
			"mfa_enabled": user.MFAEnabled,
		})
	}
}

func UpdateProfile(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uint)
		var req struct {
			Username string `json:"username"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}
		if err := db.Model(&models.User{}).Where("id = ?", userID).Update("username", req.Username).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func VerifyPassword(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			return browserReauthenticate(c, db)
		}
		return Unlock(db)(c)
	}
}

// Unlock verifies only the local screen lock; it never grants reauthentication.
func Unlock(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if middleware.CookieMode() {
			key := "unlock:" + c.Locals("token").(string)
			if allowed, _ := middleware.CheckRateLimit(key); !allowed {
				return c.SendStatus(429)
			}
		}
		var req struct {
			Password string `json:"password"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "invalid request"})
		}

		userID := c.Locals("user_id").(uint)
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			if middleware.CookieMode() {
				middleware.RecordFailedAttempt("unlock:" + c.Locals("token").(string))
			}
			return c.Status(400).JSON(fiber.Map{"message": "password incorrect"})
		}

		return c.JSON(fiber.Map{"message": "verified"})
	}
}

func ClearUserMFA(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var user models.User
		if err := db.First(&user, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "user not found"})
		}
		user.MFAEnabled = false
		user.MFASecret = ""
		user.MFARecoveryCodes = ""
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
			if middleware.CookieMode() {
				return tx.Where("user_id = ?", user.ID).Delete(&models.BrowserSession{}).Error
			}
			return nil
		})
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "ok"})
	}
}

func sendCaptchaRequired(c *fiber.Ctx) error {
	capt, err := slideCapt.Generate()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "captcha generate failed"})
	}
	block := capt.GetData()
	masterB64, _ := capt.GetMasterImage().ToBase64()
	tileB64, _ := capt.GetTileImage().ToBase64()

	id := uuid.New().String()
	captchaStore.Store(id, &captchaData{
		X: block.X, Y: block.Y,
		Width: block.Width, Height: block.Height,
		CreatedAt: time.Now(),
	})
	return c.JSON(fiber.Map{
		"captcha_required": true,
		"captcha_id":       id,
		"master_img":       masterB64,
		"tile_img":         tileB64,
		"tile_width":       block.Width,
		"tile_height":      block.Height,
		"tile_y":           block.Y,
	})
}

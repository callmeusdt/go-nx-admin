package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wenlng/go-captcha/v2/slide"

	"go-nx-admin/internal/cache"
	"go-nx-admin/internal/config"
)

type captchaData struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"w"`
	Height int `json:"h"`
}

const (
	captchaTTL     = 5 * time.Minute
	captchaFlagTTL = 30 * time.Minute
	captchaWindow  = 2 * time.Minute
)

func storeCaptcha(x, y, w, h int) (string, error) {
	id := uuid.New().String()
	raw, _ := json.Marshal(captchaData{X: x, Y: y, Width: w, Height: h})
	if err := cache.Default().Set(context.Background(), "captcha:"+id, raw, captchaTTL); err != nil {
		return "", err
	}
	return id, nil
}

func GenerateCaptcha(c *fiber.Ctx) error {
	if !config.AppConfig.Captcha.Enabled {
		return c.JSON(fiber.Map{"captcha_enabled": false})
	}
	capt, err := slideCapt.Generate()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "captcha generate failed"})
	}
	block := capt.GetData()
	masterB64, _ := capt.GetMasterImage().ToBase64()
	tileB64, _ := capt.GetTileImage().ToBase64()

	id, err := storeCaptcha(block.X, block.Y, block.Width, block.Height)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"message": "captcha generate failed"})
	}

	return c.JSON(fiber.Map{
		"captcha_enabled": true,
		"captcha_id":      id,
		"master_img":      masterB64,
		"tile_img":        tileB64,
		"tile_width":      block.Width,
		"tile_height":     block.Height,
		"tile_y":          block.Y,
	})
}

func VerifyCaptcha(id string, userX int) bool {
	raw, err := cache.Default().GetDel(context.Background(), "captcha:"+id)
	if err != nil {
		log.Printf("[captcha] verify failed: id=%s not found in store", id)
		return false
	}
	var d captchaData
	if json.Unmarshal(raw, &d) != nil {
		return false
	}
	ok := slide.Validate(userX, d.Y, d.X, d.Y, config.AppConfig.Captcha.Padding)
	if !ok {
		log.Printf("[captcha] verify failed: id=%s userX=%d correctX=%d correctY=%d padding=%d", id, userX, d.X, d.Y, config.AppConfig.Captcha.Padding)
	}
	return ok
}

// CheckCaptchaCooldown returns cooldown seconds, 0 if allowed
func CheckCaptchaCooldown(fp string) int {
	ttl, err := cache.Default().TTL(context.Background(), "captcha:lock:"+fp)
	if err != nil || ttl <= 0 {
		return 0
	}
	return int(ttl.Seconds())
}

// RecordCaptchaFail records a failed captcha attempt for the fingerprint
func RecordCaptchaFail(fp string) (cooldownSeconds int) {
	ctx := context.Background()
	n, err := cache.Default().Incr(ctx, "captcha:fail:"+fp, captchaWindow)
	if err != nil {
		return 0
	}
	var lock time.Duration
	switch {
	case n <= 2:
		lock = 3 * time.Second
	case n <= 5:
		lock = 10 * time.Second
	default:
		lock = 30 * time.Second
	}
	cache.Default().Set(ctx, "captcha:lock:"+fp, []byte("1"), lock)
	return int(lock.Seconds())
}

// FlagFp marks a fingerprint as needing captcha (after password failure)
func FlagFp(fp string) {
	if fp != "" {
		cache.Default().Set(context.Background(), "captcha:flag:"+fp, []byte("1"), captchaFlagTTL)
	}
}

// captchaRequiredFp checks if the fingerprint has been flagged for captcha
func captchaRequiredFp(fp string) bool {
	_, err := cache.Default().Get(context.Background(), "captcha:flag:"+fp)
	return err == nil
}

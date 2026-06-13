package handlers

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wenlng/go-captcha/v2/slide"

	"go-nx-admin/internal/config"
)

type captchaData struct {
	X         int
	Y         int
	Width     int
	Height    int
	CreatedAt time.Time
}

type captchaAttempt struct {
	Count       int
	FirstFail   time.Time
	LockedUntil time.Time
}

var (
	captchaStore   = sync.Map{}
	fpAttemptStore = sync.Map{}
	fpFlagStore    = sync.Map{}
)

func init() {
	go func() {
		for {
			time.Sleep(2 * time.Minute)
			now := time.Now()
			captchaStore.Range(func(k, v interface{}) bool {
				if now.Sub(v.(*captchaData).CreatedAt) > 5*time.Minute {
					captchaStore.Delete(k)
				}
				return true
			})
			fpAttemptStore.Range(func(k, v interface{}) bool {
				a := v.(*captchaAttempt)
				if now.After(a.LockedUntil) && now.Sub(a.FirstFail) > 30*time.Minute {
					fpAttemptStore.Delete(k)
				}
				return true
			})
		}
	}()
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

	id := uuid.New().String()
	captchaStore.Store(id, &captchaData{
		X: block.X, Y: block.Y,
		Width: block.Width, Height: block.Height,
		CreatedAt: time.Now(),
	})

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
	val, ok := captchaStore.LoadAndDelete(id)
	if !ok {
		log.Printf("[captcha] verify failed: id=%s not found in store", id)
		return false
	}
	d := val.(*captchaData)
	ok = slide.Validate(userX, d.Y, d.X, d.Y, config.AppConfig.Captcha.Padding)
	if !ok {
		log.Printf("[captcha] verify failed: id=%s userX=%d correctX=%d correctY=%d padding=%d", id, userX, d.X, d.Y, config.AppConfig.Captcha.Padding)
	}
	return ok
}

// CheckCaptchaCooldown returns cooldown seconds, 0 if allowed
func CheckCaptchaCooldown(fp string) int {
	val, ok := fpAttemptStore.Load(fp)
	if !ok {
		return 0
	}
	a := val.(*captchaAttempt)
	if time.Now().Before(a.LockedUntil) {
		return int(time.Until(a.LockedUntil).Seconds())
	}
	return 0
}

// RecordCaptchaFail records a failed captcha attempt for the fingerprint
func RecordCaptchaFail(fp string) (cooldownSeconds int) {
	val, _ := fpAttemptStore.LoadOrStore(fp, &captchaAttempt{
		FirstFail: time.Now(),
	})
	a := val.(*captchaAttempt)
	a.Count++
	now := time.Now()
	window := 2 * time.Minute
	if now.Sub(a.FirstFail) > window {
		a.Count = 1
		a.FirstFail = now
	}

	switch {
	case a.Count <= 2:
		a.LockedUntil = now.Add(3 * time.Second)
	case a.Count <= 5:
		a.LockedUntil = now.Add(10 * time.Second)
	default:
		a.LockedUntil = now.Add(30 * time.Second)
	}
	return int(time.Until(a.LockedUntil).Seconds())
}

// FlagFp marks a fingerprint as needing captcha (after password failure)
func FlagFp(fp string) {
	if fp != "" {
		fpFlagStore.Store(fp, true)
	}
}

// captchaRequiredFp checks if the fingerprint has been flagged for captcha
func captchaRequiredFp(fp string) bool {
	_, ok := fpFlagStore.Load(fp)
	return ok
}

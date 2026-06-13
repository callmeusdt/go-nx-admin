package middleware

import (
	"sync"
	"time"

	"go-nx-admin/internal/config"
)

type attempt struct {
	Count       int
	FirstFail   time.Time
	LockedUntil time.Time
}

var (
	rateLimitStore = sync.Map{}
	rateInitOnce   sync.Once
)

func init() {
	rateInitOnce.Do(func() {
		go func() {
			for {
				time.Sleep(5 * time.Minute)
				now := time.Now()
				rateLimitStore.Range(func(k, v interface{}) bool {
					a := v.(*attempt)
					if now.After(a.LockedUntil) && now.Sub(a.FirstFail) > 2*time.Hour {
						rateLimitStore.Delete(k)
					}
					return true
				})
			}
		}()
	})
}

func CheckRateLimit(ip string) (bool, int) {
	cfg := config.AppConfig.Login.RateLimit
	if !cfg.Enabled {
		return true, 0
	}
	key := "ip:" + ip
	val, ok := rateLimitStore.Load(key)
	if !ok {
		return true, 0
	}
	a := val.(*attempt)
	now := time.Now()

	if now.Before(a.LockedUntil) {
		remain := int(a.LockedUntil.Sub(now).Minutes())
		if remain < 1 {
			remain = 1
		}
		return false, remain
	}

	window := time.Duration(cfg.WindowMinutes) * time.Minute
	if now.After(a.FirstFail.Add(window)) {
		rateLimitStore.Delete(key)
		return true, 0
	}
	return true, 0
}

func RecordFailedAttempt(ip string) {
	cfg := config.AppConfig.Login.RateLimit
	if !cfg.Enabled {
		return
	}
	key := "ip:" + ip
	val, _ := rateLimitStore.LoadOrStore(key, &attempt{
		FirstFail: time.Now(),
	})
	a := val.(*attempt)
	a.Count++
	if a.Count >= cfg.MaxAttempts {
		a.LockedUntil = time.Now().Add(time.Duration(cfg.LockMinutes) * time.Minute)
	}
}

func ClearRateLimit(ip string) {
	rateLimitStore.Delete("ip:" + ip)
}

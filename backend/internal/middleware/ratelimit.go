package middleware

import (
	"context"
	"time"

	"go-nx-admin/internal/cache"
	"go-nx-admin/internal/config"
)

func rateKeys(id string) (count, lock string) {
	return "rl:count:" + id, "rl:lock:" + id
}

// CheckRateLimit reports whether the identity may attempt a login and, when
// locked, the remaining lock time in minutes.
func CheckRateLimit(ip string) (bool, int) {
	cfg := config.AppConfig.Login.RateLimit
	if !cfg.Enabled {
		return true, 0
	}
	_, lock := rateKeys(ip)
	ttl, err := cache.Default().TTL(context.Background(), lock)
	if err != nil {
		return true, 0
	}
	remain := int((ttl + time.Minute - 1) / time.Minute)
	if remain < 1 {
		remain = 1
	}
	return false, remain
}

func RecordFailedAttempt(ip string) {
	cfg := config.AppConfig.Login.RateLimit
	if !cfg.Enabled {
		return
	}
	ctx := context.Background()
	countKey, lockKey := rateKeys(ip)
	n, err := cache.Default().Incr(ctx, countKey, time.Duration(cfg.WindowMinutes)*time.Minute)
	if err != nil {
		return
	}
	if int(n) >= cfg.MaxAttempts {
		cache.Default().Set(ctx, lockKey, []byte("1"), time.Duration(cfg.LockMinutes)*time.Minute)
	}
}

func ClearRateLimit(ip string) {
	countKey, lockKey := rateKeys(ip)
	cache.Default().Del(context.Background(), countKey, lockKey)
}

package middleware

import (
	"context"
	"testing"

	"go-nx-admin/internal/cache"
	"go-nx-admin/internal/config"
)

func withRateLimit(t *testing.T, max, window, lock int) {
	t.Helper()
	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{Login: config.LoginConfig{RateLimit: config.RateLimitConfig{
		Enabled: true, MaxAttempts: max, WindowMinutes: window, LockMinutes: lock,
	}}}
	prevCache := cache.Set(cache.NewMemory())
	t.Cleanup(func() {
		config.AppConfig = prevCfg
		if c := cache.Set(prevCache); c != nil {
			c.Close()
		}
	})
}

func TestRateLimitLocksAfterMaxAttempts(t *testing.T) {
	withRateLimit(t, 3, 5, 15)

	for i := 0; i < 2; i++ {
		RecordFailedAttempt("1.2.3.4")
		if ok, _ := CheckRateLimit("1.2.3.4"); !ok {
			t.Fatalf("locked too early after %d failures", i+1)
		}
	}
	RecordFailedAttempt("1.2.3.4")
	ok, remain := CheckRateLimit("1.2.3.4")
	if ok {
		t.Fatal("must lock after max attempts")
	}
	if remain < 14 || remain > 15 {
		t.Fatalf("remaining minutes = %d, want ~15", remain)
	}
	if ok, _ := CheckRateLimit("5.6.7.8"); !ok {
		t.Fatal("other identities must be unaffected")
	}
}

func TestRateLimitClearUnlocks(t *testing.T) {
	withRateLimit(t, 1, 5, 15)
	RecordFailedAttempt("ip")
	if ok, _ := CheckRateLimit("ip"); ok {
		t.Fatal("expected lock")
	}
	ClearRateLimit("ip")
	if ok, _ := CheckRateLimit("ip"); !ok {
		t.Fatal("clear must unlock")
	}
}

func TestRateLimitDisabled(t *testing.T) {
	withRateLimit(t, 1, 5, 15)
	config.AppConfig.Login.RateLimit.Enabled = false
	RecordFailedAttempt("ip")
	if ok, _ := CheckRateLimit("ip"); !ok {
		t.Fatal("disabled limiter must never lock")
	}
}

func TestRateLimitWindowResetsCount(t *testing.T) {
	withRateLimit(t, 2, 5, 15)
	c := cache.Default()
	RecordFailedAttempt("ip")
	// Simulate the window elapsing by expiring the counter.
	countKey, _ := rateKeys("ip")
	c.Del(context.Background(), countKey)
	RecordFailedAttempt("ip")
	if ok, _ := CheckRateLimit("ip"); !ok {
		t.Fatal("failures outside the window must not accumulate")
	}
}

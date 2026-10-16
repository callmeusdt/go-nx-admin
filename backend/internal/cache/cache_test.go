package cache

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func suite(t *testing.T, c Cache) {
	ctx := context.Background()

	if _, err := c.Get(ctx, "missing"); !errors.Is(err, ErrMiss) {
		t.Fatalf("missing key: want ErrMiss, got %v", err)
	}

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if b, err := c.Get(ctx, "k"); err != nil || string(b) != "v" {
		t.Fatalf("get: %q %v", b, err)
	}

	if b, err := c.GetDel(ctx, "k"); err != nil || string(b) != "v" {
		t.Fatalf("getdel: %q %v", b, err)
	}
	if _, err := c.GetDel(ctx, "k"); !errors.Is(err, ErrMiss) {
		t.Fatalf("getdel twice must miss, got %v", err)
	}

	c.Set(ctx, "a", []byte("1"), 0)
	c.Set(ctx, "b", []byte("1"), 0)
	c.Del(ctx, "a", "b")
	if _, err := c.Get(ctx, "a"); !errors.Is(err, ErrMiss) {
		t.Fatal("del did not remove key")
	}

	c.Set(ctx, "ttl", []byte("1"), 2*time.Second)
	d, err := c.TTL(ctx, "ttl")
	if err != nil || d <= 0 || d > 2*time.Second {
		t.Fatalf("ttl: %v %v", d, err)
	}
	if _, err := c.TTL(ctx, "none"); !errors.Is(err, ErrMiss) {
		t.Fatalf("ttl on missing key: %v", err)
	}
	c.Set(ctx, "forever", []byte("1"), 0)
	if d, err := c.TTL(ctx, "forever"); err != nil || d != 0 {
		t.Fatalf("no-expiry ttl: %v %v", d, err)
	}

	c.Del(ctx, "n")
	for want := int64(1); want <= 3; want++ {
		if got, err := c.Incr(ctx, "n", time.Minute); err != nil || got != want {
			t.Fatalf("incr: want %d got %d %v", want, got, err)
		}
	}
	// TTL is fixed at creation and not refreshed by later increments.
	first, _ := c.TTL(ctx, "n")
	time.Sleep(20 * time.Millisecond)
	c.Incr(ctx, "n", time.Hour)
	if after, _ := c.TTL(ctx, "n"); after > first {
		t.Fatalf("incr extended ttl: %v -> %v", first, after)
	}
}

func TestMemory(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	suite(t, m)
}

func TestMemoryExpiry(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	ctx := context.Background()
	m.Set(ctx, "k", []byte("v"), 30*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	if _, err := m.Get(ctx, "k"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expired key must miss, got %v", err)
	}
	m.Incr(ctx, "c", 30*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	if n, _ := m.Incr(ctx, "c", time.Minute); n != 1 {
		t.Fatalf("counter must restart after expiry, got %d", n)
	}
}

func TestMemoryValuesAreCopied(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	ctx := context.Background()
	in := []byte("abc")
	m.Set(ctx, "k", in, 0)
	in[0] = 'X'
	got, _ := m.Get(ctx, "k")
	got[1] = 'Y'
	again, _ := m.Get(ctx, "k")
	if string(again) != "abc" {
		t.Fatalf("stored value mutated: %q", again)
	}
}

func TestMemoryConcurrentIncrAndGetDel(t *testing.T) {
	m := NewMemory()
	defer m.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); m.Incr(ctx, "n", time.Minute) }()
	}
	wg.Wait()
	if b, _ := m.Get(ctx, "n"); string(b) != "100" {
		t.Fatalf("lost increments: %s", b)
	}

	// A one-shot token must be redeemable exactly once under contention.
	m.Set(ctx, "tok", []byte("1"), time.Minute)
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := m.GetDel(ctx, "tok"); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("token redeemed %d times", wins)
	}
}

func TestNewFactory(t *testing.T) {
	if c, err := New(cfgDriver("")); err != nil {
		t.Fatal(err)
	} else {
		c.Close()
	}
	if _, err := New(cfgDriver("bogus")); err == nil {
		t.Fatal("unknown driver must fail")
	}
	if _, err := New(cfgDriver("redis")); err == nil {
		t.Fatal("redis without addr must fail")
	}
}

func TestRedis(t *testing.T) {
	addr := os.Getenv("NX_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("NX_TEST_REDIS_ADDR not set")
	}
	r, err := NewRedis(addr, "", 15, "nxtest:")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	suite(t, r)
}

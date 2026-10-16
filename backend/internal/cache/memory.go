package cache

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type memEntry struct {
	val    []byte
	expire time.Time
}

func (e *memEntry) expired(now time.Time) bool {
	return !e.expire.IsZero() && !now.Before(e.expire)
}

type Memory struct {
	mu   sync.Mutex
	data map[string]*memEntry
	stop chan struct{}
	once sync.Once
}

func NewMemory() *Memory {
	m := &Memory{data: map[string]*memEntry{}, stop: make(chan struct{})}
	go m.janitor()
	return m
}

func (m *Memory) janitor() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-m.stop:
			return
		case now := <-t.C:
			m.mu.Lock()
			for k, e := range m.data {
				if e.expired(now) {
					delete(m.data, k)
				}
			}
			m.mu.Unlock()
		}
	}
}

func expireAt(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

// live returns the entry only if present and unexpired; callers hold m.mu.
func (m *Memory) live(key string) *memEntry {
	e, ok := m.data[key]
	if !ok {
		return nil
	}
	if e.expired(time.Now()) {
		delete(m.data, key)
		return nil
	}
	return e
}

func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.live(key)
	if e == nil {
		return nil, ErrMiss
	}
	return append([]byte(nil), e.val...), nil
}

func (m *Memory) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = &memEntry{val: append([]byte(nil), val...), expire: expireAt(ttl)}
	return nil
}

func (m *Memory) GetDel(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.live(key)
	if e == nil {
		return nil, ErrMiss
	}
	delete(m.data, key)
	return e.val, nil
}

func (m *Memory) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *Memory) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.live(key)
	if e == nil {
		m.data[key] = &memEntry{val: []byte("1"), expire: expireAt(ttl)}
		return 1, nil
	}
	n, _ := strconv.ParseInt(string(e.val), 10, 64)
	n++
	e.val = []byte(strconv.FormatInt(n, 10))
	return n, nil
}

func (m *Memory) TTL(_ context.Context, key string) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.live(key)
	if e == nil {
		return 0, ErrMiss
	}
	if e.expire.IsZero() {
		return 0, nil
	}
	return time.Until(e.expire), nil
}

func (m *Memory) Close() error {
	m.once.Do(func() { close(m.stop) })
	return nil
}

package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	rdb    *redis.Client
	prefix string
}

func NewRedis(addr, password string, db int, prefix string) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, err
	}
	return &Redis{rdb: rdb, prefix: prefix}, nil
}

func (r *Redis) k(key string) string { return r.prefix + key }

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	b, err := r.rdb.Get(ctx, r.k(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrMiss
	}
	return b, err
}

func (r *Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return r.rdb.Set(ctx, r.k(key), val, max(ttl, 0)).Err()
}

func (r *Redis) GetDel(ctx context.Context, key string) ([]byte, error) {
	b, err := r.rdb.GetDel(ctx, r.k(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrMiss
	}
	return b, err
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	full := make([]string, len(keys))
	for i, k := range keys {
		full[i] = r.k(k)
	}
	return r.rdb.Del(ctx, full...).Err()
}

// incrScript sets the TTL only when the counter is created, giving a fixed window.
var incrScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 and tonumber(ARGV[1]) > 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n`)

func (r *Redis) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return incrScript.Run(ctx, r.rdb, []string{r.k(key)}, ttl.Milliseconds()).Int64()
}

func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := r.rdb.PTTL(ctx, r.k(key)).Result()
	if err != nil {
		return 0, err
	}
	if d == -2 {
		return 0, ErrMiss
	}
	if d < 0 {
		return 0, nil
	}
	return d, nil
}

func (r *Redis) Close() error { return r.rdb.Close() }

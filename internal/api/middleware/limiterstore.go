package middleware

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiterStorage implements fiber.Storage on top of Redis so that rate
// limit counters are shared by every API replica. Without it each process keeps
// its own counters and the effective limit is multiplied by the replica count.
//
// It fails open: if Redis is unreachable the limiter behaves as if no request
// had been seen, rather than turning a Redis outage into an API outage.
// Authentication itself already depends on Redis (refresh tokens, revocation),
// so those paths fail closed on their own.
type RedisLimiterStorage struct {
	rdb    *redis.Client
	prefix string
}

// NewRedisLimiterStorage returns storage whose keys are namespaced by prefix,
// so limiters sharing a client IP key do not share counters.
func NewRedisLimiterStorage(rdb *redis.Client, prefix string) *RedisLimiterStorage {
	return &RedisLimiterStorage{rdb: rdb, prefix: "ratelimit:" + prefix + ":"}
}

const limiterOpTimeout = 250 * time.Millisecond

func (s *RedisLimiterStorage) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), limiterOpTimeout)
	defer cancel()
	val, err := s.rdb.Get(ctx, s.prefix+key).Bytes()
	if err != nil {
		return nil, nil // redis.Nil (miss) and outages both mean "no state"
	}
	return val, nil
}

func (s *RedisLimiterStorage) Set(key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), limiterOpTimeout)
	defer cancel()
	_ = s.rdb.Set(ctx, s.prefix+key, val, exp).Err()
	return nil
}

func (s *RedisLimiterStorage) Delete(key string) error {
	if key == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), limiterOpTimeout)
	defer cancel()
	_ = s.rdb.Del(ctx, s.prefix+key).Err()
	return nil
}

// Reset removes this limiter's keys.
func (s *RedisLimiterStorage) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter := s.rdb.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		_ = s.rdb.Del(ctx, iter.Val()).Err()
	}
	return nil
}

// Close is a no-op: the Redis client is owned by the caller.
func (s *RedisLimiterStorage) Close() error { return nil }

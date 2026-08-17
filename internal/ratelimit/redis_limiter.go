package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var incrAndExpireScript = redis.NewScript(`
				local current = redis.call("INCR", KEYS[1])
				if current == 1 then
		        	redis.call("PEXPIRE", KEYS[1], ARGV[1])
		    	end
				return current
			`)

type RedisLimiter struct {
	client *redis.Client
	prefix string
}

func NewRedisLimiter(client *redis.Client, prefix string) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		prefix: prefix,
	}
}

func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		return true, nil
	}

	redisKey := l.prefix + ":" + key

	ttlMillis := window.Milliseconds()
	if ttlMillis <= 0 {
		return false, fmt.Errorf("rate limit window must be at least 1 millisecond")
	}

	count, err := incrAndExpireScript.Run(
		ctx,
		l.client,
		[]string{redisKey},
		ttlMillis,
	).Int64()

	if err != nil {
		return false, fmt.Errorf("run redis rate limit script: %w", err)
	}

	return count <= int64(limit), nil
}

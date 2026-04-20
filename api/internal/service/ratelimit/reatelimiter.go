package ratelimit

import (
	"context"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func New(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

var fixedWindowScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("TTL", KEYS[1])
return {current, ttl}
`)

func (r *RateLimiter) Allow(
	ctx context.Context,
	key string,
	limit int,
	window time.Duration,
) (allowed bool, current int, resetIn time.Duration, err error) {
	if limit <= 0 {
		return true, 0, 0, nil
	}

	seconds := int(window.Seconds())
	if seconds <= 0 {
		seconds = 60
	}

	result, err := fixedWindowScript.Run(ctx, r.client, []string{key}, seconds).Result()
	if err != nil {
		return false, 0, 0, fmt.Errorf("run redis rate limit script: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return false, 0, 0, fmt.Errorf("unexpected redis script result: %#v", result)
	}

	current64, ok := values[0].(int64)
	if !ok {
		return false, 0, 0, fmt.Errorf("unexpected current count type: %#v", values[0])
	}

	ttl64, ok := values[1].(int64)
	if !ok {
		return false, 0, 0, fmt.Errorf("unexpected ttl type: %#v", values[1])
	}

	current = int(current64)
	resetIn = time.Duration(ttl64) * time.Second
	allowed = current <= limit

	return allowed, current, resetIn, nil
}

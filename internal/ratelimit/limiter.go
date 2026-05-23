package ratelimit

import (
	"context"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type Limiter interface {
	Wait(ctx context.Context, key string, ratePerSec int) error
}

type redisLimiter struct {
	limiter *redis_rate.Limiter
}

func NewRedis(client *redis.Client) Limiter {
	return &redisLimiter{limiter: redis_rate.NewLimiter(client)}
}

func (r *redisLimiter) Wait(ctx context.Context, key string, ratePerSec int) error {
	for {
		res, err := r.limiter.Allow(ctx, key, redis_rate.PerSecond(ratePerSec))
		if err != nil {
			return err
		}
		if res.Allowed > 0 {
			return nil
		}
		wait := res.RetryAfter
		if wait <= 0 {
			wait = 10 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

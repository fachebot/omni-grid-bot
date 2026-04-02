package variational

import (
	"context"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/logger"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	if burst <= 0 {
		burst = 1
	}
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
	start := time.Now()
	err := rl.limiter.Wait(ctx)
	elapsed := time.Since(start)
	if elapsed > 0 {
		logger.Infof("[VariationalRateLimiter] waited %v for request", elapsed)
	}
	return err
}

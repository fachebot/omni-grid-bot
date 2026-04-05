package lighter

import (
	"context"
	"math"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/logger"
	"golang.org/x/time/rate"
)

const (
	MaxTxPerBatch = 40
)

type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(requestsPer60Seconds int) *RateLimiter {
	requestsPerSecond := float64(requestsPer60Seconds) / 60.0

	logger.Infof("[LighterRateLimiter] Created with %d requests per 60s (limit=%.3f req/s, burst=%d)",
		requestsPer60Seconds, requestsPerSecond, requestsPer60Seconds)

	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPer60Seconds),
	}
}

func (rl *RateLimiter) Wait(ctx context.Context, endpoint string) error {
	logger.Infof("[LighterRateLimiter] Wait called for endpoint %s", endpoint)
	return nil
}

func (rl *RateLimiter) WaitBatch(ctx context.Context, endpoint string, count int) error {
	logger.Infof("[LighterRateLimiter] WaitBatch called: endpoint=%s, count=%d", endpoint, count)

	batches := (count + MaxTxPerBatch - 1) / MaxTxPerBatch
	logger.Infof("[LighterRateLimiter] Will send %d batches (MaxTxPerBatch=%d)", batches, MaxTxPerBatch)

	for i := 0; i < batches; i++ {
		remaining := count - i*MaxTxPerBatch
		currentBatchSize := int(math.Min(float64(MaxTxPerBatch), float64(remaining)))

		logger.Infof("[LighterRateLimiter] Batch %d/%d: requesting %d tokens", i+1, batches, currentBatchSize)

		start := time.Now()
		err := rl.limiter.WaitN(ctx, currentBatchSize)
		elapsed := time.Since(start)

		logger.Infof("[LighterRateLimiter] Batch %d/%d: waited %v", i+1, batches, elapsed)

		if err != nil {
			logger.Errorf("[LighterRateLimiter] Batch %d/%d: error=%v", i+1, batches, err)
			return err
		}
	}

	logger.Infof("[LighterRateLimiter] WaitBatch completed successfully")
	return nil
}

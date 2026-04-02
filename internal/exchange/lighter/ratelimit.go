package lighter

import (
	"context"
	"math"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/logger"
	"golang.org/x/time/rate"
)

const (
	WeightSendTx         = 6
	WeightApikeys        = 150
	WeightTx             = 300
	WeightAccount        = 300
	WeightOrderBook      = 300
	WeightAccountTxs     = 300
	WeightAccountOrders  = 300
	WeightInactiveOrders = 100
)

var endpointWeights = map[string]int{
	"nextNonce":             WeightSendTx,
	"sendTx":                WeightSendTx,
	"sendTxBatch":           WeightSendTx,
	"apikeys":               WeightApikeys,
	"tx":                    WeightTx,
	"account":               WeightAccount,
	"accountTxs":            WeightAccountTxs,
	"accountActiveOrders":   WeightAccountOrders,
	"accountInactiveOrders": WeightInactiveOrders,
	"orderBookDetails":      WeightOrderBook,
	"orderBooks":            WeightOrderBook,
}

type RateLimiter struct {
	limiter           *rate.Limiter
	requestsPerMinute int
	batchSize         int
}

func NewRateLimiter(requestsPerMinute, batchSize int) *RateLimiter {
	if batchSize <= 0 {
		batchSize = 10
	}
	return &RateLimiter{
		limiter:           rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), requestsPerMinute),
		requestsPerMinute: requestsPerMinute,
		batchSize:         batchSize,
	}
}

func (rl *RateLimiter) BatchSize() int {
	return rl.batchSize
}

func (rl *RateLimiter) Wait(ctx context.Context, endpoint string) error {
	weight := endpointWeights[endpoint]
	if weight == 0 {
		weight = 300
	}

	start := time.Now()
	err := rl.limiter.WaitN(ctx, weight)
	elapsed := time.Since(start)
	if elapsed > 0 {
		logger.Infof("[LighterRateLimiter] waited %v for endpoint %s (weight: %d)", elapsed, endpoint, weight)
	}
	return err
}

func (rl *RateLimiter) WaitN(ctx context.Context, endpoint string, count int) error {
	weight := endpointWeights[endpoint]
	if weight == 0 {
		weight = 300
	}

	start := time.Now()
	err := rl.limiter.WaitN(ctx, weight*count)
	elapsed := time.Since(start)
	if elapsed > 0 {
		logger.Infof("[LighterRateLimiter] waited %v for endpoint %s (weight: %d x %d)", elapsed, endpoint, weight, count)
	}
	return err
}

func (rl *RateLimiter) WaitBatch(ctx context.Context, endpoint string, count, batchSize int) error {
	weight := endpointWeights[endpoint]
	if weight == 0 {
		weight = 300
	}

	batches := (count + batchSize - 1) / batchSize

	for i := 0; i < batches; i++ {
		remaining := count - i*batchSize
		currentBatchSize := int(math.Min(float64(batchSize), float64(remaining)))

		start := time.Now()
		err := rl.limiter.WaitN(ctx, weight*currentBatchSize)
		elapsed := time.Since(start)
		if elapsed > 0 {
			logger.Infof("[LighterRateLimiter] waited %v for batch %d/%d, size %d", elapsed, i+1, batches, currentBatchSize)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

package lighter

import (
	"context"
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
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		limiter:           rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), requestsPerMinute),
		requestsPerMinute: requestsPerMinute,
	}
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

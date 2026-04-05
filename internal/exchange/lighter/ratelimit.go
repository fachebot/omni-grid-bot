package lighter

import (
	"context"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/logger"
)

const (
	MaxTxPerBatch = 40
)

type RateLimiter struct {
	mu                   sync.Mutex
	allowedRequests      int       // 当前时间窗口内可用的请求数
	windowStartTime      time.Time // 当前窗口的开始时间
	requestsPer60Seconds int       // 60秒窗口允许的总请求数
}

func NewRateLimiter(requestsPer60Seconds int) *RateLimiter {
	now := time.Now()

	logger.Infof("[LighterRateLimiter] Created with %d requests per 60s",
		requestsPer60Seconds)

	return &RateLimiter{
		allowedRequests:      requestsPer60Seconds,
		windowStartTime:      now,
		requestsPer60Seconds: requestsPer60Seconds,
	}
}

func (rl *RateLimiter) Wait(ctx context.Context, endpoint string) error {
	return rl.WaitBatch(ctx, endpoint, 1)
}

func (rl *RateLimiter) WaitBatch(ctx context.Context, endpoint string, count int) error {
	logger.Infof("[LighterRateLimiter] WaitBatch called: endpoint=%s, count=%d", endpoint, count)

	if count == 0 {
		return nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowDuration := now.Sub(rl.windowStartTime)

	logger.Infof("[LighterRateLimiter] Window duration: %v, window started: %v",
		windowDuration, rl.windowStartTime.Format("15:04:05.000"))

	// 如果窗口已过期（超过60秒），重置窗口
	if windowDuration >= 60*time.Second {
		rl.allowedRequests = rl.requestsPer60Seconds
		rl.windowStartTime = now
		windowDuration = 0

		logger.Infof("[LighterRateLimiter] New 60s window started, %d requests available",
			rl.allowedRequests)
	}

	// 检查是否有足够的请求配额
	if rl.allowedRequests >= count {
		// 当前窗口有足够的配额，直接使用
		rl.allowedRequests -= count

		logger.Infof("[LighterRateLimiter] Granted %d requests, %d remaining",
			count, rl.allowedRequests)

		return nil
	}

	// 配额不足，必须等待完整的60秒
	// 因为一旦当前窗口的配额用完，就必须等待下一个60秒窗口
	logger.Infof("[LighterRateLimiter] Not enough requests. Window has %d available, need %d. Waiting full 60s for next window.",
		rl.allowedRequests, count)

	select {
	case <-time.After(60 * time.Second):
		// 等待60秒后，进入新窗口
		rl.windowStartTime = time.Now()
		rl.allowedRequests = rl.requestsPer60Seconds - count

		logger.Infof("[LighterRateLimiter] New window started after 60s wait, granted %d requests, %d remaining",
			count, rl.allowedRequests)

		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

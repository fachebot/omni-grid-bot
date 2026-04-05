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

	if windowDuration >= 60*time.Second {
		// 新的60秒窗口
		rl.allowedRequests = rl.requestsPer60Seconds
		rl.windowStartTime = now

		logger.Infof("[LighterRateLimiter] New 60s window started, %d requests available",
			rl.allowedRequests)
	}

	if rl.allowedRequests >= count {
		// 当前窗口有足够的配额
		rl.allowedRequests -= count

		logger.Infof("[LighterRateLimiter] Granted %d requests, %d remaining",
			count, rl.allowedRequests)

		return nil
	}

	// 需要等待下一个窗口
	neededRequests := count - rl.allowedRequests
	waitTime := 60*time.Second - windowDuration

	logger.Infof("[LighterRateLimiter] Not enough requests. Need %d more, window has %d available, waiting %v for next window",
		neededRequests, rl.allowedRequests, waitTime)

	if waitTime > 0 {
		select {
		case <-time.After(waitTime):
			// 等待完成，进入新窗口
			rl.windowStartTime = time.Now()
			rl.allowedRequests = rl.requestsPer60Seconds - neededRequests

			logger.Infof("[LighterRateLimiter] New window started, granted %d requests, %d remaining",
				neededRequests, rl.allowedRequests)

			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// 等待时间为0，立即进入新窗口
	rl.windowStartTime = now
	rl.allowedRequests = rl.requestsPer60Seconds - neededRequests

	logger.Infof("[LighterRateLimiter] New window started, granted %d requests, %d remaining",
		neededRequests, rl.allowedRequests)

	return nil
}

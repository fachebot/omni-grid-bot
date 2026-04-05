package lighter

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRateLimiterTimeSlot_Basic(t *testing.T) {
	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("Within window, grant immediately", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 20)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}

		if elapsed > 100*time.Millisecond {
			t.Errorf("Should be fast (within window), took %v", elapsed)
		}
	})

	t.Run("Exceeds window, should wait", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 50)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}

		if elapsed < 59*time.Second {
			t.Errorf("Should wait (exceeds window), took %v", elapsed)
		}
	})
}

func TestRateLimiterTimeSlot_WindowReset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	rl := NewRateLimiter(6)
	ctx := context.Background()

	t.Run("First window: use all quota", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 6)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}

		if elapsed > 100*time.Millisecond {
			t.Errorf("First window should be fast (use quota), took %v", elapsed)
		}
	})

	time.Sleep(61 * time.Second)

	t.Run("Second window: should have new quota", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 6)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}

		if elapsed > 100*time.Millisecond {
			t.Errorf("Second window should be fast (new quota), took %v", elapsed)
		}
	})
}

func TestRateLimiterTimeSlot_ExactBoundary(t *testing.T) {
	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("Exactly 40 requests in first window", func(t *testing.T) {
		t.Log("Test: Exactly 40 requests in first 60s window")

		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 40)
		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}
		elapsed1 := time.Since(start)
		t.Logf("First batch (40 requests): waited %v", elapsed1)

		if elapsed1 > 100*time.Millisecond {
			t.Errorf("First batch should be fast (use quota), took %v", elapsed1)
		}
	})

	time.Sleep(100 * time.Millisecond)

	t.Run("1 request exceeds window, should wait", func(t *testing.T) {
		t.Log("Test: 1 request exceeds window, should wait for next window")

		start2 := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 1)
		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}
		elapsed2 := time.Since(start2)
		t.Logf("Second batch (1 request): waited %v", elapsed2)

		if elapsed2 < 59*time.Second {
			t.Errorf("Second batch should wait ~60s (need next window), took %v", elapsed2)
		}

		if elapsed2 > 62*time.Second {
			t.Errorf("Second batch should not wait too long, took %v", elapsed2)
		}
	})
}

func TestRateLimiterTimeSlot_MultipleRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("10 requests of 4 each", func(t *testing.T) {
		t.Log("Test: 10 requests of 4 each, total 40")

		totalStart := time.Now()

		for i := 0; i < 10; i++ {
			start := time.Now()
			err := rl.WaitBatch(ctx, "sendTx", 4)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("WaitBatch failed: %v", err)
			}

			t.Logf("Request %d: waited %v", i+1, elapsed)

			if elapsed > 100*time.Millisecond {
				t.Errorf("Request %d should be fast (within window), took %v", i+1, elapsed)
			}
		}

		totalElapsed := time.Since(totalStart)
		t.Logf("Total time: %v", totalElapsed)

		if totalElapsed > 1*time.Second {
			t.Errorf("Total time should be fast (within window), took %v", totalElapsed)
		}
	})

	time.Sleep(61 * time.Second)

	t.Run("Another 10 requests of 4 each, should wait", func(t *testing.T) {
		t.Log("Test: Another 10 requests in new window")

		totalStart := time.Now()

		for i := 0; i < 10; i++ {
			start := time.Now()
			err := rl.WaitBatch(ctx, "sendTx", 4)
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("WaitBatch failed: %v", err)
			}

			t.Logf("Request %d: waited %v", i+1, elapsed)

			if elapsed > 100*time.Millisecond {
				t.Errorf("Request %d should be fast (new window), took %v", i+1, elapsed)
			}
		}

		totalElapsed := time.Since(totalStart)
		t.Logf("Total time: %v", totalElapsed)

		if totalElapsed > 1*time.Second {
			t.Errorf("Total time should be fast (new window), took %v", totalElapsed)
		}
	})
}

func TestRateLimiterTimeSlot_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("Concurrent goroutines", func(t *testing.T) {
		t.Log("Test: 3 goroutines each request 10, total 30")

		var wg sync.WaitGroup
		errors := make(chan error, 3)

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 10; j++ {
					start := time.Now()
					err := rl.WaitBatch(ctx, "sendTx", 1)
					elapsed := time.Since(start)

					if err != nil {
						errors <- err
						return
					}

					t.Logf("Goroutine %d, Request %d: waited %v", id, j+1, elapsed)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent request failed: %v", err)
		}
	})
}

func TestRateLimiterTimeSlot_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}

	rl := NewRateLimiter(10)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.Run("Cancel context during wait", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 50)
		elapsed := time.Since(start)

		t.Logf("Waited %v before cancellation", elapsed)

		if err != context.DeadlineExceeded {
			t.Errorf("Expected deadline exceeded context error, got: %v", err)
		}

		if elapsed < 2*time.Second {
			t.Errorf("Should wait until timeout, took %v", elapsed)
		}
	})
}

func TestRateLimiterTimeSlot_EdgeCases(t *testing.T) {
	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("Zero requests", func(t *testing.T) {
		err := rl.WaitBatch(ctx, "sendTx", 0)
		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}
	})

	t.Run("Exactly at window limit", func(t *testing.T) {
		start := time.Now()
		err := rl.WaitBatch(ctx, "sendTx", 40)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("WaitBatch failed: %v", err)
		}

		if elapsed > 100*time.Millisecond {
			t.Errorf("Should be fast (use quota), took %v", elapsed)
		}

		time.Sleep(61 * time.Second)

		start2 := time.Now()
		err2 := rl.WaitBatch(ctx, "sendTx", 1)
		elapsed2 := time.Since(start2)

		if err2 != nil {
			t.Errorf("WaitBatch failed: %v", err2)
		}

		if elapsed2 < 59*time.Second {
			t.Errorf("Should wait ~60s (need next window), took %v", elapsed2)
		}
	})
}

func TestRateLimiterTimeSlot_WaitMethod(t *testing.T) {
	rl := NewRateLimiter(40)
	ctx := context.Background()

	t.Run("Wait method calls WaitBatch with count=1", func(t *testing.T) {
		start := time.Now()
		err := rl.Wait(ctx, "sendTx")
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Wait failed: %v", err)
		}

		t.Logf("Wait() took %v", elapsed)
	})
}

func BenchmarkRateLimiterTimeSlot(b *testing.B) {
	rl := NewRateLimiter(1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.WaitBatch(ctx, "sendTx", 10)
	}
}

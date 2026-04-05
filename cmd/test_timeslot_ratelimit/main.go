package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("Lighter RateLimiter Time Slot Test")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Println("Initializing RateLimiter with 40 requests per 60s...")
	rl := lighter.NewRateLimiter(40)
	ctx := context.Background()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("Test Scenario 1: 40 + 10 requests in same window")
	fmt.Println("========================================")
	fmt.Println("Expected behavior:")
	fmt.Println("  - First batch (40 reqs): should be fast (use burst)")
	fmt.Println("  - Second batch (10 reqs): should wait ~60 seconds (need next window)")
	fmt.Println("  - Total time: ~60 seconds")
	fmt.Println()

	totalStart := time.Now()

	test1Start := time.Now()
	fmt.Println("Step 1: WaitBatch (40 requests)")
	err := rl.WaitBatch(ctx, "sendTx", 40)
	if err != nil {
		log.Fatalf("WaitBatch failed: %v", err)
	}
	test1Elapsed := time.Since(test1Start)
	fmt.Printf("  Waited: %v\n", test1Elapsed)
	fmt.Println()

	time.Sleep(100 * time.Millisecond)

	test2Start := time.Now()
	fmt.Println("Step 2: WaitBatch (10 requests)")
	fmt.Println("  Expected: should wait ~60 seconds (need next window)...")
	err = rl.WaitBatch(ctx, "sendTx", 10)
	if err != nil {
		log.Fatalf("WaitBatch failed: %v", err)
	}
	test2Elapsed := time.Since(test2Start)
	fmt.Printf("  Waited: %v\n", test2Elapsed)
	fmt.Println()

	totalElapsed := time.Since(totalStart)

	fmt.Println("========================================")
	fmt.Println("Test Results")
	fmt.Println("========================================")
	fmt.Printf("Test 1.1 (40 reqs):  Waited: %v\n", test1Elapsed)
	fmt.Printf("Test 1.2 (10 reqs):  Waited: %v\n", test2Elapsed)
	fmt.Printf("Total time:           %v\n", totalElapsed)
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("Validation")
	fmt.Println("========================================")

	allPassed := true

	if test1Elapsed < 100*time.Millisecond {
		fmt.Println("PASS Test 1.1: Fast (used burst)")
	} else {
		fmt.Printf("FAIL Test 1.1: Too slow (%v)\n", test1Elapsed)
		allPassed = false
	}

	if test2Elapsed >= 59*time.Second && test2Elapsed <= 61*time.Second {
		fmt.Println("PASS Test 1.2: Waited ~60 seconds (CORRECT!)")
	} else {
		fmt.Printf("FAIL Test 1.2: Unexpected wait time (%v, expected ~60s)\n", test2Elapsed)
		allPassed = false
	}

	if totalElapsed >= 59*time.Second && totalElapsed <= 61*time.Second {
		fmt.Println("PASS Total: ~60 seconds (correct time slot behavior!)")
	} else {
		fmt.Printf("FAIL Total: Unexpected time (%v, expected ~60s)\n", totalElapsed)
		allPassed = false
	}

	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("Test Scenario 2: Wait 60 seconds, then 40 requests")
	fmt.Println("========================================")
	fmt.Println("Expected behavior:")
	fmt.Println("  - Should have new window with 40 requests available")
	fmt.Println("  - Wait should be fast (use new window)")
	fmt.Println()

	time.Sleep(61 * time.Second)

	fmt.Println("Step 3: WaitBatch (40 requests after waiting 60s)")
	test3Start := time.Now()
	err = rl.WaitBatch(ctx, "sendTx", 40)
	if err != nil {
		log.Fatalf("WaitBatch failed: %v", err)
	}
	test3Elapsed := time.Since(test3Start)
	fmt.Printf("  Waited: %v\n", test3Elapsed)
	fmt.Println()

	fmt.Println("========================================")
	fmt.Println("Test 2 Results")
	fmt.Println("========================================")

	if test3Elapsed < 100*time.Millisecond {
		fmt.Println("PASS Test 2.1: Fast (new window available)")
	} else {
		fmt.Printf("FAIL Test 2.1: Too slow (%v)\n", test3Elapsed)
		allPassed = false
	}

	fmt.Println()
	if allPassed {
		fmt.Println("========================================")
		fmt.Println("ALL TESTS PASSED!")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Println("The time slot rate limiter is working correctly.")
		fmt.Println("It strictly enforces 40 requests per 60-second window.")
		fmt.Println()
		fmt.Println("Key behavior:")
		fmt.Println("  - First 40 requests: immediate (use burst)")
		fmt.Println("  - Next 10 requests: wait ~60s (need next window)")
		fmt.Println("  - After 60s: new window with 40 requests available")
	} else {
		fmt.Println("========================================")
		fmt.Println("TESTS FAILED!")
		fmt.Println("========================================")
	}
}

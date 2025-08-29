package performance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"family-budget-service/internal/application"
)

// TestConcurrentUsers simulates multiple concurrent users
func TestConcurrentUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	tests := []struct {
		name            string
		concurrency     int
		requestsPerUser int
		endpoint        string
		timeout         time.Duration
	}{
		{
			name:            "Light Load",
			concurrency:     10,
			requestsPerUser: 10,
			endpoint:        "/health",
			timeout:         10 * time.Second,
		},
		{
			name:            "Medium Load",
			concurrency:     50,
			requestsPerUser: 20,
			endpoint:        "/health",
			timeout:         30 * time.Second,
		},
		{
			name:            "Heavy Load",
			concurrency:     100,
			requestsPerUser: 10,
			endpoint:        "/health",
			timeout:         60 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			var (
				successCount int64
				errorCount   int64
				totalLatency int64
				wg           sync.WaitGroup
			)

			startTime := time.Now()

			// Launch concurrent users
			for i := range tc.concurrency {
				wg.Add(1)
				go func(userID int) {
					defer wg.Done()

					client := &http.Client{
						Timeout: 5 * time.Second,
					}

					for range tc.requestsPerUser {
						select {
						case <-ctx.Done():
							return
						default:
						}

						reqStart := time.Now()
						resp, err := client.Get(testServer.URL + tc.endpoint)
						latency := time.Since(reqStart)

						if err != nil {
							atomic.AddInt64(&errorCount, 1)
							continue
						}

						resp.Body.Close()

						if resp.StatusCode == http.StatusOK {
							atomic.AddInt64(&successCount, 1)
							atomic.AddInt64(&totalLatency, int64(latency))
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(startTime)

			// Calculate metrics
			totalRequests := successCount + errorCount
			successRate := float64(successCount) / float64(totalRequests) * 100
			avgLatency := time.Duration(totalLatency / successCount)
			rps := float64(totalRequests) / duration.Seconds()

			t.Logf("Load Test Results for %s:", tc.name)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Total Requests: %d", totalRequests)
			t.Logf("  Successful: %d", successCount)
			t.Logf("  Failed: %d", errorCount)
			t.Logf("  Success Rate: %.2f%%", successRate)
			t.Logf("  Average Latency: %v", avgLatency)
			t.Logf("  Requests/sec: %.2f", rps)

			// Performance assertions
			if successRate < 95.0 {
				t.Errorf("Success rate (%.2f%%) below threshold (95%%)", successRate)
			}

			if avgLatency > 100*time.Millisecond {
				t.Errorf("Average latency (%v) exceeds threshold (100ms)", avgLatency)
			}

			// RPS expectations vary by load level
			var minRPS float64
			switch tc.name {
			case "Light Load":
				minRPS = 50
			case "Medium Load":
				minRPS = 100
			case "Heavy Load":
				minRPS = 150
			}

			if rps < minRPS {
				t.Errorf("RPS (%.2f) below threshold (%.2f)", rps, minRPS)
			}
		})
	}
}

// TestSustainedLoad tests performance under sustained load
func TestSustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	const (
		duration    = 30 * time.Second
		concurrency = 20
		targetRPS   = 100
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var (
		requestCount int64
		errorCount   int64
		wg           sync.WaitGroup
	)

	startTime := time.Now()

	// Launch concurrent workers
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			ticker := time.NewTicker(time.Second / time.Duration(targetRPS/concurrency))
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					resp, err := client.Get(testServer.URL + "/health")
					atomic.AddInt64(&requestCount, 1)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						continue
					}

					resp.Body.Close()

					if resp.StatusCode != http.StatusOK {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}
		}()
	}

	wg.Wait()
	testDuration := time.Since(startTime)

	// Calculate final metrics
	totalRequests := atomic.LoadInt64(&requestCount)
	errors := atomic.LoadInt64(&errorCount)
	successRate := float64(totalRequests-errors) / float64(totalRequests) * 100
	actualRPS := float64(totalRequests) / testDuration.Seconds()

	t.Logf("Sustained Load Test Results:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Errors: %d", errors)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Target RPS: %d", targetRPS)
	t.Logf("  Actual RPS: %.2f", actualRPS)

	// Performance assertions
	if successRate < 98.0 {
		t.Errorf("Success rate (%.2f%%) below threshold (98%%) for sustained load", successRate)
	}

	if actualRPS < float64(targetRPS)*0.8 {
		t.Errorf("Actual RPS (%.2f) significantly below target (%d)", actualRPS, targetRPS)
	}
}

// TestRampUpLoad tests performance during gradual load increase
func TestRampUpLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ramp-up load test in short mode")
	}

	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	const (
		totalDuration  = 60 * time.Second
		maxConcurrency = 100
		stepDuration   = 10 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), totalDuration)
	defer cancel()

	var (
		requestCount int64
		errorCount   int64
		mu           sync.Mutex
		workers      = make(map[int]context.CancelFunc)
	)

	steps := int(totalDuration / stepDuration)
	concurrencyStep := maxConcurrency / steps

	t.Logf("Starting ramp-up load test: %d steps, %d users per step", steps, concurrencyStep)

	startTime := time.Now()

	for step := 1; step <= steps; step++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		currentConcurrency := step * concurrencyStep

		// Start new workers for this step
		for i := (step - 1) * concurrencyStep; i < currentConcurrency; i++ {
			workerCtx, workerCancel := context.WithCancel(ctx)

			mu.Lock()
			workers[i] = workerCancel
			mu.Unlock()

			go func(workerID int) {
				client := &http.Client{
					Timeout: 5 * time.Second,
				}

				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-workerCtx.Done():
						return
					case <-ticker.C:
						resp, err := client.Get(testServer.URL + "/health")
						atomic.AddInt64(&requestCount, 1)

						if err != nil {
							atomic.AddInt64(&errorCount, 1)
							continue
						}

						resp.Body.Close()

						if resp.StatusCode != http.StatusOK {
							atomic.AddInt64(&errorCount, 1)
						}
					}
				}
			}(i)
		}

		t.Logf("Step %d: %d concurrent users", step, currentConcurrency)
		time.Sleep(stepDuration)
	}

	// Stop all workers
	mu.Lock()
	for _, cancel := range workers {
		cancel()
	}
	mu.Unlock()

	testDuration := time.Since(startTime)

	// Calculate final metrics
	totalRequests := atomic.LoadInt64(&requestCount)
	errors := atomic.LoadInt64(&errorCount)
	successRate := float64(totalRequests-errors) / float64(totalRequests) * 100
	avgRPS := float64(totalRequests) / testDuration.Seconds()

	t.Logf("Ramp-up Load Test Results:")
	t.Logf("  Duration: %v", testDuration)
	t.Logf("  Max Concurrency: %d", maxConcurrency)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Errors: %d", errors)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Average RPS: %.2f", avgRPS)

	// Performance assertions
	if successRate < 95.0 {
		t.Errorf("Success rate (%.2f%%) below threshold (95%%) for ramp-up load", successRate)
	}

	if totalRequests == 0 {
		t.Error("No requests completed during ramp-up test")
	}
}

// TestSpikeLoad tests performance under sudden load spikes
func TestSpikeLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping spike load test in short mode")
	}

	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	phases := []struct {
		name        string
		concurrency int
		duration    time.Duration
		expectation string
	}{
		{
			name:        "Baseline",
			concurrency: 10,
			duration:    10 * time.Second,
			expectation: "stable performance",
		},
		{
			name:        "Spike",
			concurrency: 200,
			duration:    5 * time.Second,
			expectation: "handle sudden load increase",
		},
		{
			name:        "Recovery",
			concurrency: 10,
			duration:    10 * time.Second,
			expectation: "return to stable performance",
		},
	}

	var overallResults []map[string]any

	for _, phase := range phases {
		t.Logf("Starting phase: %s (%d users, %v)", phase.name, phase.concurrency, phase.duration)

		ctx, cancel := context.WithTimeout(context.Background(), phase.duration)

		var (
			requestCount int64
			errorCount   int64
			totalLatency int64
			wg           sync.WaitGroup
		)

		startTime := time.Now()

		// Launch workers for this phase
		for range phase.concurrency {
			wg.Add(1)
			go func() {
				defer wg.Done()

				client := &http.Client{
					Timeout: 5 * time.Second,
				}

				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					reqStart := time.Now()
					resp, err := client.Get(testServer.URL + "/health")
					latency := time.Since(reqStart)

					atomic.AddInt64(&requestCount, 1)

					if err != nil {
						atomic.AddInt64(&errorCount, 1)
						continue
					}

					resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&totalLatency, int64(latency))
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}()
		}

		wg.Wait()
		cancel()
		phaseDuration := time.Since(startTime)

		// Calculate phase metrics
		requests := atomic.LoadInt64(&requestCount)
		errors := atomic.LoadInt64(&errorCount)
		successRate := float64(requests-errors) / float64(requests) * 100
		avgLatency := time.Duration(atomic.LoadInt64(&totalLatency) / (requests - errors))
		rps := float64(requests) / phaseDuration.Seconds()

		results := map[string]any{
			"phase":       phase.name,
			"requests":    requests,
			"errors":      errors,
			"successRate": successRate,
			"avgLatency":  avgLatency,
			"rps":         rps,
			"duration":    phaseDuration,
		}
		overallResults = append(overallResults, results)

		t.Logf("Phase %s results:", phase.name)
		t.Logf("  Requests: %d", requests)
		t.Logf("  Errors: %d", errors)
		t.Logf("  Success Rate: %.2f%%", successRate)
		t.Logf("  Avg Latency: %v", avgLatency)
		t.Logf("  RPS: %.2f", rps)

		// Wait between phases
		if phase.name != "Recovery" {
			time.Sleep(2 * time.Second)
		}
	}

	// Analyze spike test results
	baseline := overallResults[0]
	spike := overallResults[1]
	recovery := overallResults[2]

	baselineRPS := baseline["rps"].(float64)
	spikeSuccessRate := spike["successRate"].(float64)
	recoveryRPS := recovery["rps"].(float64)

	t.Logf("Spike Load Test Summary:")
	t.Logf("  Baseline RPS: %.2f", baselineRPS)
	t.Logf("  Spike Success Rate: %.2f%%", spikeSuccessRate)
	t.Logf("  Recovery RPS: %.2f", recoveryRPS)

	// Performance assertions
	if spikeSuccessRate < 80.0 {
		t.Errorf("Spike phase success rate (%.2f%%) below threshold (80%%)", spikeSuccessRate)
	}

	rpsRecoveryRatio := recoveryRPS / baselineRPS
	if rpsRecoveryRatio < 0.8 {
		t.Errorf("Recovery RPS (%.2f) significantly lower than baseline (%.2f), ratio: %.2f",
			recoveryRPS, baselineRPS, rpsRecoveryRatio)
	}
}

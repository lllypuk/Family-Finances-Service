package performance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/application"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// TestConcurrentDomainOperations tests thread safety of domain operations
func TestConcurrentDomainOperations(t *testing.T) {
	tests := []struct {
		name        string
		goroutines  int
		operations  int
		operation   func(int)
		description string
	}{
		{
			name:       "ConcurrentBudgetCalculations",
			goroutines: runtime.NumCPU(),
			operations: 10000,
			operation: func(i int) {
				budget := &budget.Budget{
					Amount: 1000.0 + float64(i%100),
					Spent:  500.0 + float64(i%50),
				}
				_ = budget.GetRemainingAmount()
				_ = budget.GetSpentPercentage()
				_ = budget.IsOverBudget()
			},
			description: "Budget calculations should be thread-safe",
		},
		{
			name:       "ConcurrentTransactionCreation",
			goroutines: runtime.NumCPU() * 2,
			operations: 1000,
			operation: func(i int) {
				trans := transaction.NewTransaction(
					100.0+float64(i%100),
					transaction.TypeExpense,
					"Concurrent Transaction",
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					time.Now(),
				)
				_ = trans
			},
			description: "Transaction creation should be thread-safe",
		},
		{
			name:       "ConcurrentCategoryCreation",
			goroutines: runtime.NumCPU(),
			operations: 1000,
			operation: func(i int) {
				cat := category.NewCategory(
					"Concurrent Category",
					category.TypeExpense,
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				)
				_ = cat
			},
			description: "Category creation should be thread-safe",
		},
		{
			name:       "ConcurrentUserCreation",
			goroutines: runtime.NumCPU(),
			operations: 500,
			operation: func(i int) {
				u := user.NewUser(
					"test@example.com",
					"John",
					"Doe",
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					user.RoleMember,
				)
				_ = u
			},
			description: "User creation should be thread-safe",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				wg        sync.WaitGroup
				completed int64
				panics    int64
			)

			start := time.Now()

			// Launch goroutines
			for i := range tc.goroutines {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							atomic.AddInt64(&panics, 1)
							t.Errorf("Panic in goroutine %d: %v", goroutineID, r)
						}
					}()

					operationsPerGoroutine := tc.operations / tc.goroutines
					for j := range operationsPerGoroutine {
						tc.operation(goroutineID*operationsPerGoroutine + j)
						atomic.AddInt64(&completed, 1)
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			totalCompleted := atomic.LoadInt64(&completed)
			totalPanics := atomic.LoadInt64(&panics)
			operationsPerSecond := float64(totalCompleted) / duration.Seconds()

			t.Logf("Concurrent test results for %s:", tc.name)
			t.Logf("  %s", tc.description)
			t.Logf("  Goroutines: %d", tc.goroutines)
			t.Logf("  Operations completed: %d", totalCompleted)
			t.Logf("  Panics: %d", totalPanics)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Operations/sec: %.2f", operationsPerSecond)

			// Assertions
			if totalPanics > 0 {
				t.Errorf("Detected %d panics during concurrent operations", totalPanics)
			}

			expectedOperations := int64(tc.operations)
			if totalCompleted < expectedOperations {
				t.Errorf("Completed %d operations, expected %d", totalCompleted, expectedOperations)
			}

			// Performance expectations
			minOpsPerSec := float64(tc.operations) / 10.0 // Should complete within 10 seconds
			if operationsPerSecond < minOpsPerSec {
				t.Errorf("Performance too slow: %.2f ops/sec, expected at least %.2f",
					operationsPerSecond, minOpsPerSec)
			}
		})
	}
}

// TestConcurrentHTTPRequests tests HTTP server under concurrent load
func TestConcurrentHTTPRequests(t *testing.T) {
	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	tests := []struct {
		name         string
		endpoint     string
		concurrency  int
		requests     int
		timeout      time.Duration
		expectStatus int
	}{
		{
			name:         "Health Endpoint",
			endpoint:     "/health",
			concurrency:  50,
			requests:     1000,
			timeout:      30 * time.Second,
			expectStatus: http.StatusOK,
		},
		{
			name:         "Not Found Endpoint",
			endpoint:     "/nonexistent",
			concurrency:  20,
			requests:     500,
			timeout:      15 * time.Second,
			expectStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			var (
				wg           sync.WaitGroup
				successCount int64
				errorCount   int64
				timeoutCount int64
				totalLatency int64
			)

			requestsPerGoroutine := tc.requests / tc.concurrency
			start := time.Now()

			// Launch concurrent goroutines
			for i := range tc.concurrency {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()

					client := &http.Client{
						Timeout: 5 * time.Second,
					}

					for range requestsPerGoroutine {
						select {
						case <-ctx.Done():
							atomic.AddInt64(&timeoutCount, 1)
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

						if resp.StatusCode == tc.expectStatus {
							atomic.AddInt64(&successCount, 1)
							atomic.AddInt64(&totalLatency, int64(latency))
						} else {
							atomic.AddInt64(&errorCount, 1)
						}
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			// Calculate metrics
			totalRequests := successCount + errorCount + timeoutCount
			successRate := float64(successCount) / float64(totalRequests) * 100
			avgLatency := time.Duration(0)
			if successCount > 0 {
				avgLatency = time.Duration(totalLatency / successCount)
			}
			rps := float64(totalRequests) / duration.Seconds()

			t.Logf("Concurrent HTTP test results for %s:", tc.name)
			t.Logf("  Endpoint: %s", tc.endpoint)
			t.Logf("  Concurrency: %d", tc.concurrency)
			t.Logf("  Total requests: %d", totalRequests)
			t.Logf("  Successful: %d", successCount)
			t.Logf("  Errors: %d", errorCount)
			t.Logf("  Timeouts: %d", timeoutCount)
			t.Logf("  Success rate: %.2f%%", successRate)
			t.Logf("  Average latency: %v", avgLatency)
			t.Logf("  Requests/sec: %.2f", rps)
			t.Logf("  Duration: %v", duration)

			// Performance assertions
			if successRate < 95.0 {
				t.Errorf("Success rate (%.2f%%) below threshold (95%%)", successRate)
			}

			if avgLatency > 100*time.Millisecond {
				t.Errorf("Average latency (%v) exceeds threshold (100ms)", avgLatency)
			}

			minExpectedRPS := float64(tc.requests) / tc.timeout.Seconds() * 0.5
			if rps < minExpectedRPS {
				t.Errorf("RPS (%.2f) below minimum expected (%.2f)", rps, minExpectedRPS)
			}
		})
	}
}

// TestRaceConditions tests for potential race conditions
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	// Test for race conditions in budget operations
	t.Run("BudgetRaceConditions", func(t *testing.T) {
		budget := &budget.Budget{
			Amount: 1000.0,
			Spent:  0.0,
		}

		const (
			goroutines = 100
			iterations = 100
		)

		var wg sync.WaitGroup

		// Simulate concurrent budget updates (read-only operations)
		for i := range goroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for range iterations {
					// These operations should be safe to run concurrently
					remaining := budget.GetRemainingAmount()
					percentage := budget.GetSpentPercentage()
					overBudget := budget.IsOverBudget()

					// Basic consistency checks
					if remaining < 0 && !overBudget {
						t.Errorf("Inconsistent state: negative remaining (%f) but not over budget", remaining)
					}
					if percentage < 0 || percentage > 200 {
						t.Errorf("Invalid percentage: %f", percentage)
					}
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Budget race condition test completed successfully")
	})

	// Test configuration loading race conditions
	t.Run("ConfigRaceConditions", func(t *testing.T) {
		const goroutines = 50

		var wg sync.WaitGroup

		for i := range goroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Concurrent configuration loading
				config := &application.Config{
					Host: "localhost",
					Port: "8080",
				}

				// Basic validation
				if config.Host == "" {
					t.Errorf("Goroutine %d: empty host", id)
				}

				if config.Port == "" {
					t.Errorf("Goroutine %d: empty port", id)
				}
			}(i)
		}

		wg.Wait()
		t.Logf("Configuration race condition test completed successfully")
	})
}

// TestDeadlockDetection tests for potential deadlocks
func TestDeadlockDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deadlock detection test in short mode")
	}

	// Test with timeout to catch deadlocks
	timeout := 30 * time.Second
	done := make(chan bool, 1)

	go func() {
		// Simulate complex concurrent operations that might deadlock
		var wg sync.WaitGroup
		const goroutines = 20

		// Create shared resources
		budgets := make([]*budget.Budget, 10)
		for i := range budgets {
			budgets[i] = &budget.Budget{
				Amount: 1000.0,
				Spent:  float64(i * 100),
			}
		}

		// Launch goroutines that access multiple resources
		for i := range goroutines {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Access budgets in different orders to test for deadlocks
				for j := range 100 {
					idx1 := (id + j) % len(budgets)
					idx2 := (id + j + 1) % len(budgets)

					// Perform operations on multiple budgets
					_ = budgets[idx1].GetRemainingAmount()
					_ = budgets[idx2].GetSpentPercentage()

					// Small delay to increase chance of race conditions
					time.Sleep(time.Microsecond)
				}
			}(i)
		}

		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Logf("Deadlock detection test completed successfully")
	case <-time.After(timeout):
		t.Fatal("Potential deadlock detected - test timed out")
	}
}

// TestGoroutineLeaks tests for goroutine leaks
func TestGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	// Record initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Run operations that might leak goroutines
	t.Run("HTTPServerOperations", func(t *testing.T) {
		config := &application.Config{
			Host: "localhost",
			Port: "8080",
		}

		// Create and close multiple servers
		for range 5 {
			server := application.NewHTTPServer(nil, config)
			testServer := httptest.NewServer(server.Echo())

			// Make some requests
			for range 10 {
				resp, err := http.Get(testServer.URL + "/health")
				if err == nil {
					resp.Body.Close()
				}
			}

			testServer.Close()
		}
	})

	// Give time for cleanup
	time.Sleep(2 * time.Second)
	runtime.GC()
	time.Sleep(1 * time.Second)

	// Check final goroutine count
	finalGoroutines := runtime.NumGoroutine()
	goroutineDiff := finalGoroutines - initialGoroutines

	t.Logf("Goroutine leak test results:")
	t.Logf("  Initial goroutines: %d", initialGoroutines)
	t.Logf("  Final goroutines: %d", finalGoroutines)
	t.Logf("  Difference: %d", goroutineDiff)

	// Allow for some variance but detect significant leaks
	if goroutineDiff > 10 {
		t.Errorf("Potential goroutine leak: %d extra goroutines", goroutineDiff)
	}
}

// TestChannelPerformance tests channel operations performance
func TestChannelPerformance(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		messages   int
		producers  int
		consumers  int
	}{
		{
			name:       "Unbuffered Channel",
			bufferSize: 0,
			messages:   10000,
			producers:  1,
			consumers:  1,
		},
		{
			name:       "Buffered Channel",
			bufferSize: 100,
			messages:   10000,
			producers:  1,
			consumers:  1,
		},
		{
			name:       "Multiple Producers",
			bufferSize: 50,
			messages:   10000,
			producers:  5,
			consumers:  1,
		},
		{
			name:       "Multiple Consumers",
			bufferSize: 50,
			messages:   10000,
			producers:  1,
			consumers:  5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := make(chan int, tc.bufferSize)
			var wg sync.WaitGroup

			start := time.Now()

			// Start consumers
			consumeCount := int64(0)
			for range tc.consumers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range ch {
						atomic.AddInt64(&consumeCount, 1)
					}
				}()
			}

			// Start producers
			messagesPerProducer := tc.messages / tc.producers
			for i := range tc.producers {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for j := range messagesPerProducer {
						ch <- id*messagesPerProducer + j
					}
				}(i)
			}

			// Wait for all producers to finish, then close channel
			go func() {
				wg.Wait()
				close(ch)
			}()

			// Wait for all consumers to finish
			wg.Wait()
			duration := time.Since(start)

			messagesPerSecond := float64(consumeCount) / duration.Seconds()

			t.Logf("Channel performance results for %s:", tc.name)
			t.Logf("  Buffer size: %d", tc.bufferSize)
			t.Logf("  Messages: %d", consumeCount)
			t.Logf("  Producers: %d", tc.producers)
			t.Logf("  Consumers: %d", tc.consumers)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Messages/sec: %.2f", messagesPerSecond)

			// Performance expectations
			expectedMessages := int64(tc.messages)
			if consumeCount != expectedMessages {
				t.Errorf("Expected %d messages, got %d", expectedMessages, consumeCount)
			}

			minMessagesPerSec := float64(tc.messages) / 10.0
			if messagesPerSecond < minMessagesPerSec {
				t.Errorf("Channel performance too slow: %.2f msg/sec, expected at least %.2f",
					messagesPerSecond, minMessagesPerSec)
			}
		})
	}
}

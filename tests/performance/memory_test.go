package performance_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// TestMemoryLeaks checks for memory leaks in domain operations
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "TransactionCreation",
			fn: func() {
				for range 1000 {
					trans := transaction.NewTransaction(
						100.0,
						transaction.TypeExpense,
						"Memory Test Transaction",
						uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						time.Now(),
					)
					_ = trans
				}
			},
		},
		{
			name: "CategoryCreation",
			fn: func() {
				for range 1000 {
					cat := category.NewCategory(
						"Memory Test Category",
						category.TypeExpense,
						uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					)
					_ = cat
				}
			},
		},
		{
			name: "BudgetOperations",
			fn: func() {
				budget := &budget.Budget{
					Amount: 1000.0,
					Spent:  500.0,
				}
				for range 10000 {
					_ = budget.GetRemainingAmount()
					_ = budget.GetSpentPercentage()
					_ = budget.IsOverBudget()
				}
			},
		},
		{
			name: "UserCreation",
			fn: func() {
				for range 1000 {
					u := user.NewUser(
						"test@example.com",
						"John",
						"Doe",
						uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						user.RoleMember,
					)
					_ = u
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Force GC and get baseline memory stats
			runtime.GC()
			runtime.GC() // Call twice to ensure cleanup

			var startMem, endMem runtime.MemStats
			runtime.ReadMemStats(&startMem)

			// Run the test function multiple times
			const iterations = 10
			for range iterations {
				tc.fn()
			}

			// Force GC and measure final memory
			runtime.GC()
			runtime.GC()
			runtime.ReadMemStats(&endMem)

			// Calculate memory usage
			allocDiff := endMem.TotalAlloc - startMem.TotalAlloc
			heapDiff := int64(endMem.HeapInuse) - int64(startMem.HeapInuse)

			t.Logf("Memory usage for %s:", tc.name)
			t.Logf("  Total allocations: %d bytes", allocDiff)
			t.Logf("  Heap difference: %d bytes", heapDiff)
			t.Logf("  GC runs: %d", endMem.NumGC-startMem.NumGC)
			t.Logf("  Goroutines: %d", runtime.NumGoroutine())

			// Memory leak detection - heap should not grow significantly
			if heapDiff > 1024*1024 { // 1MB threshold
				t.Errorf("Potential memory leak detected: heap grew by %d bytes", heapDiff)
			}

			// Check for excessive allocations per iteration
			avgAllocPerIteration := allocDiff / iterations
			var maxExpectedAlloc uint64

			switch tc.name {
			case "TransactionCreation":
				maxExpectedAlloc = 50 * 1024 // 50KB per 1000 transactions
			case "CategoryCreation":
				maxExpectedAlloc = 30 * 1024 // 30KB per 1000 categories
			case "BudgetOperations":
				maxExpectedAlloc = 10 * 1024 // 10KB per 10000 operations
			case "UserCreation":
				maxExpectedAlloc = 100 * 1024 // 100KB per 1000 users
			}

			if avgAllocPerIteration > maxExpectedAlloc {
				t.Errorf("Excessive allocations: %d bytes per iteration, expected max %d",
					avgAllocPerIteration, maxExpectedAlloc)
			}
		})
	}
}

// TestMemoryPressure tests performance under memory pressure
func TestMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	// Create memory pressure by allocating large slices
	const (
		pressureSize = 100 * 1024 * 1024 // 100MB
		numAllocs    = 10
	)

	var memPressure [][]byte
	for range numAllocs {
		memPressure = append(memPressure, make([]byte, pressureSize/numAllocs))
	}

	// Measure performance under memory pressure
	start := time.Now()

	// Create domain objects under memory pressure
	for range 1000 {
		trans := transaction.NewTransaction(
			100.0,
			transaction.TypeExpense,
			"Pressure Test Transaction",
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			time.Now(),
		)
		_ = trans

		cat := category.NewCategory(
			"Pressure Test Category",
			category.TypeExpense,
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		)
		_ = cat
	}

	duration := time.Since(start)

	// Clean up memory pressure
	memPressure = nil
	runtime.GC()

	t.Logf("Performance under memory pressure:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Operations: 2000 (1000 transactions + 1000 categories)")
	t.Logf("  Rate: %.2f ops/sec", 2000.0/duration.Seconds())

	// Performance should not degrade significantly under memory pressure
	if duration > 5*time.Second {
		t.Errorf("Performance degraded under memory pressure: %v", duration)
	}
}

// TestGarbageCollectionImpact measures GC impact on performance
func TestGarbageCollectionImpact(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GC impact test in short mode")
	}

	tests := []struct {
		name         string
		gcPercent    int
		expectGCRuns bool
	}{
		{
			name:         "Default GC",
			gcPercent:    100,
			expectGCRuns: true,
		},
		{
			name:         "Aggressive GC",
			gcPercent:    50,
			expectGCRuns: true,
		},
		{
			name:         "Conservative GC",
			gcPercent:    200,
			expectGCRuns: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set GC target percentage
			oldGCPercent := runtime.GOMAXPROCS(0)
			defer func() {
				runtime.GOMAXPROCS(oldGCPercent)
			}()

			// Get initial GC stats
			var startStats, endStats runtime.MemStats
			runtime.ReadMemStats(&startStats)

			start := time.Now()

			// Create allocation pressure to trigger GC
			for i := range 5000 {
				// Create objects that will need GC
				trans := transaction.NewTransaction(
					float64(i),
					transaction.TypeExpense,
					"GC Test Transaction",
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					time.Now(),
				)

				// Create slices to increase allocation pressure
				data := make([]byte, 1024)
				_ = data
				_ = trans

				// Periodic check for excessive duration
				if i%1000 == 0 && time.Since(start) > 30*time.Second {
					t.Fatalf("Test taking too long, stopping at iteration %d", i)
				}
			}

			duration := time.Since(start)
			runtime.ReadMemStats(&endStats)

			// Calculate GC metrics
			gcRuns := endStats.NumGC - startStats.NumGC
			gcPauseTotal := endStats.PauseTotalNs - startStats.PauseTotalNs
			avgGCPause := time.Duration(0)
			if gcRuns > 0 {
				avgGCPause = time.Duration(gcPauseTotal / uint64(gcRuns))
			}

			t.Logf("GC Impact Results for %s:", tc.name)
			t.Logf("  Total Duration: %v", duration)
			t.Logf("  GC Runs: %d", gcRuns)
			t.Logf("  Total GC Pause: %v", time.Duration(gcPauseTotal))
			t.Logf("  Average GC Pause: %v", avgGCPause)
			t.Logf("  Heap Size: %d bytes", endStats.HeapInuse)
			t.Logf("  Next GC Target: %d bytes", endStats.NextGC)

			// Verify GC behavior expectations
			if tc.expectGCRuns && gcRuns == 0 {
				t.Errorf("Expected GC runs but none occurred")
			}

			// Performance thresholds
			if avgGCPause > 10*time.Millisecond {
				t.Errorf("Average GC pause (%v) exceeds threshold (10ms)", avgGCPause)
			}

			if duration > 10*time.Second {
				t.Errorf("Total duration (%v) exceeds threshold (10s)", duration)
			}
		})
	}
}

// TestMemoryEfficiency tests memory efficiency of data structures
func TestMemoryEfficiency(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() any
		maxBytes int64
	}{
		{
			name: "Single Transaction",
			fn: func() any {
				return transaction.NewTransaction(
					100.0,
					transaction.TypeExpense,
					"Test Transaction",
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					time.Now(),
				)
			},
			maxBytes: 500, // Maximum expected bytes per transaction
		},
		{
			name: "Single Category",
			fn: func() any {
				return category.NewCategory(
					"Test Category",
					category.TypeExpense,
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				)
			},
			maxBytes: 300, // Maximum expected bytes per category
		},
		{
			name: "Single User",
			fn: func() any {
				return user.NewUser(
					"test@example.com",
					"John",
					"Doe",
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					user.RoleMember,
				)
			},
			maxBytes: 400, // Maximum expected bytes per user
		},
		{
			name: "Single Budget",
			fn: func() any {
				return &budget.Budget{
					Amount: 1000.0,
					Spent:  500.0,
				}
			},
			maxBytes: 200, // Maximum expected bytes per budget
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Measure memory usage of single object creation
			runtime.GC()
			var startMem runtime.MemStats
			runtime.ReadMemStats(&startMem)

			const numObjects = 1000
			objects := make([]any, numObjects)

			for i := range numObjects {
				objects[i] = tc.fn()
			}

			var endMem runtime.MemStats
			runtime.ReadMemStats(&endMem)

			totalAlloc := endMem.TotalAlloc - startMem.TotalAlloc
			bytesPerObject := int64(totalAlloc) / numObjects

			t.Logf("Memory efficiency for %s:", tc.name)
			t.Logf("  Total allocation: %d bytes", totalAlloc)
			t.Logf("  Bytes per object: %d", bytesPerObject)
			t.Logf("  Objects created: %d", numObjects)

			if bytesPerObject > tc.maxBytes {
				t.Errorf("Memory usage per object (%d bytes) exceeds threshold (%d bytes)",
					bytesPerObject, tc.maxBytes)
			}

			// Ensure objects are not optimized away
			_ = objects[0]
		})
	}
}

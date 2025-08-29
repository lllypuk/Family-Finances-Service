package performance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/application"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// BenchmarkHTTPServerPerformance tests HTTP server response times
func BenchmarkHTTPServerPerformance(b *testing.B) {
	// Setup HTTP server for benchmarking
	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	// Warm up
	for range 10 {
		_, _ = http.Get(testServer.URL + "/health")
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("HealthEndpoint", func(b *testing.B) {
		for b.Loop() {
			resp, err := http.Get(testServer.URL + "/health")
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkDomainOperations tests domain logic performance
func BenchmarkDomainOperations(b *testing.B) {
	b.Run("BudgetCalculations", func(b *testing.B) {
		budget := &budget.Budget{
			Amount: 1000.0,
			Spent:  750.0,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_ = budget.GetRemainingAmount()
			_ = budget.GetSpentPercentage()
			_ = budget.IsOverBudget()
		}
	})

	b.Run("TransactionCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			trans := transaction.NewTransaction(
				100.0,
				transaction.TypeExpense,
				"Test Transaction",
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				time.Now(),
			)
			_ = trans
		}
	})

	b.Run("CategoryCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			cat := category.NewCategory(
				"Test Category",
				category.TypeExpense,
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			)
			_ = cat
		}
	})
}

// BenchmarkConfigurationLoading tests config loading performance
func BenchmarkConfigurationLoading(b *testing.B) {
	// Set test environment variables
	b.Setenv("ENVIRONMENT", "test")
	b.Setenv("SERVER_PORT", "8080")
	b.Setenv("MONGODB_URI", "mongodb://localhost:27017")

	b.ReportAllocs()

	for b.Loop() {
		config := &application.Config{
			Host: "localhost",
			Port: "8080",
		}
		_ = config
	}
}

// BenchmarkMemoryUsage tests memory allocation patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("BulkTransactionCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			transactions := make([]*transaction.Transaction, 1000)
			for i := range transactions {
				transactions[i] = transaction.NewTransaction(
					100.0,
					transaction.TypeExpense,
					"Transaction",
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					time.Now(),
				)
			}
			_ = transactions
		}
	})

	b.Run("BulkCategoryCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			categories := make([]*category.Category, 100)
			for i := range categories {
				categories[i] = category.NewCategory(
					"Category",
					category.TypeExpense,
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				)
			}
			_ = categories
		}
	})
}

// BenchmarkConcurrentOperations tests performance under concurrent load
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentBudgetCalculations", func(b *testing.B) {
		budget := &budget.Budget{
			Amount: 1000.0,
			Spent:  750.0,
		}

		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(runtime.NumCPU())

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = budget.GetRemainingAmount()
				_ = budget.GetSpentPercentage()
				_ = budget.IsOverBudget()
			}
		})
	})

	b.Run("ConcurrentTransactionCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		b.SetParallelism(runtime.NumCPU())

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				trans := transaction.NewTransaction(
					100.0,
					transaction.TypeExpense,
					"Test Transaction",
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					time.Now(),
				)
				_ = trans
			}
		})
	})
}

// BenchmarkUserOperations tests user-related operations
func BenchmarkUserOperations(b *testing.B) {
	b.Run("UserCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			u := user.NewUser(
				"test@example.com",
				"John",
				"Doe",
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				user.RoleMember,
			)
			_ = u
		}
	})

	b.Run("UserRoleCheck", func(b *testing.B) {
		u := &user.User{
			Role: user.RoleAdmin,
		}

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			_ = u.Role == user.RoleAdmin
			_ = u.Role == user.RoleMember
			_ = u.Role == user.RoleChild
		}
	})
}

// BenchmarkResponseTimes measures API response times
func BenchmarkResponseTimes(b *testing.B) {
	config := &application.Config{
		Host: "localhost",
		Port: "8080",
	}

	server := application.NewHTTPServer(nil, nil, config)
	testServer := httptest.NewServer(server.Echo())
	defer testServer.Close()

	endpoints := []struct {
		name string
		path string
	}{
		{"Health", "/health"},
		{"NotFound", "/nonexistent"},
	}

	for _, endpoint := range endpoints {
		b.Run(endpoint.name, func(b *testing.B) {
			var totalDuration time.Duration

			b.ResetTimer()

			for b.Loop() {
				start := time.Now()
				resp, err := http.Get(testServer.URL + endpoint.path)
				duration := time.Since(start)
				totalDuration += duration

				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}

			avgDuration := totalDuration / time.Duration(b.N)
			b.Logf("Average response time: %v", avgDuration)

			// Performance targets
			if endpoint.name == "Health" && avgDuration > 5*time.Millisecond {
				b.Logf("WARNING: Health endpoint response time (%v) exceeds target (5ms)", avgDuration)
			}
		})
	}
}

// BenchmarkContextOperations tests context handling performance
func BenchmarkContextOperations(b *testing.B) {
	b.Run("ContextCreation", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cancel()
			_ = ctx
		}
	})

	b.Run("ContextPropagation", func(b *testing.B) {
		ctx := context.Background()

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			childCtx := context.WithValue(ctx, "test", "value")
			_ = childCtx.Value("test")
		}
	})
}

// BenchmarkGarbageCollection measures GC impact
func BenchmarkGarbageCollection(b *testing.B) {
	b.Run("AllocationPressure", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			// Create allocation pressure
			data := make([][]byte, 1000)
			for i := range data {
				data[i] = make([]byte, 1024)
			}

			// Use data to prevent optimization
			_ = len(data)
		}

		// Force GC and measure
		runtime.GC()
	})
}

package observability_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	observability "family-budget-service/internal/observability"
)

// TestMetricsInitialization tests metrics initialization
func TestMetricsInitialization(t *testing.T) {
	t.Run("InitMetrics", func(t *testing.T) {
		// Should not panic
		observability.InitMetrics()

		// Verify metrics are accessible
		metrics := observability.GetDefaultMetrics()
		assert.NotNil(t, metrics)
		assert.NotNil(t, metrics.HTTPRequestsTotal)
		assert.NotNil(t, metrics.HTTPRequestDuration)
		assert.NotNil(t, metrics.DatabaseConnections)
		assert.NotNil(t, metrics.DatabaseOperationsTotal)
		assert.NotNil(t, metrics.FamiliesTotal)
		assert.NotNil(t, metrics.UsersTotal)
	})
}

// TestNewMetrics tests metrics creation
func TestNewMetrics(t *testing.T) {
	t.Run("CreateMetrics", func(t *testing.T) {
		metrics := observability.NewMetrics()
		require.NotNil(t, metrics)

		// Verify all metric groups are created
		assert.NotNil(t, metrics.HTTPRequestsTotal)
		assert.NotNil(t, metrics.HTTPRequestDuration)
		assert.NotNil(t, metrics.HTTPRequestsErrors)

		assert.NotNil(t, metrics.FamiliesTotal)
		assert.NotNil(t, metrics.UsersTotal)
		assert.NotNil(t, metrics.TransactionsTotal)
		assert.NotNil(t, metrics.BudgetsActive)
		assert.NotNil(t, metrics.TransactionAmount)

		assert.NotNil(t, metrics.DatabaseConnections)
		assert.NotNil(t, metrics.DatabaseOperationDuration)
		assert.NotNil(t, metrics.DatabaseOperationsTotal)

		assert.NotNil(t, metrics.ApplicationStartTime)
		assert.NotNil(t, metrics.ApplicationUptime)
	})
}

// TestObservabilityServiceMetrics tests metrics integration with observability service
func TestObservabilityServiceMetrics(t *testing.T) {
	t.Run("ServiceInitializesMetrics", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Metrics should be initialized during service creation
		metrics := observability.GetDefaultMetrics()
		assert.NotNil(t, metrics.HTTPRequestsTotal)
	})
}

// TestHTTPMetrics tests HTTP-related metrics
func TestHTTPMetrics(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	t.Run("RecordHTTPRequest", func(t *testing.T) {
		metrics.RecordHTTPRequest("GET", "/api/users", "200", 0.15)

		// Verify no panics occurred
		assert.NotNil(t, metrics.HTTPRequestsTotal)
		assert.NotNil(t, metrics.HTTPRequestDuration)
	})

	t.Run("RecordHTTPError", func(t *testing.T) {
		metrics.RecordHTTPError("POST", "/api/transactions", "validation")
		metrics.RecordHTTPError("GET", "/api/budgets", "server_error")

		// Verify no panics occurred
		assert.NotNil(t, metrics.HTTPRequestsErrors)
	})

	t.Run("GlobalHTTPFunctions", func(t *testing.T) {
		// Test global functions for backward compatibility
		observability.RecordHTTPRequest("GET", "/api/families", "200", 0.1)
		observability.RecordHTTPError("POST", "/api/users", "validation")

		// Should not panic - verify operation completed
		assert.NotNil(t, metrics)
	})
}

// TestDatabaseMetrics tests database-related metrics
func TestDatabaseMetrics(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	t.Run("RecordDatabaseOperation", func(t *testing.T) {
		metrics.RecordDatabaseOperation("find", "transactions", "success", 0.025)
		metrics.RecordDatabaseOperation("insert", "users", "success", 0.1)
		metrics.RecordDatabaseOperation("update", "budgets", "error", 0.5)

		// Verify no panics occurred
		assert.NotNil(t, metrics.DatabaseOperationsTotal)
		assert.NotNil(t, metrics.DatabaseOperationDuration)
	})

	t.Run("DatabaseConnections", func(t *testing.T) {
		metrics.DatabaseConnections.Set(5)
		metrics.DatabaseConnections.Set(10)
		metrics.DatabaseConnections.Set(3)

		// Verify no panics occurred
		assert.NotNil(t, metrics.DatabaseConnections)
	})
}

// TestBusinessMetrics tests business-specific metrics
func TestBusinessMetrics(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	t.Run("BusinessGauges", func(t *testing.T) {
		metrics.FamiliesTotal.Set(25)
		metrics.UsersTotal.Set(150)
		metrics.TransactionsTotal.Set(1500)
		metrics.BudgetsActive.Set(75)

		// Verify no panics occurred
		assert.NotNil(t, metrics.FamiliesTotal)
		assert.NotNil(t, metrics.UsersTotal)
		assert.NotNil(t, metrics.TransactionsTotal)
		assert.NotNil(t, metrics.BudgetsActive)
	})

	t.Run("TransactionAmounts", func(t *testing.T) {
		metrics.TransactionAmount.WithLabelValues("expense", "food").Observe(100.50)
		metrics.TransactionAmount.WithLabelValues("income", "salary").Observe(5000.0)
		metrics.TransactionAmount.WithLabelValues("expense", "transport").Observe(75.25)

		// Verify no panics occurred
		assert.NotNil(t, metrics.TransactionAmount)
	})
}

// TestApplicationMetrics tests application-specific metrics
func TestApplicationMetrics(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	t.Run("ApplicationUptime", func(t *testing.T) {
		// Test initial uptime
		metrics.UpdateUptime()

		// Wait a bit and update again
		time.Sleep(10 * time.Millisecond)
		metrics.UpdateUptime()

		// Verify no panics occurred
		assert.NotNil(t, metrics.ApplicationUptime)
		assert.NotNil(t, metrics.ApplicationStartTime)
	})
}

// TestMetricsRegistry tests the metrics registry functionality
func TestMetricsRegistry(t *testing.T) {
	t.Run("NewMetricsRegistry", func(t *testing.T) {
		registry := observability.NewMetricsRegistry()
		assert.NotNil(t, registry)

		// Get metrics instance
		metrics1 := registry.Get()
		assert.NotNil(t, metrics1)

		// Get again - should be same instance (singleton)
		metrics2 := registry.Get()
		assert.Same(t, metrics1, metrics2, "Should return same instance (singleton)")
	})
}

// TestMetricsGetters tests metric getter methods
func TestMetricsGetters(t *testing.T) {
	metrics := observability.NewMetrics()

	t.Run("HTTPMetricsGetters", func(t *testing.T) {
		assert.NotNil(t, metrics.GetHTTPRequestsTotal())
		assert.NotNil(t, metrics.GetHTTPRequestDuration())
		assert.NotNil(t, metrics.GetHTTPRequestsErrors())

		// Verify they return the correct instances
		assert.Same(t, metrics.HTTPRequestsTotal, metrics.GetHTTPRequestsTotal())
		assert.Same(t, metrics.HTTPRequestDuration, metrics.GetHTTPRequestDuration())
		assert.Same(t, metrics.HTTPRequestsErrors, metrics.GetHTTPRequestsErrors())
	})
}

// TestMetricsPerformance tests metrics recording performance
func TestMetricsPerformance(t *testing.T) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	t.Run("HighVolumeMetrics", func(t *testing.T) {
		start := time.Now()

		// Record many metrics quickly
		for range 1000 {
			metrics.RecordHTTPRequest("GET", "/api/test", "200", 0.001)
			metrics.RecordDatabaseOperation("find", "test", "success", 0.001)
			metrics.TransactionAmount.WithLabelValues("expense", "test").Observe(100.0)
		}

		duration := time.Since(start)

		// Should complete quickly
		assert.Less(t, duration, 100*time.Millisecond, "Metrics recording should be fast")
	})
}

// BenchmarkMetrics benchmarks metrics recording performance
func BenchmarkMetrics(b *testing.B) {
	metrics := observability.NewMetrics()
	metrics.Initialize()

	b.Run("HTTPMetrics", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			metrics.RecordHTTPRequest("GET", "/api/test", "200", 0.001)
		}
	})

	b.Run("DatabaseMetrics", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			metrics.RecordDatabaseOperation("find", "test", "success", 0.001)
		}
	})

	b.Run("BusinessMetrics", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			metrics.TransactionAmount.WithLabelValues("expense", "test").Observe(100.0)
		}
	})

	b.Run("GlobalFunctions", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			observability.RecordHTTPRequest("GET", "/api/test", "200", 0.001)
		}
	})
}

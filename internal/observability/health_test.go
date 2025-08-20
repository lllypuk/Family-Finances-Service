package observability_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	observability "family-budget-service/internal/observability"
)

// TestHealthService tests basic health service functionality
func TestHealthService(t *testing.T) {
	t.Run("BasicHealthCheck", func(t *testing.T) {
		// Create basic service
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Perform health check
		health := service.HealthService.CheckHealth(context.Background())
		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, "test-version", health.Version)
		assert.NotZero(t, health.Uptime)
		assert.NotZero(t, health.Timestamp)
	})

	t.Run("WithCustomHealthCheck", func(t *testing.T) {
		// Create service with custom check
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Add custom check that always passes
		service.AddCustomHealthCheck("test-check", func(ctx context.Context) error {
			return nil
		})

		health := service.HealthService.CheckHealth(context.Background())
		assert.Equal(t, "healthy", health.Status)
		assert.Contains(t, health.Checks, "test-check")
		assert.Equal(t, "healthy", health.Checks["test-check"].Status)
	})

	t.Run("FailingHealthCheck", func(t *testing.T) {
		// Create service with failing check
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Add custom check that always fails
		service.AddCustomHealthCheck("failing-check", func(ctx context.Context) error {
			return errors.New("test error")
		})

		health := service.HealthService.CheckHealth(context.Background())
		assert.Equal(t, "unhealthy", health.Status)
		assert.Contains(t, health.Checks, "failing-check")
		assert.Equal(t, "unhealthy", health.Checks["failing-check"].Status)
	})
}

// TestHealthHandlers tests HTTP health check handlers
func TestHealthHandlers(t *testing.T) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
	}, "test-version")
	require.NoError(t, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	// Add a test health check
	service.AddCustomHealthCheck("test", func(ctx context.Context) error {
		return nil
	})

	e := echo.New()

	t.Run("HealthHandler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := service.HealthService.HealthHandler()
		err := handler(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "healthy")
		assert.Contains(t, rec.Body.String(), "test-version")
	})

	t.Run("ReadinessHandler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := service.HealthService.ReadinessHandler()
		err := handler(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "ready")
	})

	t.Run("LivenessHandler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/live", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := service.HealthService.LivenessHandler()
		err := handler(c)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "alive")
	})
}

// TestCustomHealthChecker tests custom health checker functionality
func TestCustomHealthChecker(t *testing.T) {
	t.Run("SuccessfulCheck", func(t *testing.T) {
		checker := observability.NewCustomHealthChecker("test", func(ctx context.Context) error {
			return nil
		})

		assert.Equal(t, "test", checker.Name())

		result := checker.CheckHealth(context.Background())
		assert.Equal(t, observability.HealthStatusHealthy, result.Status)
		assert.NotZero(t, result.Duration)
		assert.NotZero(t, result.Timestamp)
	})

	t.Run("FailingCheck", func(t *testing.T) {
		testError := errors.New("test error")
		checker := observability.NewCustomHealthChecker("failing", func(ctx context.Context) error {
			return testError
		})

		result := checker.CheckHealth(context.Background())
		assert.Equal(t, "unhealthy", result.Status)
		assert.Equal(t, testError.Error(), result.Message)
		assert.NotZero(t, result.Duration)
	})

	t.Run("TimeoutCheck", func(t *testing.T) {
		checker := observability.NewCustomHealthChecker("timeout", func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		result := checker.CheckHealth(ctx)
		assert.Equal(t, "unhealthy", result.Status)
		assert.Contains(t, result.Message, "deadline exceeded")
	})
}

// TestHealthServiceConcurrency tests concurrent health checks
func TestHealthServiceConcurrency(t *testing.T) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
	}, "test-version")
	require.NoError(t, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	// Add multiple health checks
	for i := range 5 {
		name := fmt.Sprintf("check_%d", i)
		delay := time.Duration(i+1) * 10 * time.Millisecond
		service.AddCustomHealthCheck(name, func(ctx context.Context) error {
			time.Sleep(delay)
			return nil
		})
	}

	// Run health checks concurrently
	const numGoroutines = 10
	results := make(chan observability.HealthStatus, numGoroutines)

	for range numGoroutines {
		go func() {
			health := service.HealthService.CheckHealth(context.Background())
			results <- health
		}()
	}

	// Collect results
	for range numGoroutines {
		health := <-results
		assert.Equal(t, "healthy", health.Status)
		assert.Len(t, health.Checks, 5)
	}
}

// TestHealthServicePerformance tests health check performance
func TestHealthServicePerformance(t *testing.T) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
	}, "test-version")
	require.NoError(t, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	// Add a simple health check
	service.AddCustomHealthCheck("fast", func(ctx context.Context) error {
		return nil
	})

	t.Run("HealthCheckPerformance", func(t *testing.T) {
		start := time.Now()
		health := service.HealthService.CheckHealth(context.Background())
		duration := time.Since(start)

		assert.Equal(t, "healthy", health.Status)
		assert.Less(t, duration, 100*time.Millisecond, "Health check should be fast")
	})
}

// BenchmarkHealthService benchmarks health service performance
func BenchmarkHealthService(b *testing.B) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
	}, "test-version")
	require.NoError(b, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	// Add test health check
	service.AddCustomHealthCheck("benchmark", func(ctx context.Context) error {
		return nil
	})

	b.Run("CheckHealth", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			service.HealthService.CheckHealth(context.Background())
		}
	})

	b.Run("HealthHandler", func(b *testing.B) {
		e := echo.New()
		handler := service.HealthService.HealthHandler()

		b.ResetTimer()
		for b.Loop() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handler(c)
		}
	})
}

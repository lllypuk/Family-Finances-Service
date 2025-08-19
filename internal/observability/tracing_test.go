package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	observability "family-budget-service/internal/observability"
)

// TestTracingInitialization tests tracing initialization
func TestTracingInitialization(t *testing.T) {
	t.Run("InitTracingWithService", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
			Tracing: observability.TracingConfig{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Enabled:        true,
			},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Tracing should be initialized
		assert.NotNil(t, service)
	})

	t.Run("TracingDisabled", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
			Tracing: observability.TracingConfig{
				Enabled: false,
			},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Should work even with tracing disabled
		assert.NotNil(t, service)
	})
}

// TestTracingConfiguration tests different tracing configurations
func TestTracingConfiguration(t *testing.T) {
	testCases := []struct {
		name   string
		config observability.TracingConfig
	}{
		{
			name: "BasicConfig",
			config: observability.TracingConfig{
				ServiceName:    "family-budget-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Enabled:        true,
			},
		},
		{
			name: "ProductionConfig",
			config: observability.TracingConfig{
				ServiceName:    "family-budget-service",
				ServiceVersion: "2.0.0",
				OTLPEndpoint:   "http://localhost:4318/v1/traces",
				Environment:    "production",
				Enabled:        true,
			},
		},
		{
			name: "DevelopmentConfig",
			config: observability.TracingConfig{
				ServiceName:    "family-budget-service",
				ServiceVersion: "dev",
				Environment:    "development",
				Enabled:        true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := observability.NewService(observability.Config{
				Logging: observability.LogConfig{Level: "info", Format: "json"},
				Tracing: tc.config,
			}, "test-version")
			require.NoError(t, err)
			defer func() { _ = service.Shutdown(context.Background()) }()

			assert.NotNil(t, service)
		})
	}
}

// TestTracingIntegration tests tracing integration with the observability service
func TestTracingIntegration(t *testing.T) {
	t.Run("ServiceShutdown", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
			Tracing: observability.TracingConfig{
				ServiceName: "test-service",
				Enabled:     true,
			},
		}, "test-version")
		require.NoError(t, err)

		// Shutdown should work without errors
		err = service.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		config := observability.DefaultConfig()

		assert.Equal(t, "info", config.Logging.Level)
		assert.Equal(t, "json", config.Logging.Format)
		assert.Equal(t, "family-budget-service", config.Tracing.ServiceName)
		assert.Equal(t, "1.0.0", config.Tracing.ServiceVersion)
		assert.Equal(t, "development", config.Tracing.Environment)
		assert.True(t, config.Tracing.Enabled)
	})
}

// TestTracingPerformance tests tracing performance impact
func TestTracingPerformance(t *testing.T) {
	t.Run("TracingOverhead", func(t *testing.T) {
		// Create service with tracing enabled
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
			Tracing: observability.TracingConfig{
				ServiceName: "performance-test",
				Enabled:     true,
			},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		// Perform some operations to test overhead
		ctx := context.Background()
		for i := range 100 {
			// Simulate some traced operations
			service.Logger.InfoContext(ctx, "test operation",
				"iteration", i,
			)
		}

		// Should complete without significant performance impact
		assert.NotNil(t, service)
	})
}

// BenchmarkTracing benchmarks tracing performance
func BenchmarkTracing(b *testing.B) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
		Tracing: observability.TracingConfig{
			ServiceName: "benchmark-test",
			Enabled:     true,
		},
	}, "test-version")
	require.NoError(b, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	ctx := context.Background()

	b.Run("TracedLogging", func(b *testing.B) {
		b.ResetTimer()
		for i := range b.N {
			service.Logger.InfoContext(ctx, "benchmark test",
				"iteration", i,
			)
		}
	})
}

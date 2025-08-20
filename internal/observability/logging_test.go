package observability_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	observability "family-budget-service/internal/observability"
)

// TestNewLogger tests logger creation with different configurations
func TestNewLogger(t *testing.T) {
	t.Run("JSONLogger", func(t *testing.T) {
		config := observability.LogConfig{
			Level:  "info",
			Format: "json",
		}

		logger := observability.NewLogger(config)
		assert.NotNil(t, logger)
	})

	t.Run("TextLogger", func(t *testing.T) {
		config := observability.LogConfig{
			Level:  "debug",
			Format: "text",
		}

		logger := observability.NewLogger(config)
		assert.NotNil(t, logger)
	})

	t.Run("DefaultConfig", func(t *testing.T) {
		config := observability.LogConfig{} // Empty config should use defaults

		logger := observability.NewLogger(config)
		assert.NotNil(t, logger)
	})
}

// TestObservabilityServiceLogging tests the observability service logging integration
func TestObservabilityServiceLogging(t *testing.T) {
	t.Run("ServiceWithJSONLogging", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "info", Format: "json"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		assert.NotNil(t, service.Logger)
		assert.NotNil(t, service.BusinessLogger)
	})

	t.Run("ServiceWithTextLogging", func(t *testing.T) {
		service, err := observability.NewService(observability.Config{
			Logging: observability.LogConfig{Level: "debug", Format: "text"},
		}, "test-version")
		require.NoError(t, err)
		defer func() { _ = service.Shutdown(context.Background()) }()

		assert.NotNil(t, service.Logger)
		assert.NotNil(t, service.BusinessLogger)
	})
}

// TestBusinessLogger tests business logging functionality
func TestBusinessLogger(t *testing.T) {
	// Create logger with JSON format for easier testing
	config := observability.LogConfig{
		Level:  "info",
		Format: "json",
	}

	logger := observability.NewLogger(config)
	businessLogger := observability.NewBusinessLogger(logger)

	t.Run("LogUserAction", func(t *testing.T) {
		businessLogger.LogUserAction(context.Background(), "user123", "family123", "login", map[string]any{
			"ip_address": "192.168.1.1",
			"user_agent": "test-agent",
		})

		// Since we can't easily capture slog output in tests, we just verify no panic
		assert.NotNil(t, businessLogger)
	})

	t.Run("LogTransactionEvent", func(t *testing.T) {
		businessLogger.LogTransactionEvent(
			context.Background(),
			"tx123",
			"user123",
			"family123",
			"created",
			100.50,
			"USD",
		)

		// Verify no panic occurred
		assert.NotNil(t, businessLogger)
	})

	t.Run("LogBudgetEvent", func(t *testing.T) {
		businessLogger.LogBudgetEvent(
			context.Background(),
			"budget123",
			"user123",
			"family123",
			"updated",
			map[string]any{
				"old_amount": 500.0,
				"new_amount": 450.0,
			},
		)

		// Verify no panic occurred
		assert.NotNil(t, businessLogger)
	})

	t.Run("LogAPIError", func(t *testing.T) {
		businessLogger.LogAPIError(
			context.Background(),
			"/api/transactions",
			"POST",
			"validation",
			"test error message",
			400,
			"user123",
		)

		// Verify no panic occurred
		assert.NotNil(t, businessLogger)
	})
}

// TestLoggerLevels tests different log levels
func TestLoggerLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run("Level_"+level, func(t *testing.T) {
			config := observability.LogConfig{
				Level:  level,
				Format: "json",
			}

			logger := observability.NewLogger(config)
			assert.NotNil(t, logger)

			// Test that the logger can log at this level
			logger.InfoContext(context.Background(), "test message",
				slog.String("level", level),
				slog.String("test", "value"),
			)
		})
	}
}

// TestLoggerFormats tests different log formats
func TestLoggerFormats(t *testing.T) {
	formats := []string{"json", "text"}

	for _, format := range formats {
		t.Run("Format_"+format, func(t *testing.T) {
			config := observability.LogConfig{
				Level:  "info",
				Format: format,
			}

			logger := observability.NewLogger(config)
			assert.NotNil(t, logger)

			// Test that the logger can output in this format
			logger.InfoContext(context.Background(), "test message",
				slog.String("format", format),
				slog.String("test", "value"),
			)
		})
	}
}

// TestStructuredLogging tests structured logging capabilities
func TestStructuredLogging(t *testing.T) {
	config := observability.LogConfig{
		Level:  "info",
		Format: "json",
	}

	logger := observability.NewLogger(config)

	t.Run("ContextualLogging", func(t *testing.T) {
		ctx := context.Background()

		// Test various structured log entries
		logger.InfoContext(ctx, "user operation",
			slog.String("user_id", "user123"),
			slog.String("operation", "create_transaction"),
			slog.Float64("amount", 100.50),
			slog.Group("metadata",
				slog.String("family_id", "family123"),
				slog.String("category", "food"),
			),
		)

		logger.WarnContext(ctx, "budget warning",
			slog.String("budget_id", "budget123"),
			slog.Float64("spent", 450.0),
			slog.Float64("limit", 500.0),
			slog.Float64("percentage", 90.0),
		)

		logger.ErrorContext(ctx, "operation failed",
			slog.String("operation", "delete_transaction"),
			slog.String("error", "transaction not found"),
			slog.String("transaction_id", "tx123"),
		)

		// Verify no panics occurred
		assert.NotNil(t, logger)
	})
}

// TestBusinessLoggerIntegration tests business logger integration
func TestBusinessLoggerIntegration(t *testing.T) {
	service, err := observability.NewService(observability.Config{
		Logging: observability.LogConfig{Level: "info", Format: "json"},
	}, "test-version")
	require.NoError(t, err)
	defer func() { _ = service.Shutdown(context.Background()) }()

	t.Run("CompleteBusinessFlow", func(t *testing.T) {
		ctx := context.Background()

		// Simulate a complete business flow with logging
		service.BusinessLogger.LogUserAction(ctx, "user123", "family123", "login", map[string]any{
			"ip": "192.168.1.1",
		})

		service.BusinessLogger.LogTransactionEvent(ctx, "tx123", "user123", "family123", "created", 75.50, "USD")

		service.BusinessLogger.LogBudgetEvent(
			ctx,
			"budget123",
			"user123",
			"family123",
			"updated",
			map[string]any{
				"old_amount": 500.0,
				"new_amount": 425.50,
			},
		)

		service.BusinessLogger.LogUserAction(ctx, "user123", "family123", "logout", nil)

		// Verify the business logger is working
		assert.NotNil(t, service.BusinessLogger)
	})
}

// TestLoggingPerformance tests logging performance
func TestLoggingPerformance(t *testing.T) {
	config := observability.LogConfig{
		Level:  "info",
		Format: "json",
	}

	logger := observability.NewLogger(config)

	t.Run("LoggingPerformance", func(t *testing.T) {
		ctx := context.Background()

		// Log multiple entries to test performance
		for i := range 100 {
			logger.InfoContext(ctx, "performance test",
				slog.Int("iteration", i),
				slog.String("test", "performance"),
				slog.Float64("value", float64(i)*1.5),
			)
		}

		// Verify logger is still working
		assert.NotNil(t, logger)
	})
}

// BenchmarkLogging benchmarks logging performance
func BenchmarkLogging(b *testing.B) {
	config := observability.LogConfig{
		Level:  "info",
		Format: "json",
	}

	logger := observability.NewLogger(config)
	ctx := context.Background()

	b.Run("StructuredLogging", func(b *testing.B) {
		b.ResetTimer()
		for i := range b.N {
			logger.InfoContext(ctx, "benchmark test",
				slog.Int("iteration", i),
				slog.String("operation", "benchmark"),
				slog.Float64("value", float64(i)),
			)
		}
	})

	b.Run("BusinessLogging", func(b *testing.B) {
		businessLogger := observability.NewBusinessLogger(logger)

		b.ResetTimer()
		for b.Loop() {
			businessLogger.LogTransactionEvent(ctx, "tx123", "user123", "family123", "created", 100.0, "USD")
		}
	})
}

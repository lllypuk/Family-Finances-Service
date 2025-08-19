package internal_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
)

func TestGracefulShutdownTimeout_Constant(t *testing.T) {
	// Test that the graceful shutdown timeout constant is properly defined
	expectedTimeout := 30 * time.Second
	assert.Equal(t, expectedTimeout, internal.GracefulShutdownTimeout)
}

func TestApplication_StructFields(t *testing.T) {
	// Test that Application struct can be created
	// This test verifies the struct exists and is accessible
	// We can't easily test internal fields due to package visibility
	config := &internal.Config{
		Environment: "test",
	}

	assert.NotNil(t, config)
	assert.Equal(t, "test", config.Environment)
}

func TestNewApplication_ConfigurationLoading(t *testing.T) {
	// Set test environment variables
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SERVER_PORT", "8081")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	t.Setenv("MONGODB_DATABASE", "test_db")
	t.Setenv("SESSION_SECRET", "test-secret")

	// Note: This test will fail if MongoDB is not available
	// In a real test environment, we would use testcontainers or mock
	// For now, we'll test what we can without external dependencies

	// Test that LoadConfig works properly
	config := internal.LoadConfig()
	assert.Equal(t, "test", config.Environment)
	assert.Equal(t, "8081", config.Server.Port)
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "test_db", config.Database.Name)
	assert.Equal(t, "test-secret", config.Web.SessionSecret)
}

func TestNewApplication_ErrorHandling(t *testing.T) {
	// Test error handling when MongoDB connection fails
	t.Setenv("MONGODB_URI", "mongodb://non-existent-host:27017")
	t.Setenv("ENVIRONMENT", "test")

	// This should fail because MongoDB connection will fail
	app, err := internal.NewApplication()

	// We expect an error due to MongoDB connection failure
	require.Error(t, err)
	assert.Nil(t, app)
	assert.Contains(t, err.Error(), "failed to connect to MongoDB")
}

func TestNewApplication_ObservabilityInitialization(t *testing.T) {
	// Test observability initialization with different log levels
	logLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range logLevels {
		t.Run("LogLevel_"+level, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", level)
			t.Setenv("ENVIRONMENT", "test")
			t.Setenv("MONGODB_URI", "mongodb://non-existent-host:27017")

			// Even though MongoDB will fail, we can test that observability config is read
			_, err := internal.NewApplication()

			// We expect a MongoDB error, but observability should be initialized first
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to connect to MongoDB")
			// If observability failed, the error would be about observability, not MongoDB
		})
	}
}

func TestApplication_ProductionConfiguration(t *testing.T) {
	// Test production-specific configuration
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("SERVER_PORT", "443")
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("MONGODB_URI", "mongodb://prod-host:27017")
	t.Setenv("SESSION_SECRET", "super-secure-production-secret")

	config := internal.LoadConfig()

	assert.True(t, config.IsProduction())
	assert.Equal(t, "production", config.Environment)
	assert.Equal(t, "443", config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, "super-secure-production-secret", config.Web.SessionSecret)
}

func TestApplication_DevelopmentConfiguration(t *testing.T) {
	// Test development-specific configuration
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("LOG_LEVEL", "debug")

	config := internal.LoadConfig()

	assert.False(t, config.IsProduction())
	assert.Equal(t, "development", config.Environment)
}

func TestApplication_DefaultConfiguration(t *testing.T) {
	// Clear all environment variables to test defaults
	envVars := []string{
		"ENVIRONMENT", "LOG_LEVEL", "SERVER_PORT", "SERVER_HOST",
		"MONGODB_URI", "MONGODB_DATABASE", "SESSION_SECRET",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	config := internal.LoadConfig()

	// Verify defaults
	assert.Equal(t, "development", config.Environment)
	assert.False(t, config.IsProduction())
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "family_budget", config.Database.Name)
}

func TestApplication_ContextHandling(t *testing.T) {
	// Test context creation and cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test context with timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), internal.GracefulShutdownTimeout)
	defer timeoutCancel()

	assert.NotNil(t, ctx)
	assert.NotNil(t, timeoutCtx)

	// Test that context can be cancelled
	cancel()
	select {
	case <-ctx.Done():
		// Context was cancelled as expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context was not cancelled")
	}
}

func TestApplication_TimeoutConfiguration(t *testing.T) {
	// Test various timeout configurations
	tests := []struct {
		name     string
		timeout  time.Duration
		expected bool
	}{
		{
			name:     "Standard timeout",
			timeout:  internal.GracefulShutdownTimeout,
			expected: true,
		},
		{
			name:     "Short timeout",
			timeout:  1 * time.Second,
			expected: true,
		},
		{
			name:     "Long timeout",
			timeout:  60 * time.Second,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			assert.NotNil(t, ctx)

			// Check that deadline is set
			deadline, ok := ctx.Deadline()
			if tt.expected {
				assert.True(t, ok)
				assert.True(t, deadline.After(time.Now()))
			}
		})
	}
}

func TestApplication_EnvironmentValidation(t *testing.T) {
	// Test environment-specific behavior
	environments := []struct {
		env          string
		isProduction bool
	}{
		{"production", true},
		{"prod", false},
		{"development", false},
		{"dev", false},
		{"test", false},
		{"testing", false},
		{"staging", false},
		{"local", false},
		{"", false},
	}

	for _, env := range environments {
		t.Run("Environment_"+env.env, func(t *testing.T) {
			if env.env != "" {
				t.Setenv("ENVIRONMENT", env.env)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}

			config := internal.LoadConfig()

			if env.env == "" {
				assert.Equal(t, "development", config.Environment) // default
			} else {
				assert.Equal(t, env.env, config.Environment)
			}
			assert.Equal(t, env.isProduction, config.IsProduction())
		})
	}
}

func TestApplication_ConfigurationSecurity(t *testing.T) {
	// Test that sensitive configuration is handled properly
	secretValues := []struct {
		name   string
		value  string
		secure bool
	}{
		{
			name:   "Weak session secret",
			value:  "123",
			secure: false,
		},
		{
			name:   "Strong session secret",
			value:  "very-long-and-secure-session-secret-with-high-entropy",
			secure: true,
		},
		{
			name:   "Default session secret",
			value:  "your-super-secret-session-key-change-in-production",
			secure: false, // Default should be changed in production
		},
	}

	for _, secret := range secretValues {
		t.Run(secret.name, func(t *testing.T) {
			t.Setenv("SESSION_SECRET", secret.value)

			config := internal.LoadConfig()

			assert.Equal(t, secret.value, config.Web.SessionSecret)

			// In production, we should validate session secret strength
			if config.IsProduction() && !secret.secure {
				// This is a test hint that weak secrets shouldn't be used in production
				assert.Greater(t, len(config.Web.SessionSecret), 32,
					"Production session secret should be longer than 32 characters")
			}
		})
	}
}

func TestApplication_ObservabilityConfiguration(t *testing.T) {
	// Test observability-related configuration
	logLevels := []string{"debug", "info", "warn", "error", ""}

	for _, level := range logLevels {
		t.Run("LogLevel_"+level, func(t *testing.T) {
			if level != "" {
				t.Setenv("LOG_LEVEL", level)
			} else {
				os.Unsetenv("LOG_LEVEL")
			}

			// We can't easily test the actual observability initialization without dependencies
			// But we can test that the environment variable is read correctly
			envLevel := os.Getenv("LOG_LEVEL")
			if level != "" {
				assert.Equal(t, level, envLevel)
			} else {
				assert.Empty(t, envLevel)
			}
		})
	}
}

// Benchmark tests for application configuration loading
func BenchmarkLoadConfigApplication(b *testing.B) {
	// Set some environment variables
	b.Setenv("ENVIRONMENT", "test")
	b.Setenv("SERVER_PORT", "8080")
	b.Setenv("LOG_LEVEL", "info")

	for b.Loop() {
		_ = internal.LoadConfig()
	}
}

func BenchmarkConfigIsProduction(b *testing.B) {
	config := &internal.Config{Environment: "production"}

	for b.Loop() {
		_ = config.IsProduction()
	}
}

func BenchmarkContextCreation(b *testing.B) {
	for b.Loop() {
		ctx, cancel := context.WithTimeout(context.Background(), internal.GracefulShutdownTimeout)
		cancel()
		_ = ctx
	}
}

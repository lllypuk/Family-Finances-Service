package internal_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Clear environment variables to test defaults
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")

	// Execute
	config := internal.LoadConfig()

	// Assert default values
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "family_budget", config.Database.Name)
}

func TestLoadConfig_EnvironmentValues(t *testing.T) {
	// Set environment variables
	t.Setenv("SERVER_PORT", "3000")
	t.Setenv("SERVER_HOST", "0.0.0.0")
	t.Setenv("MONGODB_URI", "mongodb://remote:27017")
	t.Setenv("MONGODB_DATABASE", "test_db")

	// Execute
	config := internal.LoadConfig()

	// Assert environment values are used
	assert.Equal(t, "3000", config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, "mongodb://remote:27017", config.Database.URI)
	assert.Equal(t, "test_db", config.Database.Name)
}

func TestLoadConfig_MixedValues(t *testing.T) {
	// Set only some environment variables
	t.Setenv("SERVER_PORT", "9000")
	t.Setenv("MONGODB_URI", "mongodb://custom-host:27017")

	// Execute
	config := internal.LoadConfig()

	// Assert mix of environment and default values
	assert.Equal(t, "9000", config.Server.Port)                         // From environment
	assert.Equal(t, "localhost", config.Server.Host)                    // Default
	assert.Equal(t, "mongodb://custom-host:27017", config.Database.URI) // From environment
	assert.Equal(t, "family_budget", config.Database.Name)              // Default
}

func TestConfig_StructFields(t *testing.T) {
	// Test that Config struct has all expected fields
	config := &internal.Config{
		Server: internal.ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		Database: internal.DatabaseConfig{
			URI:  "mongodb://localhost:27017",
			Name: "test_db",
		},
	}

	// Assert all fields are accessible
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "test_db", config.Database.Name)
}

func TestServerConfig_StructFields(t *testing.T) {
	// Test that ServerConfig struct has all required fields
	serverConfig := internal.ServerConfig{
		Port: "3000",
		Host: "0.0.0.0",
	}

	// Assert all fields are accessible
	assert.Equal(t, "3000", serverConfig.Port)
	assert.Equal(t, "0.0.0.0", serverConfig.Host)
}

func TestDatabaseConfig_StructFields(t *testing.T) {
	// Test that DatabaseConfig struct has all required fields
	dbConfig := internal.DatabaseConfig{
		URI:  "mongodb://test:27017",
		Name: "test_database",
	}

	// Assert all fields are accessible
	assert.Equal(t, "mongodb://test:27017", dbConfig.URI)
	assert.Equal(t, "test_database", dbConfig.Name)
}

func TestLoadConfig_EmptyEnvironmentVariables(t *testing.T) {
	// Set environment variables to empty strings
	t.Setenv("SERVER_PORT", "")
	t.Setenv("SERVER_HOST", "")
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_DATABASE", "")

	// Execute
	config := internal.LoadConfig()

	// Assert that empty environment variables fall back to defaults
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "family_budget", config.Database.Name)
}

func TestWebConfig_StructFields(t *testing.T) {
	// Test that WebConfig struct has all required fields
	webConfig := internal.WebConfig{
		SessionSecret: "test-secret-key",
	}

	// Assert all fields are accessible
	assert.Equal(t, "test-secret-key", webConfig.SessionSecret)
}

func TestLoadConfig_WebConfiguration(t *testing.T) {
	// Test web configuration loading
	t.Setenv("SESSION_SECRET", "custom-session-secret")

	config := internal.LoadConfig()

	assert.Equal(t, "custom-session-secret", config.Web.SessionSecret)
}

func TestLoadConfig_WebConfigurationDefaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("SESSION_SECRET")

	config := internal.LoadConfig()

	assert.Equal(t, "your-super-secret-session-key-change-in-production", config.Web.SessionSecret)
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{
			name:        "Production environment",
			environment: "production",
			expected:    true,
		},
		{
			name:        "Development environment",
			environment: "development",
			expected:    false,
		},
		{
			name:        "Test environment",
			environment: "test",
			expected:    false,
		},
		{
			name:        "Staging environment",
			environment: "staging",
			expected:    false,
		},
		{
			name:        "Empty environment",
			environment: "",
			expected:    false,
		},
		{
			name:        "Unknown environment",
			environment: "unknown",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &internal.Config{
				Environment: tt.environment,
			}

			result := config.IsProduction()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfig_EnvironmentSetting(t *testing.T) {
	tests := []struct {
		name           string
		environmentVar string
		expectedEnv    string
		expectedIsProd bool
	}{
		{
			name:           "Production environment",
			environmentVar: "production",
			expectedEnv:    "production",
			expectedIsProd: true,
		},
		{
			name:           "Development environment",
			environmentVar: "development",
			expectedEnv:    "development",
			expectedIsProd: false,
		},
		{
			name:           "Default environment (no env var)",
			environmentVar: "",
			expectedEnv:    "development",
			expectedIsProd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.environmentVar != "" {
				t.Setenv("ENVIRONMENT", tt.environmentVar)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}

			config := internal.LoadConfig()

			assert.Equal(t, tt.expectedEnv, config.Environment)
			assert.Equal(t, tt.expectedIsProd, config.IsProduction())
		})
	}
}

func TestLoadConfig_CompleteConfiguration(t *testing.T) {
	// Set all environment variables
	t.Setenv("SERVER_PORT", "8443")
	t.Setenv("SERVER_HOST", "app.example.com")
	t.Setenv("MONGODB_URI", "mongodb://user:pass@prod-mongo:27017/family_budget")
	t.Setenv("MONGODB_DATABASE", "family_budget_prod")
	t.Setenv("SESSION_SECRET", "super-secure-production-secret")
	t.Setenv("ENVIRONMENT", "production")

	config := internal.LoadConfig()

	// Verify server config
	assert.Equal(t, "8443", config.Server.Port)
	assert.Equal(t, "app.example.com", config.Server.Host)

	// Verify database config
	assert.Equal(t, "mongodb://user:pass@prod-mongo:27017/family_budget", config.Database.URI)
	assert.Equal(t, "family_budget_prod", config.Database.Name)

	// Verify web config
	assert.Equal(t, "super-secure-production-secret", config.Web.SessionSecret)

	// Verify environment
	assert.Equal(t, "production", config.Environment)
	assert.True(t, config.IsProduction())
}

func TestLoadConfig_SpecialCharacters(t *testing.T) {
	// Test handling of special characters in configuration values
	t.Setenv("MONGODB_URI", "mongodb://user:p@ssw0rd!@host:27017/db?authSource=admin")
	t.Setenv("SESSION_SECRET", "secret-with-special-chars!@#$%^&*()")

	config := internal.LoadConfig()

	assert.Equal(t, "mongodb://user:p@ssw0rd!@host:27017/db?authSource=admin", config.Database.URI)
	assert.Equal(t, "secret-with-special-chars!@#$%^&*()", config.Web.SessionSecret)
}

func TestLoadConfig_NumericPorts(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{"Standard HTTP port", "80"},
		{"Standard HTTPS port", "443"},
		{"Development port", "3000"},
		{"Custom high port", "8080"},
		{"High port number", "65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SERVER_PORT", tt.port)

			config := internal.LoadConfig()

			assert.Equal(t, tt.port, config.Server.Port)
		})
	}
}

func TestLoadConfig_DatabaseURIFormats(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{
			name: "Simple local URI",
			uri:  "mongodb://localhost:27017",
		},
		{
			name: "URI with authentication",
			uri:  "mongodb://admin:password@localhost:27017",
		},
		{
			name: "URI with replica set",
			uri:  "mongodb://host1:27017,host2:27017,host3:27017/mydb?replicaSet=myReplicaSet",
		},
		{
			name: "URI with SSL",
			uri:  "mongodb://host:27017/mydb?ssl=true",
		},
		{
			name: "MongoDB Atlas URI",
			uri:  "mongodb+srv://username:password@cluster.mongodb.net/myFirstDatabase?retryWrites=true&w=majority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MONGODB_URI", tt.uri)

			config := internal.LoadConfig()

			assert.Equal(t, tt.uri, config.Database.URI)
		})
	}
}

func TestConfig_CompleteStructure(t *testing.T) {
	// Test that the complete config structure can be created and accessed
	config := &internal.Config{
		Server: internal.ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		Database: internal.DatabaseConfig{
			URI:  "mongodb://localhost:27017",
			Name: "test_db",
		},
		Web: internal.WebConfig{
			SessionSecret: "test-secret",
		},
		Environment: "test",
	}

	// Verify all nested structures are accessible
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "test_db", config.Database.Name)
	assert.Equal(t, "test-secret", config.Web.SessionSecret)
	assert.Equal(t, "test", config.Environment)
	assert.False(t, config.IsProduction())
}

// Benchmark tests for performance
func BenchmarkLoadConfig(b *testing.B) {
	// Set some environment variables
	b.Setenv("SERVER_PORT", "8080")
	b.Setenv("MONGODB_URI", "mongodb://localhost:27017")

	for b.Loop() {
		_ = internal.LoadConfig()
	}
}

func BenchmarkConfig_IsProduction(b *testing.B) {
	config := &internal.Config{Environment: "production"}

	for b.Loop() {
		_ = config.IsProduction()
	}
}

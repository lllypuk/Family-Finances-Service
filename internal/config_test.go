package internal

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Clean environment
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")

	// Execute
	config := LoadConfig()

	// Assert default values
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "mongodb://localhost:27017", config.Database.URI)
	assert.Equal(t, "family_budget", config.Database.Name)
}

func TestLoadConfig_EnvironmentValues(t *testing.T) {
	// Setup environment variables
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("SERVER_HOST", "0.0.0.0")
	os.Setenv("MONGODB_URI", "mongodb://test-host:27017")
	os.Setenv("MONGODB_DATABASE", "test_db")

	// Clean up after test
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("MONGODB_DATABASE")
	}()

	// Execute
	config := LoadConfig()

	// Assert environment values are used
	assert.Equal(t, "3000", config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, "mongodb://test-host:27017", config.Database.URI)
	assert.Equal(t, "test_db", config.Database.Name)
}

func TestLoadConfig_PartialEnvironmentValues(t *testing.T) {
	// Setup only some environment variables
	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("MONGODB_URI", "mongodb://custom-host:27017")

	// Clean up after test
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("SERVER_HOST")
		os.Unsetenv("MONGODB_DATABASE")
	}()

	// Execute
	config := LoadConfig()

	// Assert mix of environment and default values
	assert.Equal(t, "9000", config.Server.Port)                         // From environment
	assert.Equal(t, "localhost", config.Server.Host)                    // Default value
	assert.Equal(t, "mongodb://custom-host:27017", config.Database.URI) // From environment
	assert.Equal(t, "family_budget", config.Database.Name)              // Default value
}

func TestGetEnv_WithValue(t *testing.T) {
	// Setup
	key := "TEST_ENV_VAR"
	value := "test_value"
	defaultValue := "default_value"
	os.Setenv(key, value)

	// Clean up after test
	defer os.Unsetenv(key)

	// Execute
	result := getEnv(key, defaultValue)

	// Assert
	assert.Equal(t, value, result)
}

func TestGetEnv_WithoutValue(t *testing.T) {
	// Setup
	key := "NON_EXISTENT_ENV_VAR"
	defaultValue := "default_value"
	os.Unsetenv(key) // Ensure it doesn't exist

	// Execute
	result := getEnv(key, defaultValue)

	// Assert
	assert.Equal(t, defaultValue, result)
}

func TestGetEnv_EmptyValue(t *testing.T) {
	// Setup
	key := "EMPTY_ENV_VAR"
	defaultValue := "default_value"
	os.Setenv(key, "")

	// Clean up after test
	defer os.Unsetenv(key)

	// Execute
	result := getEnv(key, defaultValue)

	// Assert that empty string returns default
	assert.Equal(t, defaultValue, result)
}

func TestConfig_StructFields(t *testing.T) {
	// Test that Config struct has all expected fields
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
			Host: "localhost",
		},
		Database: DatabaseConfig{
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
	// Test that ServerConfig struct has all expected fields
	serverConfig := &ServerConfig{
		Port: "3000",
		Host: "0.0.0.0",
	}

	assert.Equal(t, "3000", serverConfig.Port)
	assert.Equal(t, "0.0.0.0", serverConfig.Host)
}

func TestDatabaseConfig_StructFields(t *testing.T) {
	// Test that DatabaseConfig struct has all expected fields
	dbConfig := &DatabaseConfig{
		URI:  "mongodb://test:27017",
		Name: "test_database",
	}

	assert.Equal(t, "mongodb://test:27017", dbConfig.URI)
	assert.Equal(t, "test_database", dbConfig.Name)
}

func TestLoadConfig_StructInitialization(t *testing.T) {
	// Clean environment
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")

	// Execute
	config := LoadConfig()

	// Assert that config is properly initialized
	assert.NotNil(t, config)
	assert.NotNil(t, config.Server)
	assert.NotNil(t, config.Database)
}

func TestGetEnv_DifferentDataTypes(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "Numeric string",
			envKey:       "NUMERIC_VAR",
			envValue:     "12345",
			defaultValue: "0",
			expected:     "12345",
		},
		{
			name:         "URL string",
			envKey:       "URL_VAR",
			envValue:     "https://example.com",
			defaultValue: "http://localhost",
			expected:     "https://example.com",
		},
		{
			name:         "Boolean-like string",
			envKey:       "BOOL_VAR",
			envValue:     "true",
			defaultValue: "false",
			expected:     "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			// Execute
			result := getEnv(tt.envKey, tt.defaultValue)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadConfig_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		envVars     map[string]string
		expected    Config
		description string
	}{
		{
			name:    "Development environment",
			envVars: map[string]string{},
			expected: Config{
				Server:   ServerConfig{Port: "8080", Host: "localhost"},
				Database: DatabaseConfig{URI: "mongodb://localhost:27017", Name: "family_budget"},
			},
			description: "Default development settings",
		},
		{
			name: "Production environment",
			envVars: map[string]string{
				"SERVER_PORT":      "80",
				"SERVER_HOST":      "0.0.0.0",
				"MONGODB_URI":      "mongodb://prod-mongo:27017",
				"MONGODB_DATABASE": "family_budget_prod",
			},
			expected: Config{
				Server:   ServerConfig{Port: "80", Host: "0.0.0.0"},
				Database: DatabaseConfig{URI: "mongodb://prod-mongo:27017", Name: "family_budget_prod"},
			},
			description: "Production settings with external MongoDB",
		},
		{
			name: "Docker environment",
			envVars: map[string]string{
				"SERVER_PORT":      "8080",
				"SERVER_HOST":      "0.0.0.0",
				"MONGODB_URI":      "mongodb://mongo:27017",
				"MONGODB_DATABASE": "family_budget",
			},
			expected: Config{
				Server:   ServerConfig{Port: "8080", Host: "0.0.0.0"},
				Database: DatabaseConfig{URI: "mongodb://mongo:27017", Name: "family_budget"},
			},
			description: "Docker compose environment",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Clean environment first
			envKeys := []string{"SERVER_PORT", "SERVER_HOST", "MONGODB_URI", "MONGODB_DATABASE"}
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Setup environment variables
			for key, value := range scenario.envVars {
				os.Setenv(key, value)
			}

			// Clean up after test
			defer func() {
				for key := range scenario.envVars {
					os.Unsetenv(key)
				}
			}()

			// Execute
			config := LoadConfig()

			// Assert
			assert.Equal(t, scenario.expected.Server.Port, config.Server.Port, scenario.description)
			assert.Equal(t, scenario.expected.Server.Host, config.Server.Host, scenario.description)
			assert.Equal(t, scenario.expected.Database.URI, config.Database.URI, scenario.description)
			assert.Equal(t, scenario.expected.Database.Name, config.Database.Name, scenario.description)
		})
	}
}

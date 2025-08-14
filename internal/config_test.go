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

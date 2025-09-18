package internal

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Configuration constants
const (
	// Server timeout defaults
	defaultServerReadTimeout  = 15 * time.Second
	defaultServerWriteTimeout = 15 * time.Second
	defaultServerIdleTimeout  = 60 * time.Second

	// Database connection defaults
	defaultMaxOpenConns    = 50
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = 1 * time.Hour
	defaultConnMaxIdleTime = 5 * time.Minute

	// Web session defaults
	defaultSessionTimeout = 24 * time.Hour

	// Test environment defaults
	testMaxOpenConns = 5
	testMaxIdleConns = 2
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Web         WebConfig
	Logging     LoggingConfig
	Environment string
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URI             string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
	Schema          string
}

type WebConfig struct {
	SessionSecret  string
	SessionTimeout time.Duration
	CSRFSecret     string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string
}

type LoggingConfig struct {
	Level      string
	Format     string
	OutputPath string
}

// IsProduction returns true if the application is running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if the application is running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTest returns true if the application is running in test environment
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}

func LoadConfig() *Config {
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "localhost"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", defaultServerReadTimeout),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", defaultServerWriteTimeout),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", defaultServerIdleTimeout),
		},
		Database: DatabaseConfig{
			URI: getEnv(
				"POSTGRESQL_URI",
				"postgres://postgres:postgres123@localhost:5432/family_budget?sslmode=disable",
			),
			Name: getEnv("POSTGRESQL_DATABASE", "family_budget"),
			MaxOpenConns: getIntEnv(
				"POSTGRESQL_MAX_OPEN_CONNS",
				defaultMaxOpenConns,
			), // Increased for better concurrency
			MaxIdleConns: getIntEnv(
				"POSTGRESQL_MAX_IDLE_CONNS",
				defaultMaxIdleConns,
			), // Increased to maintain warm connections
			ConnMaxLifetime: getDurationEnv(
				"POSTGRESQL_CONN_MAX_LIFETIME",
				defaultConnMaxLifetime,
			), // Extended to reduce reconnections
			ConnMaxIdleTime: getDurationEnv(
				"POSTGRESQL_CONN_MAX_IDLE_TIME",
				defaultConnMaxIdleTime,
			), // Optimized idle time
			SSLMode: getEnv("POSTGRESQL_SSL_MODE", "prefer"),
			Schema:  getEnv("POSTGRESQL_SCHEMA", "family_budget"),
		},
		Web: WebConfig{
			SessionSecret:  getEnv("SESSION_SECRET", "your-super-secret-session-key-change-in-production"),
			SessionTimeout: getDurationEnv("SESSION_TIMEOUT", defaultSessionTimeout),
			CSRFSecret:     getEnv("CSRF_SECRET", "your-csrf-secret-key-change-in-production"),
			CookieSecure:   getBoolEnv("COOKIE_SECURE", false),
			CookieHTTPOnly: getBoolEnv("COOKIE_HTTP_ONLY", true),
			CookieSameSite: getEnv("COOKIE_SAME_SITE", "Lax"),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	// Adjust settings based on environment
	if config.IsProduction() {
		config.Web.CookieSecure = true
		config.Database.SSLMode = "require"
		if config.Logging.Level == "debug" {
			config.Logging.Level = "info"
		}
	}

	if config.IsDevelopment() {
		config.Database.SSLMode = "disable"
		config.Web.CookieSecure = false
		if config.Logging.Level == "" {
			config.Logging.Level = "debug"
		}
	}

	if config.IsTest() {
		config.Database.SSLMode = "disable"
		config.Web.CookieSecure = false
		config.Logging.Level = "warn"
		config.Database.MaxOpenConns = testMaxOpenConns
		config.Database.MaxIdleConns = testMaxIdleConns
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Add validation logic here
	if c.Web.SessionSecret == "your-super-secret-session-key-change-in-production" && c.IsProduction() {
		return errors.New("session secret must be changed in production")
	}

	if c.Web.CSRFSecret == "your-csrf-secret-key-change-in-production" && c.IsProduction() {
		return errors.New("CSRF secret must be changed in production")
	}

	if c.Database.URI == "" {
		return errors.New("database URI is required")
	}

	return nil
}

// GetConnectionString returns the database connection string with additional parameters
func (c *Config) GetConnectionString() string {
	if c.Database.URI == "" {
		return ""
	}

	// If URI already contains query parameters, append to them
	separator := "?"
	if strings.Contains(c.Database.URI, "?") {
		separator = "&"
	}

	params := []string{
		fmt.Sprintf("sslmode=%s", c.Database.SSLMode),
		"connect_timeout=10",
		"statement_timeout=30000",                   // 30 seconds
		"idle_in_transaction_session_timeout=60000", // 1 minute
	}

	return c.Database.URI + separator + strings.Join(params, "&")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

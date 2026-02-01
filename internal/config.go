package internal

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Configuration constants
const (
	// Server timeout defaults
	defaultServerReadTimeout  = 15 * time.Second
	defaultServerWriteTimeout = 15 * time.Second
	defaultServerIdleTimeout  = 60 * time.Second

	// Web session defaults
	defaultSessionTimeout = 24 * time.Hour
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
	// SQLite configuration
	Path string
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
			Path: getEnv("DATABASE_PATH", "./data/budget.db"),
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
		if config.Logging.Level == "debug" {
			config.Logging.Level = "info"
		}
	}

	if config.IsDevelopment() {
		config.Web.CookieSecure = false
		if config.Logging.Level == "" {
			config.Logging.Level = "debug"
		}
		// Warn about default secrets in development
		if config.Web.SessionSecret == "your-super-secret-session-key-change-in-production" ||
			config.Web.CSRFSecret == "your-csrf-secret-key-change-in-production" {
			fmt.Fprintln(
				os.Stderr,
				"WARNING: Using default secrets in development mode - ensure these are changed before deploying to production",
			)
		}
	}

	if config.IsTest() {
		config.Web.CookieSecure = false
		config.Logging.Level = "warn"
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

	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	return nil
}

// GetDatabasePath returns the database file path
func (c *Config) GetDatabasePath() string {
	return c.Database.Path
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
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

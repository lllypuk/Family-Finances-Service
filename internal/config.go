package internal

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"family-budget-service/internal/auth"
)

// Configuration constants
const (
	// Server timeout defaults
	defaultServerReadTimeout  = 15 * time.Second
	defaultServerWriteTimeout = 15 * time.Second
	defaultServerIdleTimeout  = 60 * time.Second
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Logging     LoggingConfig
	Environment string
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxies — TRUSTED_PROXIES как есть: CIDR через запятую, чьи X-Forwarded-For принимаются.
	// Пусто — доверять только RemoteAddr; разбирает Validate, результат отдаёт TrustedProxyRanges.
	TrustedProxies string

	trustedProxyRanges []*net.IPNet
}

type DatabaseConfig struct {
	// SQLite configuration
	Path string
	// BackupDir — каталог файлов бэкапа. Пустой BACKUP_DIR означает
	// <dir(Path)>/backups; в контейнере каталог смонтирован отдельным томом,
	// поэтому путь задаётся явно.
	BackupDir string
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
			Port:           getEnv("SERVER_PORT", "8080"),
			Host:           getEnv("SERVER_HOST", "localhost"),
			ReadTimeout:    getDurationEnv("SERVER_READ_TIMEOUT", defaultServerReadTimeout),
			WriteTimeout:   getDurationEnv("SERVER_WRITE_TIMEOUT", defaultServerWriteTimeout),
			IdleTimeout:    getDurationEnv("SERVER_IDLE_TIMEOUT", defaultServerIdleTimeout),
			TrustedProxies: getEnv("TRUSTED_PROXIES", ""),
		},
		Database: DatabaseConfig{
			Path:      getEnv("DATABASE_PATH", "./data/budget.db"),
			BackupDir: getEnv("BACKUP_DIR", ""),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	if config.IsProduction() && config.Logging.Level == "debug" {
		config.Logging.Level = "info"
	}

	if config.IsDevelopment() && config.Logging.Level == "" {
		config.Logging.Level = "debug"
	}

	if config.IsTest() {
		config.Logging.Level = "warn"
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	ranges, err := auth.ParseTrustedProxies(c.Server.TrustedProxies)
	if err != nil {
		return fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	c.Server.trustedProxyRanges = ranges
	return nil
}

// TrustedProxyRanges — разобранный TRUSTED_PROXIES для echo.IPExtractor; заполняет Validate.
func (c *Config) TrustedProxyRanges() []*net.IPNet {
	return c.Server.trustedProxyRanges
}

// GetBackupDir returns the backup directory, falling back to <dir(database)>/backups.
func (c *Config) GetBackupDir() string {
	if c.Database.BackupDir != "" {
		return c.Database.BackupDir
	}
	return filepath.Join(filepath.Dir(c.Database.Path), "backups")
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

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

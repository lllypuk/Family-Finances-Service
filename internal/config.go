package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	// minProductionSecretLength — минимальная длина секретов в production.
	// Проверки только на плейсхолдер было мало: SESSION_SECRET=123 её проходил
	// (S-04, docs/specs/002-security-audit.md). `openssl rand -base64 32`
	// даёт 44 символа.
	minProductionSecretLength = 32

	placeholderSessionSecret = "your-super-secret-session-key-change-in-production"
	placeholderCSRFSecret    = "your-csrf-secret-key-change-in-production"
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
	// BackupDir — каталог файлов бэкапа. Пустой BACKUP_DIR означает
	// <dir(Path)>/backups; в контейнере каталог смонтирован отдельным томом,
	// поэтому путь задаётся явно.
	BackupDir string
}

type WebConfig struct {
	SessionSecret  string
	SessionTimeout time.Duration
	// CSRFSecret пока ничего не подписывает: CSRF-токен генерируется случайно
	// и хранится в сессии, которую защищает SESSION_SECRET. Значение остаётся
	// обязательным в production (и валидируется), чтобы не менять контракт
	// развёртывания задним числом; см. docs/backlog.md.
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
			Path:      getEnv("DATABASE_PATH", "./data/budget.db"),
			BackupDir: getEnv("BACKUP_DIR", ""),
		},
		Web: WebConfig{
			SessionSecret:  getEnv("SESSION_SECRET", placeholderSessionSecret),
			SessionTimeout: getDurationEnv("SESSION_TIMEOUT", defaultSessionTimeout),
			CSRFSecret:     getEnv("CSRF_SECRET", placeholderCSRFSecret),
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
		// COOKIE_SECURE остаётся управляемым и в production: `Secure` на
		// session-cookie означает, что браузер выбросит её на любом http://
		// origin, и вход превращается в бесконечный редирект. Это ровно случай
		// docker-compose.minimal.yml и первого запуска по IP/SSH-туннелю до
		// того, как перед сервисом появился TLS-прокси. По умолчанию true.
		config.Web.CookieSecure = getBoolEnv("COOKIE_SECURE", true)
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
		if config.Web.SessionSecret == placeholderSessionSecret ||
			config.Web.CSRFSecret == placeholderCSRFSecret {
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
	if c.Database.Path == "" {
		return errors.New("database path is required")
	}

	if !c.IsProduction() {
		return nil
	}

	if err := validateProductionSecret("session secret", c.Web.SessionSecret, placeholderSessionSecret); err != nil {
		return err
	}

	return validateProductionSecret("CSRF secret", c.Web.CSRFSecret, placeholderCSRFSecret)
}

// validateProductionSecret отвергает секреты, оставленные плейсхолдером или
// слишком короткие, чтобы их нельзя было подобрать.
func validateProductionSecret(name, value, placeholder string) error {
	if value == placeholder {
		return fmt.Errorf("%s must be changed in production - generate one with `openssl rand -base64 32`", name)
	}

	if len(value) < minProductionSecretLength {
		return fmt.Errorf(
			"%s must be at least %d characters in production - generate one with `openssl rand -base64 32`",
			name, minProductionSecretLength,
		)
	}

	return nil
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

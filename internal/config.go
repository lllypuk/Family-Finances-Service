package internal

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/services"
)

// Configuration constants
const (
	// Server timeout defaults
	defaultServerReadTimeout  = 15 * time.Second
	defaultServerWriteTimeout = 15 * time.Second
	defaultServerIdleTimeout  = 60 * time.Second

	// maxPort — верхняя граница TCP-порта в METRICS_ADDR.
	maxPort = 65535

	defaultLLMModel   = "gemma4:31b-cloud"
	defaultLLMTimeout = 60 * time.Second
	// maxLLMTimeout закрепляет верх цепочки дедлайнов: Budget() движка и срок чтения клиента считаются от него.
	maxLLMTimeout = 60 * time.Second
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Logging     LoggingConfig
	LLM         LLMConfig
	Environment string
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxies — TRUSTED_PROXIES как есть: CIDR через запятую, чьи X-Forwarded-For принимаются.
	// Пусто — доверять только RemoteAddr; разбирает TrustedProxyRanges.
	TrustedProxies string
	// MetricsAddr — host:port служебного слушателя /metrics. Пусто — метрики выключены
	// целиком: ни порта, ни реестра.
	MetricsAddr string
}

type DatabaseConfig struct {
	// SQLite configuration
	Path string
	// BackupDir — каталог файлов бэкапа. Пустой BACKUP_DIR означает
	// <dir(Path)>/backups; в контейнере каталог смонтирован отдельным томом,
	// поэтому путь задаётся явно.
	BackupDir string
	// BackupKeep — BACKUP_KEEP: сколько последних файлов бэкапа хранить.
	BackupKeep int
}

// LLMConfig — плечо распознавания скриншотов; ключ облачных моделей живёт у демона Ollama, не здесь.
type LLMConfig struct {
	// OllamaHost — LLM_OLLAMA_HOST; пусто — распознавание выключено (503).
	OllamaHost string
	Model      string
	// Timeout — срок одной попытки вызова модели.
	Timeout time.Duration
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
			MetricsAddr:    getEnv("METRICS_ADDR", ""),
		},
		Database: DatabaseConfig{
			Path:       getEnv("DATABASE_PATH", "./data/budget.db"),
			BackupDir:  getEnv("BACKUP_DIR", ""),
			BackupKeep: getIntEnv("BACKUP_KEEP", services.DefaultBackupKeep),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
		},
		LLM: LLMConfig{
			OllamaHost: getEnv("LLM_OLLAMA_HOST", ""),
			Model:      getEnv("LLM_MODEL", defaultLLMModel),
			Timeout:    getDurationEnv("LLM_TIMEOUT", defaultLLMTimeout),
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

	if c.Server.MetricsAddr != "" {
		// SplitHostPort только режет строку: "0.0.0.0:abc" и порт 99999 проходят её,
		// а падает уже Start — и serveAll снимает вместе со служебным слушателем основной.
		_, port, err := net.SplitHostPort(c.Server.MetricsAddr)
		if err != nil {
			return fmt.Errorf("METRICS_ADDR: %w", err)
		}
		number, convErr := strconv.Atoi(port)
		if convErr != nil || number < 0 || number > maxPort {
			return fmt.Errorf("METRICS_ADDR: invalid port %q", port)
		}
	}

	if err := c.LLM.validate(); err != nil {
		return err
	}

	_, err := c.TrustedProxyRanges()
	return err
}

// validate проверяет плечо только при заданном хосте: выключенному модель и срок не нужны.
func (c LLMConfig) validate() error {
	if c.OllamaHost == "" {
		return nil
	}

	host, err := url.Parse(c.OllamaHost)
	if err != nil || (host.Scheme != "http" && host.Scheme != "https") || host.Host == "" {
		return fmt.Errorf("LLM_OLLAMA_HOST: want http(s)://host:port, got %q", c.OllamaHost)
	}

	if c.Model == "" {
		return errors.New("LLM_MODEL is required when LLM_OLLAMA_HOST is set")
	}

	if c.Timeout <= 0 || c.Timeout > maxLLMTimeout {
		return fmt.Errorf("LLM_TIMEOUT: want 0 < t <= %s, got %s", maxLLMTimeout, c.Timeout)
	}

	return nil
}

// TrustedProxyRanges — разобранный TRUSTED_PROXIES для echo.IPExtractor.
func (c *Config) TrustedProxyRanges() ([]*net.IPNet, error) {
	ranges, err := auth.ParseTrustedProxies(c.Server.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	return ranges, nil
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

// getIntEnv — неразбираемое или неположительное значение считается незаданным.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
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

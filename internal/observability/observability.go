package observability

import (
	"context"
	"log/slog"
)

// Config общая конфигурация для observability
type Config struct {
	Logging LogConfig `json:"logging"`
}

// Service центральный сервис для observability
type Service struct {
	Logger         *slog.Logger
	BusinessLogger *BusinessLogger
	HealthService  *HealthService
	Config         Config
}

// NewService создает новый observability service
func NewService(config Config, version string) (*Service, error) {
	// Создаем logger
	logger := NewLogger(config.Logging)

	// Создаем business logger
	businessLogger := NewBusinessLogger(logger)

	// Создаем health service
	healthService := NewHealthService(version)

	service := &Service{
		Logger:         logger,
		BusinessLogger: businessLogger,
		HealthService:  healthService,
		Config:         config,
	}

	logger.InfoContext(context.Background(), "Observability service initialized",
		slog.String("log_level", config.Logging.Level),
		slog.String("log_format", config.Logging.Format),
		slog.String("service_version", version),
	)

	return service, nil
}

// PostgreSQLHealthChecker интерфейс для PostgreSQL health check
type PostgreSQLHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// AddPostgreSQLHealthCheck добавляет health check для PostgreSQL
func (s *Service) AddPostgreSQLHealthCheck(pg PostgreSQLHealthChecker) {
	checker := NewPostgreSQLHealthChecker(pg)
	s.HealthService.AddChecker(checker)
	s.Logger.InfoContext(context.Background(), "PostgreSQL health check added")
}

// AddCustomHealthCheck добавляет пользовательский health check
func (s *Service) AddCustomHealthCheck(name string, checkFunc func(ctx context.Context) error) {
	checker := NewCustomHealthChecker(name, checkFunc)
	s.HealthService.AddChecker(checker)
	s.Logger.InfoContext(context.Background(), "Custom health check added", slog.String("name", name))
}

// Shutdown корректно завершает все observability компоненты
func (s *Service) Shutdown(ctx context.Context) error {
	s.Logger.InfoContext(ctx, "Shutting down observability service")
	s.Logger.InfoContext(ctx, "Observability service shutdown completed")
	return nil
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Logging: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

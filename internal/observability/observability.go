package observability

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/mongo"
)

// Config общая конфигурация для observability
type Config struct {
	Logging LogConfig     `json:"logging"`
	Tracing TracingConfig `json:"tracing"`
}

// Service центральный сервис для observability
type Service struct {
	Logger          *slog.Logger
	BusinessLogger  *BusinessLogger
	HealthService   *HealthService
	Config          Config
	shutdownTracing func(context.Context) error
}

// NewService создает новый observability service
func NewService(config Config, version string) (*Service, error) {
	// Инициализируем метрики
	InitMetrics()

	// Создаем logger
	logger := NewLogger(config.Logging)

	// Создаем business logger
	businessLogger := NewBusinessLogger(logger)

	// Инициализируем tracing
	shutdownTracing, err := InitTracing(context.Background(), config.Tracing, logger)
	if err != nil {
		return nil, err
	}

	// Создаем health service
	healthService := NewHealthService(version)

	service := &Service{
		Logger:          logger,
		BusinessLogger:  businessLogger,
		HealthService:   healthService,
		Config:          config,
		shutdownTracing: shutdownTracing,
	}

	logger.InfoContext(context.Background(), "Observability service initialized",
		slog.String("log_level", config.Logging.Level),
		slog.String("log_format", config.Logging.Format),
		slog.Bool("tracing_enabled", config.Tracing.Enabled),
		slog.String("service_version", version),
	)

	return service, nil
}

// AddMongoHealthCheck добавляет health check для MongoDB
func (s *Service) AddMongoHealthCheck(client *mongo.Client) {
	checker := NewMongoHealthChecker(client)
	s.HealthService.AddChecker(checker)
	s.Logger.InfoContext(context.Background(), "MongoDB health check added")
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

	if s.shutdownTracing != nil {
		if err := s.shutdownTracing(ctx); err != nil {
			s.Logger.ErrorContext(ctx, "Failed to shutdown tracing", slog.String("error", err.Error()))
			return err
		}
	}

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
		Tracing: TracingConfig{
			ServiceName:    "family-budget-service",
			ServiceVersion: "1.0.0",
			OTLPEndpoint:   "http://localhost:4318/v1/traces",
			Environment:    "development",
			Enabled:        true,
		},
	}
}

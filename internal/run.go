package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"family-budget-service/internal/application"
	"family-budget-service/internal/handlers"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/observability"
)

type Application struct {
	config               *Config
	repositories         *handlers.Repositories
	httpServer           *application.HTTPServer
	mongodb              *infrastructure.MongoDB
	observabilityService *observability.Service
}

func NewApplication() (*Application, error) {
	// Загрузка конфигурации
	config := LoadConfig()

	// Настройка observability
	obsConfig := observability.DefaultConfig()
	// Настраиваем уровень логирования из переменной окружения
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		obsConfig.Logging.Level = level
	}

	observabilityService, err := observability.NewService(obsConfig, "1.0.0")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	app := &Application{
		config:               config,
		observabilityService: observabilityService,
	}

	// Подключение к MongoDB
	mongodb, err := infrastructure.NewMongoDB(config.Database.URI, config.Database.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	app.mongodb = mongodb

	// Добавляем health check для MongoDB
	app.observabilityService.AddMongoHealthCheck(mongodb.Client)

	// Инициализация репозиториев
	app.repositories = infrastructure.NewRepositories(mongodb)

	// Создание HTTP сервера с observability
	serverConfig := &application.Config{
		Port: config.Server.Port,
		Host: config.Server.Host,
	}
	app.httpServer = application.NewHTTPServerWithObservability(
		app.repositories,
		serverConfig,
		app.observabilityService,
	)

	return app, nil
}

func (a *Application) Run() error {
	// Создание контекста для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для получения сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запуск HTTP сервера в горутине
	go func() {
		a.observabilityService.Logger.InfoContext(ctx, "Starting HTTP server",
			slog.String("host", a.config.Server.Host),
			slog.String("port", a.config.Server.Port))
		if err := a.httpServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.observabilityService.Logger.ErrorContext(ctx, "HTTP server error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	select {
	case sig := <-sigChan:
		a.observabilityService.Logger.InfoContext(ctx, "Received shutdown signal", slog.String("signal", sig.String()))
	case <-ctx.Done():
		a.observabilityService.Logger.InfoContext(ctx, "Context cancelled")
	}

	return a.shutdown()
}

func (a *Application) shutdown() error {
	a.observabilityService.Logger.InfoContext(context.Background(), "Shutting down application...")

	// Контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Остановка HTTP сервера
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.observabilityService.Logger.ErrorContext(ctx, "HTTP server shutdown error", slog.String("error", err.Error()))
	} else {
		a.observabilityService.Logger.InfoContext(ctx, "HTTP server stopped")
	}

	// Закрытие подключения к MongoDB
	if a.mongodb != nil {
		if err := a.mongodb.Close(ctx); err != nil {
			a.observabilityService.Logger.ErrorContext(
				ctx,
				"MongoDB disconnect error",
				slog.String("error", err.Error()),
			)
		} else {
			a.observabilityService.Logger.InfoContext(ctx, "MongoDB disconnected")
		}
	}

	// Остановка observability сервиса
	if err := a.observabilityService.Shutdown(ctx); err != nil {
		// Используем стандартный logger для последнего сообщения
		slog.ErrorContext(ctx, "Observability service shutdown error", slog.String("error", err.Error()))
	}

	a.observabilityService.Logger.InfoContext(ctx, "Application shutdown complete")
	return nil
}

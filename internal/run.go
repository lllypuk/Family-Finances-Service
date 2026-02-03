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
	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
)

const (
	// GracefulShutdownTimeout timeout for graceful application shutdown
	GracefulShutdownTimeout = 30 * time.Second
)

type Application struct {
	config               *Config
	repositories         *handlers.Repositories
	services             *services.Services
	httpServer           *application.HTTPServer
	sqliteConn           *infrastructure.SQLiteConnection
	observabilityService *observability.Service
}

func NewApplication() (*Application, error) {
	// Загрузка конфигурации
	config := LoadConfig()

	// Валидация конфигурации (включая проверку production secrets)
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

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

	// Подключение к SQLite
	sqliteConn, err := infrastructure.NewSQLiteConnection(config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}
	app.sqliteConn = sqliteConn

	// Запуск миграций
	dbURL := fmt.Sprintf("sqlite://%s", config.Database.Path)
	migrationManager := infrastructure.NewMigrationManager(dbURL, "./migrations")
	if err = migrationManager.Up(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Добавляем health check для SQLite
	app.observabilityService.AddCustomHealthCheck("sqlite", sqliteConn.HealthCheck)

	// Инициализация репозиториев
	app.repositories = infrastructure.NewRepositoriesSQLite(sqliteConn.DB())

	// Получаем logger из observability service
	logger := app.observabilityService.Logger

	// Инициализация BackupService
	backupService := services.NewBackupService(sqliteConn.DB(), config.Database.Path, logger)

	// Инициализация сервисов
	app.services = services.NewServices(
		app.repositories.User,
		app.repositories.Family,
		app.repositories.Category,
		app.repositories.Transaction,
		app.repositories.Budget, // BudgetRepositoryForTransactions
		app.repositories.Budget, // BudgetRepository
		app.repositories.Report,
		app.repositories.Invite,
		backupService,
		logger,
	)

	// Создание HTTP сервера с observability
	serverConfig := &application.Config{
		Port:          config.Server.Port,
		Host:          config.Server.Host,
		SessionSecret: config.Web.SessionSecret,
		IsProduction:  config.IsProduction(),
	}
	app.httpServer = application.NewHTTPServerWithObservability(
		app.repositories,
		app.services,
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
	ctx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
	defer cancel()

	// Остановка HTTP сервера
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.observabilityService.Logger.ErrorContext(ctx, "HTTP server shutdown error", slog.String("error", err.Error()))
	} else {
		a.observabilityService.Logger.InfoContext(ctx, "HTTP server stopped")
	}

	// Закрытие подключения к SQLite
	if a.sqliteConn != nil {
		if closeErr := a.sqliteConn.Close(); closeErr != nil {
			a.observabilityService.Logger.ErrorContext(
				ctx,
				"SQLite close error",
				slog.String("error", closeErr.Error()),
			)
		} else {
			a.observabilityService.Logger.InfoContext(ctx, "SQLite disconnected")
		}
	}

	// Логируем завершение работы приложения перед остановкой observability сервиса
	a.observabilityService.Logger.InfoContext(ctx, "Application shutdown complete")

	// Остановка observability сервиса (последний шаг)
	if err := a.observabilityService.Shutdown(ctx); err != nil {
		// На этом этапе logger уже может быть недоступен, используем простой log
		// Альтернативно можно игнорировать эту ошибку, так как приложение завершается
		_ = err // Игнорируем ошибку при shutdown observability сервиса
	}
	return nil
}

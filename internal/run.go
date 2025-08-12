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
)

type Application struct {
	config       *Config
	repositories *handlers.Repositories
	httpServer   *application.HTTPServer
	mongodb      *infrastructure.MongoDB
	logger       *slog.Logger
}

func NewApplication() (*Application, error) {
	// Загрузка конфигурации
	config := LoadConfig()

	// Настройка логгера
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	app := &Application{
		config: config,
		logger: logger,
	}

	// Подключение к MongoDB
	mongodb, err := infrastructure.NewMongoDB(config.Database.URI, config.Database.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	app.mongodb = mongodb

	// Инициализация репозиториев
	app.repositories = infrastructure.NewRepositories(mongodb)

	// Создание HTTP сервера
	serverConfig := &application.Config{
		Port: config.Server.Port,
		Host: config.Server.Host,
	}
	app.httpServer = application.NewHTTPServer(app.repositories, serverConfig)

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
		a.logger.Info("Starting HTTP server", "host", a.config.Server.Host, "port", a.config.Server.Port)
		if err := a.httpServer.Start(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("HTTP server error", "error", err)
			cancel()
		}
	}()

	// Ожидание сигнала завершения
	select {
	case sig := <-sigChan:
		a.logger.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		a.logger.Info("Context cancelled")
	}

	return a.shutdown()
}

func (a *Application) shutdown() error {
	a.logger.Info("Shutting down application...")

	// Контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Остановка HTTP сервера
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error("HTTP server shutdown error", "error", err)
	} else {
		a.logger.Info("HTTP server stopped")
	}

	// Закрытие подключения к MongoDB
	if a.mongodb != nil {
		if err := a.mongodb.Close(ctx); err != nil {
			a.logger.Error("MongoDB disconnect error", "error", err)
		} else {
			a.logger.Info("MongoDB disconnected")
		}
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

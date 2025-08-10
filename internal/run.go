package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/application"
)

type Application struct {
	config       *Config
	repositories *application.Repositories
	httpServer   *application.HTTPServer
	mongoClient  *mongo.Client
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
	if err := app.connectToMongoDB(); err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Инициализация репозиториев
	app.initRepositories()

	// Создание HTTP сервера
	serverConfig := &application.Config{
		Port: config.Server.Port,
		Host: config.Server.Host,
	}
	app.httpServer = application.NewHTTPServer(app.repositories, serverConfig)

	return app, nil
}

func (a *Application) connectToMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(a.config.Database.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверка подключения
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	a.mongoClient = client
	a.logger.Info("Successfully connected to MongoDB")
	return nil
}

func (a *Application) initRepositories() {
	// TODO: Здесь будут инициализированы реальные MongoDB репозитории
	// Пока создаем пустую структуру
	a.repositories = &application.Repositories{
		// User:        userRepo,
		// Family:      familyRepo,
		// Category:    categoryRepo,
		// Transaction: transactionRepo,
		// Budget:      budgetRepo,
		// Report:      reportRepo,
	}
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
		if err := a.httpServer.Start(ctx); err != nil && err != http.ErrServerClosed {
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
	if a.mongoClient != nil {
		if err := a.mongoClient.Disconnect(ctx); err != nil {
			a.logger.Error("MongoDB disconnect error", "error", err)
		} else {
			a.logger.Info("MongoDB disconnected")
		}
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

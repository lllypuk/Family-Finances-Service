package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/infrastructure/llmengine"
	"family-budget-service/internal/metrics"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/version"
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
	db                   *sql.DB
	observabilityService *observability.Service
	listeners            []listener
}

// listener — слушатель, которым распоряжаются serveAll и shutdown.
type listener interface {
	Start() error
	Shutdown(ctx context.Context) error
}

// metricsListener приводит служебный *http.Server к listener: он поднимается сам, без echo.
type metricsListener struct {
	*http.Server
}

func (l metricsListener) Start() error { return l.ListenAndServe() }

// backupTimesFunc — замыкание над BackupService под metrics.BackupLister:
// пакет metrics не знает про services.BackupInfo.
type backupTimesFunc func(ctx context.Context) ([]time.Time, error)

func (f backupTimesFunc) ListBackupTimes(ctx context.Context) ([]time.Time, error) { return f(ctx) }

func NewApplication() (*Application, error) {
	// Загрузка конфигурации
	config := LoadConfig()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Настройка observability
	obsConfig := observability.DefaultConfig()
	// Настраиваем уровень логирования из переменной окружения
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		obsConfig.Logging.Level = level
	}

	observabilityService, err := observability.NewService(obsConfig, version.String())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	app := &Application{
		config:               config,
		observabilityService: observabilityService,
	}

	db, err := OpenDatabase(config)
	if err != nil {
		return nil, err
	}
	app.db = db

	// Добавляем health check для SQLite
	app.observabilityService.AddCustomHealthCheck("sqlite", db.PingContext)

	// Инициализация репозиториев
	app.repositories = infrastructure.NewRepositoriesSQLite(db)

	// Получаем logger из observability service
	logger := app.observabilityService.Logger

	// Реестр метрик — до сервисов: наблюдатели едут в них параметром.
	// Пустой METRICS_ADDR выключает метрики целиком, а не только слушателя.
	var registry *metrics.Metrics
	if config.Server.MetricsAddr != "" {
		registry = metrics.New(version.String(), logger)
	}

	// Инициализация BackupService
	backupService := services.NewBackupService(
		db,
		config.Database.Path,
		config.GetBackupDir(),
		config.Database.BackupKeep,
		logger,
		backupObserver(registry),
	)

	authService := auth.NewService(app.repositories.Session, app.repositories.User, app.repositories.Family)

	// Инициализация сервисов
	app.services = services.NewServices(
		app.repositories.User,
		app.repositories.Family,
		app.repositories.Category,
		app.repositories.Account,
		app.repositories.Holding,
		app.repositories.Balance,
		app.repositories.Transaction,
		app.repositories.Budget, // BudgetRepositoryForTransactions
		app.repositories.Budget, // BudgetRepository
		backupService,
		authService,
		recognizer(config.LLM, logger),
		recognizeObserver(registry),
		logger,
	)

	trustedProxies, err := config.TrustedProxyRanges()
	if err != nil {
		return nil, err
	}
	if config.IsProduction() && len(trustedProxies) == 0 {
		logger.WarnContext(context.Background(),
			"TRUSTED_PROXIES is empty: X-Forwarded-For is ignored and the login limiter counts only per email; "+
				"list the reverse proxy network to enable the per-IP limit")
	}

	// Создание HTTP сервера с observability
	serverConfig := &application.Config{
		Port:           config.Server.Port,
		Host:           config.Server.Host,
		ReadTimeout:    config.Server.ReadTimeout,
		WriteTimeout:   config.Server.WriteTimeout,
		IdleTimeout:    config.Server.IdleTimeout,
		TrustedProxies: trustedProxies,
		Metrics:        registry,
	}
	app.httpServer = application.NewHTTPServerWithObservability(
		app.repositories,
		app.services,
		serverConfig,
		app.observabilityService,
	)
	app.listeners = []listener{app.httpServer}

	if registry != nil {
		app.registerCollectors(registry, backupService)
		app.listeners = append(app.listeners, metricsListener{
			Server: metrics.NewServer(config.Server.MetricsAddr, registry.Handler()),
		})
	}

	return app, nil
}

// registerCollectors подключает коллекторы, которые читают состояние на скрейпе.
// Ошибка регистрации — только в лог: метрика важнее старта сервиса.
func (a *Application) registerCollectors(registry *metrics.Metrics, backupService services.BackupService) {
	logger := a.observabilityService.Logger

	collectors := []prometheus.Collector{
		metrics.NewBackupDirCollector(backupTimesFunc(func(ctx context.Context) ([]time.Time, error) {
			backups, err := backupService.ListBackups(ctx)
			if err != nil {
				return nil, err
			}
			times := make([]time.Time, 0, len(backups))
			for _, backup := range backups {
				times = append(times, backup.CreatedAt)
			}
			return times, nil
		})),
		metrics.NewStateCollector(infrastructure.NewMetricsReader(a.db), a.config.Database.Path),
		metrics.NewDBCollector(a.db),
	}

	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			logger.ErrorContext(context.Background(), "collector not registered",
				slog.String("error", err.Error()))
		}
	}
}

// backupObserver отдаёт наблюдателя реестра или заглушку, если метрики выключены.
func backupObserver(m *metrics.Metrics) services.BackupObserver {
	if m == nil {
		return services.NopBackupObserver{}
	}
	return m.Backup()
}

// recognizer собирает движок над Ollama или nil — распознавание выключено.
// Возврат интерфейсом: *llmengine.Engine(nil) внутри интерфейса сервис не отличил бы от движка.
func recognizer(cfg LLMConfig, logger *slog.Logger) services.Recognizer {
	if cfg.OllamaHost == "" {
		return nil
	}
	return llmengine.New(cfg.OllamaHost, cfg.Model, cfg.Timeout, nil, logger)
}

// recognizeObserver отдаёт наблюдателя реестра или заглушку, если метрики выключены.
func recognizeObserver(m *metrics.Metrics) services.RecognizeObserver {
	if m == nil {
		return services.NopRecognizeObserver{}
	}
	return m.Recognize()
}

func (a *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a.observabilityService.Logger.InfoContext(ctx, "Starting HTTP server",
		slog.String("host", a.config.Server.Host),
		slog.String("port", a.config.Server.Port),
		slog.String("metrics_addr", a.config.Server.MetricsAddr))

	stopErrs, serveErr := serveAll(ctx, a.listeners...)
	if serveErr != nil {
		a.observabilityService.Logger.ErrorContext(ctx, "listener error", slog.String("error", serveErr.Error()))
	}
	for _, err := range stopErrs {
		if err != nil {
			a.observabilityService.Logger.ErrorContext(ctx, "listener shutdown error",
				slog.String("error", err.Error()))
		}
	}

	a.shutdown()
	return serveErr
}

// serveAll поднимает все слушатели и ждёт отмены контекста или первой ошибки старта;
// в обоих случаях гасит остальные, дожидается их горутин и возвращает ошибки остановки
// вместе с этой ошибкой. Гасим ровно один раз: повтор в shutdown() растянул бы
// graceful-окно вдвое, если первый Shutdown упёрся в таймаут.
func serveAll(ctx context.Context, listeners ...listener) ([]error, error) {
	errCh := make(chan error, len(listeners))

	var wg sync.WaitGroup
	for _, l := range listeners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}

	var startErr error
	select {
	case startErr = <-errCh:
	case <-ctx.Done():
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
	defer cancel()
	stopErrs := shutdownAll(stopCtx, listeners)
	wg.Wait()

	return stopErrs, startErr
}

// shutdownAll гасит слушатели параллельно: последовательно один зависший съел бы
// весь таймаут, отведённый на всех.
func shutdownAll(ctx context.Context, listeners []listener) []error {
	errs := make([]error, len(listeners))

	var wg sync.WaitGroup
	for i, l := range listeners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = l.Shutdown(ctx)
		}()
	}
	wg.Wait()

	return errs
}

// shutdown закрывает базу и observability. Вызывается только после serveAll:
// коллекторы /metrics ходят в базу на скрейпе, и db.Close() до остановки
// слушателя — гонка.
func (a *Application) shutdown() {
	a.observabilityService.Logger.InfoContext(context.Background(), "Shutting down application...")

	ctx, cancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
	defer cancel()

	if a.db != nil {
		if closeErr := a.db.Close(); closeErr != nil {
			a.observabilityService.Logger.ErrorContext(
				ctx,
				"SQLite close error",
				slog.String("error", closeErr.Error()),
			)
		} else {
			a.observabilityService.Logger.InfoContext(ctx, "SQLite disconnected")
		}
	}

	a.observabilityService.Logger.InfoContext(ctx, "Application shutdown complete")

	if err := a.observabilityService.Shutdown(ctx); err != nil {
		_ = err // Игнорируем ошибку при shutdown observability сервиса
	}
}

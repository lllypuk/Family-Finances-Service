package application

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/metrics"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/version"
)

const (
	// HTTPRequestTimeout timeout for HTTP requests
	HTTPRequestTimeout = 30 * time.Second

	// recognizeRoute живёт на своих дедлайнах (handlers.RecognizeHandler), общий таймаут его не касается.
	recognizeRoute = "/api/v1/transactions/recognize"
)

type HTTPServer struct {
	echo                 *echo.Echo
	server               *http.Server
	services             *services.Services
	config               *Config
	observabilityService *observability.Service
	healthService        *observability.HealthService

	// API Handlers
	authHandler           *handlers.AuthHandler
	meHandler             *handlers.MeHandler
	userHandler           *handlers.UserHandler
	familyHandler         *handlers.FamilyHandler
	categoryHandler       *handlers.CategoryHandler
	accountHandler        *handlers.AccountHandler
	holdingHandler        *handlers.HoldingHandler
	transactionHandler    *handlers.TransactionHandler
	budgetHandler         *handlers.BudgetHandler
	statsHandler          *handlers.StatsHandler
	reconciliationHandler *handlers.ReconciliationHandler
	backupHandler         *handlers.BackupHandler
	recognizeHandler      *handlers.RecognizeHandler
}

type Config struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxies — сети, чей X-Forwarded-For определяет c.RealIP(); пусто — только RemoteAddr.
	TrustedProxies []*net.IPNet
	// LoginLimiter — лимитер POST /auth/login; nil — auth.NewRateLimiter, причём без TrustedProxies
	// корзина per-IP отключена: за прокси все клиенты — один адрес, лимит закрывал бы вход всей семье.
	LoginLimiter *auth.RateLimiter
	// Metrics — реестр Prometheus; nil — HTTP-метрики не пишутся (NewHTTPServer без Config.Metrics,
	// юнит-тесты хендлеров). Интеграционный стенд реестр задаёт.
	Metrics *metrics.Metrics
	// RecognizeUploadTimeout — срок приёма тела распознавания; 0 — handlers.RecognizeUploadTimeout.
	RecognizeUploadTimeout time.Duration
}

// NewHTTPServer создает HTTP сервер без observability (для обратной совместимости)
func NewHTTPServer(repositories *handlers.Repositories, services *services.Services, config *Config) *HTTPServer {
	return NewHTTPServerWithObservability(repositories, services, config, nil)
}

// NewHTTPServerWithObservability создает HTTP сервер с observability
func NewHTTPServerWithObservability(
	repositories *handlers.Repositories,
	services *services.Services,
	config *Config,
	obsService *observability.Service,
) *HTTPServer {
	e := echo.New()

	// Без observability /health всё равно отвечает по схеме Health (в том числе setup_complete).
	healthService := observability.NewHealthService(version.String())
	logger := slog.Default()
	if obsService != nil {
		healthService = obsService.HealthService
		logger = obsService.Logger
	}
	healthService.SetSetupChecker(services.Family.IsSetupComplete)

	limiter := config.LoginLimiter
	if limiter == nil {
		var opts []auth.LimiterOption
		if len(config.TrustedProxies) == 0 {
			opts = append(opts, auth.WithoutIPLimit())
		}
		limiter = auth.NewRateLimiter(nil, opts...)
	}

	e.Validator = &CustomValidator{validator: validator.New()}
	e.IPExtractor = auth.IPExtractor(config.TrustedProxies)
	e.HTTPErrorHandler = newAPIErrorHandler(logger)

	// Базовые middleware. Метрики — первыми, чтобы считать и запросы, отвергнутые дальше по цепочке.
	if config.Metrics != nil {
		e.Use(metrics.EchoMiddleware(config.Metrics.HTTP()))
	}
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: panicLogger(config.Metrics, logger),
	}))
	e.Use(middleware.RequestID())

	// Timeout для всех запросов
	e.Use(middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
		Skipper: func(c echo.Context) bool { return c.Path() == recognizeRoute },
		Timeout: HTTPRequestTimeout,
	}))

	// Добавляем observability middleware если сервис доступен
	if obsService != nil {
		// Structured logging
		e.Use(observability.LoggingMiddleware(obsService.Logger))
	} else {
		// Fallback к стандартному логированию
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogStatus: true,
			LogURI:    true,
			LogError:  true,
			LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
				e.Logger.Info("request",
					"uri", v.URI,
					"status", v.Status,
					"error", v.Error,
				)
				return nil
			},
		}))
	}

	uploadTimeout := config.RecognizeUploadTimeout
	if uploadTimeout == 0 {
		uploadTimeout = handlers.RecognizeUploadTimeout
	}

	server := &HTTPServer{
		echo:                 e,
		services:             services,
		config:               config,
		observabilityService: obsService,
		healthService:        healthService,

		// Инициализация API handlers
		authHandler:           handlers.NewAuthHandler(services.Auth, limiter, logger, loginObserver(config.Metrics)),
		meHandler:             handlers.NewMeHandler(services.User, services.Auth),
		userHandler:           handlers.NewUserHandler(services.User, services.Auth),
		familyHandler:         handlers.NewFamilyHandler(services.Family),
		categoryHandler:       handlers.NewCategoryHandler(services.Category),
		accountHandler:        handlers.NewAccountHandler(services.Account),
		holdingHandler:        handlers.NewHoldingHandler(services.Holding),
		transactionHandler:    handlers.NewTransactionHandler(services.Transaction),
		budgetHandler:         handlers.NewBudgetHandler(repositories, services.Budget),
		statsHandler:          handlers.NewStatsHandler(services.Stats),
		reconciliationHandler: handlers.NewReconciliationHandler(services.Reconciliation),
		backupHandler:         handlers.NewBackupHandler(services.Backup),
		recognizeHandler:      handlers.NewRecognizeHandler(services.Recognize, uploadTimeout),
	}

	server.setupRoutes()
	// e.Server — то, что гасит echo.Shutdown; configureServer его не заполняет,
	// поэтому связать экземпляр надо здесь.
	server.server = server.buildNetHTTPServer(net.JoinHostPort(config.Host, config.Port))
	e.Server = server.server
	return server
}

// loginObserver отдаёт счётчик входов реестра или заглушку, если метрики выключены.
func loginObserver(m *metrics.Metrics) handlers.LoginObserver {
	if m == nil {
		return handlers.NopLoginObserver{}
	}
	return m.Login()
}

// panicLogger считает панику и пишет стек в slog; штатный вывод Recover при заданном
// callback выключается. Возврат nil подавил бы 500 — ошибку нужно вернуть как есть.
func panicLogger(m *metrics.Metrics, logger *slog.Logger) middleware.LogErrorFunc {
	return func(c echo.Context, err error, stack []byte) error {
		if m != nil {
			m.HTTP().ObservePanic()
		}
		logger.ErrorContext(c.Request().Context(), "HTTP handler panicked",
			slog.String("method", c.Request().Method),
			slog.String("path", c.Request().URL.Path),
			slog.String("error", err.Error()),
			slog.String("stack", string(stack)),
		)

		return err
	}
}

// Echo returns the echo instance for testing purposes
func (s *HTTPServer) Echo() *echo.Echo {
	return s.echo
}

func (s *HTTPServer) setupRoutes() {
	s.echo.GET("/health", s.healthService.HealthHandler())

	// Явный корневой catch-all: иначе у несматченного пути c.Path() пуст либо достаётся
	// соседнему шаблону, и метка route у метрик врёт (/healthz → /health).
	s.echo.RouteNotFound("/*", echo.NotFoundHandler)

	// Единственный публичный маршрут API. Зарегистрирован мимо группы, поэтому
	// RequireBearer его не касается.
	s.echo.POST("/api/v1/auth/login", s.authHandler.Login)

	api := s.echo.Group("/api/v1", auth.RequireBearer(s.services.Auth))

	api.POST("/auth/logout", s.authHandler.Logout)
	api.GET("/auth/sessions", s.authHandler.ListSessions)
	api.DELETE("/auth/sessions/:id", s.authHandler.RevokeSession)

	api.GET("/me", s.meHandler.GetMe)
	api.PUT("/me", s.meHandler.UpdateMe)
	api.PUT("/me/password", s.meHandler.ChangePassword)

	s.setupResourceRoutes(api)
}

// setupHoldingRoutes — активы и пассивы; удаление уносит историю снимков, поэтому только админ.
func (s *HTTPServer) setupHoldingRoutes(api *echo.Group, financeAccess, adminOnly echo.MiddlewareFunc) {
	holdings := api.Group("/holdings", financeAccess)
	holdings.GET("", s.holdingHandler.ListHoldings)
	holdings.POST("", s.holdingHandler.CreateHolding)
	holdings.PUT("/:id", s.holdingHandler.UpdateHolding)
	holdings.DELETE("/:id", s.holdingHandler.DeleteHolding, adminOnly)
	holdings.GET("/:id/values", s.holdingHandler.ListHoldingValues)
	holdings.PUT("/:id/values/:date", s.holdingHandler.PutHoldingValue)
	holdings.DELETE("/:id/values/:date", s.holdingHandler.DeleteHoldingValue)
}

// setupResourceRoutes — ролевая модель: управление пользователями — только админ,
// финансовые разделы — админ и member. Удаление категории закрыто до админа:
// через API оно необратимо и без подтверждения.
func (s *HTTPServer) setupResourceRoutes(api *echo.Group) {
	adminOnly := auth.RequireRole(user.RoleAdmin)
	financeAccess := auth.RequireRole(user.RoleAdmin, user.RoleMember)

	// Семья: читают все роли, меняет только админ.
	family := api.Group("/family")
	family.GET("", s.familyHandler.GetFamily)
	family.PUT("", s.familyHandler.UpdateFamily, adminOnly)

	users := api.Group("/users", adminOnly)
	users.POST("", s.userHandler.CreateUser)
	users.GET("", s.userHandler.GetUsers)
	users.GET("/:id", s.userHandler.GetUserByID)
	users.PUT("/:id", s.userHandler.UpdateUser)
	users.PATCH("/:id", s.userHandler.PatchUser)
	users.PUT("/:id/password", s.userHandler.SetUserPassword)

	categories := api.Group("/categories", financeAccess)
	categories.POST("", s.categoryHandler.CreateCategory)
	categories.GET("", s.categoryHandler.GetCategories)
	categories.GET("/:id", s.categoryHandler.GetCategoryByID)
	categories.PUT("/:id", s.categoryHandler.UpdateCategory)
	categories.DELETE("/:id", s.categoryHandler.DeleteCategory, adminOnly)

	accounts := api.Group("/accounts", financeAccess)
	accounts.GET("", s.accountHandler.ListAccounts)
	accounts.POST("", s.accountHandler.CreateAccount)
	accounts.PUT("/:id", s.accountHandler.UpdateAccount)
	accounts.DELETE("/:id", s.accountHandler.DeleteAccount, adminOnly)
	accounts.PUT("/:id/balances/:month", s.reconciliationHandler.PutAccountBalance)
	accounts.DELETE("/:id/balances/:month", s.reconciliationHandler.DeleteAccountBalance)

	s.setupHoldingRoutes(api, financeAccess, adminOnly)

	transactions := api.Group("/transactions", financeAccess)
	transactions.POST("", s.transactionHandler.CreateTransaction)
	transactions.GET("", s.transactionHandler.GetTransactions)
	transactions.POST("/bulk-delete", s.transactionHandler.BulkDeleteTransactions)
	transactions.POST("/recognize", s.recognizeHandler.Recognize, middleware.BodyLimit(handlers.RecognizeBodyLimit))
	transactions.GET("/:id", s.transactionHandler.GetTransactionByID)
	transactions.PUT("/:id", s.transactionHandler.UpdateTransaction)
	transactions.DELETE("/:id", s.transactionHandler.DeleteTransaction)

	budgets := api.Group("/budgets", financeAccess)
	budgets.POST("", s.budgetHandler.CreateBudget)
	budgets.GET("", s.budgetHandler.GetBudgets)
	budgets.GET("/:id", s.budgetHandler.GetBudgetByID)
	budgets.PUT("/:id", s.budgetHandler.UpdateBudget)
	budgets.DELETE("/:id", s.budgetHandler.DeleteBudget)

	stats := api.Group("/stats", financeAccess)
	stats.GET("/summary", s.statsHandler.GetSummary)
	stats.GET("/monthly", s.statsHandler.GetMonthly)
	stats.GET("/net-worth", s.statsHandler.GetNetWorth)
	stats.GET("/reconciliation", s.reconciliationHandler.GetReconciliationStats)

	backups := api.Group("/backups", adminOnly)
	backups.POST("", s.backupHandler.CreateBackup)
	backups.GET("", s.backupHandler.ListBackups)
	backups.GET("/:name/download", s.backupHandler.DownloadBackup)
	backups.DELETE("/:name", s.backupHandler.DeleteBackup)
}

// Start слушает до остановки; сам *http.Server собран в конструкторе, чтобы
// Shutdown был возможен и до того, как горутина Start успела стартовать.
func (s *HTTPServer) Start() error {
	return s.echo.StartServer(s.server)
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *HTTPServer) buildNetHTTPServer(address string) *http.Server {
	return &http.Server{
		Addr:         address,
		Handler:      s.echo,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
}

// CustomValidator wraps go-playground/validator for Echo
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates structs using go-playground/validator
func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

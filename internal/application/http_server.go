package application

import (
	"context"
	"fmt"
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
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/version"
)

const (
	// HTTPRequestTimeout timeout for HTTP requests
	HTTPRequestTimeout = 30 * time.Second
)

type HTTPServer struct {
	echo                 *echo.Echo
	repositories         *handlers.Repositories
	services             *services.Services
	config               *Config
	observabilityService *observability.Service
	healthService        *observability.HealthService

	// API Handlers
	authHandler        *handlers.AuthHandler
	meHandler          *handlers.MeHandler
	userHandler        *handlers.UserHandler
	familyHandler      *handlers.FamilyHandler
	categoryHandler    *handlers.CategoryHandler
	transactionHandler *handlers.TransactionHandler
	budgetHandler      *handlers.BudgetHandler
	reportHandler      *handlers.ReportHandler
	statsHandler       *handlers.StatsHandler
	backupHandler      *handlers.BackupHandler
}

type Config struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxies — сети, чей X-Forwarded-For определяет c.RealIP(); пусто — только RemoteAddr.
	TrustedProxies []*net.IPNet
	// LoginLimiter — лимитер POST /auth/login; nil — auth.NewRateLimiter(nil).
	LoginLimiter *auth.RateLimiter
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
		limiter = auth.NewRateLimiter(nil)
	}

	e.Validator = &CustomValidator{validator: validator.New()}
	e.IPExtractor = auth.IPExtractor(config.TrustedProxies)
	e.HTTPErrorHandler = newAPIErrorHandler(logger)

	// Базовые middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Timeout для всех запросов
	e.Use(middleware.ContextTimeoutWithConfig(middleware.ContextTimeoutConfig{
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

	server := &HTTPServer{
		echo:                 e,
		repositories:         repositories,
		services:             services,
		config:               config,
		observabilityService: obsService,
		healthService:        healthService,

		// Инициализация API handlers
		authHandler:        handlers.NewAuthHandler(services.Auth, limiter, logger),
		meHandler:          handlers.NewMeHandler(services.User, services.Auth),
		userHandler:        handlers.NewUserHandler(services.User, services.Auth),
		familyHandler:      handlers.NewFamilyHandler(services.Family),
		categoryHandler:    handlers.NewCategoryHandler(repositories, services.Category),
		transactionHandler: handlers.NewTransactionHandler(repositories, services.Transaction),
		budgetHandler:      handlers.NewBudgetHandler(repositories, services.Budget),
		reportHandler:      handlers.NewReportHandler(repositories, services.Report),
		statsHandler:       handlers.NewStatsHandler(services.Stats),
		backupHandler:      handlers.NewBackupHandler(services.Backup),
	}

	server.setupRoutes()
	return server
}

// Echo returns the echo instance for testing purposes
func (s *HTTPServer) Echo() *echo.Echo {
	return s.echo
}

func (s *HTTPServer) setupRoutes() {
	s.echo.GET("/health", s.healthService.HealthHandler())

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

// setupResourceRoutes — ролевая модель: управление пользователями — только админ,
// финансовые разделы — админ и member, роль child к ним не допущена. Удаление
// категории закрыто до админа: через API оно необратимо и без подтверждения.
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

	transactions := api.Group("/transactions", financeAccess)
	transactions.POST("", s.transactionHandler.CreateTransaction)
	transactions.GET("", s.transactionHandler.GetTransactions)
	transactions.POST("/bulk-delete", s.transactionHandler.BulkDeleteTransactions)
	transactions.GET("/:id", s.transactionHandler.GetTransactionByID)
	transactions.PUT("/:id", s.transactionHandler.UpdateTransaction)
	transactions.DELETE("/:id", s.transactionHandler.DeleteTransaction)

	budgets := api.Group("/budgets", financeAccess)
	budgets.POST("", s.budgetHandler.CreateBudget)
	budgets.GET("", s.budgetHandler.GetBudgets)
	budgets.GET("/:id", s.budgetHandler.GetBudgetByID)
	budgets.PUT("/:id", s.budgetHandler.UpdateBudget)
	budgets.DELETE("/:id", s.budgetHandler.DeleteBudget)

	reports := api.Group("/reports", financeAccess)
	reports.POST("", s.reportHandler.CreateReport)
	reports.GET("", s.reportHandler.GetReports)
	reports.GET("/:id", s.reportHandler.GetReportByID)
	reports.GET("/:id/export", s.reportHandler.ExportReport)
	reports.DELETE("/:id", s.reportHandler.DeleteReport)

	stats := api.Group("/stats", financeAccess)
	stats.GET("/summary", s.statsHandler.GetSummary)

	backups := api.Group("/backups", adminOnly)
	backups.POST("", s.backupHandler.CreateBackup)
	backups.GET("", s.backupHandler.ListBackups)
	backups.GET("/:name/download", s.backupHandler.DownloadBackup)
	backups.DELETE("/:name", s.backupHandler.DeleteBackup)
}

func (s *HTTPServer) Start(_ context.Context) error {
	address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	server := s.buildNetHTTPServer(address)
	s.echo.Server = server
	return s.echo.StartServer(server)
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

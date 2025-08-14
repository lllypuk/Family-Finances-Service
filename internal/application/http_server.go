package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"family-budget-service/internal/handlers"
	"family-budget-service/internal/observability"
)

type HTTPServer struct {
	echo                 *echo.Echo
	repositories         *handlers.Repositories
	config               *Config
	observabilityService *observability.Service

	// Handlers
	userHandler        *handlers.UserHandler
	familyHandler      *handlers.FamilyHandler
	categoryHandler    *handlers.CategoryHandler
	transactionHandler *handlers.TransactionHandler
	budgetHandler      *handlers.BudgetHandler
	reportHandler      *handlers.ReportHandler
}

type Config struct {
	Port string
	Host string
}

// NewHTTPServer создает HTTP сервер без observability (для обратной совместимости)
func NewHTTPServer(repositories *handlers.Repositories, config *Config) *HTTPServer {
	return NewHTTPServerWithObservability(repositories, config, nil)
}

// NewHTTPServerWithObservability создает HTTP сервер с observability
func NewHTTPServerWithObservability(
	repositories *handlers.Repositories,
	config *Config,
	obsService *observability.Service,
) *HTTPServer {
	e := echo.New()

	// Базовые middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	// Timeout для всех запросов
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	// Добавляем observability middleware если сервис доступен
	if obsService != nil {
		// OpenTelemetry tracing
		e.Use(otelecho.Middleware("family-budget-service"))

		// Prometheus metrics
		e.Use(observability.PrometheusMiddleware())

		// Structured logging
		e.Use(observability.LoggingMiddleware(obsService.Logger))

		// Health check middleware (исключает health endpoints из метрик)
		e.Use(observability.HealthCheckMiddleware())
	} else {
		// Fallback к стандартному логированию
		e.Use(middleware.Logger())
	}

	server := &HTTPServer{
		echo:                 e,
		repositories:         repositories,
		config:               config,
		observabilityService: obsService,

		// Инициализация handlers
		userHandler:        handlers.NewUserHandler(repositories),
		familyHandler:      handlers.NewFamilyHandler(repositories),
		categoryHandler:    handlers.NewCategoryHandler(repositories),
		transactionHandler: handlers.NewTransactionHandler(repositories),
		budgetHandler:      handlers.NewBudgetHandler(repositories),
		reportHandler:      handlers.NewReportHandler(repositories),
	}

	server.setupRoutes()
	return server
}

// Echo returns the echo instance for testing purposes
func (s *HTTPServer) Echo() *echo.Echo {
	return s.echo
}

func (s *HTTPServer) setupRoutes() {
	// Observability endpoints
	if s.observabilityService != nil {
		s.echo.GET("/metrics", observability.MetricsHandler())
		s.echo.GET("/health", s.observabilityService.HealthService.HealthHandler())
		s.echo.GET("/ready", s.observabilityService.HealthService.ReadinessHandler())
		s.echo.GET("/live", s.observabilityService.HealthService.LivenessHandler())
	} else {
		// Fallback health check
		s.echo.GET("/health", s.healthCheck)
	}

	// API версионирование
	api := s.echo.Group("/api/v1")

	// Маршруты для пользователей и семей
	users := api.Group("/users")
	users.POST("", s.userHandler.CreateUser)
	users.GET("/:id", s.userHandler.GetUserByID)
	users.PUT("/:id", s.userHandler.UpdateUser)
	users.DELETE("/:id", s.userHandler.DeleteUser)

	families := api.Group("/families")
	families.POST("", s.familyHandler.CreateFamily)
	families.GET("/:id", s.familyHandler.GetFamilyByID)
	families.GET("/:id/members", s.familyHandler.GetFamilyMembers)

	// Маршруты для категорий
	categories := api.Group("/categories")
	categories.POST("", s.categoryHandler.CreateCategory)
	categories.GET("", s.categoryHandler.GetCategories)
	categories.GET("/:id", s.categoryHandler.GetCategoryByID)
	categories.PUT("/:id", s.categoryHandler.UpdateCategory)
	categories.DELETE("/:id", s.categoryHandler.DeleteCategory)

	// Маршруты для транзакций
	transactions := api.Group("/transactions")
	transactions.POST("", s.transactionHandler.CreateTransaction)
	transactions.GET("", s.transactionHandler.GetTransactions)
	transactions.GET("/:id", s.transactionHandler.GetTransactionByID)
	transactions.PUT("/:id", s.transactionHandler.UpdateTransaction)
	transactions.DELETE("/:id", s.transactionHandler.DeleteTransaction)

	// Маршруты для бюджетов
	budgets := api.Group("/budgets")
	budgets.POST("", s.budgetHandler.CreateBudget)
	budgets.GET("", s.budgetHandler.GetBudgets)
	budgets.GET("/:id", s.budgetHandler.GetBudgetByID)
	budgets.PUT("/:id", s.budgetHandler.UpdateBudget)
	budgets.DELETE("/:id", s.budgetHandler.DeleteBudget)

	// Маршруты для отчетов
	reports := api.Group("/reports")
	reports.POST("", s.reportHandler.CreateReport)
	reports.GET("", s.reportHandler.GetReports)
	reports.GET("/:id", s.reportHandler.GetReportByID)
	reports.DELETE("/:id", s.reportHandler.DeleteReport)
}

func (s *HTTPServer) Start(_ context.Context) error {
	address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	return s.echo.Start(address)
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.echo.Shutdown(ctx)
}

func (s *HTTPServer) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

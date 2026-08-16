package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web"
)

const (
	// HTTPRequestTimeout timeout for HTTP requests
	HTTPRequestTimeout = 30 * time.Second

	// DefaultTemplatesDir — путь к шаблонам по умолчанию (относительно CWD процесса)
	DefaultTemplatesDir = "internal/web/templates"
	// DefaultStaticDir — путь к статике по умолчанию (относительно CWD процесса)
	DefaultStaticDir = "internal/web/static"
)

type HTTPServer struct {
	echo                 *echo.Echo
	repositories         *handlers.Repositories
	services             *services.Services
	config               *Config
	observabilityService *observability.Service

	// API Handlers
	userHandler        *handlers.UserHandler
	categoryHandler    *handlers.CategoryHandler
	transactionHandler *handlers.TransactionHandler
	budgetHandler      *handlers.BudgetHandler
	reportHandler      *handlers.ReportHandler

	// Web Interface
	webServer        *web.Server
	webServerInitErr error
}

type Config struct {
	Port          string
	Host          string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	SessionSecret string
	// CookieSecure — флаг Secure на session-cookie (и на flash-cookie).
	// Обычно совпадает с production, но управляется отдельно (COOKIE_SECURE):
	// на http:// origin браузер выбрасывает Secure-cookie, и вход становится
	// невозможен — см. internal/config.go.
	CookieSecure bool

	// TemplatesDir — путь к каталогу HTML-шаблонов. Пустое значение означает
	// DefaultTemplatesDir, который резолвится относительно рабочего каталога процесса.
	TemplatesDir string
	// StaticDir — путь к каталогу статики (css/js/img). Пустое значение означает
	// DefaultStaticDir, который так же резолвится относительно CWD процесса.
	StaticDir string
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

	// Настройка валидации
	e.Validator = &CustomValidator{validator: validator.New()}

	// Базовые middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
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

		// Инициализация API handlers
		userHandler:        handlers.NewUserHandler(repositories, services.User),
		categoryHandler:    handlers.NewCategoryHandler(repositories, services.Category),
		transactionHandler: handlers.NewTransactionHandler(repositories, services.Transaction),
		budgetHandler:      handlers.NewBudgetHandler(repositories, services.Budget),
		reportHandler:      handlers.NewReportHandler(repositories, services.Report),
	}

	// Инициализация веб-интерфейса
	templatesDir := config.TemplatesDir
	if templatesDir == "" {
		templatesDir = DefaultTemplatesDir
	}

	staticDir := config.StaticDir
	if staticDir == "" {
		staticDir = DefaultStaticDir
	}

	webServer, err := web.NewWebServer(
		e, repositories, services,
		web.Paths{TemplatesDir: templatesDir, StaticDir: staticDir},
		config.SessionSecret, config.CookieSecure,
	)
	if err != nil {
		// Не прерываем работу сервера, но ошибку обязательно логируем и сохраняем,
		// чтобы вызывающий код (в том числе тестовый хелпер) мог её увидеть.
		server.webServerInitErr = err
		if obsService != nil {
			obsService.Logger.Error("Failed to initialize web server", "error", err)
		} else {
			e.Logger.Errorf("Failed to initialize web server (templates dir %q): %v", templatesDir, err)
		}
	} else {
		server.webServer = webServer
	}

	server.setupRoutes()
	return server
}

// Echo returns the echo instance for testing purposes
func (s *HTTPServer) Echo() *echo.Echo {
	return s.echo
}

// WebServerInitError возвращает ошибку инициализации веб-интерфейса, если она была.
// nil означает, что веб-слой (сессии, CSRF, HTML-маршруты) зарегистрирован.
func (s *HTTPServer) WebServerInitError() error {
	return s.webServerInitErr
}

func (s *HTTPServer) setupRoutes() {
	// Health check endpoints
	if s.observabilityService != nil {
		s.echo.GET("/health", s.observabilityService.HealthService.HealthHandler())
	} else {
		// Fallback health check
		s.echo.GET("/health", s.healthCheck)
	}

	// Веб-интерфейс маршруты
	if s.webServer != nil {
		s.webServer.SetupRoutes()
	}

	// API версионирование.
	// RequireAPIAuth закрывает всю группу: без валидной сессии — 401 и JSON-ошибка
	// (находка S-01). /health и веб-маршруты регистрируются выше и не задеты.
	apiAuth := []echo.MiddlewareFunc{handlers.RequireAPIAuth()}
	// Та же перепроверка сессии по БД, что и в вебе. Сервер на моках (unit-тесты
	// application) собирается без сервисов — там перепроверять нечем и не по чему.
	if s.services != nil && s.services.User != nil {
		apiAuth = append(apiAuth, handlers.RequireAPIActiveUser(s.services.User))
	}

	api := s.echo.Group("/api/v1", apiAuth...)

	// Ролевая модель API повторяет веб (internal/web/web.go):
	// управление пользователями — только админ (там RequireAdmin), финансовые
	// разделы — админ и member (там RequireAdminOrMember), роль child к ним не
	// допущена. Удаление категории дополнительно закрыто до админа: через API
	// оно необратимо и не имеет подтверждения, которое есть в UI.
	adminOnly := handlers.RequireAPIRole(user.RoleAdmin)
	financeAccess := handlers.RequireAPIRole(user.RoleAdmin, user.RoleMember)

	// Маршруты для пользователей — целиком под админом, как /users в вебе.
	users := api.Group("/users", adminOnly)
	users.POST("", s.userHandler.CreateUser)
	users.GET("/:id", s.userHandler.GetUserByID)
	users.PUT("/:id", s.userHandler.UpdateUser)
	users.DELETE("/:id", s.userHandler.DeleteUser)

	// Маршруты для категорий
	categories := api.Group("/categories", financeAccess)
	categories.POST("", s.categoryHandler.CreateCategory)
	categories.GET("", s.categoryHandler.GetCategories)
	categories.GET("/:id", s.categoryHandler.GetCategoryByID)
	categories.PUT("/:id", s.categoryHandler.UpdateCategory)
	categories.DELETE("/:id", s.categoryHandler.DeleteCategory, adminOnly)

	// Маршруты для транзакций
	transactions := api.Group("/transactions", financeAccess)
	transactions.POST("", s.transactionHandler.CreateTransaction)
	transactions.GET("", s.transactionHandler.GetTransactions)
	transactions.GET("/:id", s.transactionHandler.GetTransactionByID)
	transactions.PUT("/:id", s.transactionHandler.UpdateTransaction)
	transactions.DELETE("/:id", s.transactionHandler.DeleteTransaction)

	// Маршруты для бюджетов
	budgets := api.Group("/budgets", financeAccess)
	budgets.POST("", s.budgetHandler.CreateBudget)
	budgets.GET("", s.budgetHandler.GetBudgets)
	budgets.GET("/:id", s.budgetHandler.GetBudgetByID)
	budgets.PUT("/:id", s.budgetHandler.UpdateBudget)
	budgets.DELETE("/:id", s.budgetHandler.DeleteBudget)

	// Маршруты для отчетов
	reports := api.Group("/reports", financeAccess)
	reports.POST("", s.reportHandler.CreateReport)
	reports.GET("", s.reportHandler.GetReports)
	reports.GET("/:id", s.reportHandler.GetReportByID)
	reports.DELETE("/:id", s.reportHandler.DeleteReport)
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

func (s *HTTPServer) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// CustomValidator wraps go-playground/validator for Echo
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates structs using go-playground/validator
func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

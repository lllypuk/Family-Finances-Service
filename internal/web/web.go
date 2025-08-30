package web

import (
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

// Server представляет веб-сервер для HTML интерфейса
type Server struct {
	echo         *echo.Echo
	repositories *handlers.Repositories
	services     *services.Services
	renderer     *TemplateRenderer

	// Handlers
	dashboardHandler   *webHandlers.DashboardHandler
	authHandler        *webHandlers.AuthHandler
	userHandler        *webHandlers.UserHandler
	categoryHandler    *webHandlers.CategoryHandler
	transactionHandler *webHandlers.TransactionHandler
	budgetHandler      *webHandlers.BudgetHandler
	reportHandler      *webHandlers.ReportHandler
}

// NewWebServer создает новый веб-сервер
func NewWebServer(
	e *echo.Echo,
	repositories *handlers.Repositories,
	services *services.Services,
	templatesDir, sessionSecret string,
	isProduction bool,
) (*Server, error) {
	// Создаем рендерер шаблонов
	renderer, err := NewTemplateRenderer(templatesDir)
	if err != nil {
		return nil, err
	}

	// Устанавливаем рендерер для Echo
	e.Renderer = renderer

	// Настраиваем middleware
	e.Use(middleware.SessionStore(sessionSecret, isProduction))
	e.Use(middleware.CSRFProtection())

	ws := &Server{
		echo:         e,
		repositories: repositories,
		services:     services,
		renderer:     renderer,

		// Инициализируем handlers
		dashboardHandler:   webHandlers.NewDashboardHandler(repositories, services),
		authHandler:        webHandlers.NewAuthHandler(repositories, services),
		userHandler:        webHandlers.NewUserHandler(repositories, services),
		categoryHandler:    webHandlers.NewCategoryHandler(repositories, services),
		transactionHandler: webHandlers.NewTransactionHandler(repositories, services),
		budgetHandler:      webHandlers.NewBudgetHandler(repositories, services),
		reportHandler:      webHandlers.NewReportHandler(repositories, services),
	}

	return ws, nil
}

// SetupRoutes настраивает маршруты для веб-интерфейса
func (ws *Server) SetupRoutes() {
	// Статические файлы
	ws.echo.Static("/static", "internal/web/static")

	// Аутентификация (доступна без авторизации)
	ws.echo.GET("/login", ws.authHandler.LoginPage, middleware.RedirectIfAuthenticated("/"))
	ws.echo.POST("/login", ws.authHandler.Login, middleware.RedirectIfAuthenticated("/"))
	ws.echo.GET("/register", ws.authHandler.RegisterPage, middleware.RedirectIfAuthenticated("/"))
	ws.echo.POST("/register", ws.authHandler.Register, middleware.RedirectIfAuthenticated("/"))
	ws.echo.POST("/logout", ws.authHandler.Logout)

	// Защищенные маршруты (требуют аутентификации)
	protected := ws.echo.Group("", middleware.RequireAuth())

	// Главная страница
	protected.GET("/", ws.dashboardHandler.Dashboard)

	// Управление пользователями (только для Admin)
	users := protected.Group("/users", middleware.RequireAdmin())
	users.GET("", ws.userHandler.Index)
	users.GET("/new", ws.userHandler.New)
	users.POST("", ws.userHandler.Create)

	// Категории (Admin и Member)
	categories := protected.Group("/categories", middleware.RequireAdminOrMember())
	categories.GET("", ws.categoryHandler.Index)
	categories.GET("/new", ws.categoryHandler.New)
	categories.POST("", ws.categoryHandler.Create)
	categories.GET("/:id/edit", ws.categoryHandler.Edit)
	categories.PUT("/:id", ws.categoryHandler.Update)
	categories.DELETE("/:id", ws.categoryHandler.Delete)

	// Транзакции (Admin и Member)
	transactions := protected.Group("/transactions", middleware.RequireAdminOrMember())
	transactions.GET("", ws.transactionHandler.Index)
	transactions.GET("/new", ws.transactionHandler.New)
	transactions.POST("", ws.transactionHandler.Create)
	transactions.GET("/:id/edit", ws.transactionHandler.Edit)
	transactions.PUT("/:id", ws.transactionHandler.Update)
	transactions.DELETE("/:id", ws.transactionHandler.Delete)
	transactions.POST("/bulk-delete", ws.transactionHandler.BulkDelete)

	// Бюджеты (Admin и Member)
	budgets := protected.Group("/budgets", middleware.RequireAdminOrMember())
	budgets.GET("", ws.budgetHandler.Index)
	budgets.GET("/new", ws.budgetHandler.New)
	budgets.POST("", ws.budgetHandler.Create)
	budgets.GET("/:id", ws.budgetHandler.Show)
	budgets.GET("/:id/edit", ws.budgetHandler.Edit)
	budgets.PUT("/:id", ws.budgetHandler.Update)
	budgets.DELETE("/:id", ws.budgetHandler.Delete)
	budgets.POST("/:id/activate", ws.budgetHandler.Activate)
	budgets.POST("/:id/deactivate", ws.budgetHandler.Deactivate)

	// Budget alerts
	budgets.GET("/alerts", ws.budgetHandler.Alerts)
	budgets.POST("/alerts", ws.budgetHandler.CreateAlert)
	budgets.DELETE("/alerts/:alert_id", ws.budgetHandler.DeleteAlert)

	// Отчеты (Admin и Member)
	reports := protected.Group("/reports", middleware.RequireAdminOrMember())
	reports.GET("", ws.reportHandler.Index)
	reports.GET("/new", ws.reportHandler.New)
	reports.POST("", ws.reportHandler.Create)
	reports.GET("/:id", ws.reportHandler.Show)
	reports.DELETE("/:id", ws.reportHandler.Delete)
	reports.GET("/:id/export", ws.reportHandler.Export)

	// HTMX endpoints
	htmx := protected.Group("/htmx", middleware.RequireAuth())

	// HTMX для dashboard
	htmx.GET("/dashboard/stats", ws.dashboardHandler.DashboardStats)
	htmx.GET("/transactions/recent", ws.dashboardHandler.RecentTransactions, middleware.RequireAdminOrMember())
	htmx.GET("/budgets/overview", ws.dashboardHandler.BudgetOverview, middleware.RequireAdminOrMember())

	// HTMX для категорий
	htmx.GET("/categories/search", ws.categoryHandler.Search, middleware.RequireAdminOrMember())
	htmx.GET("/categories/select", ws.categoryHandler.Select, middleware.RequireAdminOrMember())

	// HTMX для транзакций
	htmx.GET("/transactions/filter", ws.transactionHandler.Filter, middleware.RequireAdminOrMember())
	htmx.GET("/transactions/list", ws.transactionHandler.List, middleware.RequireAdminOrMember())
	htmx.DELETE("/transactions/bulk-delete", ws.transactionHandler.BulkDelete, middleware.RequireAdminOrMember())
	htmx.DELETE("/transactions/:id", ws.transactionHandler.Delete, middleware.RequireAdminOrMember())

	// HTMX для бюджетов
	htmx.GET("/budgets/:id/progress", ws.budgetHandler.Progress, middleware.RequireAdminOrMember())

	// HTMX для отчетов
	htmx.POST("/reports/generate", ws.reportHandler.Generate, middleware.RequireAdminOrMember())
}

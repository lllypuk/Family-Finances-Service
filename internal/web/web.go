package web

import (
	"errors"
	"net/http"

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

	// Настраиваем обработчик ошибок
	e.HTTPErrorHandler = customHTTPErrorHandler(renderer)

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

	// Настраиваем маршруты аутентификации
	ws.setupAuthRoutes()

	// Защищенные маршруты (требуют аутентификации)
	protected := ws.echo.Group("", middleware.RequireAuth())

	// Главная страница
	protected.GET("/", ws.dashboardHandler.Dashboard)

	// Настраиваем группы защищенных маршрутов
	ws.setupProtectedRoutes(protected)

	// Настраиваем HTMX endpoints
	ws.setupHTMXRoutes(protected)
}

// setupAuthRoutes настраивает маршруты аутентификации
func (ws *Server) setupAuthRoutes() {
	ws.echo.GET("/login", ws.authHandler.LoginPage, middleware.RedirectIfAuthenticated("/"))
	ws.echo.POST("/login", ws.authHandler.Login, middleware.RedirectIfAuthenticated("/"))
	ws.echo.GET("/register", ws.authHandler.RegisterPage, middleware.RedirectIfAuthenticated("/"))
	ws.echo.POST("/register", ws.authHandler.Register, middleware.RedirectIfAuthenticated("/"))
	ws.echo.GET("/logout", ws.authHandler.Logout)
	ws.echo.POST("/logout", ws.authHandler.Logout)
}

// setupProtectedRoutes настраивает защищенные маршруты
func (ws *Server) setupProtectedRoutes(protected *echo.Group) {
	ws.setupUserRoutes(protected)
	ws.setupCategoryRoutes(protected)
	ws.setupTransactionRoutes(protected)
	ws.setupBudgetRoutes(protected)
	ws.setupReportRoutes(protected)
}

// setupUserRoutes настраивает маршруты управления пользователями
func (ws *Server) setupUserRoutes(protected *echo.Group) {
	users := protected.Group("/users", middleware.RequireAdmin())
	users.GET("", ws.userHandler.Index)
	users.GET("/new", ws.userHandler.New)
	users.POST("", ws.userHandler.Create)
}

// setupCategoryRoutes настраивает маршруты категорий
func (ws *Server) setupCategoryRoutes(protected *echo.Group) {
	categories := protected.Group("/categories", middleware.RequireAdminOrMember())
	categories.GET("", ws.categoryHandler.Index)
	categories.GET("/new", ws.categoryHandler.New)
	categories.POST("", ws.categoryHandler.Create)
	categories.GET("/:id", ws.categoryHandler.Show)
	categories.GET("/:id/edit", ws.categoryHandler.Edit)
	categories.PUT("/:id", ws.categoryHandler.Update)
	categories.DELETE("/:id", ws.categoryHandler.Delete)
}

// setupTransactionRoutes настраивает маршруты транзакций
func (ws *Server) setupTransactionRoutes(protected *echo.Group) {
	transactions := protected.Group("/transactions", middleware.RequireAdminOrMember())
	transactions.GET("", ws.transactionHandler.Index)
	transactions.GET("/new", ws.transactionHandler.New)
	transactions.POST("", ws.transactionHandler.Create)
	transactions.GET("/:id/edit", ws.transactionHandler.Edit)
	transactions.PUT("/:id", ws.transactionHandler.Update)
	transactions.DELETE("/:id", ws.transactionHandler.Delete)
	transactions.POST("/bulk-delete", ws.transactionHandler.BulkDelete)
}

// setupBudgetRoutes настраивает маршруты бюджетов
func (ws *Server) setupBudgetRoutes(protected *echo.Group) {
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
}

// setupReportRoutes настраивает маршруты отчетов
func (ws *Server) setupReportRoutes(protected *echo.Group) {
	reports := protected.Group("/reports", middleware.RequireAdminOrMember())
	reports.GET("", ws.reportHandler.Index)
	reports.GET("/new", ws.reportHandler.New)
	reports.POST("", ws.reportHandler.Create)
	reports.GET("/:id", ws.reportHandler.Show)
	reports.DELETE("/:id", ws.reportHandler.Delete)
	reports.GET("/:id/export", ws.reportHandler.Export)
}

// setupHTMXRoutes настраивает HTMX endpoints
func (ws *Server) setupHTMXRoutes(protected *echo.Group) {
	htmx := protected.Group("/htmx", middleware.RequireAuth())

	// HTMX для dashboard
	htmx.GET("/dashboard/stats", ws.dashboardHandler.DashboardStats)
	htmx.GET("/dashboard/filter", ws.dashboardHandler.DashboardFilter)
	htmx.GET("/dashboard/category-insights", ws.dashboardHandler.CategoryInsights, middleware.RequireAdminOrMember())
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

// customHTTPErrorHandler создает кастомный обработчик HTTP ошибок
func customHTTPErrorHandler(renderer *TemplateRenderer) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var (
			code = http.StatusInternalServerError
			msg  any
		)

		// Определяем тип ошибки
		var he *echo.HTTPError
		if errors.As(err, &he) {
			code = he.Code
			msg = he.Message
		} else {
			msg = err.Error()
		}

		// Если ответ уже отправлен, не делаем ничего
		if c.Response().Committed {
			return
		}

		// Для HTMX запросов возвращаем простой текст
		if c.Request().Header.Get("Hx-Request") == "true" {
			_ = c.String(code, "Error: "+err.Error())
			return
		}

		// Для обычных запросов рендерим страницу ошибки
		data := map[string]any{
			"PageData": map[string]any{
				"Title": getErrorTitle(code),
			},
			"StatusCode":   code,
			"ErrorMessage": msg,
		}

		// Пытаемся отрендерить страницу ошибки
		if renderErr := renderer.Render(c.Response(), "pages/error", data, c); renderErr != nil {
			// Fallback: простой текстовый ответ
			_ = c.String(code, "Error "+string(rune(code))+": "+renderErr.Error())
		}
	}
}

// getErrorTitle возвращает заголовок для страницы ошибки
func getErrorTitle(code int) string {
	switch code {
	case http.StatusNotFound:
		return "Страница не найдена"
	case http.StatusInternalServerError:
		return "Внутренняя ошибка сервера"
	case http.StatusForbidden:
		return "Доступ запрещен"
	case http.StatusUnauthorized:
		return "Требуется авторизация"
	case http.StatusBadRequest:
		return "Неверный запрос"
	default:
		return "Произошла ошибка"
	}
}

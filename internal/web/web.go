package web

import (
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

// Server представляет веб-сервер для HTML интерфейса
type Server struct {
	echo         *echo.Echo
	repositories *handlers.Repositories
	renderer     *TemplateRenderer

	// Handlers
	dashboardHandler *webHandlers.DashboardHandler
	authHandler      *webHandlers.AuthHandler
}

// NewWebServer создает новый веб-сервер
func NewWebServer(
	e *echo.Echo,
	repositories *handlers.Repositories,
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
		renderer:     renderer,

		// Инициализируем handlers
		dashboardHandler: webHandlers.NewDashboardHandler(repositories),
		authHandler:      webHandlers.NewAuthHandler(repositories),
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

	// HTMX endpoints
	htmx := protected.Group("/htmx")
	htmx.GET("/dashboard/stats", ws.dashboardHandler.DashboardStats)
}

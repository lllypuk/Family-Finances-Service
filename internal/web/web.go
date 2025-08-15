package web

import (
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/handlers"
	webHandlers "family-budget-service/internal/web/handlers"
)

// Server представляет веб-сервер для HTML интерфейса
type Server struct {
	echo         *echo.Echo
	repositories *handlers.Repositories
	renderer     *TemplateRenderer

	// Handlers
	dashboardHandler *webHandlers.DashboardHandler
}

// NewWebServer создает новый веб-сервер
func NewWebServer(e *echo.Echo, repositories *handlers.Repositories, templatesDir string) (*Server, error) {
	// Создаем рендерер шаблонов
	renderer, err := NewTemplateRenderer(templatesDir)
	if err != nil {
		return nil, err
	}

	// Устанавливаем рендерер для Echo
	e.Renderer = renderer

	ws := &Server{
		echo:         e,
		repositories: repositories,
		renderer:     renderer,

		// Инициализируем handlers
		dashboardHandler: webHandlers.NewDashboardHandler(repositories),
	}

	return ws, nil
}

// SetupRoutes настраивает маршруты для веб-интерфейса
func (ws *Server) SetupRoutes() {
	// Статические файлы
	ws.echo.Static("/static", "internal/web/static")

	// Главная страница
	ws.echo.GET("/", ws.dashboardHandler.Dashboard)

	// HTMX endpoints
	htmx := ws.echo.Group("/htmx")
	htmx.GET("/dashboard/stats", ws.dashboardHandler.DashboardStats)

	// TODO: Добавить остальные маршруты
	// ws.echo.GET("/login", ws.authHandler.LoginPage)
	// ws.echo.POST("/login", ws.authHandler.Login)
	// ws.echo.GET("/register", ws.authHandler.RegisterPage)
	// ws.echo.POST("/register", ws.authHandler.Register)
	// ws.echo.POST("/logout", ws.authHandler.Logout)
}

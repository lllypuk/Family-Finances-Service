package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
)

// Paths — каталоги, которые веб-слой резолвит на диске. Оба пути
// резолвятся относительно рабочего каталога процесса, если заданы
// относительными, поэтому интеграционные тесты передают абсолютные.
type Paths struct {
	TemplatesDir string
	StaticDir    string
}

// Server представляет веб-сервер для HTML интерфейса
type Server struct {
	echo         *echo.Echo
	repositories *handlers.Repositories
	services     *services.Services
	renderer     *TemplateRenderer
	staticDir    string

	// Handlers
	dashboardHandler   *webHandlers.DashboardHandler
	authHandler        *webHandlers.AuthHandler
	userHandler        *webHandlers.UserHandler
	adminHandler       *webHandlers.AdminHandler
	categoryHandler    *webHandlers.CategoryHandler
	transactionHandler *webHandlers.TransactionHandler
	budgetHandler      *webHandlers.BudgetHandler
	reportHandler      *webHandlers.ReportHandler
	backupHandler      *webHandlers.BackupHandler
}

// NewWebServer создает новый веб-сервер
func NewWebServer(
	e *echo.Echo,
	repositories *handlers.Repositories,
	services *services.Services,
	paths Paths,
	sessionSecret string,
	cookieSecure bool,
) (*Server, error) {
	// Создаем рендерер шаблонов
	renderer, err := NewTemplateRenderer(paths.TemplatesDir)
	if err != nil {
		return nil, err
	}

	// Устанавливаем рендерер для Echo
	e.Renderer = renderer

	// Настраиваем middleware
	e.Use(middleware.SessionStore(sessionSecret, cookieSecure))
	e.Use(middleware.CSRFProtection())

	// Тот же флаг Secure на flash-cookie.
	webHandlers.SetCookieSecureForProduction(cookieSecure)

	// Настраиваем обработчик ошибок
	e.HTTPErrorHandler = customHTTPErrorHandler(renderer)

	ws := &Server{
		echo:         e,
		repositories: repositories,
		services:     services,
		renderer:     renderer,
		staticDir:    paths.StaticDir,

		// Инициализируем handlers
		dashboardHandler:   webHandlers.NewDashboardHandler(repositories, services),
		authHandler:        webHandlers.NewAuthHandler(repositories, services),
		userHandler:        webHandlers.NewUserHandler(repositories, services),
		adminHandler:       webHandlers.NewAdminHandler(repositories, services),
		categoryHandler:    webHandlers.NewCategoryHandler(repositories, services),
		transactionHandler: webHandlers.NewTransactionHandler(repositories, services),
		budgetHandler:      webHandlers.NewBudgetHandler(repositories, services),
		reportHandler:      webHandlers.NewReportHandler(repositories, services),
		backupHandler:      webHandlers.NewBackupHandler(repositories, services),
	}

	return ws, nil
}

// SetupRoutes настраивает маршруты для веб-интерфейса
func (ws *Server) SetupRoutes() {
	// Статические файлы
	ws.echo.Static("/static", ws.staticDir)

	// Setup middleware — redirects to /setup if family doesn't exist
	ws.echo.Use(middleware.RequireSetup(ws.services.Family))

	// Настраиваем маршруты аутентификации
	ws.setupAuthRoutes()

	// Защищенные маршруты (требуют аутентификации).
	// RequireActiveUser сверяет владельца подписанной cookie с БД: иначе
	// удалённый или пониженный в правах пользователь работал бы до конца
	// SessionTimeout (24 часа), и отозвать доступ было бы нечем.
	protected := ws.echo.Group("",
		middleware.RequireAuth(),
		middleware.RequireActiveUser(ws.services.User),
	)

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
	ws.echo.GET("/setup", ws.authHandler.SetupPage)
	ws.echo.POST("/setup", ws.authHandler.Setup)
	ws.echo.GET("/logout", ws.authHandler.Logout)
	ws.echo.POST("/logout", ws.authHandler.Logout)

	// Invite registration routes (public, but token-gated)
	ws.echo.GET("/invite/:token", ws.authHandler.InviteRegisterPage)
	ws.echo.POST("/invite/:token", ws.authHandler.InviteRegister)
}

// setupProtectedRoutes настраивает защищенные маршруты
func (ws *Server) setupProtectedRoutes(protected *echo.Group) {
	ws.setupUserRoutes(protected)
	ws.setupAdminRoutes(protected)
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

// setupAdminRoutes настраивает маршруты администрирования
func (ws *Server) setupAdminRoutes(protected *echo.Group) {
	admin := protected.Group("/admin", middleware.RequireAdmin())

	// User management with invites
	admin.GET("/users", ws.adminHandler.ListUsers)
	admin.POST("/users/invite", ws.adminHandler.CreateInvite)
	admin.DELETE("/users/:id", ws.adminHandler.DeleteUser)
	admin.DELETE("/invites/:id", ws.adminHandler.RevokeInvite)

	// Backup management
	admin.GET("/backup", ws.backupHandler.BackupPage)
	admin.POST("/backup/create", ws.backupHandler.CreateBackup)
	admin.GET("/backup/download/:filename", ws.backupHandler.DownloadBackup)
	admin.DELETE("/backup/:filename", ws.backupHandler.DeleteBackup)
	admin.POST("/backup/restore/:filename", ws.backupHandler.RestoreBackup)
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
	// Никакого повторного middleware.RequireAuth() здесь быть не должно: Echo
	// склеивает middleware родительской группы с middleware дочерней, поэтому
	// второй RequireAuth отработал бы ПОСЛЕ RequireActiveUser и перезаписал бы
	// в контексте свежую (прочитанную из БД) SessionData той, что лежит в
	// подписанной cookie. Ролевые проверки ниже читали бы устаревшую роль, и
	// понижение прав не действовало бы до конца SessionTimeout.
	htmx := protected.Group("/htmx")

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
			// Текст произвольной ошибки наружу не отдаём: там оказываются имена
			// шаблонов и полей структур (renderer.go оборачивает ошибку
			// исполнения) и обёрнутые ошибки репозиториев/сервисов. Клиент
			// получает обобщённую формулировку, подробности — только в лог.
			c.Logger().Errorf("unhandled error on %s %s: %v",
				c.Request().Method, c.Request().URL.Path, err)
			msg = getErrorTitle(code)
		}

		// Если ответ уже отправлен, не делаем ничего
		if c.Response().Committed {
			return
		}

		// Для HTMX запросов возвращаем простой текст
		if c.Request().Header.Get("Hx-Request") == "true" {
			_ = c.String(code, fmt.Sprintf("Error: %v", msg))
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

		// Страница ошибки собирается в буфер: запись прямо в c.Response()
		// отдавала её со статусом 200 (WriteHeader никто не вызывал), то есть
		// любые 404/500 выглядели для клиента успешными ответами.
		var buf bytes.Buffer
		if renderErr := renderer.Render(&buf, "pages/error", data, c); renderErr != nil {
			// Fallback: простой текстовый ответ
			_ = c.String(code, fmt.Sprintf("Error %d: %v", code, msg))
			return
		}

		_ = c.HTMLBlob(code, buf.Bytes())
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

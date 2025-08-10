package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HTTPServer struct {
	echo         *echo.Echo
	repositories *Repositories
	config       *Config
}

type Config struct {
	Port string
	Host string
}

func NewHTTPServer(repositories *Repositories, config *Config) *HTTPServer {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	// Timeout для всех запросов
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	server := &HTTPServer{
		echo:         e,
		repositories: repositories,
		config:       config,
	}

	server.setupRoutes()
	return server
}

func (s *HTTPServer) setupRoutes() {
	// Health check
	s.echo.GET("/health", s.healthCheck)

	// API версионирование
	api := s.echo.Group("/api/v1")

	// Маршруты для пользователей и семей
	users := api.Group("/users")
	users.POST("", s.createUser)
	users.GET("/:id", s.getUserByID)
	users.PUT("/:id", s.updateUser)
	users.DELETE("/:id", s.deleteUser)

	families := api.Group("/families")
	families.POST("", s.createFamily)
	families.GET("/:id", s.getFamilyByID)
	families.GET("/:id/members", s.getFamilyMembers)

	// Маршруты для категорий
	categories := api.Group("/categories")
	categories.POST("", s.createCategory)
	categories.GET("", s.getCategories)
	categories.GET("/:id", s.getCategoryByID)
	categories.PUT("/:id", s.updateCategory)
	categories.DELETE("/:id", s.deleteCategory)

	// Маршруты для транзакций
	transactions := api.Group("/transactions")
	transactions.POST("", s.createTransaction)
	transactions.GET("", s.getTransactions)
	transactions.GET("/:id", s.getTransactionByID)
	transactions.PUT("/:id", s.updateTransaction)
	transactions.DELETE("/:id", s.deleteTransaction)

	// Маршруты для бюджетов
	budgets := api.Group("/budgets")
	budgets.POST("", s.createBudget)
	budgets.GET("", s.getBudgets)
	budgets.GET("/:id", s.getBudgetByID)
	budgets.PUT("/:id", s.updateBudget)
	budgets.DELETE("/:id", s.deleteBudget)

	// Маршруты для отчетов
	reports := api.Group("/reports")
	reports.POST("", s.createReport)
	reports.GET("", s.getReports)
	reports.GET("/:id", s.getReportByID)
	reports.DELETE("/:id", s.deleteReport)
}

func (s *HTTPServer) Start(ctx context.Context) error {
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

// Заглушки для handlers - будут реализованы в отдельных файлах
func (s *HTTPServer) createUser(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getUserByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) updateUser(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) deleteUser(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) createFamily(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getFamilyByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getFamilyMembers(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) createCategory(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getCategories(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getCategoryByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) updateCategory(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) deleteCategory(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) createTransaction(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getTransactions(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getTransactionByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) updateTransaction(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) deleteTransaction(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) createBudget(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getBudgets(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getBudgetByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) updateBudget(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) deleteBudget(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) createReport(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getReports(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) getReportByID(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

func (s *HTTPServer) deleteReport(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, map[string]string{"message": "not implemented"})
}

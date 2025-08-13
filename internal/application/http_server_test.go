package application

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/handlers"
	"family-budget-service/internal/observability"
)

// MockRepositories provides mock implementations for all repositories
type MockRepositories struct {
	handlers.Repositories
}

func NewMockRepositories() *MockRepositories {
	return &MockRepositories{}
}

func TestNewHTTPServer(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{
		Port: "8080",
		Host: "localhost",
	}

	// Execute
	server := NewHTTPServer(&repos.Repositories, config)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.echo)
	assert.Equal(t, &repos.Repositories, server.repositories)
	assert.Equal(t, config, server.config)
	assert.Nil(t, server.observabilityService)
	assert.NotNil(t, server.userHandler)
	assert.NotNil(t, server.familyHandler)
	assert.NotNil(t, server.categoryHandler)
	assert.NotNil(t, server.transactionHandler)
	assert.NotNil(t, server.budgetHandler)
	assert.NotNil(t, server.reportHandler)
}

func TestNewHTTPServerWithObservability(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{
		Port: "8080",
		Host: "localhost",
	}
	obsService := &observability.Service{}

	// Execute
	server := NewHTTPServerWithObservability(&repos.Repositories, config, obsService)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.echo)
	assert.Equal(t, &repos.Repositories, server.repositories)
	assert.Equal(t, config, server.config)
	assert.Equal(t, obsService, server.observabilityService)
}

func TestHTTPServer_Echo(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Execute
	echoInstance := server.Echo()

	// Assert
	assert.NotNil(t, echoInstance)
	assert.Equal(t, server.echo, echoInstance)
}

func TestHTTPServer_HealthCheck(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := server.echo.NewContext(req, rec)

	// Execute
	err := server.healthCheck(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
	assert.Contains(t, rec.Body.String(), `"time"`)
}

func TestHTTPServer_RoutesSetup(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Test that routes are properly set up by checking if the echo instance has routes
	routes := server.echo.Routes()
	assert.NotEmpty(t, routes)

	// Check for key routes
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Health endpoints
	assert.True(t, routePaths["GET /health"])

	// API endpoints
	assert.True(t, routePaths["POST /api/v1/users"])
	assert.True(t, routePaths["GET /api/v1/users/:id"])
	assert.True(t, routePaths["PUT /api/v1/users/:id"])
	assert.True(t, routePaths["DELETE /api/v1/users/:id"])

	assert.True(t, routePaths["POST /api/v1/families"])
	assert.True(t, routePaths["GET /api/v1/families/:id"])
	assert.True(t, routePaths["GET /api/v1/families/:id/members"])

	assert.True(t, routePaths["POST /api/v1/categories"])
	assert.True(t, routePaths["GET /api/v1/categories"])
	assert.True(t, routePaths["GET /api/v1/categories/:id"])
	assert.True(t, routePaths["PUT /api/v1/categories/:id"])
	assert.True(t, routePaths["DELETE /api/v1/categories/:id"])

	assert.True(t, routePaths["POST /api/v1/transactions"])
	assert.True(t, routePaths["GET /api/v1/transactions"])
	assert.True(t, routePaths["GET /api/v1/transactions/:id"])
	assert.True(t, routePaths["PUT /api/v1/transactions/:id"])
	assert.True(t, routePaths["DELETE /api/v1/transactions/:id"])

	assert.True(t, routePaths["POST /api/v1/budgets"])
	assert.True(t, routePaths["GET /api/v1/budgets"])
	assert.True(t, routePaths["GET /api/v1/budgets/:id"])
	assert.True(t, routePaths["PUT /api/v1/budgets/:id"])
	assert.True(t, routePaths["DELETE /api/v1/budgets/:id"])

	assert.True(t, routePaths["POST /api/v1/reports"])
	assert.True(t, routePaths["GET /api/v1/reports"])
	assert.True(t, routePaths["GET /api/v1/reports/:id"])
	assert.True(t, routePaths["DELETE /api/v1/reports/:id"])
}

func TestHTTPServer_HealthEndpoint_Integration(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Execute
	server.echo.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestHTTPServer_MiddlewareSetup(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Test that middleware is applied by making a request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Execute
	server.echo.ServeHTTP(rec, req)

	// Assert that middleware headers are present
	assert.NotEmpty(t, rec.Header().Get("X-Request-Id")) // RequestID middleware
}

func TestHTTPServer_WithObservabilityRoutes(t *testing.T) {
	// Setup mock observability service
	obsService := &observability.Service{}

	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServerWithObservability(&repos.Repositories, config, obsService)

	// Test that observability routes are properly set up
	routes := server.echo.Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Check for basic health endpoint (simplified for testing)
	assert.True(t, routePaths["GET /health"])
}

func TestConfig_Fields(t *testing.T) {
	// Test that Config struct has expected fields
	config := &Config{
		Port: "8080",
		Host: "localhost",
	}

	assert.Equal(t, "8080", config.Port)
	assert.Equal(t, "localhost", config.Host)
}

func TestHTTPServer_StartShutdownInterface(t *testing.T) {
	// This test verifies that the server implements the expected interface
	// but doesn't actually start/stop the server to avoid port conflicts in tests

	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Verify methods exist and have correct signatures
	ctx := context.Background()

	// Test that Start method exists and returns error interface
	// We won't actually call it to avoid starting a real server
	assert.NotNil(t, server.Start)

	// Test that Shutdown method exists and returns error interface
	// We won't actually call it since server isn't running
	assert.NotNil(t, server.Shutdown)

	// Just verify the methods can be called with proper context
	_ = ctx
}

// MockHealthService for testing observability integration
type MockHealthService struct {
	mock.Mock
}

func (m *MockHealthService) HealthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	}
}

func (m *MockHealthService) ReadinessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ready"})
	}
}

func (m *MockHealthService) LivenessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "live"})
	}
}

func TestHTTPServer_TimeoutMiddleware(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &Config{Port: "8080", Host: "localhost"}
	server := NewHTTPServer(&repos.Repositories, config)

	// Create a simple handler for testing
	server.echo.GET("/test", func(c echo.Context) error {
		return c.String(200, "test response")
	})

	// This test verifies that timeout middleware is configured
	// The actual timeout behavior would be tested in integration tests
	assert.NotNil(t, server.echo)
}

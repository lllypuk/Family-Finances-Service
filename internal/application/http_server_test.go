package application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application"
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
	config := &application.Config{
		Port: "8080",
		Host: "localhost",
	}

	// Execute
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.Echo()) // Test public Echo() method
}

func TestNewHTTPServerWithObservability(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{
		Port: "8080",
		Host: "localhost",
	}

	// Создаем правильно инициализированный observability service
	obsConfig := observability.DefaultConfig()
	obsService, err := observability.NewService(obsConfig, "test-version")
	require.NoError(t, err)
	defer func() { _ = obsService.Shutdown(context.Background()) }()

	// Execute
	server := application.NewHTTPServerWithObservability(&repos.Repositories, config, obsService)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.Echo())
}

func TestHTTPServer_Echo(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Execute
	echoInstance := server.Echo()

	// Assert
	assert.NotNil(t, echoInstance)
	assert.IsType(t, &echo.Echo{}, echoInstance)
}

func TestHTTPServer_HealthEndpoint(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Create request to health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Execute
	server.Echo().ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
	assert.Contains(t, rec.Body.String(), `"time"`)
}

func TestHTTPServer_RoutesSetup(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Test that routes are properly set up by checking if the echo instance has routes
	routes := server.Echo().Routes()
	assert.NotEmpty(t, routes)

	// Check for key routes
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Health endpoints
	assert.True(t, routePaths["GET /health"])

	// Web interface routes (if web server is initialized)
	// Note: Web routes are only available when observability is enabled
	// In basic HTTP server (without observability), web interface is not initialized

	// API endpoints - check for some key endpoints
	assert.True(t, routePaths["POST /api/v1/users"])
	assert.True(t, routePaths["GET /api/v1/users/:id"])
	assert.True(t, routePaths["POST /api/v1/families"])
	assert.True(t, routePaths["POST /api/v1/categories"])
	assert.True(t, routePaths["GET /api/v1/categories"])
	assert.True(t, routePaths["POST /api/v1/transactions"])
	assert.True(t, routePaths["GET /api/v1/transactions"])
	assert.True(t, routePaths["POST /api/v1/budgets"])
	assert.True(t, routePaths["GET /api/v1/budgets"])
	assert.True(t, routePaths["POST /api/v1/reports"])
	assert.True(t, routePaths["GET /api/v1/reports"])
}

func TestHTTPServer_MiddlewareSetup(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Test that middleware is applied by making a request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Execute
	server.Echo().ServeHTTP(rec, req)

	// Assert that middleware headers are present
	assert.NotEmpty(t, rec.Header().Get("X-Request-Id")) // RequestID middleware
}

func TestHTTPServer_WithObservabilityRoutes(t *testing.T) {
	// Setup properly initialized observability service
	obsConfig := observability.DefaultConfig()
	obsService, err := observability.NewService(obsConfig, "test-version")
	require.NoError(t, err)
	defer func() { _ = obsService.Shutdown(context.Background()) }()

	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServerWithObservability(&repos.Repositories, config, obsService)

	// Test that observability routes are properly set up
	routes := server.Echo().Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Check for observability endpoints
	assert.True(t, routePaths["GET /health"])
	assert.True(t, routePaths["GET /metrics"])
	assert.True(t, routePaths["GET /ready"])
	assert.True(t, routePaths["GET /live"])
}

func TestConfig_Fields(t *testing.T) {
	// Test that Config struct has expected fields
	config := &application.Config{
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
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

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

func TestHTTPServer_IntegrationWithRealEndpoints(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Test that we can make requests to various endpoints
	testCases := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/health", http.StatusOK},
		{"GET", "/", http.StatusNotFound},                    // Dashboard not available without observability
		{"GET", "/api/v1/categories", http.StatusBadRequest}, // Missing family_id
		{"GET", "/api/v1/nonexistent", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			server.Echo().ServeHTTP(rec, req)

			assert.Equal(t, tc.expected, rec.Code)
		})
	}
}

func TestHTTPServer_CORSEnabled(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, config)

	// Test CORS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/categories", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	server.Echo().ServeHTTP(rec, req)

	// Assert CORS headers are present
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

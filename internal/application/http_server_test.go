package application_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// MockRepositories provides mock implementations for all repositories
type MockRepositories struct {
	handlers.Repositories
}

func NewMockRepositories() *MockRepositories {
	return &MockRepositories{}
}

// MockUserService provides mock implementation for UserService
type MockUserService struct {
	mock.Mock
}

// Implement the UserService interface methods

func (m *MockUserService) CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUsers(ctx context.Context) ([]*user.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) ChangeUserRole(ctx context.Context, id uuid.UUID, newRole user.Role) error {
	args := m.Called(ctx, id, newRole)
	return args.Error(0)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) ValidateUserAccess(ctx context.Context, userID uuid.UUID, targetFamilyID uuid.UUID) error {
	args := m.Called(ctx, userID, targetFamilyID)
	return args.Error(0)
}

// MockFamilyService provides mock implementation for FamilyService
type MockFamilyService struct {
	mock.Mock
}

func (m *MockFamilyService) SetupFamily(ctx context.Context, req dto.SetupFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) GetFamily(ctx context.Context) (*user.Family, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) UpdateFamily(ctx context.Context, req dto.UpdateFamilyDTO) (*user.Family, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyService) IsSetupComplete(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

// MockCategoryService is a mock for category service
type MockCategoryService struct {
	mock.Mock
}

//nolint:revive // test mock
func (m *MockCategoryService) CreateCategory(
	ctx context.Context,
	req dto.CreateCategoryDTO,
) (*category.Category, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockCategoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	return nil, nil //nolint:nilnil // test mock
}

func (m *MockCategoryService) GetCategories(
	ctx context.Context,
	typeFilter *category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

//nolint:revive // test mock
func (m *MockCategoryService) UpdateCategory(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateCategoryDTO,
) (*category.Category, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockCategoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockCategoryService) GetCategoryHierarchy(ctx context.Context) ([]*category.Category, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockCategoryService) ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockCategoryService) CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	return false, nil
}

//nolint:revive // test mock
func (m *MockCategoryService) CreateDefaultCategories(ctx context.Context) error {
	return nil
}

// MockTransactionService is a mock for transaction service
type MockTransactionService struct {
	mock.Mock
}

//nolint:revive // test mock
func (m *MockTransactionService) CreateTransaction(
	ctx context.Context,
	req dto.CreateTransactionDTO,
) (*transaction.Transaction, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockTransactionService) GetTransactionByID(
	ctx context.Context,
	id uuid.UUID,
) (*transaction.Transaction, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockTransactionService) GetTransactions(
	ctx context.Context,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockTransactionService) UpdateTransaction(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateTransactionDTO,
) (*transaction.Transaction, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockTransactionService) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockTransactionService) BulkCategorizeTransactions(
	ctx context.Context,
	transactionIDs []uuid.UUID,
	categoryID uuid.UUID,
) error {
	return nil
}

//nolint:revive // test mock
func (m *MockTransactionService) ValidateTransactionLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
) error {
	return nil
}

//nolint:revive // test mock
func (m *MockTransactionService) GetAllTransactions(
	ctx context.Context,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockTransactionService) GetTransactionsByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockTransactionService) GetTransactionsByDateRange(
	ctx context.Context,
	from, to time.Time,
) ([]*transaction.Transaction, error) {
	return nil, nil
}

// MockBudgetService is a mock for budget service
type MockBudgetService struct {
	mock.Mock
}

//nolint:revive // test mock
func (m *MockBudgetService) CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockBudgetService) GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockBudgetService) GetBudgets(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockBudgetService) UpdateBudget(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateBudgetDTO,
) (*budget.Budget, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockBudgetService) DeleteBudget(ctx context.Context, id uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockBudgetService) CalculateBudgetUtilization(
	ctx context.Context,
	budgetID uuid.UUID,
) (*dto.BudgetUtilizationDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockBudgetService) GetBudgetsByCategory(ctx context.Context, categoryID uuid.UUID) ([]*budget.Budget, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockBudgetService) ValidateBudgetPeriod(
	ctx context.Context,
	categoryID *uuid.UUID,
	startDate, endDate time.Time,
) error {
	return nil
}

//nolint:revive // test mock
func (m *MockBudgetService) CheckBudgetLimits(ctx context.Context, categoryID uuid.UUID, amount float64) error {
	return nil
}

//nolint:revive // test mock
func (m *MockBudgetService) GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockBudgetService) GetActiveBudgets(ctx context.Context, date time.Time) ([]*budget.Budget, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockBudgetService) UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error {
	return nil
}

//nolint:revive // test mock
func (m *MockBudgetService) GetAllBudgets(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockBudgetService) RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error {
	return nil
}

// MockReportService is a mock for report service
type MockReportService struct {
	mock.Mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateExpenseReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.ExpenseReportDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateIncomeReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.IncomeReportDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateBudgetComparisonReport(
	ctx context.Context,
	period report.Period,
) (*dto.BudgetComparisonDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateCashFlowReport(
	ctx context.Context,
	from, to time.Time,
) (*dto.CashFlowReportDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) SaveReport(
	ctx context.Context,
	reportData any,
	reportType report.Type,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GetReportByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GetReports(ctx context.Context, typeFilter *report.Type) ([]*report.Report, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) DeleteReport(ctx context.Context, id uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockReportService) ExportReport(
	ctx context.Context,
	reportID uuid.UUID,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) ExportReportData(
	ctx context.Context,
	reportData any,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) ScheduleReport(
	ctx context.Context,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GetScheduledReports(ctx context.Context) ([]*dto.ScheduledReportDTO, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) UpdateScheduledReport(
	ctx context.Context,
	id uuid.UUID,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) DeleteScheduledReport(ctx context.Context, id uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockReportService) ExecuteScheduledReport(ctx context.Context, scheduledReportID uuid.UUID) error {
	return nil
}

//nolint:revive // test mock
func (m *MockReportService) CalculateBenchmarks(ctx context.Context) (*dto.BenchmarkComparisonDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

//nolint:revive // test mock
func (m *MockReportService) GenerateFinancialInsights(ctx context.Context) ([]dto.RecommendationDTO, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) GenerateSpendingForecast(ctx context.Context, months int) ([]dto.ForecastDTO, error) {
	return nil, nil
}

//nolint:revive // test mock
func (m *MockReportService) GenerateTrendAnalysis(
	ctx context.Context,
	categoryID *uuid.UUID,
	period report.Period,
) (*dto.TrendAnalysisDTO, error) {
	return nil, nil //nolint:nilnil // test mock
}

// Create a services struct that satisfies the services.Services interface
func NewMockServices() *services.Services {
	return &services.Services{
		User:        &MockUserService{},
		Family:      &MockFamilyService{},
		Category:    &MockCategoryService{},
		Transaction: &MockTransactionService{},
		Budget:      &MockBudgetService{},
		Report:      &MockReportService{},
	}
}

func TestNewHTTPServer(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	mockServices := NewMockServices()
	config := &application.Config{
		Port: "8080",
		Host: "localhost",
	}

	// Execute
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.Echo()) // Test public Echo() method
}

func TestNewHTTPServerWithObservability(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	mockServices := NewMockServices()
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
	server := application.NewHTTPServerWithObservability(&repos.Repositories, mockServices, config, obsService)

	// Assert
	assert.NotNil(t, server)
	assert.NotNil(t, server.Echo())
}

func TestHTTPServer_Echo(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	mockServices := NewMockServices()
	config := &application.Config{Port: "8080", Host: "localhost"}
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

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
	mockServices := NewMockServices()
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

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
	mockServices := NewMockServices()
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

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
	mockServices := NewMockServices()
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

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
	mockServices := NewMockServices()
	server := application.NewHTTPServerWithObservability(&repos.Repositories, mockServices, config, obsService)

	// Test that observability routes are properly set up
	routes := server.Echo().Routes()
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Method+" "+route.Path] = true
	}

	// Check for health check endpoints
	assert.True(t, routePaths["GET /health"])
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
	mockServices := NewMockServices()
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

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

	// Setup mock category service to return empty list
	mockCategoryService := &MockCategoryService{}
	mockCategoryService.On("GetCategories", mock.Anything, (*category.Type)(nil)).
		Return([]*category.Category{}, nil)

	mockServices := &services.Services{
		User:        &MockUserService{},
		Family:      &MockFamilyService{},
		Category:    mockCategoryService,
		Transaction: &MockTransactionService{},
		Budget:      &MockBudgetService{},
		Report:      &MockReportService{},
	}
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

	// Test that we can make requests to various endpoints
	testCases := []struct {
		method   string
		path     string
		expected int
	}{
		{"GET", "/health", http.StatusOK},
		{"GET", "/", http.StatusNotFound},            // Dashboard not available without observability
		{"GET", "/api/v1/categories", http.StatusOK}, // Returns empty list in single-family model
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
	mockServices := NewMockServices()
	server := application.NewHTTPServer(&repos.Repositories, mockServices, config)

	// Test CORS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/categories", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	server.Echo().ServeHTTP(rec, req)

	// Assert CORS headers are present
	assert.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

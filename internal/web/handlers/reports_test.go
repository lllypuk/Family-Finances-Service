package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
)

// MockReportService is a mock implementation of ReportService
type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) GenerateExpenseReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.ExpenseReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ExpenseReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateIncomeReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.IncomeReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.IncomeReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateBudgetComparisonReport(
	ctx context.Context,
	period report.Period,
) (*dto.BudgetComparisonDTO, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetComparisonDTO), args.Error(1)
}

func (m *MockReportService) GenerateCashFlowReport(
	ctx context.Context,
	from, to time.Time,
) (*dto.CashFlowReportDTO, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CashFlowReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CategoryBreakdownDTO), args.Error(1)
}

func (m *MockReportService) SaveReport(
	ctx context.Context,
	reportData any,
	reportType report.Type,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	args := m.Called(ctx, reportData, reportType, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) GetReportByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) GetReports(
	ctx context.Context,
	typeFilter *report.Type,
) ([]*report.Report, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportService) DeleteReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportService) ExportReport(
	ctx context.Context,
	reportID uuid.UUID,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportID, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) ExportReportData(
	ctx context.Context,
	reportData any,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportData, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) ScheduleReport(
	ctx context.Context,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) GetScheduledReports(ctx context.Context) ([]*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) UpdateScheduledReport(
	ctx context.Context,
	id uuid.UUID,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) DeleteScheduledReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportService) CancelScheduledReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportService) ExecuteScheduledReport(ctx context.Context, scheduledReportID uuid.UUID) error {
	args := m.Called(ctx, scheduledReportID)
	return args.Error(0)
}

func (m *MockReportService) GenerateTrendAnalysis(
	ctx context.Context,
	categoryID *uuid.UUID,
	period report.Period,
) (*dto.TrendAnalysisDTO, error) {
	args := m.Called(ctx, categoryID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TrendAnalysisDTO), args.Error(1)
}

func (m *MockReportService) GenerateSpendingForecast(
	ctx context.Context,
	months int,
) ([]dto.ForecastDTO, error) {
	args := m.Called(ctx, months)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ForecastDTO), args.Error(1)
}

func (m *MockReportService) GenerateFinancialInsights(ctx context.Context) ([]dto.RecommendationDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.RecommendationDTO), args.Error(1)
}

func (m *MockReportService) CalculateBenchmarks(ctx context.Context) (*dto.BenchmarkComparisonDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BenchmarkComparisonDTO), args.Error(1)
}

// mockRenderer is a simple test renderer
type mockRenderer struct{}

func (r *mockRenderer) Render(_ io.Writer, _ string, _ interface{}, _ echo.Context) error {
	return nil
}

const testReportFormData = "type=expenses&period=monthly&start_date=2025-01-01&end_date=2025-01-31&name=Test+Report"

// setupReportHandlerTest creates a test context with session data for report handler tests
func setupReportHandlerTest(htmx bool) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Renderer = &mockRenderer{}

	req := httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(testReportFormData))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	if htmx {
		req.Header.Set("Hx-Request", "true")
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session data to context using "user" key (used by GetUserFromContext)
	sessionData := &middleware.SessionData{
		UserID: uuid.New(),
		Role:   user.RoleAdmin,
	}
	c.Set("user", sessionData)

	return c, rec
}

// TestReportHandler_Create_HTMXGenerationError tests that HTMX error response
// is handled correctly and doesn't cause nil pointer dereference (BUG-001)
func TestReportHandler_Create_HTMXGenerationError(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate error during report generation
	mockReportService.On("GenerateExpenseReport", mock.Anything, mock.Anything).
		Return(nil, errors.New("failed to get transactions"))

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := NewReportHandler(nil, svc)

	// Create test context with HTMX header and valid form data
	// Using testReportFormData constant
	c, rec := setupReportHandlerTest(true)

	// Execute handler - this should NOT panic
	err := handler.Create(c)

	// Verify no error returned (HTMX response was rendered)
	require.NoError(t, err)

	// Verify mock was called
	mockReportService.AssertCalled(t, "GenerateExpenseReport", mock.Anything, mock.Anything)

	// Verify response status (should be 200 OK with rendered error template)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestReportHandler_Create_Success tests successful report creation
func TestReportHandler_Create_Success(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	expenseReport := &dto.ExpenseReportDTO{
		ID:            uuid.New(),
		Name:          "Test Report",
		TotalExpenses: 1000.0,
	}
	mockReportService.On("GenerateExpenseReport", mock.Anything, mock.Anything).
		Return(expenseReport, nil)

	// Simulate successful save
	savedReport := &report.Report{
		ID:   uuid.New(),
		Name: "Test Report",
		Type: report.TypeExpenses,
	}
	mockReportService.On("SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything).
		Return(savedReport, nil)

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := NewReportHandler(nil, svc)

	// Create test context with valid form data (non-HTMX)
	// Using testReportFormData constant
	c, rec := setupReportHandlerTest(false)

	// Execute handler
	err := handler.Create(c)

	// Verify redirect response
	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/reports/")

	// Verify mocks were called
	mockReportService.AssertCalled(t, "GenerateExpenseReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything)
}

// TestReportHandler_Create_HTMXSuccess tests successful report creation with HTMX
func TestReportHandler_Create_HTMXSuccess(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	expenseReport := &dto.ExpenseReportDTO{
		ID:            uuid.New(),
		Name:          "Test Report",
		TotalExpenses: 1000.0,
	}
	mockReportService.On("GenerateExpenseReport", mock.Anything, mock.Anything).
		Return(expenseReport, nil)

	// Simulate successful save
	savedReport := &report.Report{
		ID:   uuid.New(),
		Name: "Test Report",
		Type: report.TypeExpenses,
	}
	mockReportService.On("SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything).
		Return(savedReport, nil)

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := NewReportHandler(nil, svc)

	// Create test context with HTMX header and valid form data
	// Using testReportFormData constant
	c, rec := setupReportHandlerTest(true)

	// Execute handler
	err := handler.Create(c)

	// Verify response with HTMX redirect header
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Hx-Redirect"), "/reports/")

	// Verify mocks were called
	mockReportService.AssertCalled(t, "GenerateExpenseReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything)
}

// TestReportHandler_Create_SaveError tests error handling during report save
func TestReportHandler_Create_SaveError(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	expenseReport := &dto.ExpenseReportDTO{
		ID:            uuid.New(),
		Name:          "Test Report",
		TotalExpenses: 1000.0,
	}
	mockReportService.On("GenerateExpenseReport", mock.Anything, mock.Anything).
		Return(expenseReport, nil)

	// Simulate save error
	mockReportService.On("SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything).
		Return(nil, errors.New("database error"))

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := NewReportHandler(nil, svc)

	// Create test context with HTMX header and valid form data
	// Using testReportFormData constant
	c, _ := setupReportHandlerTest(true)

	// Execute handler - save errors are propagated as-is
	err := handler.Create(c)

	// SaveReport errors are not wrapped in HTMX response, they're returned directly
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	// Verify mocks were called
	mockReportService.AssertCalled(t, "GenerateExpenseReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, expenseReport, report.TypeExpenses, mock.Anything)
}

// TestErrHTMXResponseSent verifies that errHTMXResponseSent is properly defined
func TestErrHTMXResponseSent(t *testing.T) {
	// Verify the sentinel error exists and has correct message
	require.Error(t, errHTMXResponseSent)
	assert.Equal(t, "HTMX response already sent", errHTMXResponseSent.Error())
}

package handlers_test

import (
	"context"
	"errors"
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
	"family-budget-service/internal/web/handlers"
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

func (m *MockReportService) GenerateReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) SaveReport(ctx context.Context, reportEntity *report.Report) error {
	args := m.Called(ctx, reportEntity)
	return args.Error(0)
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

func (m *MockReportService) GetReportsByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, userID)
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

func (m *MockReportService) GenerateFinancialInsights(ctx context.Context) ([]dto.RecommendationDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.RecommendationDTO), args.Error(1)
}

const testReportFormData = "type=expenses&period=monthly&start_date=2025-01-01&end_date=2025-01-31&name=Test+Report"

// setupReportHandlerTest creates a test context with session data for report handler tests
func setupReportHandlerTest(htmx bool) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Renderer = &MockRenderer{}

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
	mockReportService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(nil, errors.New("failed to get transactions"))

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := handlers.NewReportHandler(nil, svc, false)

	// Create test context with HTMX header and valid form data
	// Using testReportFormData constant
	c, rec := setupReportHandlerTest(true)

	// Execute handler - this should NOT panic
	err := handler.Create(c)

	// Verify no error returned (HTMX response was rendered)
	require.NoError(t, err)

	// Verify mock was called
	mockReportService.AssertCalled(t, "GenerateReport", mock.Anything, mock.Anything)

	// Verify response status (should be 200 OK with rendered error template)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestReportHandler_Create_Success tests successful report creation
func TestReportHandler_Create_Success(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	generatedReport := &report.Report{
		ID:   uuid.New(),
		Name: "Test Report",
		Type: report.TypeExpenses,
	}
	mockReportService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(generatedReport, nil)
	mockReportService.On("SaveReport", mock.Anything, generatedReport).Return(nil)

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := handlers.NewReportHandler(nil, svc, false)

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
	mockReportService.AssertCalled(t, "GenerateReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, generatedReport)
}

// TestReportHandler_Create_HTMXSuccess tests successful report creation with HTMX
func TestReportHandler_Create_HTMXSuccess(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	generatedReport := &report.Report{
		ID:   uuid.New(),
		Name: "Test Report",
		Type: report.TypeExpenses,
	}
	mockReportService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(generatedReport, nil)
	mockReportService.On("SaveReport", mock.Anything, generatedReport).Return(nil)

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := handlers.NewReportHandler(nil, svc, false)

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
	mockReportService.AssertCalled(t, "GenerateReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, generatedReport)
}

// TestReportHandler_Create_SaveError tests error handling during report save
func TestReportHandler_Create_SaveError(t *testing.T) {
	// Setup mock service
	mockReportService := new(MockReportService)

	// Simulate successful report generation
	generatedReport := &report.Report{
		ID:   uuid.New(),
		Name: "Test Report",
		Type: report.TypeExpenses,
	}
	mockReportService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(generatedReport, nil)

	// Simulate save error
	mockReportService.On("SaveReport", mock.Anything, generatedReport).
		Return(errors.New("database error"))

	// Create services container with mock
	svc := &services.Services{
		Report: mockReportService,
	}

	// Create handler
	handler := handlers.NewReportHandler(nil, svc, false)

	// Create test context with HTMX header and valid form data
	// Using testReportFormData constant
	c, rec := setupReportHandlerTest(true)

	// Execute handler
	err := handler.Create(c)

	// Ошибка сохранения идёт тем же путём, что и ошибка генерации: HTMX-партиал с текстом ошибки
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify mocks were called
	mockReportService.AssertCalled(t, "GenerateReport", mock.Anything, mock.Anything)
	mockReportService.AssertCalled(t, "SaveReport", mock.Anything, generatedReport)
}

// setupReportHandlerWithUser собирает обработчик отчётов с моками сервисов
// отчётов и пользователей: buildPageData дочитывает имя и фамилию владельца
// сессии через UserService, поэтому без него страницы отчётов не собираются.
func setupReportHandlerWithUser() (*handlers.ReportHandler, *MockReportService, *MockUserService) {
	mockReportService := new(MockReportService)
	mockUserService := new(MockUserService)

	svc := &services.Services{
		Report: mockReportService,
		User:   mockUserService,
	}

	return handlers.NewReportHandler(nil, svc, false), mockReportService, mockUserService
}

// TestReportHandler_PageDataContract проверяет контракт данных страниц отчётов:
// шаблон шапки читает `.CurrentUser` из **корня** контекста
// (`{{if .CurrentUser}}`), поэтому *PageData обязан быть встроен, а не лежать
// в map под ключом "PageData". Регрессия U-02
// (docs/specs/003-ui-ux-audit.md#u-02) возникла именно из-за вложенности
// на уровень глубже. Заодно фиксируются русские заголовки (U-05).
func TestReportHandler_PageDataContract(t *testing.T) {
	userID := uuid.New()
	testReport := &report.Report{
		ID:     uuid.New(),
		Name:   "Февраль",
		Type:   report.TypeExpenses,
		Period: report.PeriodMonthly,
		UserID: userID,
	}

	newHandler := func(t *testing.T) *handlers.ReportHandler {
		t.Helper()

		handler, mockReportService, mockUserService := setupReportHandlerWithUser()

		mockReportService.On("GetReports", mock.Anything, mock.Anything).
			Return([]*report.Report{testReport}, nil).Maybe()
		mockReportService.On("GetReportByID", mock.Anything, testReport.ID).
			Return(testReport, nil).Maybe()
		mockUserService.On("GetUserByID", mock.Anything, userID).
			Return(&user.User{ID: userID, Email: "test@example.com", FirstName: "John", LastName: "Doe"}, nil)

		return handler
	}

	t.Run("Index", func(t *testing.T) {
		handler := newHandler(t)

		c, renderer := newCapturingContext(http.MethodGet, "/reports")
		withSession(c, userID, user.RoleAdmin)

		require.NoError(t, handler.Index(c))

		out, err := renderer.renderPageData()
		require.NoError(t, err, "данные хендлера не дают шаблону .CurrentUser в корне контекста")
		assert.Equal(t, "John Doe|admin|Отчёты", out)
	})

	t.Run("New", func(t *testing.T) {
		handler := newHandler(t)

		c, renderer := newCapturingContext(http.MethodGet, "/reports/new")
		withSession(c, userID, user.RoleAdmin)

		require.NoError(t, handler.New(c))

		out, err := renderer.renderPageData()
		require.NoError(t, err)
		assert.Equal(t, "John Doe|admin|Новый отчёт", out)
	})

	t.Run("Show", func(t *testing.T) {
		handler := newHandler(t)

		c, renderer := newCapturingContext(http.MethodGet, "/reports/"+testReport.ID.String())
		c.SetParamNames("id")
		c.SetParamValues(testReport.ID.String())
		withSession(c, userID, user.RoleAdmin)

		require.NoError(t, handler.Show(c))

		out, err := renderer.renderPageData()
		require.NoError(t, err)
		assert.Equal(t, "John Doe|admin|Отчёт: Февраль", out)
	})

	t.Run("FormWithErrors", func(t *testing.T) {
		handler := newHandler(t)

		// Пустое тело — форма не проходит валидацию, хендлер перерисовывает
		// pages/reports/new с ошибками. Шапка обязана остаться на месте.
		c, renderer := newCapturingContext(http.MethodPost, "/reports")
		withSession(c, userID, user.RoleAdmin)

		// Форма невалидна: хендлер обязан перерисовать её, а не свалиться в 5xx.
		createErr := handler.Create(c)
		if createErr != nil {
			var httpErr *echo.HTTPError
			require.ErrorAs(t, createErr, &httpErr)
			assert.Less(t, httpErr.Code, http.StatusInternalServerError,
				"перерисовка формы с ошибками не должна давать 5xx")
		}

		out, err := renderer.renderPageData()
		require.NoError(t, err)
		assert.Equal(t, "John Doe|admin|Новый отчёт", out)
	})
}

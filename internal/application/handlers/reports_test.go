package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// MockReportRepository is a mock implementation of report repository
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) Create(ctx context.Context, rpt *report.Report) error {
	args := m.Called(ctx, rpt)
	return args.Error(0)
}

func (m *MockReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportRepository) GetAll(ctx context.Context) ([]*report.Report, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// setupReportHandler creates a new report handler with mock repositories
func setupReportHandler() (*handlers.ReportHandler, *MockReportRepository) {
	mockRepo := &MockReportRepository{}
	repositories := &handlers.Repositories{
		Report: mockRepo,
	}
	handler := handlers.NewReportHandler(repositories)
	return handler, mockRepo
}

// setupReportHandlerWithService creates a handler backed by a mocked report service
func setupReportHandlerWithService() (*handlers.ReportHandler, *MockReportService) {
	mockService := &MockReportService{}
	handler := handlers.NewReportHandler(&handlers.Repositories{}, mockService)
	return handler, mockService
}

// generatedTestReport is the report a mocked GenerateReport returns
func generatedTestReport(userID uuid.UUID) *report.Report {
	generated := report.NewReport(
		"Monthly Expenses Report",
		report.TypeExpenses,
		report.PeriodMonthly,
		userID,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
	)
	generated.Data = report.Data{TotalExpenses: 100}
	return generated
}

// postReportRequest builds a POST /reports context with the given body
func postReportRequest(t *testing.T, body any) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(raw))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	return e.NewContext(httpReq, rec), rec
}

// createValidReportRequest creates a valid report request for testing
func createValidReportRequest() handlers.CreateReportRequest {
	return handlers.CreateReportRequest{
		Name:      "Monthly Expenses Report",
		Type:      "expenses",
		Period:    "monthly",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
	}
}

func TestReportHandler_CreateReport_Success(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	userID := uuid.New()
	generated := generatedTestReport(userID)

	mockService.On("GenerateReport", mock.Anything, mock.MatchedBy(func(req dto.ReportRequestDTO) bool {
		return req.UserID == userID && req.Type == report.TypeExpenses
	})).Return(generated, nil)
	mockService.On("SaveReport", mock.Anything, generated).Return(nil)

	c, rec := postReportRequest(t, createValidReportRequest())
	withSessionUser(c, userID)

	require.NoError(t, handler.CreateReport(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response handlers.APIResponse[handlers.ReportResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, generated.ID, response.Data.ID)
	assert.Equal(t, userID, response.Data.UserID)
	assert.Equal(t, "expenses", response.Data.Type)

	mockService.AssertExpectations(t)
}

// TestReportHandler_CreateReport_NoSession — автор отчёта берётся только из сессии,
// поэтому без неё роут отвечает 401, не читая тело (S-01).
func TestReportHandler_CreateReport_NoSession(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	c, rec := postReportRequest(t, createValidReportRequest())

	require.NoError(t, handler.CreateReport(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "UNAUTHORIZED", response.Error.Code)

	mockService.AssertNotCalled(t, "GenerateReport", mock.Anything, mock.Anything)
}

func TestReportHandler_CreateReport_UnsupportedType(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	mockService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(nil, services.ErrUnsupportedReportType)

	c, rec := postReportRequest(t, createValidReportRequest())
	withSessionUser(c, uuid.New())

	require.NoError(t, handler.CreateReport(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INVALID_REQUEST", response.Error.Code)

	mockService.AssertNotCalled(t, "SaveReport", mock.Anything, mock.Anything)
}

func TestReportHandler_CreateReport_GenerationError(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	mockService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(nil, errors.New("repository failure"))

	c, rec := postReportRequest(t, createValidReportRequest())
	withSessionUser(c, uuid.New())

	require.NoError(t, handler.CreateReport(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "GENERATION_FAILED", response.Error.Code)
}

func TestReportHandler_CreateReport_SaveError(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	userID := uuid.New()
	mockService.On("GenerateReport", mock.Anything, mock.Anything).
		Return(generatedTestReport(userID), nil)
	mockService.On("SaveReport", mock.Anything, mock.Anything).
		Return(errors.New("repository failure"))

	c, rec := postReportRequest(t, createValidReportRequest())
	withSessionUser(c, userID)

	require.NoError(t, handler.CreateReport(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "SAVE_FAILED", response.Error.Code)

	mockService.AssertExpectations(t)
}

func TestReportHandler_ExportReport_Success(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	reportID := uuid.New()
	stored := generatedTestReport(uuid.New())
	stored.ID = reportID
	csvBody := []byte("Category,Amount,Percentage,Transaction Count\n")

	mockService.On("GetReportByID", mock.Anything, reportID).Return(stored, nil)
	mockService.On("ExportReport", mock.Anything, reportID, "csv", dto.ExportOptionsDTO{}).
		Return(csvBody, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String()+"/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	require.NoError(t, handler.ExportReport(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "text/csv")
	assert.Contains(t, rec.Header().Get(echo.HeaderContentDisposition), "report-"+reportID.String()+".csv")
	assert.Equal(t, string(csvBody), rec.Body.String())

	mockService.AssertExpectations(t)
}

func TestReportHandler_ExportReport_NotFound(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	reportID := uuid.New()
	mockService.On("GetReportByID", mock.Anything, reportID).Return(nil, errors.New("report not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String()+"/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	require.NoError(t, handler.ExportReport(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	mockService.AssertNotCalled(t, "ExportReport", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReportHandler_ExportReport_InvalidID(t *testing.T) {
	handler, mockService := setupReportHandlerWithService()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/not-a-uuid/export", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	require.NoError(t, handler.ExportReport(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INVALID_ID", response.Error.Code)

	mockService.AssertNotCalled(t, "GetReportByID", mock.Anything, mock.Anything)
}

func TestReportHandler_CreateReport_InvalidRequest(t *testing.T) {
	handler, _ := setupReportHandler()

	tests := []struct {
		name        string
		requestBody any
		expectedMsg string
	}{
		{
			name:        "Invalid JSON",
			requestBody: "invalid json",
			expectedMsg: "Invalid request body",
		},
		{
			name: "Missing name",
			requestBody: map[string]any{
				"type":       "expenses",
				"period":     "monthly",
				"family_id":  uuid.New().String(),
				"user_id":    uuid.New().String(),
				"start_date": time.Now(),
				"end_date":   time.Now().AddDate(0, 1, 0),
			},
			expectedMsg: "",
		},
		{
			name: "Invalid type",
			requestBody: map[string]any{
				"name":       "Test Report",
				"type":       "invalid",
				"period":     "monthly",
				"family_id":  uuid.New().String(),
				"user_id":    uuid.New().String(),
				"start_date": time.Now(),
				"end_date":   time.Now().AddDate(0, 1, 0),
			},
			expectedMsg: "",
		},
		{
			name: "Invalid period",
			requestBody: map[string]any{
				"name":       "Test Report",
				"type":       "expenses",
				"period":     "invalid",
				"family_id":  uuid.New().String(),
				"user_id":    uuid.New().String(),
				"start_date": time.Now(),
				"end_date":   time.Now().AddDate(0, 1, 0),
			},
			expectedMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)
			withSessionUser(c, uuid.New())

			// Act
			err = handler.CreateReport(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestReportHandler_GetReports_ByFamily_Success(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	familyID := uuid.New()
	expectedReports := []*report.Report{
		{
			ID:          uuid.New(),
			Name:        "Monthly Expenses",
			Type:        report.TypeExpenses,
			Period:      report.PeriodMonthly,
			UserID:      uuid.New(),
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
			Data:        report.Data{},
			GeneratedAt: time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Weekly Income",
			Type:        report.TypeIncome,
			Period:      report.PeriodWeekly,
			UserID:      uuid.New(),
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2025, 1, 7, 23, 59, 59, 0, time.UTC),
			Data:        report.Data{},
			GeneratedAt: time.Now(),
		},
	}

	mockRepo.On("GetAll", mock.Anything).Return(expectedReports, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reports?family_id=%s", familyID), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetReports(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.ReportResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Data, 2)
	assert.Equal(t, expectedReports[0].Name, response.Data[0].Name)
	assert.Equal(t, string(expectedReports[0].Type), response.Data[0].Type)
	assert.Equal(t, expectedReports[1].Name, response.Data[1].Name)
	assert.Equal(t, string(expectedReports[1].Type), response.Data[1].Type)

	mockRepo.AssertExpectations(t)
}

func TestReportHandler_GetReports_ByUser_Success(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	familyID := uuid.New()
	userID := uuid.New()
	expectedReports := []*report.Report{
		{
			ID:          uuid.New(),
			Name:        "User Expenses",
			Type:        report.TypeExpenses,
			Period:      report.PeriodMonthly,
			UserID:      userID,
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
			Data:        report.Data{},
			GeneratedAt: time.Now(),
		},
	}

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(expectedReports, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/reports?family_id=%s&user_id=%s", familyID, userID),
		nil,
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetReports(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[[]handlers.ReportResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Data, 1)
	assert.Equal(t, expectedReports[0].Name, response.Data[0].Name)
	assert.Equal(t, userID, response.Data[0].UserID)

	mockRepo.AssertExpectations(t)
}

// TestReportHandler_GetReports_MissingFamilyID is deprecated in single-family model
// family_id is no longer required
/*
func TestReportHandler_GetReports_MissingFamilyID(t *testing.T) {
	handler, _ := setupReportHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetReports(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "MISSING_FAMILY_ID", response.Error.Code)
}
*/

// TestReportHandler_GetReports_InvalidFamilyID is deprecated in single-family model
// family_id is no longer required
/*
func TestReportHandler_GetReports_InvalidFamilyID(t *testing.T) {
	handler, _ := setupReportHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports?family_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetReports(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_FAMILY_ID", response.Error.Code)
}
*/

func TestReportHandler_GetReports_InvalidUserID(t *testing.T) {
	handler, _ := setupReportHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports?family_id="+uuid.New().String()+"&user_id=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err := handler.GetReports(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_USER_ID", response.Error.Code)
}

func TestReportHandler_GetReportByID_Success(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	reportID := uuid.New()
	expectedReport := &report.Report{
		ID:        reportID,
		Name:      "Budget Analysis",
		Type:      report.TypeBudget,
		Period:    report.PeriodMonthly,
		UserID:    uuid.New(),
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
		Data: report.Data{
			TotalIncome:   5000.0,
			TotalExpenses: 3500.0,
			NetIncome:     1500.0,
		},
		GeneratedAt: time.Now(),
	}

	mockRepo.On("GetByID", mock.Anything, reportID).Return(expectedReport, nil)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	// Act
	err := handler.GetReportByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.ReportResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedReport.ID, response.Data.ID)
	assert.Equal(t, expectedReport.Name, response.Data.Name)
	assert.Equal(t, string(expectedReport.Type), response.Data.Type)
	// Проверяем что данные присутствуют (они имеют тип any в response)
	assert.NotNil(t, response.Data.Data)

	mockRepo.AssertExpectations(t)
}

func TestReportHandler_GetReportByID_InvalidID(t *testing.T) {
	handler, _ := setupReportHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.GetReportByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "INVALID_ID", response.Error.Code)
}

func TestReportHandler_GetReportByID_NotFound(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	reportID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, reportID).Return(nil, errors.New("not found"))

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, "/reports/"+reportID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	// Act
	err := handler.GetReportByID(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "REPORT_NOT_FOUND", response.Error.Code)

	mockRepo.AssertExpectations(t)
}

func TestReportHandler_DeleteReport_Success(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	reportID := uuid.New()
	mockRepo.On("Delete", mock.Anything, reportID).Return(nil)

	e := echo.New()
	httpReq := httptest.NewRequest(
		http.MethodDelete,
		"/reports/"+reportID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	// Act
	err := handler.DeleteReport(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	mockRepo.AssertExpectations(t)
}

func TestReportHandler_DeleteReport_InvalidID(t *testing.T) {
	handler, _ := setupReportHandler()

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodDelete, "/reports/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	// Act
	err := handler.DeleteReport(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReportHandler_DeleteReport_RepositoryError(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	reportID := uuid.New()
	mockRepo.On("Delete", mock.Anything, reportID).Return(errors.New("database error"))

	e := echo.New()
	httpReq := httptest.NewRequest(
		http.MethodDelete,
		"/reports/"+reportID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())

	// Act
	err := handler.DeleteReport(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "DELETE_FAILED", response.Error.Code)

	mockRepo.AssertExpectations(t)
}

func TestReportHandler_ReportTypes_Validation(t *testing.T) {
	validTypes := []string{"expenses", "income", "budget", "cash_flow", "category_breakdown"}

	for _, reportType := range validTypes {
		t.Run(fmt.Sprintf("Valid type: %s", reportType), func(t *testing.T) {
			handler, mockService := setupReportHandlerWithService()
			userID := uuid.New()
			mockService.On("GenerateReport", mock.Anything, mock.Anything).Return(generatedTestReport(userID), nil)
			mockService.On("SaveReport", mock.Anything, mock.Anything).Return(nil)

			req := createValidReportRequest()
			req.Type = reportType

			c, rec := postReportRequest(t, req)
			withSessionUser(c, userID)

			require.NoError(t, handler.CreateReport(c))
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestReportHandler_ReportPeriods_Validation(t *testing.T) {
	validPeriods := []string{"daily", "weekly", "monthly", "yearly", "custom"}

	for _, period := range validPeriods {
		t.Run(fmt.Sprintf("Valid period: %s", period), func(t *testing.T) {
			handler, mockService := setupReportHandlerWithService()
			userID := uuid.New()
			mockService.On("GenerateReport", mock.Anything, mock.Anything).Return(generatedTestReport(userID), nil)
			mockService.On("SaveReport", mock.Anything, mock.Anything).Return(nil)

			req := createValidReportRequest()
			req.Period = period

			c, rec := postReportRequest(t, req)
			withSessionUser(c, userID)

			require.NoError(t, handler.CreateReport(c))
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestReportHandler_DateRange_Validation(t *testing.T) {
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
	}{
		{
			name:      "Valid date range",
			startDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			name:      "Same start and end date",
			startDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 1, 23, 59, 59, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockService := setupReportHandlerWithService()
			userID := uuid.New()
			mockService.On("GenerateReport", mock.Anything, mock.Anything).Return(generatedTestReport(userID), nil)
			mockService.On("SaveReport", mock.Anything, mock.Anything).Return(nil)

			req := createValidReportRequest()
			req.StartDate = tt.startDate
			req.EndDate = tt.endDate

			c, rec := postReportRequest(t, req)
			withSessionUser(c, userID)

			require.NoError(t, handler.CreateReport(c))
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkReportHandler_CreateReport(b *testing.B) {
	handler, mockService := setupReportHandlerWithService()

	userID := uuid.New()
	mockService.On("GenerateReport", mock.Anything, mock.Anything).Return(generatedTestReport(userID), nil)
	mockService.On("SaveReport", mock.Anything, mock.Anything).Return(nil)

	body, _ := json.Marshal(createValidReportRequest())

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
		httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)
		withSessionUser(c, userID)

		handler.CreateReport(c)
	}
}

func BenchmarkReportHandler_GetReports(b *testing.B) {
	handler, mockRepo := setupReportHandler()

	// Setup mock to return empty slice for all calls
	mockRepo.On("GetAll", mock.Anything).
		Return([]*report.Report{}, nil)

	familyID := uuid.New()

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/reports?family_id=%s", familyID), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.GetReports(c)
	}
}

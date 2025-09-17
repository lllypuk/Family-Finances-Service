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

func (m *MockReportRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, familyID)
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

// createValidReportRequest creates a valid report request for testing
func createValidReportRequest() handlers.CreateReportRequest {
	return handlers.CreateReportRequest{
		Name:      "Monthly Expenses Report",
		Type:      "expenses",
		Period:    "monthly",
		FamilyID:  uuid.New(),
		UserID:    uuid.New(),
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
	}
}

func TestReportHandler_CreateReport_Success(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	req := createValidReportRequest()
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*report.Report")).Return(nil)

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateReport(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var response handlers.APIResponse[handlers.ReportResponse]
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, req.Name, response.Data.Name)
	assert.Equal(t, req.Type, response.Data.Type)
	assert.Equal(t, req.Period, response.Data.Period)
	assert.Equal(t, req.FamilyID, response.Data.FamilyID)
	assert.Equal(t, req.UserID, response.Data.UserID)
	assert.Equal(t, req.StartDate, response.Data.StartDate)
	assert.Equal(t, req.EndDate, response.Data.EndDate)
	assert.False(t, response.Data.GeneratedAt.IsZero())

	mockRepo.AssertExpectations(t)
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

			// Act
			err = handler.CreateReport(c)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestReportHandler_CreateReport_RepositoryError(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	// Arrange
	req := createValidReportRequest()
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*report.Report")).
		Return(errors.New("database error"))

	// Prepare HTTP request
	body, err := json.Marshal(req)
	require.NoError(t, err)

	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
	httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(httpReq, rec)

	// Act
	err = handler.CreateReport(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "CREATE_FAILED", response.Error.Code)
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
			FamilyID:    familyID,
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
			FamilyID:    familyID,
			UserID:      uuid.New(),
			StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2025, 1, 7, 23, 59, 59, 0, time.UTC),
			Data:        report.Data{},
			GeneratedAt: time.Now(),
		},
	}

	mockRepo.On("GetByFamilyID", mock.Anything, familyID).Return(expectedReports, nil)

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
			FamilyID:    familyID,
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
		FamilyID:  uuid.New(),
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
	httpReq := httptest.NewRequest(http.MethodDelete, "/reports/"+reportID.String(), nil)
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
	httpReq := httptest.NewRequest(http.MethodDelete, "/reports/"+reportID.String(), nil)
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
	handler, mockRepo := setupReportHandler()

	validTypes := []string{"expenses", "income", "budget", "cash_flow", "category_break"}

	for _, reportType := range validTypes {
		t.Run(fmt.Sprintf("Valid type: %s", reportType), func(t *testing.T) {
			req := createValidReportRequest()
			req.Type = reportType

			mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *report.Report) bool {
				return string(r.Type) == reportType
			})).Return(nil).Once()

			body, err := json.Marshal(req)
			require.NoError(t, err)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err = handler.CreateReport(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestReportHandler_ReportPeriods_Validation(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	validPeriods := []string{"daily", "weekly", "monthly", "yearly", "custom"}

	for _, period := range validPeriods {
		t.Run(fmt.Sprintf("Valid period: %s", period), func(t *testing.T) {
			req := createValidReportRequest()
			req.Period = period

			mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *report.Report) bool {
				return string(r.Period) == period
			})).Return(nil).Once()

			body, err := json.Marshal(req)
			require.NoError(t, err)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err = handler.CreateReport(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, rec.Code)
		})
	}
}

func TestReportHandler_DateRange_Validation(t *testing.T) {
	handler, mockRepo := setupReportHandler()

	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		valid     bool
	}{
		{
			name:      "Valid date range",
			startDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
			valid:     true,
		},
		{
			name:      "Same start and end date",
			startDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 1, 1, 23, 59, 59, 0, time.UTC),
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createValidReportRequest()
			req.StartDate = tt.startDate
			req.EndDate = tt.endDate

			// Add mock expectation for valid cases
			if tt.valid {
				mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*report.Report")).Return(nil).Once()
			}

			body, err := json.Marshal(req)
			require.NoError(t, err)

			e := echo.New()
			httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
			httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(httpReq, rec)

			err = handler.CreateReport(c)
			require.NoError(t, err)

			if tt.valid {
				assert.Equal(t, http.StatusCreated, rec.Code)
			} else {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkReportHandler_CreateReport(b *testing.B) {
	handler, mockRepo := setupReportHandler()

	// Setup mock to return nil for all calls
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*report.Report")).Return(nil)

	req := createValidReportRequest()
	body, _ := json.Marshal(req)

	for b.Loop() {
		e := echo.New()
		httpReq := httptest.NewRequest(http.MethodPost, "/reports", bytes.NewBuffer(body))
		httpReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(httpReq, rec)

		handler.CreateReport(c)
	}
}

func BenchmarkReportHandler_GetReports(b *testing.B) {
	handler, mockRepo := setupReportHandler()

	// Setup mock to return empty slice for all calls
	mockRepo.On("GetByFamilyID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
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

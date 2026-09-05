package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) Summary(ctx context.Context, from, to time.Time) (*dto.StatsSummary, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.StatsSummary), args.Error(1)
}

func statsRequest(target string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	httpReq := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()

	return e.NewContext(httpReq, rec), rec
}

func TestStatsHandler_GetSummary_Success(t *testing.T) {
	mockService := &MockStatsService{}
	handler := handlers.NewStatsHandler(mockService)

	from := time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2025, 3, 31, 23, 59, 59, 0, time.Local)
	summary := &dto.StatsSummary{
		From:    from,
		To:      to,
		Current: dto.PeriodTotals{Income: 500, Expenses: 200, Net: 300, TransactionCount: 3},
	}
	mockService.On("Summary", mock.Anything, from, to).Return(summary, nil)

	c, rec := statsRequest("/stats/summary?from=2025-03-01&to=2025-03-31")

	require.NoError(t, handler.GetSummary(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.InDelta(t, 500.0, response.Data.Current.Income, 0.001)
	assert.Equal(t, 3, response.Data.Current.TransactionCount)

	mockService.AssertExpectations(t)
}

// TestStatsHandler_GetSummary_DefaultPeriod — без параметров период равен текущему месяцу.
func TestStatsHandler_GetSummary_DefaultPeriod(t *testing.T) {
	mockService := &MockStatsService{}
	handler := handlers.NewStatsHandler(mockService)

	now := time.Now()
	expectedFrom := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	expectedTo := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	mockService.On("Summary", mock.Anything, expectedFrom, expectedTo).Return(&dto.StatsSummary{}, nil)

	c, rec := statsRequest("/stats/summary")

	require.NoError(t, handler.GetSummary(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	mockService.AssertExpectations(t)
}

func TestStatsHandler_GetSummary_InvalidDate(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{name: "invalid from", target: "/stats/summary?from=01-03-2025"},
		{name: "invalid to", target: "/stats/summary?to=2025-13-45"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockStatsService{}
			handler := handlers.NewStatsHandler(mockService)

			c, rec := statsRequest(tt.target)

			require.NoError(t, handler.GetSummary(c))
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
			require.Len(t, response.Error.Details, 1)
			assert.Equal(t, "INVALID_QUERY_PARAM", response.Error.Details[0].Code)

			mockService.AssertNotCalled(t, "Summary", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestStatsHandler_GetSummary_InvertedPeriod(t *testing.T) {
	mockService := &MockStatsService{}
	handler := handlers.NewStatsHandler(mockService)

	mockService.On("Summary", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, services.ErrInvalidStatsPeriod)

	c, rec := statsRequest("/stats/summary?from=2025-03-31&to=2025-03-01")

	require.NoError(t, handler.GetSummary(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "from", response.Error.Details[0].Field)
}

func TestStatsHandler_GetSummary_ServiceError(t *testing.T) {
	mockService := &MockStatsService{}
	handler := handlers.NewStatsHandler(mockService)

	mockService.On("Summary", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("repository failure"))

	c, rec := statsRequest("/stats/summary")

	require.NoError(t, handler.GetSummary(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "INTERNAL_ERROR", response.Error.Code)
}

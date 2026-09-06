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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) Summary(ctx context.Context, from, to *date.Date) (*dto.StatsSummary, error) {
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

	from := date.New(2025, time.March, 1)
	to := date.New(2025, time.March, 31)
	summary := &dto.StatsSummary{
		From: from,
		To:   to,
		Current: dto.PeriodTotals{
			IncomeMinor: 50_000, ExpensesMinor: 20_000, NetMinor: 30_000, TransactionCount: 3,
		},
	}
	mockService.On("Summary", mock.Anything, &from, &to).Return(summary, nil)

	c, rec := statsRequest("/stats/summary?from=2025-03-01&to=2025-03-31")

	require.NoError(t, handler.GetSummary(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, money.Minor(50_000), response.Data.Current.IncomeMinor)
	assert.Equal(t, 3, response.Data.Current.TransactionCount)

	mockService.AssertExpectations(t)
}

// TestStatsHandler_GetSummary_DefaultPeriod — без параметров границы не задаются:
// текущий месяц по зоне семьи считает сервис.
func TestStatsHandler_GetSummary_DefaultPeriod(t *testing.T) {
	mockService := &MockStatsService{}
	handler := handlers.NewStatsHandler(mockService)

	mockService.On("Summary", mock.Anything, (*date.Date)(nil), (*date.Date)(nil)).
		Return(&dto.StatsSummary{}, nil)

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

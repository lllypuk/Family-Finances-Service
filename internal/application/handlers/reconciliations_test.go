package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type mockReconciliationService struct {
	mock.Mock
}

func (m *mockReconciliationService) PutBalance(
	ctx context.Context,
	accountID uuid.UUID,
	month date.Date,
	balance money.Minor,
) (*reconciliation.Balance, error) {
	args := m.Called(ctx, accountID, month, balance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*reconciliation.Balance), args.Error(1)
}

func (m *mockReconciliationService) DeleteBalance(ctx context.Context, accountID uuid.UUID, month date.Date) error {
	return m.Called(ctx, accountID, month).Error(0)
}

func (m *mockReconciliationService) Summary(ctx context.Context, month *date.Date) (*dto.ReconciliationStats, error) {
	args := m.Called(ctx, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ReconciliationStats), args.Error(1)
}

func callBalance(
	t *testing.T,
	handler echo.HandlerFunc,
	method, id, month, body string,
) (int, handlers.ErrorResponse) {
	t.Helper()

	c, rec := principalContext(method, "/", body, nil)
	c.SetParamNames("id", "month")
	c.SetParamValues(id, month)
	require.NoError(t, handler(c))
	if rec.Code < http.StatusBadRequest {
		return rec.Code, handlers.ErrorResponse{}
	}

	return rec.Code, errorCode(t, rec.Body.Bytes())
}

func TestReconciliationHandler_PutAccountBalance(t *testing.T) {
	svc := &mockReconciliationService{}
	h := handlers.NewReconciliationHandler(svc)
	id := uuid.New()
	aug := date.New(2026, time.August, 1)
	svc.On("PutBalance", mock.Anything, id, aug, money.Minor(0)).
		Return(&reconciliation.Balance{AccountID: id, Month: "2026-08"}, nil)
	svc.On("PutBalance", mock.Anything, id, aug, money.Minor(-1)).
		Return(nil, reconciliation.ErrBalanceOutOfRange)
	svc.On("PutBalance", mock.Anything, id, date.New(2026, time.December, 1), money.Minor(1)).
		Return(nil, services.ErrReconciliationMonthInFuture)
	unknown := uuid.New()
	svc.On("PutBalance", mock.Anything, unknown, aug, money.Minor(1)).Return(nil, account.ErrNotFound)

	code, _ := callBalance(t, h.PutAccountBalance, http.MethodPut, id.String(), "2026-08", `{"balance_minor":0}`)
	assert.Equal(t, http.StatusOK, code, "0 — законный остаток")

	tests := []struct {
		name, id, month, body string
		status                int
		code, field           string
	}{
		{"bad id", "not-a-uuid", "2026-08", `{"balance_minor":1}`, http.StatusBadRequest,
			handlers.ErrCodeInvalidID, ""},
		{"bad month", id.String(), "2026-8", `{"balance_minor":1}`, http.StatusUnprocessableEntity,
			handlers.ErrCodeValidationError, "month"},
		{"future month", id.String(), "2026-12", `{"balance_minor":1}`, http.StatusUnprocessableEntity,
			handlers.ErrCodeValidationError, "month"},
		{"out of range", id.String(), "2026-08", `{"balance_minor":-1}`, http.StatusUnprocessableEntity,
			handlers.ErrCodeValidationError, "balance_minor"},
		{"missing", id.String(), "2026-08", `{}`, http.StatusUnprocessableEntity,
			handlers.ErrCodeValidationError, "balance_minor"},
		{"unknown account", unknown.String(), "2026-08", `{"balance_minor":1}`, http.StatusNotFound,
			handlers.ErrCodeAccountNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, resp := callBalance(t, h.PutAccountBalance, http.MethodPut, tt.id, tt.month, tt.body)
			assert.Equal(t, tt.status, status)
			assert.Equal(t, tt.code, resp.Error.Code)
			if tt.field != "" {
				require.Len(t, resp.Error.Details, 1)
				assert.Equal(t, tt.field, resp.Error.Details[0].Field)
			}
		})
	}
}

func TestReconciliationHandler_DeleteAccountBalance(t *testing.T) {
	svc := &mockReconciliationService{}
	h := handlers.NewReconciliationHandler(svc)
	id := uuid.New()
	aug := date.New(2026, time.August, 1)
	svc.On("DeleteBalance", mock.Anything, id, aug).Return(reconciliation.ErrBalanceNotFound).Once()
	svc.On("DeleteBalance", mock.Anything, id, aug).Return(nil).Once()
	dec := date.New(2026, time.December, 1)
	svc.On("DeleteBalance", mock.Anything, id, dec).Return(services.ErrReconciliationMonthInFuture)
	broken := uuid.New()
	svc.On("DeleteBalance", mock.Anything, broken, aug).Return(errors.New("disk I/O error"))

	status, resp := callBalance(t, h.DeleteAccountBalance, http.MethodDelete, id.String(), "2026-08", "")
	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, handlers.ErrCodeBalanceNotFound, resp.Error.Code)

	status, _ = callBalance(t, h.DeleteAccountBalance, http.MethodDelete, id.String(), "2026-08", "")
	assert.Equal(t, http.StatusNoContent, status)

	status, _ = callBalance(t, h.DeleteAccountBalance, http.MethodDelete, "x", "2026-08", "")
	assert.Equal(t, http.StatusBadRequest, status)

	status, resp = callBalance(t, h.DeleteAccountBalance, http.MethodDelete, id.String(), "2026-12", "")
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	require.Len(t, resp.Error.Details, 1)
	assert.Equal(t, "month", resp.Error.Details[0].Field)

	status, _ = callBalance(t, h.DeleteAccountBalance, http.MethodDelete, broken.String(), "2026-08", "")
	assert.Equal(t, http.StatusInternalServerError, status)
}

func TestReconciliationHandler_GetReconciliationStats(t *testing.T) {
	svc := &mockReconciliationService{}
	h := handlers.NewReconciliationHandler(svc)
	svc.On("Summary", mock.Anything, (*date.Date)(nil)).Return(&dto.ReconciliationStats{Month: "2026-09"}, nil)
	future := date.New(2027, time.January, 1)
	svc.On("Summary", mock.Anything, &future).Return(nil, services.ErrReconciliationMonthInFuture)

	c, rec := principalContext(http.MethodGet, "/api/v1/stats/reconciliation", "", nil)
	require.NoError(t, h.GetReconciliationStats(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	for _, month := range []string{"2026-09-01", "2027-01"} {
		c, rec = principalContext(http.MethodGet, "/api/v1/stats/reconciliation?month="+month, "", nil)
		require.NoError(t, h.GetReconciliationStats(c))
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, month)
		assert.Equal(t, "month", errorCode(t, rec.Body.Bytes()).Error.Details[0].Field, month)
	}
}

package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/services/dto"
)

type mockReconciliationService struct {
	mock.Mock
}

func (m *mockReconciliationService) Put(
	ctx context.Context,
	accountID uuid.UUID,
	month date.Date,
	bankExpense money.Minor,
	note string,
) (*reconciliation.Reconciliation, error) {
	args := m.Called(ctx, accountID, month, bankExpense, note)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*reconciliation.Reconciliation), args.Error(1)
}

func (m *mockReconciliationService) Delete(ctx context.Context, accountID uuid.UUID, month date.Date) error {
	return m.Called(ctx, accountID, month).Error(0)
}

func (m *mockReconciliationService) Summary(ctx context.Context, month *date.Date) (*dto.ReconciliationStats, error) {
	args := m.Called(ctx, month)
	return args.Get(0).(*dto.ReconciliationStats), args.Error(1)
}

func TestReconciliationHandler_PutReconciliation_Path(t *testing.T) {
	svc := &mockReconciliationService{}
	h := handlers.NewReconciliationHandler(svc)
	id := uuid.New()
	sept := date.New(2026, time.September, 1)
	svc.On("Put", mock.Anything, id, sept, money.Minor(0), "").
		Return(&reconciliation.Reconciliation{AccountID: id, Month: "2026-09"}, nil)

	c, rec := principalContext(http.MethodPut, "/", `{"bank_expense_minor":0}`, nil)
	c.SetParamNames("id", "month")
	c.SetParamValues(id.String(), "2026-09")
	require.NoError(t, h.PutReconciliation(c))
	assert.Equal(t, http.StatusOK, rec.Code, "0 — законная сумма")

	c, rec = principalContext(http.MethodPut, "/", `{"bank_expense_minor":1}`, nil)
	c.SetParamNames("id", "month")
	c.SetParamValues("not-a-uuid", "2026-09")
	require.NoError(t, h.PutReconciliation(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	c, rec = principalContext(http.MethodPut, "/", `{"bank_expense_minor":1}`, nil)
	c.SetParamNames("id", "month")
	c.SetParamValues(id.String(), "2026-9")
	require.NoError(t, h.PutReconciliation(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, "month", errorCode(t, rec.Body.Bytes()).Error.Details[0].Field)
	svc.AssertNumberOfCalls(t, "Put", 1)
}

func TestReconciliationHandler_GetReconciliationStats_DefaultMonth(t *testing.T) {
	svc := &mockReconciliationService{}
	h := handlers.NewReconciliationHandler(svc)
	svc.On("Summary", mock.Anything, (*date.Date)(nil)).Return(&dto.ReconciliationStats{Month: "2026-09"}, nil)

	c, rec := principalContext(http.MethodGet, "/api/v1/stats/reconciliation", "", nil)
	require.NoError(t, h.GetReconciliationStats(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	c, rec = principalContext(http.MethodGet, "/api/v1/stats/reconciliation?month=2026-09-01", "", nil)
	require.NoError(t, h.GetReconciliationStats(c))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Equal(t, "month", errorCode(t, rec.Body.Bytes()).Error.Details[0].Field)
}

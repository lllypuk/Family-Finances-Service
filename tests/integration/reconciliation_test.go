package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

func decodeBalance(t *testing.T, rec *httptest.ResponseRecorder) handlers.AccountBalanceResponse {
	t.Helper()

	var resp handlers.APIResponse[handlers.AccountBalanceResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Data
}

func (s *accountStand) reconciliationStats(t *testing.T, month string) dto.ReconciliationStats {
	t.Helper()

	rec := s.do(t, http.MethodGet, "/api/v1/stats/reconciliation?month="+month, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp handlers.APIResponse[dto.ReconciliationStats]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Data
}

func (s *accountStand) currentMonth() date.Date {
	first, _ := date.Today(s.ts.AuthFamily.Location()).MonthBounds()

	return first
}

func balancePath(id uuid.UUID, month string) string {
	return "/api/v1/accounts/" + id.String() + "/balances/" + month
}

func TestReconciliationAPI_Access(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	path := balancePath(id, "2026-08")

	for _, p := range []string{path, "/api/v1/stats/reconciliation"} {
		rec := doAccountRequest(t, s.ts, nil, http.MethodGet, p, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, p)
	}

	rec := s.do(t, http.MethodPut, path, `{"balance_minor":100}`)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	_, member := s.ts.AuthAs(t, user.RoleMember)
	rec = doAccountRequest(t, s.ts, member, http.MethodPut, path, `{"balance_minor":200}`)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, money.Minor(200), decodeBalance(t, rec).BalanceMinor)
	rec = doAccountRequest(t, s.ts, member, http.MethodGet, "/api/v1/stats/reconciliation", "")
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doAccountRequest(t, s.ts, member, http.MethodDelete, path, "")
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestReconciliationAPI_UpsertAndGap(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	month := s.currentMonth().MonthKey()

	stats := s.reconciliationStats(t, month)
	require.Len(t, stats.Accounts, 1)
	assert.Equal(t, money.Minor(0), *stats.Accounts[0].OpeningMinor, "счёт заведён в этом месяце")
	assert.Nil(t, stats.ClosingMinor)
	assert.Nil(t, stats.GapMinor)
	assert.False(t, stats.Complete)

	rec := s.do(t, http.MethodPut, balancePath(id, month), `{"balance_minor":-2500}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	first := decodeBalance(t, rec)
	assert.Equal(t, month, first.Month)

	rec = s.do(t, http.MethodPut, balancePath(id, month), `{"balance_minor":5000}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	second := decodeBalance(t, rec)
	assert.True(t, second.UpdatedAt.After(first.UpdatedAt))

	stats = s.reconciliationStats(t, month)
	assert.True(t, stats.Complete)
	assert.Equal(t, money.Minor(5000), *stats.ClosingMinor)
	assert.Equal(t, money.Minor(5000), *stats.GapMinor)
	assert.True(t, stats.Accounts[0].UpdatedAt.Equal(second.UpdatedAt), "одна запись, свежее время")

	rec = s.do(t, http.MethodDelete, "/api/v1/accounts/"+id.String(), "")
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, handlers.ErrCodeAccountInUse, errorCodeOf(t, rec))
}

func TestReconciliationAPI_ArchivedAccountAcceptsBalance(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Старая карта")
	s.archive(t, id)

	rec := s.do(t, http.MethodPut, balancePath(id, s.currentMonth().AddMonths(-1).MonthKey()), `{"balance_minor":700}`)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestReconciliationAPI_CorrectionClosesGap(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	month := s.currentMonth().MonthKey()
	today := date.Today(s.ts.AuthFamily.Location()).String()

	expense := func(amount money.Minor) {
		t.Helper()

		rec := s.do(t, http.MethodPost, "/api/v1/transactions", fmt.Sprintf(
			`{"amount_minor":%d,"type":"expense","description":"Покупка","category_id":%q,"date":%q}`,
			amount, s.category, today))
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	}

	rec := s.do(
		t,
		http.MethodPut,
		balancePath(id, s.currentMonth().AddMonths(-1).MonthKey()),
		`{"balance_minor":10000}`,
	)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = s.do(t, http.MethodPut, balancePath(id, month), `{"balance_minor":7000}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	expense(1000)

	stats := s.reconciliationStats(t, month)
	require.True(t, stats.Complete)
	assert.Equal(t, money.Minor(10000), *stats.OpeningMinor, "строка прошлого месяца побеждает дату заведения")
	assert.Equal(t, money.Minor(1000), stats.ExpenseMinor)
	require.NotNil(t, stats.GapMinor)
	assert.Equal(t, money.Minor(-2000), *stats.GapMinor)

	expense(-*stats.GapMinor)

	stats = s.reconciliationStats(t, month)
	require.NotNil(t, stats.GapMinor)
	assert.Equal(t, money.Minor(0), *stats.GapMinor)
}

func TestReconciliationAPI_ZeroBalanceLocks(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")

	rec := s.do(t, http.MethodPut, balancePath(id, s.currentMonth().MonthKey()), `{"balance_minor":0}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = s.do(t, http.MethodPut, "/api/v1/family", `{"currency":"USD"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeCurrencyLocked, errorCodeOf(t, rec))

	rec = s.do(t, http.MethodDelete, "/api/v1/accounts/"+id.String(), "")
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeAccountInUse, errorCodeOf(t, rec))
}

func TestReconciliationAPI_Errors(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	next := s.currentMonth().AddMonths(1).MonthKey()

	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"unknown account", http.MethodPut, balancePath(uuid.New(), "2026-08"), `{"balance_minor":1}`,
			http.StatusNotFound, handlers.ErrCodeAccountNotFound},
		{"delete unknown account", http.MethodDelete, balancePath(uuid.New(), "2026-08"), "",
			http.StatusNotFound, handlers.ErrCodeAccountNotFound},
		{"no balance", http.MethodDelete, balancePath(id, "2026-08"), "",
			http.StatusNotFound, handlers.ErrCodeBalanceNotFound},
		{"bad month", http.MethodPut, balancePath(id, "2026-13"), `{"balance_minor":1}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"future month", http.MethodPut, balancePath(id, next), `{"balance_minor":1}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"out of range", http.MethodPut, balancePath(id, "2026-08"), `{"balance_minor":-100000000000000}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"missing amount", http.MethodPut, balancePath(id, "2026-08"), `{}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"stats bad month", http.MethodGet, "/api/v1/stats/reconciliation?month=2026-13", "",
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"stats future month", http.MethodGet, "/api/v1/stats/reconciliation?month=" + next, "",
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := s.do(t, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.code, errorCodeOf(t, rec))
		})
	}
}

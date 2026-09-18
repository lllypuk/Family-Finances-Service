package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

func decodeReconciliation(t *testing.T, rec *httptest.ResponseRecorder) handlers.ReconciliationResponse {
	t.Helper()

	var resp handlers.APIResponse[handlers.ReconciliationResponse]
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

func TestReconciliationAPI_Access(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	path := "/api/v1/accounts/" + id.String() + "/reconciliations/2026-09"

	for _, p := range []string{path, "/api/v1/stats/reconciliation"} {
		rec := doAccountRequest(t, s.ts, nil, http.MethodGet, p, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, p)
	}

	_, member := s.ts.AuthAs(t, user.RoleMember)
	rec := doAccountRequest(t, s.ts, member, http.MethodPut, path, `{"bank_expense_minor":100}`)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doAccountRequest(t, s.ts, member, http.MethodGet, "/api/v1/stats/reconciliation", "")
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = doAccountRequest(t, s.ts, member, http.MethodDelete, path, "")
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestReconciliationAPI_UpsertAndDiff(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	path := "/api/v1/accounts/" + id.String() + "/reconciliations/2026-09"

	rec := s.do(t, http.MethodPut, path, `{"bank_expense_minor":25000,"note":"выписка"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	first := decodeReconciliation(t, rec)
	assert.Equal(t, "выписка", first.Note)
	assert.Equal(t, "2026-09", first.Month)

	rec = s.do(t, http.MethodPut, path, `{"bank_expense_minor":20000}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	second := decodeReconciliation(t, rec)
	assert.True(t, second.UpdatedAt.After(first.UpdatedAt))

	stats := s.reconciliationStats(t, "2026-09")
	require.Len(t, stats.Accounts, 1)
	row := stats.Accounts[0]
	assert.Equal(t, money.Minor(20000), *row.BankExpenseMinor)
	assert.Empty(t, *row.Note, "PUT без note очищает заметку")
	assert.Equal(t, money.Minor(20000), *row.DiffMinor)
	assert.True(t, row.UpdatedAt.Equal(second.UpdatedAt), "одна запись, свежее время")

	s.create(t, `,"account_id":"`+id.String()+`"`)
	s.create(t, "")
	stats = s.reconciliationStats(t, "2026-09")
	assert.Equal(t, money.Minor(10000), stats.Accounts[0].RecordedMinor)
	assert.Equal(t, money.Minor(10000), *stats.Accounts[0].DiffMinor, "операция меняет diff без нового PUT")
	assert.Equal(t, money.Minor(10000), stats.UnassignedMinor)

	rec = s.do(t, http.MethodDelete, "/api/v1/accounts/"+id.String(), "")
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, handlers.ErrCodeAccountInUse, errorCodeOf(t, rec))
}

func TestReconciliationAPI_Errors(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")
	base := "/api/v1/accounts/" + id.String() + "/reconciliations/"

	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"unknown account", http.MethodPut, "/api/v1/accounts/" + uuid.NewString() + "/reconciliations/2026-09",
			`{"bank_expense_minor":1}`, http.StatusNotFound, handlers.ErrCodeAccountNotFound},
		{"no reconciliation", http.MethodDelete, base + "2026-09", "",
			http.StatusNotFound, handlers.ErrCodeReconciliationNotFound},
		{"bad month", http.MethodPut, base + "2026-13", `{"bank_expense_minor":1}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"negative", http.MethodPut, base + "2026-09", `{"bank_expense_minor":-1}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"missing amount", http.MethodPut, base + "2026-09", `{"note":"x"}`,
			http.StatusUnprocessableEntity, handlers.ErrCodeValidationError},
		{"stats bad month", http.MethodGet, "/api/v1/stats/reconciliation?month=2026-13", "",
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

func TestReconciliationAPI_ZeroReconciliationLocksCurrency(t *testing.T) {
	s := newAccountStand(t)
	id := s.account(t, "Сбер")

	rec := s.do(t, http.MethodPut, "/api/v1/accounts/"+id.String()+"/reconciliations/2026-09",
		`{"bank_expense_minor":0}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = s.do(t, http.MethodPut, "/api/v1/family", `{"currency":"USD"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeCurrencyLocked, errorCodeOf(t, rec))
}

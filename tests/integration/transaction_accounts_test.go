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
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/testhelpers"
)

type accountStand struct {
	ts       *testhelpers.TestServer
	admin    *testhelpers.AuthSession
	category uuid.UUID
}

func newAccountStand(t *testing.T) *accountStand {
	t.Helper()

	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	cat := testhelpers.CreateTestCategory(ts.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(t.Context(), cat))

	return &accountStand{ts: ts, admin: admin, category: cat.ID}
}

func (s *accountStand) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	return doAccountRequest(t, s.ts, s.admin, method, path, body)
}

func (s *accountStand) account(t *testing.T, name string) uuid.UUID {
	t.Helper()

	rec := s.do(t, http.MethodPost, "/api/v1/accounts", fmt.Sprintf(`{"name":%q}`, name))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	return decodeAccount(t, rec).ID
}

func (s *accountStand) archive(t *testing.T, id uuid.UUID) {
	t.Helper()

	rec := s.do(t, http.MethodPut, "/api/v1/accounts/"+id.String(), `{"is_archived":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// createBody — расход на 100.00; extra дописывается в тело как есть.
func (s *accountStand) createBody(extra string) string {
	return fmt.Sprintf(`{"amount_minor":10000,"type":"expense","description":"Покупка","category_id":%q,`+
		`"date":"2026-09-01"%s}`, s.category, extra)
}

func (s *accountStand) create(t *testing.T, extra string) handlers.TransactionResponse {
	t.Helper()

	rec := s.do(t, http.MethodPost, "/api/v1/transactions", s.createBody(extra))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	return decodeTransaction(t, rec)
}

func decodeTransaction(t *testing.T, rec *httptest.ResponseRecorder) handlers.TransactionResponse {
	t.Helper()

	var resp handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	return resp.Data
}

func accountField(id uuid.UUID) string {
	return fmt.Sprintf(`,"account_id":%q`, id)
}

func requireFieldError(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	var resp handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Error.Details)
	assert.Equal(t, field, resp.Error.Details[0].Field)
}

func TestTransactionAccounts_CreateAndFilter(t *testing.T) {
	s := newAccountStand(t)
	card := s.account(t, "Карта")

	linked := s.create(t, accountField(card))
	require.NotNil(t, linked.AccountID)
	assert.Equal(t, card, *linked.AccountID)
	s.create(t, accountField(card))
	cash := s.create(t, "")
	assert.Nil(t, cash.AccountID)

	rec := s.do(t, http.MethodGet, "/api/v1/transactions/"+cash.ID.String(), "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"account_id":null`)

	for _, tc := range []struct {
		query string
		want  int
	}{
		{"account_id=" + card.String(), 2},
		{"unassigned=true", 1},
		{"account_id=" + card.String() + "&limit=1", 2},
	} {
		rec = s.do(t, http.MethodGet, "/api/v1/transactions?"+tc.query, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var resp handlers.APIResponse[[]handlers.TransactionResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.NotNil(t, resp.Meta.Pagination)
		assert.Equal(t, tc.want, resp.Meta.Pagination.Total, tc.query)
		assert.Len(t, resp.Data, min(tc.want, resp.Meta.Pagination.Limit), tc.query)
	}

	rec = s.do(t, http.MethodGet, "/api/v1/transactions?unassigned=true&account_id="+card.String(), "")
	requireFieldError(t, rec, "unassigned")
}

func TestTransactionAccounts_RefusedAccounts(t *testing.T) {
	s := newAccountStand(t)
	archived := s.account(t, "Старая карта")
	s.archive(t, archived)

	requireFieldError(t, s.do(t, http.MethodPost, "/api/v1/transactions", s.createBody(accountField(archived))),
		"account_id")
	requireFieldError(t, s.do(t, http.MethodPost, "/api/v1/transactions", s.createBody(accountField(uuid.New()))),
		"account_id")

	tx := s.create(t, "")
	path := "/api/v1/transactions/" + tx.ID.String()
	requireFieldError(t, s.do(t, http.MethodPut, path, fmt.Sprintf(`{"account_id":%q}`, archived)), "account_id")
	requireFieldError(t, s.do(t, http.MethodPut, path, fmt.Sprintf(`{"account_id":%q}`, uuid.New())), "account_id")
}

func TestTransactionAccounts_Update(t *testing.T) {
	s := newAccountStand(t)
	card := s.account(t, "Карта")
	tx := s.create(t, accountField(card))
	path := "/api/v1/transactions/" + tx.ID.String()

	rec := s.do(t, http.MethodPut, path, `{"description":"Без поля счёта"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, decodeTransaction(t, rec).AccountID)

	s.archive(t, card)
	rec = s.do(t, http.MethodPut, path, fmt.Sprintf(`{"description":"Счёт в архиве","account_id":%q}`, card))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, card, *decodeTransaction(t, rec).AccountID)

	rec = s.do(t, http.MethodPut, path, fmt.Sprintf(`{"account_id":%q,"clear_account":true}`, card))
	requireFieldError(t, rec, "clear_account")

	rec = s.do(t, http.MethodPut, path, `{"clear_account":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Nil(t, decodeTransaction(t, rec).AccountID)
}

func TestTransactionAccounts_DeleteAccountInUse(t *testing.T) {
	s := newAccountStand(t)
	card := s.account(t, "Карта")
	s.create(t, accountField(card))

	rec := s.do(t, http.MethodDelete, "/api/v1/accounts/"+card.String(), "")
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeAccountInUse, errorCodeOf(t, rec))
}

func TestTransactionAccounts_IdempotentCreate(t *testing.T) {
	s := newAccountStand(t)
	card := s.account(t, "Карта")
	other := s.account(t, "Другая")
	id := uuid.New()
	body := s.createBody(fmt.Sprintf(`,"id":%q%s`, id, accountField(card)))

	rec := s.do(t, http.MethodPost, "/api/v1/transactions", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	s.archive(t, card)
	rec = s.do(t, http.MethodPost, "/api/v1/transactions", body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, card, *decodeTransaction(t, rec).AccountID)

	rec = s.do(t, http.MethodPost, "/api/v1/transactions",
		s.createBody(fmt.Sprintf(`,"id":%q%s`, id, accountField(other))))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, card, *decodeTransaction(t, rec).AccountID)
}

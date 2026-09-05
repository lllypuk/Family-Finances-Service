package integration_test

import (
	"bytes"
	"context"
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
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

const paginationTxCount = 5

// seedTransactions создаёт count расходных транзакций семьи, под которой залогинен ts.Auth.
func seedTransactions(t *testing.T, ts *testhelpers.TestServer, count int) []*transaction.Transaction {
	t.Helper()

	ts.Auth(t)

	testCategory := testhelpers.CreateTestCategory(ts.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(context.Background(), testCategory))

	created := make([]*transaction.Transaction, 0, count)
	for range count {
		tx := testhelpers.CreateTestTransaction(
			ts.AuthFamily.ID, ts.AuthUser.ID, testCategory.ID, transaction.TypeExpense,
		)
		require.NoError(t, ts.Repos.Transaction.Create(context.Background(), tx))
		created = append(created, tx)
	}

	return created
}

func getJSON[T any](t *testing.T, ts *testhelpers.TestServer, target string) (int, T) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, target, nil)
	ts.Auth(t).Apply(req)
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	var body T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	return rec.Code, body
}

func TestAPIPagination_Transactions_TotalMatchesCreated(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	seedTransactions(t, testServer, paginationTxCount)

	code, response := getJSON[handlers.APIResponse[[]handlers.TransactionResponse]](
		t, testServer, "/api/v1/transactions?limit=2",
	)

	require.Equal(t, http.StatusOK, code)
	assert.Len(t, response.Data, 2)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 2, response.Meta.Pagination.Limit)
	assert.Equal(t, 0, response.Meta.Pagination.Offset)
	assert.Equal(t, paginationTxCount, response.Meta.Pagination.Total)
}

func TestAPIPagination_Transactions_LimitAboveMax(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	seedTransactions(t, testServer, 1)

	code, response := getJSON[handlers.ErrorResponse](t, testServer, "/api/v1/transactions?limit=500")

	require.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, handlers.ErrCodeInvalidQueryParam, response.Error.Code)
}

func TestAPIPagination_Transactions_OffsetBeyondTotal(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	seedTransactions(t, testServer, paginationTxCount)

	code, response := getJSON[handlers.APIResponse[[]handlers.TransactionResponse]](
		t, testServer, "/api/v1/transactions?offset=100",
	)

	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, response.Data)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 100, response.Meta.Pagination.Offset)
	assert.Equal(t, paginationTxCount, response.Meta.Pagination.Total)
}

// Пагинация обязательна во всех списках, включая короткие (A-08).
func TestAPIPagination_AllListsCarryPagination(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	seedTransactions(t, testServer, 1)

	for _, path := range []string{
		"/api/v1/transactions",
		"/api/v1/categories",
		"/api/v1/budgets",
		"/api/v1/reports",
		"/api/v1/users",
		"/api/v1/backups",
	} {
		t.Run(path, func(t *testing.T) {
			code, response := getJSON[handlers.APIResponse[json.RawMessage]](t, testServer, path)

			require.Equal(t, http.StatusOK, code)
			require.NotNil(t, response.Meta.Pagination)
			assert.Equal(t, 50, response.Meta.Pagination.Limit)
		})
	}
}

func TestAPIPagination_Categories_OffsetAndTotal(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	for range 3 {
		testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
		testCategory.Name = uuid.NewString()
		require.NoError(t, testServer.Repos.Category.Create(context.Background(), testCategory))
	}

	code, response := getJSON[handlers.APIResponse[[]json.RawMessage]](
		t, testServer, "/api/v1/categories?limit=1&offset=2",
	)

	require.Equal(t, http.StatusOK, code)
	assert.Len(t, response.Data, 1)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 3, response.Meta.Pagination.Total)
}

func TestAPIPagination_Budgets_TotalIgnoresLimit(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	testServer.Auth(t)

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(context.Background(), testCategory))

	for range 3 {
		testBudget := testhelpers.CreateTestBudget(testServer.AuthFamily.ID, testCategory.ID)
		testBudget.Name = uuid.NewString()
		require.NoError(t, testServer.Repos.Budget.Create(context.Background(), testBudget))
	}

	code, response := getJSON[handlers.APIResponse[[]handlers.BudgetResponse]](
		t, testServer, "/api/v1/budgets?limit=2",
	)

	require.Equal(t, http.StatusOK, code)
	assert.Len(t, response.Data, 2)
	require.NotNil(t, response.Meta.Pagination)
	assert.Equal(t, 3, response.Meta.Pagination.Total)
}

func TestAPIBulkDelete_Transactions_SkipsUnknownIDs(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	created := seedTransactions(t, testServer, 2)

	ids := []uuid.UUID{created[0].ID, uuid.New(), created[1].ID}
	body, err := json.Marshal(handlers.BulkDeleteRequest{IDs: ids})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/bulk-delete", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	testServer.Auth(t).Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response handlers.APIResponse[handlers.BulkDeleteResponse]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, 2, response.Data.Deleted)

	for _, id := range []uuid.UUID{created[0].ID, created[1].ID} {
		code, _ := getJSON[handlers.APIResponse[handlers.TransactionResponse]](
			t, testServer, fmt.Sprintf("/api/v1/transactions/%s", id),
		)
		assert.Equal(t, http.StatusNotFound, code)
	}
}

func TestAPIBulkDelete_Transactions_EmptyIDsRejected(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	seedTransactions(t, testServer, 1)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/transactions/bulk-delete",
		bytes.NewBufferString(`{"ids":[]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	testServer.Auth(t).Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

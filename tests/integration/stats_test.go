package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

// TestStatsAPI_Summary — сводка за период совпадает с созданными операциями.
func TestStatsAPI_Summary(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	expenseCat := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, expenseCat))

	expense := testhelpers.CreateTestTransaction(
		testServer.AuthFamily.ID, testServer.AuthUser.ID, expenseCat.ID, transaction.TypeExpense,
	)
	expense.AmountMinor = 20_000
	// Зона семьи, не UTC: сервис считает текущий месяц по family.Timezone.
	expense.Date = date.Today(testServer.AuthFamily.Location())
	require.NoError(t, testServer.Repos.Transaction.Create(ctx, expense))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary", nil)
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, money.Minor(20_000), response.Data.Current.ExpensesMinor)
	assert.Equal(t, 1, response.Data.Current.TransactionCount)
	require.Len(t, response.Data.ExpenseCategories, 1)
	assert.Equal(t, expenseCat.Name, response.Data.ExpenseCategories[0].Name)
}

// TestStatsAPI_Summary_InvalidDate — нераспознанная дата отбивается 422, сервис не вызывается.
func TestStatsAPI_Summary_InvalidDate(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary?from=01.03.2025", nil)
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "from", response.Error.Details[0].Field)
}

// statsBulkTransactions — число операций, на котором прежняя постраничная выборка обрывалась молча.
const statsBulkTransactions = 20_001

// TestStatsAPI_Summary_BeyondFormerPageCeiling — сводка считается в SQL, поэтому потолка в 20 000 операций нет.
func TestStatsAPI_Summary_BeyondFormerPageCeiling(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	expenseCat := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, expenseCat))

	today := date.Today(testServer.AuthFamily.Location())
	insertBulkExpenses(t, testServer, expenseCat.ID, today, statsBulkTransactions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/summary", nil)
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, statsBulkTransactions, response.Data.Current.TransactionCount)
	assert.Equal(t, money.Minor(statsBulkTransactions*100), response.Data.Current.ExpensesMinor)
	require.Len(t, response.Data.ExpenseCategories, 1)
	assert.Equal(t, statsBulkTransactions, response.Data.ExpenseCategories[0].TransactionCount)
}

// insertBulkExpenses пишет операции напрямую в базу: через репозиторий по одной транзакции это минуты.
func insertBulkExpenses(
	t *testing.T,
	testServer *testhelpers.TestServer,
	categoryID uuid.UUID,
	day date.Date,
	count int,
) {
	t.Helper()

	tx, err := testServer.Container.DB.Begin()
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT INTO transactions
		(id, amount_minor, description, date, type, category_id, user_id, family_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	require.NoError(t, err)
	defer func() { _ = stmt.Close() }()

	for range count {
		_, execErr := stmt.Exec(
			uuid.New().String(), 100, "bulk", day.String(), string(transaction.TypeExpense),
			categoryID.String(), testServer.AuthUser.ID.String(), testServer.AuthFamily.ID.String(),
		)
		require.NoError(t, execErr)
	}

	require.NoError(t, tx.Commit())
}

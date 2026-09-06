package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

func TestTransactionHandler_Integration(t *testing.T) {
	t.Run("CreateTransaction_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		request := handlers.CreateTransactionRequest{
			AmountMinor: 15_050,
			Type:        "expense",
			Description: "Grocery shopping",
			CategoryID:  testCategory.ID,
			Date:        date.Today(time.UTC),
			Tags:        []string{"food", "essentials"},
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.AmountMinor, response.Data.AmountMinor)
		assert.Equal(t, request.Type, response.Data.Type)
		assert.Equal(t, request.Description, response.Data.Description)
		assert.Equal(t, request.CategoryID, response.Data.CategoryID)
		// Автор берётся из сессии, а не из тела запроса (S-01, задача 5 плана).
		assert.Equal(t, testServer.AuthUser.ID, response.Data.UserID)
		assert.Equal(t, request.Tags, response.Data.Tags)
		assert.NotZero(t, response.Data.ID)
		assert.NotZero(t, response.Data.CreatedAt)
		assert.NotZero(t, response.Data.UpdatedAt)
	})

	t.Run("CreateTransaction_ValidationError", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		tests := []struct {
			name    string
			request handlers.CreateTransactionRequest
			field   string
		}{
			{
				name: "negative_amount_minor",
				request: handlers.CreateTransactionRequest{
					AmountMinor: -10_000,
					Type:        "expense",
					Description: "Test transaction",
					CategoryID:  uuid.New(),
					Date:        date.Today(time.UTC),
				},
				field: "amount_minor",
			},
			{
				name: "zero_amount_minor",
				request: handlers.CreateTransactionRequest{
					AmountMinor: 0,
					Type:        "expense",
					Description: "Test transaction",
					CategoryID:  uuid.New(),
					Date:        date.Today(time.UTC),
				},
				field: "amount_minor",
			},
			{
				name: "invalid_type",
				request: handlers.CreateTransactionRequest{
					AmountMinor: 10_000,
					Type:        "invalid_type",
					Description: "Test transaction",
					CategoryID:  uuid.New(),
					Date:        date.Today(time.UTC),
				},
				field: "type",
			},
			{
				name: "empty_description",
				request: handlers.CreateTransactionRequest{
					AmountMinor: 10_000,
					Type:        "expense",
					Description: "",
					CategoryID:  uuid.New(),
					Date:        date.Today(time.UTC),
				},
				field: "description",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
				testServer.Auth(t).Apply(req)
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

				var response handlers.ErrorResponse
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.Error.Details)
				found := false
				for _, validationError := range response.Error.Details {
					if validationError.Field == tt.field {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected validation error for field %s", tt.field)
			})
		}
	})

	t.Run("CreateTransaction_WithNonExistentCategory", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Test with non-existent category - database enforces foreign key constraints
		// so this should fail with validation error
		request := handlers.CreateTransactionRequest{
			AmountMinor: 10_000,
			Type:        "expense",
			Description: "Test transaction",
			CategoryID:  uuid.New(), // non-existent category
			Date:        date.Today(time.UTC),
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// Database enforces foreign key constraints, so this should fail
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		var response handlers.ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
		assert.Contains(t, response.Error.Message, "Invalid category")
	})

	t.Run("GetTransactionByID_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testTransaction := testhelpers.CreateTestTransaction(
			family.ID, user.ID, testCategory.ID, transaction.TypeExpense,
		)
		err = testServer.Repos.Transaction.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.Equal(t, testTransaction.AmountMinor, response.Data.AmountMinor)
		assert.Equal(t, string(testTransaction.Type), response.Data.Type)
		assert.Equal(t, testTransaction.Description, response.Data.Description)
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)
		assert.Equal(t, testTransaction.UserID, response.Data.UserID)
	})

	t.Run("GetTransactionByID_NotFound", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions/%s", nonExistentID), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetTransactionByID_InvalidUUID", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions/invalid-uuid", nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetTransactions_ByFamily", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		// Create test transactions
		transaction1 := testhelpers.CreateTestTransaction(family.ID, user.ID, testCategory.ID, transaction.TypeExpense)
		transaction2 := testhelpers.CreateTestTransaction(family.ID, user.ID, testCategory.ID, transaction.TypeIncome)

		err = testServer.Repos.Transaction.Create(context.Background(), transaction1)
		require.NoError(t, err)
		err = testServer.Repos.Transaction.Create(context.Background(), transaction2)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2)

		transactionIDs := []uuid.UUID{response.Data[0].ID, response.Data[1].ID}
		assert.Contains(t, transactionIDs, transaction1.ID)
		assert.Contains(t, transactionIDs, transaction2.ID)
	})

	// Test removed: GetTransactions_MissingFamilyID - no longer relevant in single-family model

	t.Run("UpdateTransaction_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testTransaction := testhelpers.CreateTestTransaction(
			family.ID, user.ID, testCategory.ID, transaction.TypeExpense,
		)
		err = testServer.Repos.Transaction.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		newAmount := money.Minor(20_075)
		newDescription := "Updated grocery shopping"
		updateRequest := handlers.UpdateTransactionRequest{
			AmountMinor: &newAmount,
			Description: &newDescription,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.Equal(t, newAmount, response.Data.AmountMinor)
		assert.Equal(t, newDescription, response.Data.Description)
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)  // unchanged
		assert.NotEqual(t, testTransaction.UpdatedAt, response.Data.UpdatedAt) // should be updated
	})

	t.Run("UpdateTransaction_PartialUpdate", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testTransaction := testhelpers.CreateTestTransaction(
			family.ID, user.ID, testCategory.ID, transaction.TypeExpense,
		)
		err = testServer.Repos.Transaction.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		// Only update description
		newDescription := "Partially updated transaction"
		updateRequest := handlers.UpdateTransactionRequest{
			Description: &newDescription,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.Equal(t, testTransaction.AmountMinor, response.Data.AmountMinor) // unchanged
		assert.Equal(t, newDescription, response.Data.Description)              // updated
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)   // unchanged
	})

	t.Run("DeleteTransaction_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testTransaction := testhelpers.CreateTestTransaction(
			family.ID, user.ID, testCategory.ID, transaction.TypeExpense,
		)
		err = testServer.Repos.Transaction.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify transaction is deleted by trying to get it
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID), nil)
		testServer.Auth(t).Apply(getReq)
		getRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusNotFound, getRec.Code)
	})

	t.Run("DeleteTransaction_NotFound", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/transactions/%s", uuid.New()), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code, "тело: %s", rec.Body.String())
	})
}

func TestTransactionHandler_Integration_Filters(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	// Setup test data
	family := testhelpers.CreateTestFamily()
	err := testServer.Repos.Family.Create(context.Background(), family)
	require.NoError(t, err)

	user := testhelpers.CreateTestUser(family.ID)
	err = testServer.Repos.User.Create(context.Background(), user)
	require.NoError(t, err)

	expenseCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
	expenseCategory.Name = "Groceries"
	err = testServer.Repos.Category.Create(context.Background(), expenseCategory)
	require.NoError(t, err)

	incomeCategory := testhelpers.CreateTestCategory(family.ID, category.TypeIncome)
	incomeCategory.Name = "Salary"
	err = testServer.Repos.Category.Create(context.Background(), incomeCategory)
	require.NoError(t, err)

	// Create test transactions with different properties
	expenseTransaction := testhelpers.CreateTestTransaction(
		family.ID, user.ID, expenseCategory.ID, transaction.TypeExpense,
	)
	expenseTransaction.AmountMinor = 10_000
	expenseTransaction.Date = date.Today(time.UTC).AddDays(-1) // yesterday
	err = testServer.Repos.Transaction.Create(context.Background(), expenseTransaction)
	require.NoError(t, err)

	incomeTransaction := testhelpers.CreateTestTransaction(
		family.ID, user.ID, incomeCategory.ID, transaction.TypeIncome,
	)
	incomeTransaction.AmountMinor = 50_000
	incomeTransaction.Date = date.Today(time.UTC).AddDays(-2) // 2 days ago
	err = testServer.Repos.Transaction.Create(context.Background(), incomeTransaction)
	require.NoError(t, err)

	t.Run("GetTransactions_ByType", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/transactions?family_id=%s&type=expense", family.ID),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, "expense", response.Data[0].Type)
		assert.Equal(t, expenseTransaction.ID, response.Data[0].ID)
	})

	t.Run("GetTransactions_ByCategory", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/transactions?family_id=%s&category_id=%s", family.ID, incomeCategory.ID),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, incomeCategory.ID, response.Data[0].CategoryID)
		assert.Equal(t, incomeTransaction.ID, response.Data[0].ID)
	})

	t.Run("GetTransactions_ByDateRange", func(t *testing.T) {
		// Use a consistent time reference to avoid timing issues in CI
		today := date.Today(time.UTC)
		dateFrom := today.AddDays(-3).String() // 3 дня назад
		dateTo := today.String()               // сегодня — обе операции внутри диапазона
		query := url.Values{}
		query.Set("family_id", family.ID.String())
		query.Set("date_from", dateFrom)
		query.Set("date_to", dateTo)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/transactions?"+query.Encode(),
			nil,
		)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2) // both transactions should be in this range
	})
}

// TestTransactionAPI_MinorUnitsLifecycle — полный цикл на копейках: три операции по 33
// копейки складываются в 99, а не в 0.99 с потерей, и переживают фильтр и отчёт.
func TestTransactionAPI_MinorUnitsLifecycle(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(time.UTC)
	for i := range 3 {
		body := mustJSON(t, map[string]any{
			"amount_minor": 33,
			"type":         "expense",
			"description":  fmt.Sprintf("Копейка %d", i),
			"category_id":  testCategory.ID,
			"date":         today,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())
	}

	query := url.Values{}
	query.Set("date_from", today.String())
	query.Set("date_to", today.String())
	query.Set("amount_from_minor", "33")
	query.Set("amount_to_minor", "33")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/transactions?"+query.Encode(), nil)
	session.Apply(listReq)
	listRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code, "тело: %s", listRec.Body.String())

	var list handlers.APIResponse[[]handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &list))
	require.Len(t, list.Data, 3)

	var total money.Minor
	for _, tx := range list.Data {
		assert.Equal(t, money.Minor(33), tx.AmountMinor)
		assert.Equal(t, today, tx.Date)
		total += tx.AmountMinor
	}
	assert.Equal(t, money.Minor(99), total)

	reportBody := mustJSON(t, map[string]any{
		"name":       "Копейки",
		"type":       "expenses",
		"period":     "custom",
		"start_date": today.String(),
		"end_date":   today.String(),
	})
	reportReq := httptest.NewRequest(http.MethodPost, "/api/v1/reports", bytes.NewBuffer(reportBody))
	reportReq.Header.Set("Content-Type", "application/json")
	session.Apply(reportReq)
	reportRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(reportRec, reportReq)
	require.Equal(t, http.StatusCreated, reportRec.Code, "тело: %s", reportRec.Body.String())

	var created struct {
		Data struct {
			Data report.Data `json:"data"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(reportRec.Body.Bytes(), &created))
	assert.Equal(t, money.Minor(99), created.Data.Data.TotalExpensesMinor)
}

// TestTransactionAPI_CreateWithClientID_Idempotent — повтор POST с тем же id
// возвращает ту же запись и не плодит вторую (A-07).
func TestTransactionAPI_CreateWithClientID_Idempotent(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	clientID := uuid.New()
	body := mustJSON(t, map[string]any{
		"id":           clientID,
		"amount_minor": 12_345,
		"type":         "expense",
		"description":  "Повторяемая операция",
		"category_id":  testCategory.ID,
		"date":         date.Today(time.UTC),
	})

	post := func(t *testing.T) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	first := post(t)
	require.Equal(t, http.StatusCreated, first.Code, "тело: %s", first.Body.String())

	second := post(t)
	require.Equal(t, http.StatusOK, second.Code, "тело: %s", second.Body.String())

	var repeated handlers.APIResponse[handlers.TransactionResponse]
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &repeated))
	assert.Equal(t, clientID, repeated.Data.ID)
	assert.Equal(t, money.Minor(12_345), repeated.Data.AmountMinor)

	stored, err := testServer.Repos.Transaction.GetByFilter(ctx, transaction.Filter{Limit: 100})
	require.NoError(t, err)
	assert.Len(t, stored, 1, "повтор POST не должен создавать вторую запись")
}

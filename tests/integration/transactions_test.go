package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/testhelpers"
)

func TestTransactionHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateTransaction_Success", func(t *testing.T) {
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
			Amount:      150.50,
			Type:        "expense",
			Description: "Grocery shopping",
			CategoryID:  testCategory.ID,
			UserID:      user.ID,
			FamilyID:    family.ID,
			Date:        time.Now(),
			Tags:        []string{"food", "essentials"},
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.InEpsilon(t, request.Amount, response.Data.Amount, 0.001)
		assert.Equal(t, request.Type, response.Data.Type)
		assert.Equal(t, request.Description, response.Data.Description)
		assert.Equal(t, request.CategoryID, response.Data.CategoryID)
		assert.Equal(t, request.UserID, response.Data.UserID)
		assert.Equal(t, request.FamilyID, response.Data.FamilyID)
		assert.Equal(t, request.Tags, response.Data.Tags)
		assert.NotZero(t, response.Data.ID)
		assert.NotZero(t, response.Data.CreatedAt)
		assert.NotZero(t, response.Data.UpdatedAt)
	})

	t.Run("CreateTransaction_ValidationError", func(t *testing.T) {
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
				name: "negative_amount",
				request: handlers.CreateTransactionRequest{
					Amount:      -100.0,
					Type:        "expense",
					Description: "Test transaction",
					CategoryID:  uuid.New(),
					UserID:      uuid.New(),
					FamilyID:    family.ID,
					Date:        time.Now(),
				},
				field: "Amount",
			},
			{
				name: "invalid_type",
				request: handlers.CreateTransactionRequest{
					Amount:      100.0,
					Type:        "invalid_type",
					Description: "Test transaction",
					CategoryID:  uuid.New(),
					UserID:      uuid.New(),
					FamilyID:    family.ID,
					Date:        time.Now(),
				},
				field: "Type",
			},
			{
				name: "empty_description",
				request: handlers.CreateTransactionRequest{
					Amount:      100.0,
					Type:        "expense",
					Description: "",
					CategoryID:  uuid.New(),
					UserID:      uuid.New(),
					FamilyID:    family.ID,
					Date:        time.Now(),
				},
				field: "Description",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				testServer.Server.Echo().ServeHTTP(rec, req)

				assert.Equal(t, http.StatusBadRequest, rec.Code)

				var response handlers.APIResponse[any]
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.Errors)
				found := false
				for _, validationError := range response.Errors {
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
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		user := testhelpers.CreateTestUser(family.ID)
		err = testServer.Repos.User.Create(context.Background(), user)
		require.NoError(t, err)

		// Test with non-existent category - MongoDB doesn't enforce foreign key constraints
		// so this should succeed (this is by design in the current system)
		request := handlers.CreateTransactionRequest{
			Amount:      100.0,
			Type:        "expense",
			Description: "Test transaction",
			CategoryID:  uuid.New(), // non-existent category
			UserID:      user.ID,
			FamilyID:    family.ID,
			Date:        time.Now(),
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(requestBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// MongoDB doesn't enforce foreign key constraints, so this succeeds
		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.CategoryID, response.Data.CategoryID)
		assert.Equal(t, request.UserID, response.Data.UserID)
		assert.Equal(t, request.FamilyID, response.Data.FamilyID)
	})

	t.Run("GetTransactionByID_Success", func(t *testing.T) {
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
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.InEpsilon(t, testTransaction.Amount, response.Data.Amount, 0.001)
		assert.Equal(t, string(testTransaction.Type), response.Data.Type)
		assert.Equal(t, testTransaction.Description, response.Data.Description)
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)
		assert.Equal(t, testTransaction.UserID, response.Data.UserID)
		assert.Equal(t, testTransaction.FamilyID, response.Data.FamilyID)
	})

	t.Run("GetTransactionByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions/%s", nonExistentID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetTransactionByID_InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions/invalid-uuid", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetTransactions_ByFamily", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions?family_id=%s", family.ID), nil)
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

		for _, txn := range response.Data {
			assert.Equal(t, family.ID, txn.FamilyID)
		}
	})

	t.Run("GetTransactions_MissingFamilyID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("UpdateTransaction_Success", func(t *testing.T) {
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

		newAmount := 200.75
		newDescription := "Updated grocery shopping"
		updateRequest := handlers.UpdateTransactionRequest{
			Amount:      &newAmount,
			Description: &newDescription,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.InEpsilon(t, newAmount, response.Data.Amount, 0.001)
		assert.Equal(t, newDescription, response.Data.Description)
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)  // unchanged
		assert.NotEqual(t, testTransaction.UpdatedAt, response.Data.UpdatedAt) // should be updated
	})

	t.Run("UpdateTransaction_PartialUpdate", func(t *testing.T) {
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
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testTransaction.ID, response.Data.ID)
		assert.InEpsilon(t, testTransaction.Amount, response.Data.Amount, 0.001) // unchanged
		assert.Equal(t, newDescription, response.Data.Description)               // updated
		assert.Equal(t, testTransaction.CategoryID, response.Data.CategoryID)    // unchanged
	})

	t.Run("DeleteTransaction_Success", func(t *testing.T) {
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

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify transaction is deleted by trying to get it
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/transactions/%s", testTransaction.ID), nil)
		getRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusNotFound, getRec.Code)
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
	expenseTransaction.Amount = 100.0
	expenseTransaction.Date = time.Now().AddDate(0, 0, -1) // yesterday
	err = testServer.Repos.Transaction.Create(context.Background(), expenseTransaction)
	require.NoError(t, err)

	incomeTransaction := testhelpers.CreateTestTransaction(
		family.ID, user.ID, incomeCategory.ID, transaction.TypeIncome,
	)
	incomeTransaction.Amount = 500.0
	incomeTransaction.Date = time.Now().AddDate(0, 0, -2) // 2 days ago
	err = testServer.Repos.Transaction.Create(context.Background(), incomeTransaction)
	require.NoError(t, err)

	t.Run("GetTransactions_ByType", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/transactions?family_id=%s&type=expense", family.ID),
			nil,
		)
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
		dateFrom := time.Now().AddDate(0, 0, -3).Format(time.RFC3339) // 3 days ago
		dateTo := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)   // yesterday

		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/transactions?family_id=%s&date_from=%s&date_to=%s", family.ID, dateFrom, dateTo),
			nil,
		)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.TransactionResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2) // both transactions should be in this range
	})
}

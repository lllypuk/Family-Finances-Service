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
	"family-budget-service/internal/testhelpers"
)

func TestBudgetHandler_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateBudget_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		startDate := time.Now()
		endDate := startDate.AddDate(0, 1, 0) // one month later

		request := handlers.CreateBudgetRequest{
			Name:       "Monthly Groceries Budget",
			Amount:     800.00,
			Period:     "monthly",
			CategoryID: &testCategory.ID,
			FamilyID:   family.ID,
			StartDate:  startDate,
			EndDate:    endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Name, response.Data.Name)
		assert.InEpsilon(t, request.Amount, response.Data.Amount, 0.001)
		assert.Equal(t, request.Period, response.Data.Period)
		assert.Equal(t, *request.CategoryID, *response.Data.CategoryID)
		assert.Zero(t, response.Data.Spent)
		assert.InEpsilon(t, request.Amount, response.Data.Remaining, 0.001)
		assert.True(t, response.Data.IsActive)
		assert.NotZero(t, response.Data.ID)
		assert.NotZero(t, response.Data.CreatedAt)
		assert.NotZero(t, response.Data.UpdatedAt)
	})

	t.Run("CreateBudget_ValidationError", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		tests := []struct {
			name    string
			request handlers.CreateBudgetRequest
			field   string
		}{
			{
				name: "negative_amount",
				request: handlers.CreateBudgetRequest{
					Name:   "Test Budget",
					Amount: -100.0,
					Period: "monthly",

					StartDate: time.Now(),
					EndDate:   time.Now().AddDate(0, 1, 0),
				},
				field: "Amount",
			},
			{
				name: "invalid_period",
				request: handlers.CreateBudgetRequest{
					Name:   "Test Budget",
					Amount: 100.0,
					Period: "invalid_period",

					StartDate: time.Now(),
					EndDate:   time.Now().AddDate(0, 1, 0),
				},
				field: "Period",
			},
			{
				name: "empty_name",
				request: handlers.CreateBudgetRequest{
					Name:   "",
					Amount: 100.0,
					Period: "monthly",

					StartDate: time.Now(),
					EndDate:   time.Now().AddDate(0, 1, 0),
				},
				field: "Name",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
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

	t.Run("CreateBudget_DateValidation", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		// Create budget where start_date > end_date
		// Note: This test validates that the budget can be created even with invalid date logic
		// since date validation is not implemented at the handler level
		startDate := time.Now()
		endDate := startDate.AddDate(0, -1, 0) // one month earlier

		request := handlers.CreateBudgetRequest{
			Name:   "Date Test Budget",
			Amount: 100.0,
			Period: "monthly",

			StartDate: startDate,
			EndDate:   endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		// Date validation is implemented at repository level, so this should fail
		if rec.Code != http.StatusInternalServerError {
			t.Logf("Date validation test failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		var response handlers.ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have CREATE_FAILED error for invalid date range
		assert.Equal(t, "CREATE_FAILED", response.Error.Code)
		assert.Equal(t, "Failed to create budget", response.Error.Message)
	})

	t.Run("GetBudgetByID_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		err = testServer.Repos.Budget.Create(context.Background(), testBudget)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testBudget.ID, response.Data.ID)
		assert.Equal(t, testBudget.Name, response.Data.Name)
		assert.InEpsilon(t, testBudget.Amount, response.Data.Amount, 0.001)
		assert.Equal(t, string(testBudget.Period), response.Data.Period)
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)
		assert.Equal(t, testBudget.IsActive, response.Data.IsActive)
	})

	t.Run("GetBudgetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets/%s", nonExistentID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetBudgetByID_InvalidUUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/invalid-uuid", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetBudgets_ByFamily", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		// Create test budgets
		budget1 := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		budget1.Name = "Budget 1"
		budget2 := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		budget2.Name = "Budget 2"

		err = testServer.Repos.Budget.Create(context.Background(), budget1)
		require.NoError(t, err)
		err = testServer.Repos.Budget.Create(context.Background(), budget2)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets?family_id=%s", family.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 2)

		budgetIDs := []uuid.UUID{response.Data[0].ID, response.Data[1].ID}
		assert.Contains(t, budgetIDs, budget1.ID)
		assert.Contains(t, budgetIDs, budget2.ID)

		for _, budget := range response.Data {
			assert.Equal(t, family.ID, budget.FamilyID)
		}
	})

	t.Run("GetBudgets_MissingFamilyID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets", nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetBudgets_ActiveOnly", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		// Create active and inactive budgets
		activeBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		activeBudget.Name = "Active Budget"
		activeBudget.IsActive = true

		inactiveBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		inactiveBudget.Name = "Inactive Budget"
		inactiveBudget.IsActive = false

		err = testServer.Repos.Budget.Create(context.Background(), activeBudget)
		require.NoError(t, err)
		err = testServer.Repos.Budget.Create(context.Background(), inactiveBudget)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/api/v1/budgets?family_id=%s&active_only=true", family.ID),
			nil,
		)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[[]handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, activeBudget.ID, response.Data[0].ID)
		assert.True(t, response.Data[0].IsActive)
	})

	t.Run("UpdateBudget_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		err = testServer.Repos.Budget.Create(context.Background(), testBudget)
		require.NoError(t, err)

		newName := "Updated Budget Name"
		newAmount := 1500.00
		updateRequest := handlers.UpdateBudgetRequest{
			Name:   &newName,
			Amount: &newAmount,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Update budget failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testBudget.ID, response.Data.ID)
		assert.Equal(t, newName, response.Data.Name)
		assert.InEpsilon(t, newAmount, response.Data.Amount, 0.001)
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)  // unchanged
		assert.NotEqual(t, testBudget.UpdatedAt, response.Data.UpdatedAt) // should be updated
	})

	t.Run("UpdateBudget_PartialUpdate", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		err = testServer.Repos.Budget.Create(context.Background(), testBudget)
		require.NoError(t, err)

		// Only update name
		newName := "Partially Updated Budget"
		updateRequest := handlers.UpdateBudgetRequest{
			Name: &newName,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Partial update budget failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testBudget.ID, response.Data.ID)
		assert.Equal(t, newName, response.Data.Name)                        // updated
		assert.InEpsilon(t, testBudget.Amount, response.Data.Amount, 0.001) // unchanged
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)    // unchanged
	})

	t.Run("UpdateBudget_ToggleActive", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		testBudget.IsActive = true
		err = testServer.Repos.Budget.Create(context.Background(), testBudget)
		require.NoError(t, err)

		// Toggle active status
		newIsActive := false
		updateRequest := handlers.UpdateBudgetRequest{
			IsActive: &newIsActive,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Logf("Toggle active budget failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testBudget.ID, response.Data.ID)
		assert.Equal(t, newIsActive, response.Data.IsActive)                // updated
		assert.Equal(t, testBudget.Name, response.Data.Name)                // unchanged
		assert.InEpsilon(t, testBudget.Amount, response.Data.Amount, 0.001) // unchanged
	})

	t.Run("DeleteBudget_Success", func(t *testing.T) {
		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		err = testServer.Repos.Budget.Create(context.Background(), testBudget)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID), nil)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Logf("Delete budget failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify budget is soft deleted (is_active = false) by getting it and checking status
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID), nil)
		getRec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(getRec, getReq)

		assert.Equal(t, http.StatusOK, getRec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(getRec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Budget should be marked as inactive after soft delete
		assert.False(t, response.Data.IsActive)
	})
}

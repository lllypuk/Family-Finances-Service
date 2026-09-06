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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

func TestBudgetHandler_Integration(t *testing.T) {
	t.Run("CreateBudget_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		testCategory := testhelpers.CreateTestCategory(family.ID, category.TypeExpense)
		err = testServer.Repos.Category.Create(context.Background(), testCategory)
		require.NoError(t, err)

		startDate := date.Today(time.UTC)
		endDate := startDate.AddDays(30)

		request := handlers.CreateBudgetRequest{
			Name:        "Monthly Groceries Budget",
			AmountMinor: 80_000,
			Period:      "monthly",
			CategoryID:  &testCategory.ID,
			StartDate:   startDate,
			EndDate:     endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, request.Name, response.Data.Name)
		assert.Equal(t, request.AmountMinor, response.Data.AmountMinor)
		assert.Equal(t, request.Period, response.Data.Period)
		assert.Equal(t, *request.CategoryID, *response.Data.CategoryID)
		assert.Zero(t, response.Data.SpentMinor)
		assert.Equal(t, request.AmountMinor, response.Data.RemainingMinor)
		assert.True(t, response.Data.IsActive)
		assert.NotZero(t, response.Data.ID)
		assert.NotZero(t, response.Data.CreatedAt)
		assert.NotZero(t, response.Data.UpdatedAt)
	})

	t.Run("CreateBudget_ValidationError", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
				name: "negative_amount_minor",
				request: handlers.CreateBudgetRequest{
					Name:        "Test Budget",
					AmountMinor: -10_000,
					Period:      "monthly",
					StartDate:   date.Today(time.UTC),
					EndDate:     date.Today(time.UTC).AddDays(30),
				},
				field: "amount_minor",
			},
			{
				name: "zero_amount_minor",
				request: handlers.CreateBudgetRequest{
					Name:        "Test Budget",
					AmountMinor: 0,
					Period:      "monthly",
					StartDate:   date.Today(time.UTC),
					EndDate:     date.Today(time.UTC).AddDays(30),
				},
				field: "amount_minor",
			},
			{
				name: "invalid_period",
				request: handlers.CreateBudgetRequest{
					Name:        "Test Budget",
					AmountMinor: 10_000,
					Period:      "invalid_period",
					StartDate:   date.Today(time.UTC),
					EndDate:     date.Today(time.UTC).AddDays(30),
				},
				field: "period",
			},
			{
				name: "empty_name",
				request: handlers.CreateBudgetRequest{
					Name:        "",
					AmountMinor: 10_000,
					Period:      "monthly",
					StartDate:   date.Today(time.UTC),
					EndDate:     date.Today(time.UTC).AddDays(30),
				},
				field: "name",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				requestBodyBytes, err := json.Marshal(tt.request)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
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

	t.Run("CreateBudget_DateValidation", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		// Setup test data
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		// Порядок дат — ошибка ввода, а не сбой: 422, не 500.
		startDate := date.Today(time.UTC)
		endDate := startDate.AddDays(-30)

		request := handlers.CreateBudgetRequest{
			Name:        "Date Test Budget",
			AmountMinor: 10_000,
			Period:      "monthly",
			StartDate:   startDate,
			EndDate:     endDate,
		}

		requestBodyBytes, err := json.Marshal(request)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(requestBodyBytes))
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

		var response handlers.ErrorResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "VALIDATION_ERROR", response.Error.Code)
	})

	t.Run("GetBudgetByID_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, testBudget.ID, response.Data.ID)
		assert.Equal(t, testBudget.Name, response.Data.Name)
		assert.Equal(t, testBudget.AmountMinor, response.Data.AmountMinor)
		assert.Equal(t, string(testBudget.Period), response.Data.Period)
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)
		assert.Equal(t, testBudget.IsActive, response.Data.IsActive)
	})

	t.Run("GetBudgetByID_NotFound", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		nonExistentID := uuid.New()

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets/%s", nonExistentID), nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GetBudgetByID_InvalidUUID", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/invalid-uuid", nil)
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetBudgets_ByFamily", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
	})

	// Test removed: GetBudgets_MissingFamilyID - no longer relevant in single-family model

	t.Run("GetBudgets_ActiveOnly", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
		testServer := testhelpers.SetupHTTPServer(t)

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
		newAmount := money.Minor(150_000)
		updateRequest := handlers.UpdateBudgetRequest{
			Name:        &newName,
			AmountMinor: &newAmount,
		}

		requestBodyBytes, err := json.Marshal(updateRequest)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID),
			bytes.NewBuffer(requestBodyBytes),
		)
		testServer.Auth(t).Apply(req)
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
		assert.Equal(t, newAmount, response.Data.AmountMinor)
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)  // unchanged
		assert.NotEqual(t, testBudget.UpdatedAt, response.Data.UpdatedAt) // should be updated
	})

	t.Run("UpdateBudget_PartialUpdate", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
		assert.Equal(t, newName, response.Data.Name)                       // updated
		assert.Equal(t, testBudget.AmountMinor, response.Data.AmountMinor) // unchanged
		assert.Equal(t, testBudget.CategoryID, response.Data.CategoryID)   // unchanged
	})

	t.Run("UpdateBudget_ToggleActive", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
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
		assert.Equal(t, newIsActive, response.Data.IsActive)               // updated
		assert.Equal(t, testBudget.Name, response.Data.Name)               // unchanged
		assert.Equal(t, testBudget.AmountMinor, response.Data.AmountMinor) // unchanged
	})

	t.Run("DeleteBudget_Success", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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
		testServer.Auth(t).Apply(req)
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Logf("Delete budget failed with status %d, response: %s", rec.Code, rec.Body.String())
		}
		assert.Equal(t, http.StatusNoContent, rec.Code)

		// Verify budget is soft deleted (is_active = false) by getting it and checking status
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID), nil)
		testServer.Auth(t).Apply(getReq)
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

// TestBudgetAPI_CreateWithClientID_Idempotent — повтор POST с тем же id отдаёт
// существующий бюджет, а не создаёт второй (A-07).
func TestBudgetAPI_CreateWithClientID_Idempotent(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	clientID := uuid.New()
	today := date.Today(time.UTC)
	body := mustJSON(t, map[string]any{
		"id":           clientID,
		"name":         "Повторяемый бюджет",
		"amount_minor": 100_000,
		"period":       "monthly",
		"category_id":  testCategory.ID,
		"start_date":   today.String(),
		"end_date":     today.AddDays(30).String(),
	})

	post := func(t *testing.T) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
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

	var repeated handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &repeated))
	assert.Equal(t, clientID, repeated.Data.ID)
	assert.Equal(t, money.Minor(100_000), repeated.Data.AmountMinor)

	stored, err := testServer.Repos.Budget.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, stored, 1, "повтор POST не должен создавать второй бюджет")
}

// TestCategoryAPI_CreateWithClientID_Idempotent — то же для категорий.
func TestCategoryAPI_CreateWithClientID_Idempotent(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	clientID := uuid.New()
	body := mustJSON(t, map[string]any{
		"id":    clientID,
		"name":  "Повторяемая категория",
		"type":  "expense",
		"color": "#FF00FF",
		"icon":  "repeat",
	})

	post := func(t *testing.T) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewBuffer(body))
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

	var repeated handlers.APIResponse[dto.CategoryAPIResponse]
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &repeated))
	assert.Equal(t, clientID, repeated.Data.ID)

	stored, err := testServer.Repos.Category.GetAll(context.Background())
	require.NoError(t, err)
	matching := 0
	for _, c := range stored {
		if c.Name == "Повторяемая категория" {
			matching++
		}
	}
	assert.Equal(t, 1, matching, "повтор POST не должен создавать вторую категорию")
}

// TestBudgetAPI_UtilizationIsPercent — utilization на /budgets — проценты 0…100,
// в отличие от доли 0…1 в /stats/summary. Ошибка в 100 раз должна ронять тест.
func TestBudgetAPI_UtilizationIsPercent(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	body := mustJSON(t, map[string]any{
		"name":         "Бюджет с тратами",
		"amount_minor": 100_000,
		"period":       "monthly",
		"category_id":  testCategory.ID,
		"start_date":   today.String(),
		"end_date":     today.AddDays(30).String(),
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
	createReq.Header.Set("Content-Type", "application/json")
	session.Apply(createReq)
	createRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code, "тело: %s", createRec.Body.String())

	var created handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))

	txBody := mustJSON(t, map[string]any{
		"amount_minor": 25_000,
		"type":         "expense",
		"description":  "Трата в бюджете",
		"category_id":  testCategory.ID,
		"date":         today.String(),
	})
	txReq := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", bytes.NewBuffer(txBody))
	txReq.Header.Set("Content-Type", "application/json")
	session.Apply(txReq)
	txRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(txRec, txReq)
	require.Equal(t, http.StatusCreated, txRec.Code, "тело: %s", txRec.Body.String())

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/"+created.Data.ID.String(), nil)
	session.Apply(getReq)
	getRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code, "тело: %s", getRec.Body.String())

	var fetched handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &fetched))
	assert.Equal(t, money.Minor(25_000), fetched.Data.SpentMinor)
	assert.Equal(t, money.Minor(75_000), fetched.Data.RemainingMinor)
	assert.InDelta(t, 25.0, fetched.Data.Utilization, 0.001)
}

// TestBudgetAPI_CreateInvalidStartDate — неразобранная дата в теле называет своё поле,
// а не общее "date".
func TestBudgetAPI_CreateInvalidStartDate(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	body := []byte(`{"name":"Плохая дата","amount_minor":100,"period":"monthly",` +
		`"start_date":"2026-13-01","end_date":"2026-12-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "start_date", response.Error.Details[0].Field)
}

// TestBudgetAPI_CreateMissingStartDate — пропущенная дата отбивается валидатором (422),
// а не доезжает до репозитория как 500: `required` на date.Date работает только с
// validator.WithRequiredStructEnabled.
func TestBudgetAPI_CreateMissingStartDate(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	body := []byte(`{"name":"Без начала","amount_minor":100,"period":"monthly","end_date":"2026-12-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "start_date", response.Error.Details[0].Field)
}

// TestBudgetAPI_CreateAmountAboveMaximum — сумма выше потолка Money из openapi отбивается
// сервисом (422), а не валидацией репозитория (500).
func TestBudgetAPI_CreateAmountAboveMaximum(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)

	today := date.Today(time.UTC)
	body := mustJSON(t, map[string]any{
		"name":         "Слишком большой",
		"amount_minor": 100_000_000_000,
		"period":       "monthly",
		"start_date":   today.String(),
		"end_date":     today.AddDays(30).String(),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	session.Apply(req)
	rec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())
}

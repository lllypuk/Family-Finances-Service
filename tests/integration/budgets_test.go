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
	"family-budget-service/internal/domain/transaction"
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
		budget2.StartDate = budget1.EndDate.AddDays(1)
		budget2.EndDate = budget2.StartDate.AddDays(30)

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
		inactiveBudget.StartDate = activeBudget.EndDate.AddDays(1)
		inactiveBudget.EndDate = inactiveBudget.StartDate.AddDays(30)

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

	t.Run("UpdateBudget_IgnoresIsActive", func(t *testing.T) {
		testServer := testhelpers.SetupHTTPServer(t)

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

		body := `{"name":"Renamed Budget","is_active":false}`

		req := httptest.NewRequest(
			http.MethodPut,
			fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID),
			bytes.NewBufferString(body),
		)
		testServer.Auth(t).Apply(req)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testServer.Server.Echo().ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var response handlers.APIResponse[handlers.BudgetResponse]
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Renamed Budget", response.Data.Name)
		assert.True(t, response.Data.IsActive)
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

		// Удалённый бюджет невидим по id для всех трёх методов
		path := fmt.Sprintf("/api/v1/budgets/%s", testBudget.ID)
		for _, tc := range []struct {
			method string
			body   string
		}{
			{http.MethodGet, ""},
			{http.MethodPut, `{"name":"Renamed Budget"}`},
			{http.MethodDelete, ""},
		} {
			afterReq := httptest.NewRequest(tc.method, path, bytes.NewBufferString(tc.body))
			testServer.Auth(t).Apply(afterReq)
			afterReq.Header.Set("Content-Type", "application/json")
			afterRec := httptest.NewRecorder()

			testServer.Server.Echo().ServeHTTP(afterRec, afterReq)

			assert.Equal(t, http.StatusNotFound, afterRec.Code, "%s %s: %s", tc.method, path, afterRec.Body.String())
		}
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

	var created, repeated handlers.APIResponse[dto.CategoryAPIResponse]
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &created))
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &repeated))
	assert.Equal(t, clientID, repeated.Data.ID)
	assert.Equal(t, created.Data, repeated.Data, "повтор обязан отдать то же тело, что и 201")
	assert.Equal(t, "#FF00FF", repeated.Data.Color)
	assert.Equal(t, "repeat", repeated.Data.Icon)

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

// TestBudgetAPI_CreateDuplicateNameSamePeriod — имя занято на этот период, пусть и в другой
// категории: индекс имени не знает о категориях, поэтому занятость проверяется второй веткой
// предиката. Код отличается от пересечения области: здесь поможет только переименование.
func TestBudgetAPI_CreateDuplicateNameSamePeriod(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	first := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, first))

	second := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	second.Name = "Другая категория"
	require.NoError(t, testServer.Repos.Category.Create(ctx, second))

	today := date.Today(time.UTC)
	post := func(categoryID uuid.UUID) *httptest.ResponseRecorder {
		body := mustJSON(t, map[string]any{
			"name":         "Один и тот же бюджет",
			"amount_minor": 50_000,
			"period":       "monthly",
			"category_id":  categoryID.String(),
			"start_date":   today.String(),
			"end_date":     today.AddDays(30).String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	require.Equal(t, http.StatusCreated, post(first.ID).Code)

	rec := post(second.ID)
	require.Equal(t, http.StatusConflict, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetNameExists, response.Error.Code)
	assert.Empty(t, response.Error.Details)
}

// TestBudgetAPI_CreateOverlappingPeriod_Conflict — пересечение периодов одной категории
// это бизнес-отказ (409), а не ошибка формы.
func TestBudgetAPI_CreateOverlappingPeriod_Conflict(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	post := func(name string, start, end date.Date) *httptest.ResponseRecorder {
		body := mustJSON(t, map[string]any{
			"name":         name,
			"amount_minor": 50_000,
			"period":       "monthly",
			"category_id":  testCategory.ID,
			"start_date":   start.String(),
			"end_date":     end.String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	require.Equal(t, http.StatusCreated, post("Первый", today, today.AddDays(30)).Code)

	rec := post("Второй", today.AddDays(10), today.AddDays(40))
	require.Equal(t, http.StatusConflict, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetOverlap, response.Error.Code)
	assert.Empty(t, response.Error.Details)
}

// TestBudgetAPI_UpdateAmountBelowSpent_Conflict — сумма ниже уже потраченного за период → 409.
func TestBudgetAPI_UpdateAmountBelowSpent_Conflict(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	body := mustJSON(t, map[string]any{
		"name":         "Бюджет с расходом",
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
		"amount_minor": 40_000,
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

	updBody := mustJSON(t, map[string]any{"amount_minor": 10_000})
	updReq := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/budgets/"+created.Data.ID.String(),
		bytes.NewBuffer(updBody),
	)
	updReq.Header.Set("Content-Type", "application/json")
	session.Apply(updReq)
	updRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(updRec, updReq)

	require.Equal(t, http.StatusConflict, updRec.Code, "тело: %s", updRec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(updRec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetBelowSpent, response.Error.Code)
	assert.Empty(t, response.Error.Details)
}

// TestBudgetAPI_CreateWithDeletedID_Conflict — id мягко удалённого бюджета занят навсегда:
// идемпотентность его не видит (GetByID фильтрует is_active), а PRIMARY KEY занят. Отказ
// должен называть свою причину, а не «имя занято» — переименование тут не помогает.
func TestBudgetAPI_CreateWithDeletedID_Conflict(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	clientID := uuid.New()
	today := date.Today(testServer.AuthFamily.Location())
	post := func(name string) *httptest.ResponseRecorder {
		body := mustJSON(t, map[string]any{
			"id":           clientID,
			"name":         name,
			"amount_minor": 100_000,
			"period":       "monthly",
			"category_id":  testCategory.ID,
			"start_date":   today.String(),
			"end_date":     today.AddDays(30).String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	require.Equal(t, http.StatusCreated, post("Удаляемый бюджет").Code)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/budgets/"+clientID.String(), nil)
	session.Apply(delReq)
	delRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusNoContent, delRec.Code, "тело: %s", delRec.Body.String())

	// Другое имя и другие даты: остаётся один конфликт — по id.
	rec := post("Другое имя")
	require.Equal(t, http.StatusConflict, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetIDExists, response.Error.Code)
}

// TestBudgetAPI_RecreateDeletedNameAndPeriod — имя и период мягко удалённого бюджета
// освобождаются вместе с ним: иначе клиент получал бы 409 на бюджет, которого не видит,
// и «создать заново» после удаления было бы невозможно.
func TestBudgetAPI_RecreateDeletedNameAndPeriod(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	post := func(id uuid.UUID) *httptest.ResponseRecorder {
		body := mustJSON(t, map[string]any{
			"id":           id,
			"name":         "Повторяемый бюджет",
			"amount_minor": 70_000,
			"period":       "monthly",
			"category_id":  testCategory.ID,
			"start_date":   today.String(),
			"end_date":     today.AddDays(30).String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	firstID := uuid.New()
	require.Equal(t, http.StatusCreated, post(firstID).Code)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/budgets/"+firstID.String(), nil)
	session.Apply(delReq)
	delRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusNoContent, delRec.Code, "тело: %s", delRec.Body.String())

	rec := post(uuid.New())
	require.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())
}

// TestBudgetAPI_CreateSharedBoundaryDay_Conflict — включительные границы проверяются через
// настоящий SQL: предикат занятости обязан отдать соседа, у которого общий с новым только
// один день. Юнит-тесты сервиса этот запрос мокают.
func TestBudgetAPI_CreateSharedBoundaryDay_Conflict(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	post := func(name string, start, end date.Date) *httptest.ResponseRecorder {
		body := mustJSON(t, map[string]any{
			"name":         name,
			"amount_minor": 50_000,
			"period":       "monthly",
			"category_id":  testCategory.ID,
			"start_date":   start.String(),
			"end_date":     end.String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	require.Equal(t, http.StatusCreated, post("Первый", today, today.AddDays(29)).Code)

	shared := post("Общий день", today.AddDays(29), today.AddDays(59))
	require.Equal(t, http.StatusConflict, shared.Code, "общий день не считается пересечением: %s",
		shared.Body.String())
	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(shared.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetOverlap, response.Error.Code)

	next := post("Следующий день", today.AddDays(30), today.AddDays(60))
	assert.Equal(t, http.StatusCreated, next.Code, "период со следующего дня — не пересечение: %s",
		next.Body.String())
}

// TestBudgetAPI_UpdateToTakenName_Conflict — переименование в занятое имя проверяется тем же
// предикатом, что и INSERT: репозиторий обязан отдать тот же сентинел, иначе UPDATE вернёт 500.
func TestBudgetAPI_UpdateToTakenName_Conflict(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	first := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, first))
	second := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	second.Name = "Вторая категория"
	require.NoError(t, testServer.Repos.Category.Create(ctx, second))

	today := date.Today(testServer.AuthFamily.Location())
	create := func(name string, categoryID uuid.UUID) handlers.BudgetResponse {
		body := mustJSON(t, map[string]any{
			"name":         name,
			"amount_minor": 50_000,
			"period":       "monthly",
			"category_id":  categoryID.String(),
			"start_date":   today.String(),
			"end_date":     today.AddDays(30).String(),
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		session.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())

		var created handlers.APIResponse[handlers.BudgetResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

		return created.Data
	}

	taken := create("Занятое имя", first.ID)
	renamed := create("Переименуемый", second.ID)

	putBody := mustJSON(t, map[string]any{"name": taken.Name})
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/budgets/"+renamed.ID.String(),
		bytes.NewBuffer(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	session.Apply(putReq)
	putRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(putRec, putReq)

	require.Equal(t, http.StatusConflict, putRec.Code, "тело: %s", putRec.Body.String())
	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(putRec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetNameExists, response.Error.Code)
}

// TestBudgetAPI_UpdateEmptyBody — тело без единого поля: minProperties: 1 в контракте,
// значит 422, а не молчаливая запись с новым updated_at.
func TestBudgetAPI_UpdateEmptyBody(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	ctx := context.Background()

	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(ctx, testCategory))

	today := date.Today(testServer.AuthFamily.Location())
	body := mustJSON(t, map[string]any{
		"name":         "Бюджет без правок",
		"amount_minor": 50_000,
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

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/budgets/"+created.Data.ID.String(),
		bytes.NewBufferString(`{}`))
	putReq.Header.Set("Content-Type", "application/json")
	session.Apply(putReq)
	putRec := httptest.NewRecorder()
	testServer.Server.Echo().ServeHTTP(putRec, putReq)

	require.Equal(t, http.StatusUnprocessableEntity, putRec.Code, "тело: %s", putRec.Body.String())
	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(putRec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
}

// recurringAPI — сервер, сессия и категория для сценариев серии: проверяется только контракт,
// материализация идёт на чтении списка, как в бою.
type recurringAPI struct {
	t        *testing.T
	server   *testhelpers.TestServer
	session  *testhelpers.AuthSession
	category *category.Category
	today    date.Date
}

func newRecurringAPI(t *testing.T) *recurringAPI {
	t.Helper()

	testServer := testhelpers.SetupHTTPServer(t)
	session := testServer.Auth(t)
	testCategory := testhelpers.CreateTestCategory(testServer.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, testServer.Repos.Category.Create(context.Background(), testCategory))

	return &recurringAPI{
		t:        t,
		server:   testServer,
		session:  session,
		category: testCategory,
		today:    date.Today(testServer.AuthFamily.Location()),
	}
}

func (a *recurringAPI) do(method, path string, body map[string]any) *httptest.ResponseRecorder {
	a.t.Helper()

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBuffer(mustJSON(a.t, body)))
		req.Header.Set("Content-Type", "application/json")
	}
	a.session.Apply(req)
	rec := httptest.NewRecorder()
	a.server.Server.Echo().ServeHTTP(rec, req)

	return rec
}

// monthOf — границы месяца today+offset. Шаг делается от первого числа: AddMonths не
// зажимает день, и 31-го числа сдвиг назад уехал бы на месяц вперёд.
func (a *recurringAPI) monthOf(offsetMonths int) (date.Date, date.Date) {
	first, _ := a.today.MonthBounds()

	return first.AddMonths(offsetMonths).MonthBounds()
}

// createRecurring заводит хвост серии на месяц today+offset: даты выровнены по календарю.
func (a *recurringAPI) createRecurring(name string, offsetMonths int) handlers.BudgetResponse {
	a.t.Helper()

	start, end := a.monthOf(offsetMonths)
	rec := a.do(http.MethodPost, "/api/v1/budgets", map[string]any{
		"name":         name,
		"amount_minor": 50_000,
		"period":       "monthly",
		"category_id":  a.category.ID,
		"start_date":   start.String(),
		"end_date":     end.String(),
		"recurring":    true,
	})
	require.Equal(a.t, http.StatusCreated, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(a.t, json.Unmarshal(rec.Body.Bytes(), &response))

	return response.Data
}

func (a *recurringAPI) list(query string) []handlers.BudgetResponse {
	a.t.Helper()

	rec := a.do(http.MethodGet, "/api/v1/budgets"+query, nil)
	require.Equal(a.t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.APIResponse[[]handlers.BudgetResponse]
	require.NoError(a.t, json.Unmarshal(rec.Body.Bytes(), &response))

	return response.Data
}

func (a *recurringAPI) put(id uuid.UUID, body map[string]any) *httptest.ResponseRecorder {
	a.t.Helper()

	return a.do(http.MethodPut, "/api/v1/budgets/"+id.String(), body)
}

func (a *recurringAPI) spend(amount money.Minor, on date.Date) {
	a.t.Helper()

	tx := testhelpers.CreateTestTransaction(
		a.server.AuthFamily.ID, a.server.AuthUser.ID, a.category.ID, transaction.TypeExpense,
	)
	tx.AmountMinor = amount
	tx.Date = on
	require.NoError(a.t, a.server.Repos.Transaction.Create(context.Background(), tx))
}

// TestBudgetAPI_RecurringSeries_MaterializesOnRead — серия, заведённая три месяца назад,
// достраивается на чтении: хвост — текущий месяц со своим расходом, прошлые инстансы остаются
// историей с тем же series_id.
func TestBudgetAPI_RecurringSeries_MaterializesOnRead(t *testing.T) {
	api := newRecurringAPI(t)

	const seriesMonthsBack = -3
	created := api.createRecurring("Продукты", seriesMonthsBack)
	require.True(t, created.Recurring)
	require.NotNil(t, created.SeriesID)
	require.Equal(t, created.ID, *created.SeriesID)

	api.spend(12_000, api.today)

	active := api.list("?active_only=true")
	require.Len(t, active, 1)
	currentStart, _ := api.today.MonthBounds()
	assert.Equal(t, currentStart, active[0].StartDate)
	assert.True(t, active[0].Recurring)
	require.NotNil(t, active[0].SeriesID)
	assert.Equal(t, created.ID, *active[0].SeriesID)
	assert.Equal(t, money.Minor(12_000), active[0].SpentMinor)
	assert.Equal(t, created.AmountMinor, active[0].AmountMinor)

	all := api.list("")
	require.Len(t, all, -seriesMonthsBack+1)
	for _, b := range all {
		require.NotNil(t, b.SeriesID)
		assert.Equal(t, created.ID, *b.SeriesID)
		assert.Equal(t, b.ID == active[0].ID, b.Recurring, "флаг только у хвоста")
	}

	statsRec := api.do(http.MethodGet, "/api/v1/stats/summary", nil)
	require.Equal(t, http.StatusOK, statsRec.Code, "тело: %s", statsRec.Body.String())

	var stats handlers.APIResponse[dto.StatsSummary]
	require.NoError(t, json.Unmarshal(statsRec.Body.Bytes(), &stats))
	require.Len(t, stats.Data.Budgets, 1)
	assert.Equal(t, active[0].ID, stats.Data.Budgets[0].ID)
	assert.Equal(t, money.Minor(12_000), stats.Data.Budgets[0].SpentMinor)
}

// TestBudgetAPI_RecurringSeries_StopAndResume — снятый флаг останавливает серию, возвращённый
// достраивает пропущенные месяцы.
func TestBudgetAPI_RecurringSeries_StopAndResume(t *testing.T) {
	api := newRecurringAPI(t)

	const seriesMonthsBack = -3
	created := api.createRecurring("Продукты", seriesMonthsBack)

	stopRec := api.put(created.ID, map[string]any{"recurring": false})
	require.Equal(t, http.StatusOK, stopRec.Code, "тело: %s", stopRec.Body.String())

	var stopped handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(stopRec.Body.Bytes(), &stopped))
	assert.False(t, stopped.Data.Recurring)

	assert.Len(t, api.list(""), 1, "остановленная серия не растёт")

	resumeRec := api.put(created.ID, map[string]any{"recurring": true})
	require.Equal(t, http.StatusOK, resumeRec.Code, "тело: %s", resumeRec.Body.String())

	assert.Len(t, api.list(""), -seriesMonthsBack+1)
}

// TestBudgetAPI_RecurringSeries_StopUnlocksDates — серия из одного инстанса при остановке
// распускается: иначе series_id навсегда запретил бы правку дат обычного бюджета.
func TestBudgetAPI_RecurringSeries_StopUnlocksDates(t *testing.T) {
	api := newRecurringAPI(t)

	created := api.createRecurring("Продукты", 0)

	stopRec := api.put(created.ID, map[string]any{"recurring": false})
	require.Equal(t, http.StatusOK, stopRec.Code, "тело: %s", stopRec.Body.String())

	var stopped handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(stopRec.Body.Bytes(), &stopped))
	assert.False(t, stopped.Data.Recurring)
	assert.Nil(t, stopped.Data.SeriesID)

	start, end := api.monthOf(-6)
	moveRec := api.put(created.ID, map[string]any{"start_date": start.String(), "end_date": end.String()})
	require.Equal(t, http.StatusOK, moveRec.Code, "тело: %s", moveRec.Body.String())

	var moved handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(moveRec.Body.Bytes(), &moved))
	assert.Equal(t, start, moved.Data.StartDate)
}

// TestBudgetAPI_RecurringSeries_StaleTailConflict — форма прошлого инстанса устарела: серия
// продвинулась, и снимать флаг клиенту нужно с нового хвоста.
func TestBudgetAPI_RecurringSeries_StaleTailConflict(t *testing.T) {
	api := newRecurringAPI(t)

	const seriesMonthsBack = -3
	created := api.createRecurring("Продукты", seriesMonthsBack)
	require.Len(t, api.list(""), -seriesMonthsBack+1)

	rec := api.put(created.ID, map[string]any{"recurring": false})
	require.Equal(t, http.StatusConflict, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeBudgetNotTail, response.Error.Code)

	tail := api.list("?active_only=true")
	require.Len(t, tail, 1)
	retry := api.put(tail[0].ID, map[string]any{"recurring": false})
	require.Equal(t, http.StatusOK, retry.Code, "тело: %s", retry.Body.String())

	var stopped handlers.APIResponse[handlers.BudgetResponse]
	require.NoError(t, json.Unmarshal(retry.Body.Bytes(), &stopped))
	assert.False(t, stopped.Data.Recurring)
}

// TestBudgetAPI_RecurringSeries_SkipsManualMonth — ручной бюджет, заведённый на месяц серии
// заранее, занимает период: серия перешагивает его и материализует следующий со своим лимитом.
func TestBudgetAPI_RecurringSeries_SkipsManualMonth(t *testing.T) {
	api := newRecurringAPI(t)

	const seriesMonthsBack = -2
	created := api.createRecurring("Продукты", seriesMonthsBack)

	manualStart, manualEnd := api.monthOf(-1)
	const manualAmount = money.Minor(10_000)
	manualRec := api.do(http.MethodPost, "/api/v1/budgets", map[string]any{
		"name":         "Ручной",
		"amount_minor": int64(manualAmount),
		"period":       "monthly",
		"category_id":  api.category.ID,
		"start_date":   manualStart.String(),
		"end_date":     manualEnd.String(),
	})
	require.Equal(t, http.StatusCreated, manualRec.Code, "тело: %s", manualRec.Body.String())

	active := api.list("?active_only=true")
	require.Len(t, active, 1)
	currentStart, _ := api.today.MonthBounds()
	assert.Equal(t, currentStart, active[0].StartDate)
	assert.Equal(t, created.AmountMinor, active[0].AmountMinor, "лимит серии, а не ручного бюджета")
	assert.True(t, active[0].Recurring)
	require.NotNil(t, active[0].SeriesID)
	assert.Equal(t, created.ID, *active[0].SeriesID)

	all := api.list("")
	require.Len(t, all, 3, "хвост, ручной месяц и новый инстанс")
	for _, b := range all {
		if b.Name == "Ручной" {
			assert.Nil(t, b.SeriesID)
			assert.Equal(t, manualAmount, b.AmountMinor)
		}
	}
}

// TestBudgetAPI_RecurringSeries_MemberDatesFixed — перенос прошлого инстанса за хвост сломал бы
// поиск хвоста по MAX(start_date), поэтому даты члена серии неизменны.
func TestBudgetAPI_RecurringSeries_MemberDatesFixed(t *testing.T) {
	api := newRecurringAPI(t)

	created := api.createRecurring("Продукты", -1)
	require.Len(t, api.list(""), 2)

	start, end := api.monthOf(-6)
	rec := api.put(created.ID, map[string]any{"start_date": start.String(), "end_date": end.String()})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())

	var response handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
	require.Len(t, response.Error.Details, 1)
	assert.Equal(t, "start_date", response.Error.Details[0].Field)
}

// TestBudgetAPI_CreateRecurring_Rejected — период без длины повторять нечем, невыровненные даты
// разъехались бы с календарным шагом серии.
func TestBudgetAPI_CreateRecurring_Rejected(t *testing.T) {
	api := newRecurringAPI(t)

	start, end := api.today.MonthBounds()
	tests := []struct {
		name   string
		period string
		start  date.Date
		end    date.Date
		field  string
	}{
		{name: "custom", period: "custom", start: start, end: end, field: "recurring"},
		{name: "not_aligned", period: "monthly", start: start.AddDays(1), end: end, field: "start_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := api.do(http.MethodPost, "/api/v1/budgets", map[string]any{
				"name":         "Продукты " + tt.name,
				"amount_minor": 50_000,
				"period":       tt.period,
				"category_id":  api.category.ID,
				"start_date":   tt.start.String(),
				"end_date":     tt.end.String(),
				"recurring":    true,
			})
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "тело: %s", rec.Body.String())

			var response handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, handlers.ErrCodeValidationError, response.Error.Code)
			require.Len(t, response.Error.Details, 1)
			assert.Equal(t, tt.field, response.Error.Details[0].Field)
		})
	}
}

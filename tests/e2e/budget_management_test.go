package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// TestBudgetManagementWorkflow tests complete budget management scenarios
func TestBudgetManagementWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e budget management test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID, budgetID string

	// Setup test data
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Budget Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create user
		userData := map[string]any{
			"email":      "budget@test.com",
			"password":   "password123",
			"first_name": "Budget",
			"last_name":  "Tester",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		userID = extractIDFromResponse(userResponse)

		// Create category
		categoryData := map[string]any{
			"name":      "Food",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#FF5733",
			"icon":      "utensils",
		}
		body, _ = json.Marshal(categoryData)
		resp, err = http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoryResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&categoryResponse)
		require.NoError(t, err)
		categoryID = extractIDFromResponse(categoryResponse)
	})

	t.Run("BudgetLifecycle", func(t *testing.T) {
		// Step 1: Create Budget
		t.Run("CreateBudget", func(t *testing.T) {
			budgetData := map[string]any{
				"name":        "Food Budget",
				"amount":      500.0,
				"period":      "monthly",
				"category_id": categoryID,
				"family_id":   familyID,
				"start_date":  time.Now().Format(time.RFC3339),
				"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
			}

			body, _ := json.Marshal(budgetData)
			resp, err := http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			budgetID = extractIDFromResponse(response)
			assert.NotEmpty(t, budgetID)

			// Extract budget data from nested response
			budgetData, ok := response["data"].(map[string]any)
			require.True(t, ok, "Response should contain data field")
			assert.InDelta(t, 500.0, budgetData["amount"], 0.01)
			assert.InDelta(t, 0.0, budgetData["spent"], 0.01)
		})

		// Step 2: Add transactions and track spending
		t.Run("TrackSpending", func(t *testing.T) {
			transactions := []map[string]any{
				{
					"amount":      100.0,
					"type":        "expense",
					"description": "Groceries Week 1",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      75.0,
					"type":        "expense",
					"description": "Restaurant Dinner",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      150.0,
					"type":        "expense",
					"description": "Groceries Week 2",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
			}

			totalSpent := 0.0
			for _, transactionData := range transactions {
				body, _ := json.Marshal(transactionData)
				resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusCreated, resp.StatusCode)
				totalSpent += transactionData["amount"].(float64)
			}

			// Verify budget reflects spending
			resp, err := http.Get(fmt.Sprintf("%s/budgets/%s", baseURL, budgetID))
			require.NoError(t, err)
			defer resp.Body.Close()

			var budget map[string]any
			err = json.NewDecoder(resp.Body).Decode(&budget)
			require.NoError(t, err)

			// Extract budget data from nested response
			budgetData, ok := budget["data"].(map[string]any)
			require.True(t, ok, "Budget response should contain data field")

			spent := budgetData["spent"].(float64)
			assert.InDelta(t, totalSpent, spent, 0.01, "Budget should reflect total spending")
			assert.InDelta(t, 175.0, budgetData["remaining"].(float64), 0.01, "Remaining should be 500 - 325 = 175")
		})

		// Step 3: Test budget over-spending
		t.Run("TestOverSpending", func(t *testing.T) {
			overSpendTransaction := map[string]any{
				"amount":      200.0, // This will put us over budget (325 + 200 = 525 > 500)
				"type":        "expense",
				"description": "Emergency Food Purchase",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
			}

			body, _ := json.Marshal(overSpendTransaction)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			// Check budget status after over-spending
			resp, err = http.Get(fmt.Sprintf("%s/budgets/%s", baseURL, budgetID))
			require.NoError(t, err)
			defer resp.Body.Close()

			var budget map[string]any
			err = json.NewDecoder(resp.Body).Decode(&budget)
			require.NoError(t, err)

			spent := budget["spent"].(float64)
			remaining := budget["remaining"].(float64)
			assert.InEpsilon(t, 525.0, spent, 0.01, "Total spent should be 525")
			assert.InEpsilon(t, -25.0, remaining, 0.01, "Should be 25 over budget")
		})

		// Step 4: Update Budget
		t.Run("UpdateBudget", func(t *testing.T) {
			updateData := map[string]any{
				"amount": 600.0, // Increase budget to accommodate spending
			}

			body, _ := json.Marshal(updateData)
			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("%s/budgets/%s", baseURL, budgetID),
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var budget map[string]any
			err = json.NewDecoder(resp.Body).Decode(&budget)
			require.NoError(t, err)

			assert.InEpsilon(t, 600.0, budget["amount"], 0.01)
			assert.InEpsilon(t, 525.0, budget["spent"], 0.01)
			assert.InEpsilon(t, 75.0, budget["remaining"], 0.01, "Should now have 75 remaining")
		})
	})
}

// TestMultipleBudgetCategories tests managing budgets across multiple categories
func TestMultipleBudgetCategories(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e multiple budget test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID string
	var categoryIDs []string
	var budgetIDs []string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Multi-Budget Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create user
		userData := map[string]any{
			"email":      "multi@budget.com",
			"password":   "password123",
			"first_name": "Multi",
			"last_name":  "Budget",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		userID = extractIDFromResponse(userResponse)

		// Create multiple categories
		categories := []map[string]any{
			{"name": "Food", "type": "expense", "family_id": familyID, "color": "#FF5733", "icon": "utensils"},
			{"name": "Transportation", "type": "expense", "family_id": familyID, "color": "#33FF57", "icon": "car"},
			{"name": "Entertainment", "type": "expense", "family_id": familyID, "color": "#3357FF", "icon": "film"},
			{"name": "Utilities", "type": "expense", "family_id": familyID, "color": "#FF33F5", "icon": "bolt"},
		}

		for _, categoryData := range categories {
			body, _ := json.Marshal(categoryData)
			resp, err := http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			var categoryResponse map[string]any
			err = json.NewDecoder(resp.Body).Decode(&categoryResponse)
			require.NoError(t, err)
			categoryIDs = append(categoryIDs, extractIDFromResponse(categoryResponse))
		}
	})

	t.Run("CreateMultipleBudgets", func(t *testing.T) {
		if len(categoryIDs) < 4 {
			t.Fatalf("Expected at least 4 category IDs, got %d. Setup test may have failed.", len(categoryIDs))
		}

		budgets := []map[string]any{
			{"name": "Food Budget", "amount": 600.0, "period": "monthly", "category_id": categoryIDs[0], "family_id": familyID, "start_date": time.Now().Format(time.RFC3339), "end_date": time.Now().AddDate(0, 1, 0).Format(time.RFC3339)}, // Food
			{
				"name":        "Transportation Budget",
				"amount":      300.0,
				"period":      "monthly",
				"category_id": categoryIDs[1],
				"family_id":   familyID,
				"start_date":  time.Now().Format(time.RFC3339),
				"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
			}, // Transportation
			{
				"name":        "Entertainment Budget",
				"amount":      200.0,
				"period":      "monthly",
				"category_id": categoryIDs[2],
				"family_id":   familyID,
				"start_date":  time.Now().Format(time.RFC3339),
				"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
			}, // Entertainment
			{"name": "Utilities Budget", "amount": 400.0, "period": "monthly", "category_id": categoryIDs[3], "family_id": familyID, "start_date": time.Now().Format(time.RFC3339), "end_date": time.Now().AddDate(0, 1, 0).Format(time.RFC3339)}, // Utilities
		}

		for _, budgetData := range budgets {
			body, _ := json.Marshal(budgetData)
			resp, err := http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)
			budgetIDs = append(budgetIDs, extractIDFromResponse(response))
		}

		assert.Len(t, budgetIDs, 4)
	})

	t.Run("DistributeTransactions", func(t *testing.T) {
		transactions := []map[string]any{
			// Food transactions
			{
				"amount":      120.0,
				"type":        "expense",
				"description": "Groceries",
				"category_id": categoryIDs[0],
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      80.0,
				"type":        "expense",
				"description": "Restaurant",
				"category_id": categoryIDs[0],
				"user_id":     userID,
				"family_id":   familyID,
			},
			// Transportation transactions
			{
				"amount":      60.0,
				"type":        "expense",
				"description": "Gas",
				"category_id": categoryIDs[1],
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      120.0,
				"type":        "expense",
				"description": "Car Maintenance",
				"category_id": categoryIDs[1],
				"user_id":     userID,
				"family_id":   familyID,
			},
			// Entertainment transactions
			{
				"amount":      45.0,
				"type":        "expense",
				"description": "Movies",
				"category_id": categoryIDs[2],
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      25.0,
				"type":        "expense",
				"description": "Games",
				"category_id": categoryIDs[2],
				"user_id":     userID,
				"family_id":   familyID,
			},
			// Utilities transactions
			{
				"amount":      150.0,
				"type":        "expense",
				"description": "Electricity",
				"category_id": categoryIDs[3],
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      100.0,
				"type":        "expense",
				"description": "Internet",
				"category_id": categoryIDs[3],
				"user_id":     userID,
				"family_id":   familyID,
			},
		}

		for _, transactionData := range transactions {
			body, _ := json.Marshal(transactionData)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}
	})

	t.Run("VerifyBudgetDistribution", func(t *testing.T) {
		expectedSpending := []float64{200.0, 180.0, 70.0, 250.0} // Food, Transportation, Entertainment, Utilities
		expectedRemaining := []float64{400.0, 120.0, 130.0, 150.0}

		for i, budgetID := range budgetIDs {
			resp, err := http.Get(fmt.Sprintf("%s/budgets/%s", baseURL, budgetID))
			require.NoError(t, err)
			defer resp.Body.Close()

			var budget map[string]any
			err = json.NewDecoder(resp.Body).Decode(&budget)
			require.NoError(t, err)

			spent := budget["spent"].(float64)
			remaining := budget["remaining"].(float64)

			assert.InEpsilon(t, expectedSpending[i], spent, 0.01, "Budget %d spending mismatch", i)
			assert.InEpsilon(t, expectedRemaining[i], remaining, 0.01, "Budget %d remaining mismatch", i)
		}
	})

	t.Run("FamilyBudgetSummary", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/budgets?family_id=%s", baseURL, familyID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var budgets []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&budgets)
		require.NoError(t, err)

		assert.Len(t, budgets, 4)

		totalBudgeted := 0.0
		totalSpent := 0.0
		totalRemaining := 0.0

		for _, budget := range budgets {
			totalBudgeted += budget["amount"].(float64)
			totalSpent += budget["spent"].(float64)
			totalRemaining += budget["remaining"].(float64)
		}

		assert.InDelta(t, 1500.0, totalBudgeted, 0.01, "Total budgeted should be 1500")
		assert.InDelta(t, 700.0, totalSpent, 0.01, "Total spent should be 700")
		assert.InDelta(t, 800.0, totalRemaining, 0.01, "Total remaining should be 800")
	})
}

// TestBudgetPeriods tests different budget periods (monthly, yearly)
func TestBudgetPeriods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e budget periods test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, categoryID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Period Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create user
		userData := map[string]any{
			"email":      "period@test.com",
			"password":   "password123",
			"first_name": "Period",
			"last_name":  "Test",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		_ = extractIDFromResponse(userResponse) // userID not used elsewhere

		// Create category
		categoryData := map[string]any{
			"name":      "Vacation",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#33A5FF",
			"icon":      "plane",
		}
		body, _ = json.Marshal(categoryData)
		resp, err = http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoryResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&categoryResponse)
		require.NoError(t, err)
		categoryID = extractIDFromResponse(categoryResponse)
	})

	t.Run("MonthlyBudget", func(t *testing.T) {
		budgetData := map[string]any{
			"name":        "Monthly Vacation Budget",
			"amount":      1000.0,
			"period":      "monthly",
			"category_id": categoryID,
			"family_id":   familyID,
			"start_date":  time.Now().Format(time.RFC3339),
			"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
		}

		body, _ := json.Marshal(budgetData)
		resp, err := http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "monthly", response["period"])
		assert.InDelta(t, 1000.0, response["amount"], 0.01)
	})

	t.Run("YearlyBudget", func(t *testing.T) {
		// Delete the monthly budget first
		resp, err := http.Get(fmt.Sprintf("%s/budgets?family_id=%s", baseURL, familyID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var budgets []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&budgets)
		require.NoError(t, err)

		if len(budgets) > 0 {
			deleteBudgetID := budgets[0]["id"].(string)
			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/budgets/%s", baseURL, deleteBudgetID), nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
		}

		// Create yearly budget
		budgetData := map[string]any{
			"name":        "Yearly Vacation Budget",
			"amount":      12000.0,
			"period":      "yearly",
			"category_id": categoryID,
			"family_id":   familyID,
			"start_date":  time.Now().Format(time.RFC3339),
			"end_date":    time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
		}

		body, _ := json.Marshal(budgetData)
		resp, err = http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "yearly", response["period"])
		assert.InDelta(t, 12000.0, response["amount"], 0.01)
	})
}

// TestBudgetReporting tests budget-related reporting
func TestBudgetReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e budget reporting test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID string
	var budgetID string // Will be assigned in nested test functions
	_ = budgetID        // Used in nested test scopes

	// Setup with budget and transactions
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Reporting Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID = extractIDFromResponse(familyResponse)

		// Create user
		userData := map[string]any{
			"email":      "reporting@test.com",
			"password":   "password123",
			"first_name": "Reporting",
			"last_name":  "Test",
			"role":       "admin",
			"family_id":  familyID,
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var userResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&userResponse)
		require.NoError(t, err)
		userID = extractIDFromResponse(userResponse)

		// Create category
		categoryData := map[string]any{
			"name":      "Shopping",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#FF3377",
			"icon":      "shopping-bag",
		}
		body, _ = json.Marshal(categoryData)
		resp, err = http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoryResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&categoryResponse)
		require.NoError(t, err)
		categoryID = extractIDFromResponse(categoryResponse)

		// Create budget
		budgetData := map[string]any{
			"name":        "Shopping Budget",
			"amount":      800.0,
			"period":      "monthly",
			"category_id": categoryID,
			"family_id":   familyID,
			"start_date":  time.Now().Format(time.RFC3339),
			"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
		}
		body, _ = json.Marshal(budgetData)
		resp, err = http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var budgetResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&budgetResponse)
		require.NoError(t, err)
		budgetID = budgetResponse["id"].(string)

		// Add transactions
		transactions := []map[string]any{
			{
				"amount":      150.0,
				"type":        "expense",
				"description": "Clothes",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      200.0,
				"type":        "expense",
				"description": "Electronics",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
			},
			{
				"amount":      100.0,
				"type":        "expense",
				"description": "Books",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
			},
		}

		for _, transactionData := range transactions {
			body, _ := json.Marshal(transactionData)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()
		}
	})

	t.Run("GenerateBudgetReport", func(t *testing.T) {
		reportData := map[string]any{
			"type":      "budget_analysis",
			"family_id": familyID,
			"date_from": time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
			"date_to":   time.Now().Format("2006-01-02"),
		}

		body, _ := json.Marshal(reportData)
		resp, err := http.Post(baseURL+"/reports", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response map[string]any
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		reportID := extractIDFromResponse(response)
		assert.NotEmpty(t, reportID)

		// Retrieve and verify report
		resp, err = http.Get(fmt.Sprintf("%s/reports/%s", baseURL, reportID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var report map[string]any
		err = json.NewDecoder(resp.Body).Decode(&report)
		require.NoError(t, err)

		assert.Equal(t, "budget_analysis", report["type"])
		assert.Equal(t, familyID, report["family_id"])
	})
}

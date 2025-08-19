package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// TestCompleteAPIWorkflow tests end-to-end API workflow
func TestCompleteAPIWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Setup test server
	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	// Test data
	familyData := map[string]any{
		"name":     "Test Family",
		"currency": "USD",
	}

	userData := map[string]any{
		"email":      "test@example.com",
		"password":   "password123",
		"first_name": "John",
		"last_name":  "Doe",
		"role":       "admin",
	}

	categoryData := map[string]any{
		"name": "Food",
		"type": "expense",
	}

	transactionData := map[string]any{
		"amount":      100.50,
		"type":        "expense",
		"description": "Grocery shopping",
	}

	budgetData := map[string]any{
		"amount": 500.0,
		"period": "monthly",
	}

	t.Run("CompleteWorkflow", func(t *testing.T) {
		var familyID, userID, categoryID, transactionID, budgetID string

		// Step 1: Create Family
		t.Run("CreateFamily", func(t *testing.T) {
			body, _ := json.Marshal(familyData)
			resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			familyID = response["id"].(string)
			assert.NotEmpty(t, familyID)
		})

		// Step 2: Create User
		t.Run("CreateUser", func(t *testing.T) {
			userData["family_id"] = familyID
			body, _ := json.Marshal(userData)
			resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			userID = response["id"].(string)
			assert.NotEmpty(t, userID)
		})

		// Step 3: Create Category
		t.Run("CreateCategory", func(t *testing.T) {
			categoryData["family_id"] = familyID
			body, _ := json.Marshal(categoryData)
			resp, err := http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			categoryID = response["id"].(string)
			assert.NotEmpty(t, categoryID)
		})

		// Step 4: Create Transaction
		t.Run("CreateTransaction", func(t *testing.T) {
			transactionData["category_id"] = categoryID
			transactionData["user_id"] = userID
			transactionData["family_id"] = familyID
			body, _ := json.Marshal(transactionData)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			transactionID = response["id"].(string)
			assert.NotEmpty(t, transactionID)
		})

		// Step 5: Create Budget
		t.Run("CreateBudget", func(t *testing.T) {
			budgetData["category_id"] = categoryID
			budgetData["family_id"] = familyID
			body, _ := json.Marshal(budgetData)
			resp, err := http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			budgetID = response["id"].(string)
			assert.NotEmpty(t, budgetID)
		})

		// Step 6: Generate Report
		t.Run("GenerateReport", func(t *testing.T) {
			reportData := map[string]any{
				"type":      "expense_summary",
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

			reportID := response["id"].(string)
			assert.NotEmpty(t, reportID)
		})

		// Step 7: Verify Data Integrity
		t.Run("VerifyDataIntegrity", func(t *testing.T) {
			// Check family exists
			resp, err := http.Get(fmt.Sprintf("%s/families/%s", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check user exists
			resp, err = http.Get(fmt.Sprintf("%s/users/%s", baseURL, userID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check category exists
			resp, err = http.Get(fmt.Sprintf("%s/categories/%s", baseURL, categoryID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check transaction exists
			resp, err = http.Get(fmt.Sprintf("%s/transactions/%s", baseURL, transactionID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Check budget exists
			resp, err = http.Get(fmt.Sprintf("%s/budgets/%s", baseURL, budgetID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Step 8: Test Filtering and Pagination
		t.Run("TestFiltering", func(t *testing.T) {
			// Test transaction filtering by family
			resp, err := http.Get(fmt.Sprintf("%s/transactions?family_id=%s", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var transactions []map[string]any
			err = json.NewDecoder(resp.Body).Decode(&transactions)
			require.NoError(t, err)
			assert.Len(t, transactions, 1)
			assert.Equal(t, transactionID, transactions[0]["id"])
		})

		// Step 9: Cleanup (Test Delete Operations)
		t.Run("Cleanup", func(t *testing.T) {
			// Delete transaction
			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/transactions/%s", baseURL, transactionID), nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Delete budget
			req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/budgets/%s", baseURL, budgetID), nil)
			resp, err = http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Delete category
			req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/categories/%s", baseURL, categoryID), nil)
			resp, err = http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Delete user
			req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/users/%s", baseURL, userID), nil)
			resp, err = http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})
}

// TestConcurrentAPIAccess tests concurrent access to API
func TestConcurrentAPIAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e concurrent test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	// Create test family first
	familyData := map[string]any{
		"name":     "Concurrent Test Family",
		"currency": "USD",
	}

	body, _ := json.Marshal(familyData)
	resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var familyResponse map[string]any
	err = json.NewDecoder(resp.Body).Decode(&familyResponse)
	require.NoError(t, err)
	familyID := familyResponse["id"].(string)

	// Test concurrent user creation
	t.Run("ConcurrentUserCreation", func(t *testing.T) {
		const numGoroutines = 10
		results := make(chan error, numGoroutines)

		for i := range numGoroutines {
			go func(index int) {
				userData := map[string]any{
					"email":      fmt.Sprintf("user%d@example.com", index),
					"password":   "password123",
					"first_name": fmt.Sprintf("User%d", index),
					"last_name":  "Test",
					"role":       "member",
					"family_id":  familyID,
				}

				body, _ := json.Marshal(userData)
				resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					results <- fmt.Errorf("expected status 201, got %d", resp.StatusCode)
					return
				}

				results <- nil
			}(i)
		}

		// Collect results
		for range numGoroutines {
			err := <-results
			assert.NoError(t, err)
		}
	})
}

// TestDataConsistency tests data consistency across operations
func TestDataConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e consistency test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	t.Run("BudgetTransactionConsistency", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Consistency Test Family",
			"currency": "USD",
		}
		body, _ := json.Marshal(familyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var familyResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&familyResponse)
		require.NoError(t, err)
		familyID := familyResponse["id"].(string)

		// Create user
		userData := map[string]any{
			"email":      "consistency@example.com",
			"password":   "password123",
			"first_name": "Consistency",
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
		userID := userResponse["id"].(string)

		// Create category
		categoryData := map[string]any{
			"name":      "Food",
			"type":      "expense",
			"family_id": familyID,
		}
		body, _ = json.Marshal(categoryData)
		resp, err = http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoryResponse map[string]any
		err = json.NewDecoder(resp.Body).Decode(&categoryResponse)
		require.NoError(t, err)
		categoryID := categoryResponse["id"].(string)

		// Create budget
		budgetData := map[string]any{
			"amount":      500.0,
			"period":      "monthly",
			"category_id": categoryID,
			"family_id":   familyID,
		}
		body, _ = json.Marshal(budgetData)
		resp, err = http.Post(baseURL+"/budgets", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Create multiple transactions to test budget consistency
		transactions := []float64{100.0, 150.0, 200.0}
		for i, amount := range transactions {
			transactionData := map[string]any{
				"amount":      amount,
				"type":        "expense",
				"description": fmt.Sprintf("Transaction %d", i+1),
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
			}

			body, _ = json.Marshal(transactionData)
			resp, err = http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
			resp.Body.Close()
		}

		// Verify budget reflects all transactions
		resp, err = http.Get(fmt.Sprintf("%s/budgets?family_id=%s", baseURL, familyID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var budgets []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&budgets)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)

		budget := budgets[0]
		spent := budget["spent"].(float64)
		expectedSpent := 450.0 // 100 + 150 + 200
		assert.InEpsilon(t, expectedSpent, spent, 0.001, "Budget spent amount should reflect all transactions")
	})
}

// TestErrorHandling tests error scenarios in e2e workflow
func TestErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e error handling test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	t.Run("InvalidDataHandling", func(t *testing.T) {
		// Test invalid family creation
		invalidFamilyData := map[string]any{
			"name": "", // Empty name should fail
		}
		body, _ := json.Marshal(invalidFamilyData)
		resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Test creating user with non-existent family
		userData := map[string]any{
			"email":      "orphan@example.com",
			"password":   "password123",
			"first_name": "Orphan",
			"last_name":  "User",
			"role":       "member",
			"family_id":  uuid.New().String(), // Non-existent family
		}
		body, _ = json.Marshal(userData)
		resp, err = http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("NotFoundHandling", func(t *testing.T) {
		// Test accessing non-existent resources
		nonExistentID := uuid.New().String()

		endpoints := []string{
			fmt.Sprintf("/families/%s", nonExistentID),
			fmt.Sprintf("/users/%s", nonExistentID),
			fmt.Sprintf("/categories/%s", nonExistentID),
			fmt.Sprintf("/transactions/%s", nonExistentID),
			fmt.Sprintf("/budgets/%s", nonExistentID),
		}

		for _, endpoint := range endpoints {
			resp, err := http.Get(baseURL + endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Endpoint: %s", endpoint)
		}
	})
}

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

// TestTransactionWorkflow tests complete transaction management workflow
func TestTransactionWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e transaction workflow test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID, transactionID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create family
		familyData := map[string]any{
			"name":     "Transaction Test Family",
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
			"email":      "transaction@test.com",
			"password":   "TransactionPass123!",
			"first_name": "Transaction",
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
			"name":      "Groceries",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#FF5733",
			"icon":      "shopping-cart",
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

	t.Run("TransactionLifecycle", func(t *testing.T) {
		// Step 1: Create Transaction
		t.Run("CreateTransaction", func(t *testing.T) {
			transactionData := map[string]any{
				"amount":      125.50,
				"type":        "expense",
				"description": "Weekly grocery shopping",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
				"date":        time.Now().Format(time.RFC3339),
				"tags":        []string{"grocery", "weekly", "food"},
			}

			body, _ := json.Marshal(transactionData)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			transactionID = extractIDFromResponse(response)
			assert.NotEmpty(t, transactionID)
			assert.InDelta(t, 125.50, response["amount"], 0.01)
			assert.Equal(t, "expense", response["type"])
			assert.Equal(t, "Weekly grocery shopping", response["description"])
		})

		// Step 2: Retrieve Transaction
		t.Run("RetrieveTransaction", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/transactions/%s", baseURL, transactionID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var transaction map[string]any
			err = json.NewDecoder(resp.Body).Decode(&transaction)
			require.NoError(t, err)

			assert.Equal(t, transactionID, transaction["id"])
			assert.InDelta(t, 125.50, transaction["amount"], 0.01)
			assert.Equal(t, categoryID, transaction["category_id"])
			assert.Equal(t, userID, transaction["user_id"])
			assert.Equal(t, familyID, transaction["family_id"])
		})

		// Step 3: Update Transaction
		t.Run("UpdateTransaction", func(t *testing.T) {
			updateData := map[string]any{
				"amount":      135.75,
				"description": "Weekly grocery shopping - updated amount",
				"tags":        []string{"grocery", "weekly", "food", "organic"},
			}

			body, _ := json.Marshal(updateData)
			req, _ := http.NewRequest(
				http.MethodPut,
				fmt.Sprintf("%s/transactions/%s", baseURL, transactionID),
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var transaction map[string]any
			err = json.NewDecoder(resp.Body).Decode(&transaction)
			require.NoError(t, err)

			assert.InEpsilon(t, 135.75, transaction["amount"], 0.001)
			assert.Equal(t, "Weekly grocery shopping - updated amount", transaction["description"])

			tags := transaction["tags"].([]any)
			assert.Len(t, tags, 4)
			assert.Contains(t, tags, "organic")
		})

		// Step 4: List Transactions with Filtering
		t.Run("ListTransactionsWithFiltering", func(t *testing.T) {
			// Create additional transactions for filtering tests
			additionalTransactions := []map[string]any{
				{
					"amount":      50.0,
					"type":        "expense",
					"description": "Gas station",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().AddDate(0, 0, -1).Format(time.RFC3339),
				},
				{
					"amount":      200.0,
					"type":        "income",
					"description": "Freelance work",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().AddDate(0, 0, -2).Format(time.RFC3339),
				},
			}

			for _, transactionData := range additionalTransactions {
				body, _ := json.Marshal(transactionData)
				resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()
			}

			// Test filtering by family
			resp, err := http.Get(fmt.Sprintf("%s/transactions?family_id=%s", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()

			var transactions []map[string]any
			err = json.NewDecoder(resp.Body).Decode(&transactions)
			require.NoError(t, err)

			assert.Len(t, transactions, 3, "Should return all family transactions")

			// Test filtering by type
			resp, err = http.Get(fmt.Sprintf("%s/transactions?family_id=%s&type=expense", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(&transactions)
			require.NoError(t, err)

			assert.Len(t, transactions, 2, "Should return only expense transactions")
			for _, transaction := range transactions {
				assert.Equal(t, "expense", transaction["type"])
			}

			// Test filtering by amount range
			resp, err = http.Get(fmt.Sprintf("%s/transactions?family_id=%s&amount_from=100", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(&transactions)
			require.NoError(t, err)

			assert.Len(t, transactions, 2, "Should return transactions >= 100")
			for _, transaction := range transactions {
				amount := transaction["amount"].(float64)
				assert.GreaterOrEqual(t, amount, 100.0)
			}
		})

		// Step 5: Delete Transaction
		t.Run("DeleteTransaction", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/transactions/%s", baseURL, transactionID), nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Verify transaction is deleted
			resp, err = http.Get(fmt.Sprintf("%s/transactions/%s", baseURL, transactionID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})
}

// TestTransactionValidation tests transaction data validation
func TestTransactionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e transaction validation test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		// Create test data
		familyData := map[string]any{
			"name":     "Validation Test Family",
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

		userData := map[string]any{
			"email":      "validation@test.com",
			"password":   "ValidationPass123!",
			"first_name": "Validation",
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

		categoryData := map[string]any{
			"name":      "Test Category",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#33FF57",
			"icon":      "tag",
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

	t.Run("InvalidTransactionData", func(t *testing.T) {
		invalidTransactions := []struct {
			name string
			data map[string]any
		}{
			{
				name: "NegativeAmount",
				data: map[string]any{
					"amount":      -100.0,
					"type":        "expense",
					"description": "Invalid negative amount",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
				},
			},
			{
				name: "ZeroAmount",
				data: map[string]any{
					"amount":      0.0,
					"type":        "expense",
					"description": "Invalid zero amount",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
				},
			},
			{
				name: "EmptyDescription",
				data: map[string]any{
					"amount":      100.0,
					"type":        "expense",
					"description": "",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
				},
			},
			{
				name: "InvalidType",
				data: map[string]any{
					"amount":      100.0,
					"type":        "invalid_type",
					"description": "Invalid transaction type",
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
				},
			},
			{
				name: "MissingCategory",
				data: map[string]any{
					"amount":      100.0,
					"type":        "expense",
					"description": "Missing category",
					"user_id":     userID,
					"family_id":   familyID,
				},
			},
			{
				name: "MissingUser",
				data: map[string]any{
					"amount":      100.0,
					"type":        "expense",
					"description": "Missing user",
					"category_id": categoryID,
					"family_id":   familyID,
				},
			},
			{
				name: "MissingFamily",
				data: map[string]any{
					"amount":      100.0,
					"type":        "expense",
					"description": "Missing family",
					"category_id": categoryID,
					"user_id":     userID,
				},
			},
		}

		for _, testCase := range invalidTransactions {
			t.Run(testCase.name, func(t *testing.T) {
				body, _ := json.Marshal(testCase.data)
				resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
					"Invalid transaction data should be rejected: %s", testCase.name)
			})
		}
	})

	t.Run("ValidTransactionData", func(t *testing.T) {
		validTransaction := map[string]any{
			"amount":      100.50,
			"type":        "expense",
			"description": "Valid transaction",
			"category_id": categoryID,
			"user_id":     userID,
			"family_id":   familyID,
			"date":        time.Now().Format(time.RFC3339),
			"tags":        []string{"test", "valid"},
		}

		body, _ := json.Marshal(validTransaction)
		resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode,
			"Valid transaction should be accepted")
	})
}

// TestTransactionConcurrency tests concurrent transaction operations
func TestTransactionConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e transaction concurrency test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID string

	// Setup
	t.Run("Setup", func(t *testing.T) {
		familyData := map[string]any{
			"name":     "Concurrency Test Family",
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

		userData := map[string]any{
			"email":      "concurrency@test.com",
			"password":   "ConcurrencyPass123!",
			"first_name": "Concurrency",
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

		categoryData := map[string]any{
			"name":      "Concurrent Category",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#3357FF",
			"icon":      "layers",
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

	t.Run("ConcurrentTransactionCreation", func(t *testing.T) {
		const numGoroutines = 20
		results := make(chan error, numGoroutines)
		createdIDs := make(chan string, numGoroutines)

		for i := range numGoroutines {
			go func(index int) {
				transactionData := map[string]any{
					"amount":      float64(100 + index),
					"type":        "expense",
					"description": fmt.Sprintf("Concurrent transaction %d", index),
					"category_id": categoryID,
					"user_id":     userID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				}

				body, _ := json.Marshal(transactionData)
				resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					results <- fmt.Errorf("expected status 201, got %d", resp.StatusCode)
					return
				}

				var response map[string]any
				err = json.NewDecoder(resp.Body).Decode(&response)
				if err != nil {
					results <- err
					return
				}

				createdIDs <- extractIDFromResponse(response)
				results <- nil
			}(i)
		}

		// Collect results
		var successCount int
		var transactionIDs []string
		for range numGoroutines {
			err := <-results
			if err == nil {
				successCount++
				transactionIDs = append(transactionIDs, <-createdIDs)
			}
		}

		assert.Equal(t, numGoroutines, successCount, "All concurrent transactions should succeed")
		assert.Len(t, transactionIDs, numGoroutines, "Should have created all transactions")

		// Verify all transactions exist
		resp, err := http.Get(fmt.Sprintf("%s/transactions?family_id=%s", baseURL, familyID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var transactions []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&transactions)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(transactions), numGoroutines,
			"Should find at least %d transactions", numGoroutines)
	})
}

// TestTransactionReporting tests transaction-based reporting
func TestTransactionReporting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e transaction reporting test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var familyID, userID, categoryID string

	// Setup with sample transactions
	t.Run("Setup", func(t *testing.T) {
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

		userData := map[string]any{
			"email":      "reporting@test.com",
			"password":   "ReportingPass123!",
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

		categoryData := map[string]any{
			"name":      "Food",
			"type":      "expense",
			"family_id": familyID,
			"color":     "#FF33F5",
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

		// Create sample transactions for reporting
		transactions := []map[string]any{
			{
				"amount":      100.0,
				"type":        "expense",
				"description": "Groceries",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
				"date":        time.Now().AddDate(0, 0, -7).Format(time.RFC3339),
			},
			{
				"amount":      200.0,
				"type":        "expense",
				"description": "Restaurant",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
				"date":        time.Now().AddDate(0, 0, -3).Format(time.RFC3339),
			},
			{
				"amount":      500.0,
				"type":        "income",
				"description": "Salary",
				"category_id": categoryID,
				"user_id":     userID,
				"family_id":   familyID,
				"date":        time.Now().AddDate(0, 0, -1).Format(time.RFC3339),
			},
		}

		for _, transactionData := range transactions {
			body, _ := json.Marshal(transactionData)
			resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()
		}
	})

	t.Run("GenerateTransactionReport", func(t *testing.T) {
		reportData := map[string]any{
			"type":      "transaction_summary",
			"family_id": familyID,
			"date_from": time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
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

		assert.Equal(t, "transaction_summary", report["type"])
		assert.Equal(t, familyID, report["family_id"])
	})
}

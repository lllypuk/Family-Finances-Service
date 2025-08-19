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

// TestFamilySetupWorkflow tests complete family setup workflow
func TestFamilySetupWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e family setup test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	t.Run("CompleteFamilySetup", func(t *testing.T) {
		var familyID string

		// Step 1: Create Family
		t.Run("CreateFamily", func(t *testing.T) {
			familyData := map[string]any{
				"name":     "The Smith Family",
				"currency": "USD",
			}

			body, _ := json.Marshal(familyData)
			resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			familyID = extractIDFromResponse(response)
			assert.NotEmpty(t, familyID)

			// Extract family data from nested response
			familyData, ok := response["data"].(map[string]any)
			require.True(t, ok, "Response should contain data field")
			assert.Equal(t, "The Smith Family", familyData["name"])
			assert.Equal(t, "USD", familyData["currency"])
		})

		// Step 2: Create Admin User
		var adminUserID string
		t.Run("CreateAdminUser", func(t *testing.T) {
			userData := map[string]any{
				"email":      "admin@smith-family.com",
				"password":   "SecurePassword123!",
				"first_name": "John",
				"last_name":  "Smith",
				"role":       "admin",
				"family_id":  familyID,
			}

			body, _ := json.Marshal(userData)
			resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			adminUserID = extractIDFromResponse(response)
			assert.NotEmpty(t, adminUserID)

			// Extract user data from nested response
			userData, ok := response["data"].(map[string]any)
			require.True(t, ok, "Response should contain data field")
			assert.Equal(t, "admin", userData["role"])
		})

		// Step 3: Create Family Members
		var memberUserIDs []string
		t.Run("CreateFamilyMembers", func(t *testing.T) {
			members := []map[string]any{
				{
					"email":      "jane@smith-family.com",
					"password":   "SecurePassword123!",
					"first_name": "Jane",
					"last_name":  "Smith",
					"role":       "member",
					"family_id":  familyID,
				},
				{
					"email":      "alice@smith-family.com",
					"password":   "SecurePassword123!",
					"first_name": "Alice",
					"last_name":  "Smith",
					"role":       "child",
					"family_id":  familyID,
				},
				{
					"email":      "bob@smith-family.com",
					"password":   "SecurePassword123!",
					"first_name": "Bob",
					"last_name":  "Smith",
					"role":       "child",
					"family_id":  familyID,
				},
			}

			for _, memberData := range members {
				body, _ := json.Marshal(memberData)
				resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response map[string]any
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				memberID := extractIDFromResponse(response)
				memberUserIDs = append(memberUserIDs, memberID)
				assert.NotEmpty(t, memberID)
			}

			assert.Len(t, memberUserIDs, 3)
		})

		// Step 4: Setup Family Categories
		var categoryIDs []string
		t.Run("SetupCategories", func(t *testing.T) {
			categories := []map[string]any{
				{
					"name":      "Food & Groceries",
					"type":      "expense",
					"family_id": familyID,
					"color":     "#FF5733",
					"icon":      "shopping-cart",
				},
				{
					"name":      "Transportation",
					"type":      "expense",
					"family_id": familyID,
					"color":     "#33FF57",
					"icon":      "car",
				},
				{
					"name":      "Entertainment",
					"type":      "expense",
					"family_id": familyID,
					"color":     "#3357FF",
					"icon":      "film",
				},
				{
					"name":      "Salary",
					"type":      "income",
					"family_id": familyID,
					"color":     "#FF33F5",
					"icon":      "dollar-sign",
				},
				{
					"name":      "Freelance",
					"type":      "income",
					"family_id": familyID,
					"color":     "#FF7733",
					"icon":      "briefcase",
				},
			}

			for _, categoryData := range categories {
				body, _ := json.Marshal(categoryData)
				resp, err := http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusCreated, resp.StatusCode)

				var response map[string]any
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				categoryID := extractIDFromResponse(response)
				categoryIDs = append(categoryIDs, categoryID)
				assert.NotEmpty(t, categoryID)
			}

			assert.Len(t, categoryIDs, 5)
		})

		// Step 5: Setup Monthly Budgets
		var budgetIDs []string
		t.Run("SetupBudgets", func(t *testing.T) {
			budgets := []map[string]any{
				{
					"name":        "Food & Groceries Budget",
					"amount":      600.0,
					"period":      "monthly",
					"category_id": categoryIDs[0], // Food & Groceries
					"family_id":   familyID,
					"start_date":  time.Now().Format(time.RFC3339),
					"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
				},
				{
					"name":        "Transportation Budget",
					"amount":      300.0,
					"period":      "monthly",
					"category_id": categoryIDs[1], // Transportation
					"family_id":   familyID,
					"start_date":  time.Now().Format(time.RFC3339),
					"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
				},
				{
					"name":        "Entertainment Budget",
					"amount":      200.0,
					"period":      "monthly",
					"category_id": categoryIDs[2], // Entertainment
					"family_id":   familyID,
					"start_date":  time.Now().Format(time.RFC3339),
					"end_date":    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
				},
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

				budgetID := extractIDFromResponse(response)
				budgetIDs = append(budgetIDs, budgetID)
				assert.NotEmpty(t, budgetID)
			}

			assert.Len(t, budgetIDs, 3)
		})

		// Step 6: Add Sample Transactions
		t.Run("AddSampleTransactions", func(t *testing.T) {
			transactions := []map[string]any{
				// Income transactions
				{
					"amount":      5000.0,
					"type":        "income",
					"description": "Monthly Salary - John",
					"category_id": categoryIDs[3], // Salary
					"user_id":     adminUserID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      3000.0,
					"type":        "income",
					"description": "Monthly Salary - Jane",
					"category_id": categoryIDs[3], // Salary
					"user_id":     memberUserIDs[0],
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      500.0,
					"type":        "income",
					"description": "Freelance Work",
					"category_id": categoryIDs[4], // Freelance
					"user_id":     adminUserID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				// Expense transactions
				{
					"amount":      150.0,
					"type":        "expense",
					"description": "Weekly Groceries",
					"category_id": categoryIDs[0], // Food & Groceries
					"user_id":     memberUserIDs[0],
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      80.0,
					"type":        "expense",
					"description": "Gas Station",
					"category_id": categoryIDs[1], // Transportation
					"user_id":     adminUserID,
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
				{
					"amount":      45.0,
					"type":        "expense",
					"description": "Movie Night",
					"category_id": categoryIDs[2], // Entertainment
					"user_id":     memberUserIDs[1],
					"family_id":   familyID,
					"date":        time.Now().Format(time.RFC3339),
				},
			}

			createdTransactions := 0
			for _, transactionData := range transactions {
				body, _ := json.Marshal(transactionData)
				resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
				require.NoError(t, err)
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusCreated {
					createdTransactions++
				}
			}

			assert.Equal(t, len(transactions), createdTransactions, "All transactions should be created successfully")
		})

		// Step 7: Verify Family Members
		t.Run("VerifyFamilyMembers", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/families/%s/members", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var membersResponse struct {
				Data []map[string]any `json:"data"`
			}
			err = json.NewDecoder(resp.Body).Decode(&membersResponse)
			require.NoError(t, err)

			members := membersResponse.Data

			assert.Len(t, members, 4, "Family should have 4 members (1 admin + 3 members)")

			// Verify roles
			roleCount := map[string]int{}
			for _, member := range members {
				role := member["role"].(string)
				roleCount[role]++
			}

			assert.Equal(t, 1, roleCount["admin"], "Should have 1 admin")
			assert.Equal(t, 1, roleCount["member"], "Should have 1 member")
			assert.Equal(t, 2, roleCount["child"], "Should have 2 children")
		})

		// Step 8: Check Budget Status
		t.Run("CheckBudgetStatus", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/budgets?family_id=%s", baseURL, familyID))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var budgetsResponse struct {
				Data []map[string]any `json:"data"`
			}
			err = json.NewDecoder(resp.Body).Decode(&budgetsResponse)
			require.NoError(t, err)

			budgets := budgetsResponse.Data

			assert.Len(t, budgets, 3, "Should have 3 budgets")

			// Verify budget spending
			for _, budget := range budgets {
				amount := budget["amount"].(float64)

				// Handle both int and float types for spent field
				var spent float64
				switch s := budget["spent"].(type) {
				case float64:
					spent = s
				case int:
					spent = float64(s)
				default:
					t.Fatalf("Unexpected type for spent field: %T", budget["spent"])
				}

				assert.GreaterOrEqual(t, spent, 0.0, "Spent amount should be non-negative")
				assert.LessOrEqual(t, spent, amount*2, "Spent amount should be reasonable") // Allow for over-budget
			}
		})

		// Step 9: Generate Family Report
		t.Run("GenerateFamilyReport", func(t *testing.T) {
			reportData := map[string]any{
				"name":       "Family Summary Report",
				"type":       "expenses",
				"period":     "monthly",
				"family_id":  familyID,
				"user_id":    adminUserID,
				"start_date": time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
				"end_date":   time.Now().Format(time.RFC3339),
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

			// Verify report can be retrieved
			resp, err = http.Get(fmt.Sprintf("%s/reports/%s", baseURL, reportID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	})
}

// TestMultiFamilyIsolation tests data isolation between families
func TestMultiFamilyIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e multi-family test in short mode")
	}

	testServer := testhelpers.SetupHTTPServer(t)
	defer testServer.MongoDB.CleanupCollections(t)

	baseURL := "http://localhost:8080/api/v1"

	var family1ID, family2ID string
	var user1ID, user2ID string

	// Create two families
	t.Run("CreateFamilies", func(t *testing.T) {
		families := []map[string]any{
			{"name": "Family One", "currency": "USD"},
			{"name": "Family Two", "currency": "EUR"},
		}

		familyIDs := []*string{&family1ID, &family2ID}

		for i, familyData := range families {
			body, _ := json.Marshal(familyData)
			resp, err := http.Post(baseURL+"/families", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			*familyIDs[i] = extractIDFromResponse(response)
		}
	})

	// Create users for each family
	t.Run("CreateUsers", func(t *testing.T) {
		users := []map[string]any{
			{
				"email":      "user1@family1.com",
				"password":   "password123",
				"first_name": "User",
				"last_name":  "One",
				"role":       "admin",
				"family_id":  family1ID,
			},
			{
				"email":      "user2@family2.com",
				"password":   "password123",
				"first_name": "User",
				"last_name":  "Two",
				"role":       "admin",
				"family_id":  family2ID,
			},
		}

		userIDs := []*string{&user1ID, &user2ID}

		for i, userData := range users {
			body, _ := json.Marshal(userData)
			resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			*userIDs[i] = extractIDFromResponse(response)
		}
	})

	// Create categories for each family
	var category1ID, category2ID string
	t.Run("CreateCategories", func(t *testing.T) {
		categories := []map[string]any{
			{
				"name":      "Food Family 1",
				"type":      "expense",
				"family_id": family1ID,
				"color":     "#FF5733",
				"icon":      "utensils",
			},
			{
				"name":      "Food Family 2",
				"type":      "expense",
				"family_id": family2ID,
				"color":     "#33FF57",
				"icon":      "utensils",
			},
		}

		categoryIDs := []*string{&category1ID, &category2ID}

		for i, categoryData := range categories {
			body, _ := json.Marshal(categoryData)
			resp, err := http.Post(baseURL+"/categories", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var response map[string]any
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			*categoryIDs[i] = extractIDFromResponse(response)
		}
	})

	// Create transactions for each family
	t.Run("CreateTransactions", func(t *testing.T) {
		transactions := []map[string]any{
			{
				"amount":      100.0,
				"type":        "expense",
				"description": "Family 1 Transaction",
				"category_id": category1ID,
				"user_id":     user1ID,
				"family_id":   family1ID,
				"date":        time.Now().Format(time.RFC3339),
			},
			{
				"amount":      200.0,
				"type":        "expense",
				"description": "Family 2 Transaction",
				"category_id": category2ID,
				"user_id":     user2ID,
				"family_id":   family2ID,
				"date":        time.Now().Format(time.RFC3339),
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

	// Verify isolation - Family 1 should only see their data
	t.Run("VerifyFamily1Isolation", func(t *testing.T) {
		// Check family members
		resp, err := http.Get(fmt.Sprintf("%s/families/%s/members", baseURL, family1ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var membersResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&membersResponse)
		require.NoError(t, err)

		members := membersResponse.Data
		assert.Len(t, members, 1)
		assert.Equal(t, user1ID, members[0]["id"])

		// Check transactions
		resp, err = http.Get(fmt.Sprintf("%s/transactions?family_id=%s", baseURL, family1ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var transactionsResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&transactionsResponse)
		require.NoError(t, err)

		transactions := transactionsResponse.Data
		assert.Len(t, transactions, 1)
		assert.Equal(t, "Family 1 Transaction", transactions[0]["description"])

		// Check categories
		resp, err = http.Get(fmt.Sprintf("%s/categories?family_id=%s", baseURL, family1ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoriesResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&categoriesResponse)
		require.NoError(t, err)

		categories := categoriesResponse.Data
		assert.Len(t, categories, 1)
		assert.Equal(t, "Food Family 1", categories[0]["name"])
	})

	// Verify isolation - Family 2 should only see their data
	t.Run("VerifyFamily2Isolation", func(t *testing.T) {
		// Check family members
		resp, err := http.Get(fmt.Sprintf("%s/families/%s/members", baseURL, family2ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var membersResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&membersResponse)
		require.NoError(t, err)

		members := membersResponse.Data
		assert.Len(t, members, 1)
		assert.Equal(t, user2ID, members[0]["id"])

		// Check transactions
		resp, err = http.Get(fmt.Sprintf("%s/transactions?family_id=%s", baseURL, family2ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var transactionsResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&transactionsResponse)
		require.NoError(t, err)

		transactions := transactionsResponse.Data
		assert.Len(t, transactions, 1)
		assert.Equal(t, "Family 2 Transaction", transactions[0]["description"])

		// Check categories
		resp, err = http.Get(fmt.Sprintf("%s/categories?family_id=%s", baseURL, family2ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var categoriesResponse struct {
			Data []map[string]any `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&categoriesResponse)
		require.NoError(t, err)

		categories := categoriesResponse.Data
		assert.Len(t, categories, 1)
		assert.Equal(t, "Food Family 2", categories[0]["name"])
	})
}

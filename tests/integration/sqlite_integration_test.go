package integration

import (
	"context"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
)

// TestFullWorkflowIntegration tests complete application workflow with SQLite
func TestFullWorkflowIntegration(t *testing.T) {
	// Setup SQLite in-memory database
	container := testutils.SetupSQLiteTestDB(t)

	db := container.GetTestDatabase(t)
	ctx := context.Background()

	// Initialize repositories
	userRepo := userrepo.NewSQLiteRepository(db)
	familyRepo := userrepo.NewSQLiteFamilyRepository(db)
	categoryRepo := categoryrepo.NewSQLiteRepository(db)
	transactionRepo := transactionrepo.NewSQLiteRepository(db)
	budgetRepo := budgetrepo.NewSQLiteRepository(db)

	// Step 1: Create family
	family := &user.Family{
		ID:       uuid.New(),
		Name:     "Smith Family",
		Currency: "USD",
	}

	err := familyRepo.Create(ctx, family)
	require.NoError(t, err)

	// Step 2: Create users
	adminUser := &user.User{
		ID:        uuid.New(),
		Email:     "john@smith.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Smith",
		Role:      user.RoleAdmin,
		FamilyID:  family.ID,
	}

	memberUser := &user.User{
		ID:        uuid.New(),
		Email:     "jane@smith.com",
		Password:  "hashed_password",
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      user.RoleMember,
		FamilyID:  family.ID,
	}

	err = userRepo.Create(ctx, adminUser)
	require.NoError(t, err)
	err = userRepo.Create(ctx, memberUser)
	require.NoError(t, err)

	// Step 3: Create category hierarchy
	// Root categories
	foodCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Food & Beverages",
		Type:     category.TypeExpense,
		FamilyID: family.ID,
		IsActive: true,
	}

	transportCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Transportation",
		Type:     category.TypeExpense,
		FamilyID: family.ID,
		IsActive: true,
	}

	salaryCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Salary",
		Type:     category.TypeIncome,
		FamilyID: family.ID,
		IsActive: true,
	}

	err = categoryRepo.Create(ctx, foodCategory)
	require.NoError(t, err)
	err = categoryRepo.Create(ctx, transportCategory)
	require.NoError(t, err)
	err = categoryRepo.Create(ctx, salaryCategory)
	require.NoError(t, err)

	// Subcategories
	groceryCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Groceries",
		Type:     category.TypeExpense,
		ParentID: &foodCategory.ID,
		FamilyID: family.ID,
		IsActive: true,
	}

	gasCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Gas",
		Type:     category.TypeExpense,
		ParentID: &transportCategory.ID,
		FamilyID: family.ID,
		IsActive: true,
	}

	err = categoryRepo.Create(ctx, groceryCategory)
	require.NoError(t, err)
	err = categoryRepo.Create(ctx, gasCategory)
	require.NoError(t, err)

	// Step 4: Create budgets
	monthlyFoodBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       "Monthly Food Budget",
		Amount:     800.00,
		Spent:      0.00,
		Period:     budget.PeriodMonthly,
		CategoryID: &foodCategory.ID,
		FamilyID:   family.ID,
		StartDate:  time.Now().Truncate(24 * time.Hour),
		EndDate:    time.Now().AddDate(0, 1, 0).Truncate(24 * time.Hour),
		IsActive:   true,
	}

	monthlyTransportBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       "Monthly Transport Budget",
		Amount:     300.00,
		Spent:      0.00,
		Period:     budget.PeriodMonthly,
		CategoryID: &transportCategory.ID,
		FamilyID:   family.ID,
		StartDate:  time.Now().Truncate(24 * time.Hour),
		EndDate:    time.Now().AddDate(0, 1, 0).Truncate(24 * time.Hour),
		IsActive:   true,
	}

	err = budgetRepo.Create(ctx, monthlyFoodBudget)
	require.NoError(t, err)
	err = budgetRepo.Create(ctx, monthlyTransportBudget)
	require.NoError(t, err)

	// Step 5: Create transactions
	now := time.Now()
	transactions := []*transaction.Transaction{
		{
			ID:          uuid.New(),
			Amount:      3000.00,
			Type:        transaction.TypeIncome,
			Description: "Monthly salary",
			CategoryID:  salaryCategory.ID,
			UserID:      adminUser.ID,
			FamilyID:    family.ID,
			Date:        now.AddDate(0, 0, -1),
			Tags:        []string{"salary", "monthly"},
		},
		{
			ID:          uuid.New(),
			Amount:      2000.00,
			Type:        transaction.TypeIncome,
			Description: "Spouse salary",
			CategoryID:  salaryCategory.ID,
			UserID:      memberUser.ID,
			FamilyID:    family.ID,
			Date:        now.AddDate(0, 0, -1),
			Tags:        []string{"salary", "monthly"},
		},
		{
			ID:          uuid.New(),
			Amount:      120.50,
			Type:        transaction.TypeExpense,
			Description: "Weekly groceries",
			CategoryID:  groceryCategory.ID,
			UserID:      memberUser.ID,
			FamilyID:    family.ID,
			Date:        now.AddDate(0, 0, -2),
			Tags:        []string{"grocery", "weekly", "food"},
		},
		{
			ID:          uuid.New(),
			Amount:      85.00,
			Type:        transaction.TypeExpense,
			Description: "Gas station",
			CategoryID:  gasCategory.ID,
			UserID:      adminUser.ID,
			FamilyID:    family.ID,
			Date:        now.AddDate(0, 0, -3),
			Tags:        []string{"gas", "car", "transport"},
		},
		{
			ID:          uuid.New(),
			Amount:      45.75,
			Type:        transaction.TypeExpense,
			Description: "Coffee and snacks",
			CategoryID:  foodCategory.ID,
			UserID:      adminUser.ID,
			FamilyID:    family.ID,
			Date:        now,
			Tags:        []string{"coffee", "snacks"},
		},
	}

	for _, tx := range transactions {
		err = transactionRepo.Create(ctx, tx)
		require.NoError(t, err)
	}

	// Step 6: Verify data integrity and calculations

	// Test category hierarchy
	t.Run("VerifyCategoryHierarchy", func(t *testing.T) {
		children, err := categoryRepo.GetCategoryChildren(ctx, foodCategory.ID)
		require.NoError(t, err)
		assert.Len(t, children, 2) // Food + Groceries

		path, err := categoryRepo.GetCategoryPath(ctx, groceryCategory.ID)
		require.NoError(t, err)
		assert.Len(t, path, 2) // Food -> Groceries
	})

	// Test transaction filtering
	t.Run("VerifyTransactionFiltering", func(t *testing.T) {
		// Get all family transactions
		filter := transaction.Filter{
			FamilyID: family.ID,
			Limit:    10,
		}
		allTransactions, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, allTransactions, 5)

		// Filter by expense type
		expenseType := transaction.TypeExpense
		filter.Type = &expenseType
		expenseTransactions, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, expenseTransactions, 3)

		// Filter by tag
		filter.Type = nil
		filter.Tags = []string{"salary"}
		salaryTransactions, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, salaryTransactions, 2)

		// Filter by date range
		filter.Tags = nil
		dateFrom := now.AddDate(0, 0, -2)
		dateTo := now.AddDate(0, 0, 1)
		filter.DateFrom = &dateFrom
		filter.DateTo = &dateTo
		recentTransactions, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, recentTransactions, 4) // Monthly salary, Spouse salary, Weekly groceries, Coffee and snacks
	})

	// Test transaction summary
	t.Run("VerifyTransactionSummary", func(t *testing.T) {
		startDate := now.AddDate(0, 0, -7)
		endDate := now.AddDate(0, 0, 1)

		summary, err := transactionRepo.GetSummary(ctx, family.ID, startDate, endDate)
		require.NoError(t, err)

		assert.Equal(t, 5, summary.TotalCount)
		assert.Equal(t, 2, summary.IncomeCount)
		assert.Equal(t, 3, summary.ExpenseCount)
		assert.InEpsilon(t, 5000.00, summary.TotalIncome, 0.01)  // 3000 + 2000
		assert.InEpsilon(t, 251.25, summary.TotalExpenses, 0.01) // 120.50 + 85.00 + 45.75
		assert.InEpsilon(t, 4748.75, summary.Balance, 0.01)      // 5000 - 251.25
		assert.InEpsilon(t, 2500.00, summary.AvgIncome, 0.01)    // 5000 / 2
		assert.InEpsilon(t, 83.75, summary.AvgExpense, 0.01)     // 251.25 / 3
	})

	// Test budget tracking
	t.Run("VerifyBudgetTracking", func(t *testing.T) {
		budgets, err := budgetRepo.GetActiveBudgets(ctx, family.ID)
		require.NoError(t, err)
		assert.Len(t, budgets, 2)

		// Check budget usage stats
		stats, err := budgetRepo.GetUsageStats(ctx, family.ID)
		require.NoError(t, err)
		assert.Len(t, stats, 2)

		// Find food budget stats
		var foodBudgetStats *budgetrepo.UsageStats
		for _, stat := range stats {
			if stat.BudgetID == monthlyFoodBudget.ID {
				foodBudgetStats = stat
				break
			}
		}
		require.NotNil(t, foodBudgetStats)

		// Food budget should have some spending (only direct food category transactions, not subcategories)
		expectedFoodSpending := 45.75 // Coffee only, groceries are in subcategory
		assert.InDelta(t, expectedFoodSpending, foodBudgetStats.SpentAmount, 0.01)
		assert.InDelta(t, 800.00-expectedFoodSpending, foodBudgetStats.RemainingAmount, 0.01)
		expectedPercentage := (expectedFoodSpending / 800.00) * 100
		assert.InDelta(t, expectedPercentage, foodBudgetStats.UsagePercentage, 0.01)
	})

	// Test user permissions and family isolation
	t.Run("VerifyUserAndFamilyIsolation", func(t *testing.T) {
		// Get users by family
		familyUsers, err := userRepo.GetByFamilyID(ctx, family.ID)
		require.NoError(t, err)
		assert.Len(t, familyUsers, 2)

		// Get admin users
		adminUsers, err := userRepo.GetUsersByRole(ctx, family.ID, user.RoleAdmin)
		require.NoError(t, err)
		assert.Len(t, adminUsers, 1)
		assert.Equal(t, adminUser.ID, adminUsers[0].ID)

		// Get family statistics
		familyStats, err := familyRepo.GetFamilyStatistics(ctx, family.ID)
		require.NoError(t, err)
		assert.Equal(t, family.ID, familyStats.ID)
		assert.Equal(t, 2, familyStats.UserCount)
		assert.Equal(t, 5, familyStats.CategoryCount) // 3 root + 2 subcategories
		assert.Equal(t, 5, familyStats.TransactionCount)
		assert.Equal(t, 2, familyStats.BudgetCount)
		assert.InEpsilon(t, 100000.00, familyStats.TotalIncome, 0.01)
		assert.InEpsilon(t, 5025.00, familyStats.TotalExpenses, 0.01)
		assert.InEpsilon(t, 94975.00, familyStats.Balance, 0.01)
	})

	// Test complex queries and edge cases
	t.Run("VerifyComplexQueries", func(t *testing.T) {
		// Test monthly summary
		now := time.Now()
		monthlySummary, err := transactionRepo.GetMonthlySummary(ctx, family.ID, now.Year(), int(now.Month()))
		require.NoError(t, err)
		assert.NotEmpty(t, monthlySummary)

		// Test categories by type
		expenseCategories, err := categoryRepo.GetByFamilyIDAndType(ctx, family.ID, category.TypeExpense)
		require.NoError(t, err)
		assert.Len(t, expenseCategories, 4) // Food, Transport, Groceries, Gas

		incomeCategories, err := categoryRepo.GetByFamilyIDAndType(ctx, family.ID, category.TypeIncome)
		require.NoError(t, err)
		assert.Len(t, incomeCategories, 1) // Salary

		// Test pagination
		filter := transaction.Filter{
			FamilyID: family.ID,
			Limit:    2,
			Offset:   0,
		}
		page1, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		filter.Offset = 2
		page2, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// Ensure different transactions on different pages
		page1IDs := make(map[uuid.UUID]bool)
		for _, tx := range page1 {
			page1IDs[tx.ID] = true
		}
		for _, tx := range page2 {
			assert.False(t, page1IDs[tx.ID], "Transaction should not appear on both pages")
		}
	})
}

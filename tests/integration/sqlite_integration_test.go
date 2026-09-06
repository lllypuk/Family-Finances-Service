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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
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
	categoryRepo := categoryrepo.NewSQLiteRepository(db)
	familyRepo := userrepo.NewSQLiteFamilyRepository(db, categoryRepo, userRepo)
	transactionRepo := transactionrepo.NewSQLiteRepository(db)
	budgetRepo := budgetrepo.NewSQLiteRepository(db)

	// Step 1: Create family
	family := &user.Family{
		ID:       uuid.New(),
		Name:     "Smith Family",
		Currency: "USD",
		Timezone: "Europe/Moscow",
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
	}

	memberUser := &user.User{
		ID:        uuid.New(),
		Email:     "jane@smith.com",
		Password:  "hashed_password",
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      user.RoleMember,
	}

	err = userRepo.Create(ctx, adminUser)
	require.NoError(t, err)
	err = userRepo.Create(ctx, memberUser)
	require.NoError(t, err)

	// Step 3: Create category hierarchy
	// Root categories
	foodCategory := &category.Category{
		ID:   uuid.New(),
		Name: "Food & Beverages",
		Type: category.TypeExpense,

		IsActive: true,
	}

	transportCategory := &category.Category{
		ID:   uuid.New(),
		Name: "Transportation",
		Type: category.TypeExpense,

		IsActive: true,
	}

	salaryCategory := &category.Category{
		ID:   uuid.New(),
		Name: "Salary",
		Type: category.TypeIncome,

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

		IsActive: true,
	}

	gasCategory := &category.Category{
		ID:       uuid.New(),
		Name:     "Gas",
		Type:     category.TypeExpense,
		ParentID: &transportCategory.ID,

		IsActive: true,
	}

	err = categoryRepo.Create(ctx, groceryCategory)
	require.NoError(t, err)
	err = categoryRepo.Create(ctx, gasCategory)
	require.NoError(t, err)

	// Step 4: Create budgets
	monthlyFoodBudget := &budget.Budget{
		ID:          uuid.New(),
		Name:        "Monthly Food Budget",
		AmountMinor: 80_000,
		SpentMinor:  0,
		Period:      budget.PeriodMonthly,
		CategoryID:  &foodCategory.ID,
		StartDate:   date.Today(time.UTC),
		EndDate:     date.Today(time.UTC).AddDays(30),
		IsActive:    true,
	}

	monthlyTransportBudget := &budget.Budget{
		ID:          uuid.New(),
		Name:        "Monthly Transport Budget",
		AmountMinor: 30_000,
		SpentMinor:  0,
		Period:      budget.PeriodMonthly,
		CategoryID:  &transportCategory.ID,
		StartDate:   date.Today(time.UTC),
		EndDate:     date.Today(time.UTC).AddDays(30),
		IsActive:    true,
	}

	err = budgetRepo.Create(ctx, monthlyFoodBudget)
	require.NoError(t, err)
	err = budgetRepo.Create(ctx, monthlyTransportBudget)
	require.NoError(t, err)

	// Step 5: Create transactions
	now := date.Today(time.UTC)
	transactions := []*transaction.Transaction{
		{
			ID:          uuid.New(),
			AmountMinor: 300_000,
			Type:        transaction.TypeIncome,
			Description: "Monthly salary",
			CategoryID:  salaryCategory.ID,
			UserID:      adminUser.ID,
			Date:        now.AddDays(-1),
			Tags:        []string{"salary", "monthly"},
		},
		{
			ID:          uuid.New(),
			AmountMinor: 200_000,
			Type:        transaction.TypeIncome,
			Description: "Spouse salary",
			CategoryID:  salaryCategory.ID,
			UserID:      memberUser.ID,
			Date:        now.AddDays(-1),
			Tags:        []string{"salary", "monthly"},
		},
		{
			ID:          uuid.New(),
			AmountMinor: 12_050,
			Type:        transaction.TypeExpense,
			Description: "Weekly groceries",
			CategoryID:  groceryCategory.ID,
			UserID:      memberUser.ID,
			Date:        now.AddDays(-2),
			Tags:        []string{"grocery", "weekly", "food"},
		},
		{
			ID:          uuid.New(),
			AmountMinor: 8_500,
			Type:        transaction.TypeExpense,
			Description: "Gas station",
			CategoryID:  gasCategory.ID,
			UserID:      adminUser.ID,
			Date:        now.AddDays(-3),
			Tags:        []string{"gas", "car", "transport"},
		},
		{
			ID:          uuid.New(),
			AmountMinor: 4_575,
			Type:        transaction.TypeExpense,
			Description: "Coffee and snacks",
			CategoryID:  foodCategory.ID,
			UserID:      adminUser.ID,
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

			Limit: 10,
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
		dateFrom := now.AddDays(-2)
		dateTo := now.AddDays(1)
		filter.DateFrom = &dateFrom
		filter.DateTo = &dateTo
		recentTransactions, err := transactionRepo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, recentTransactions, 4) // Monthly salary, Spouse salary, Weekly groceries, Coffee and snacks
	})

	// Test budget tracking
	t.Run("VerifyBudgetTracking", func(t *testing.T) {
		budgets, err := budgetRepo.GetActiveBudgets(ctx, date.Today(time.UTC))
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
		expectedFoodSpending := money.Minor(4_575) // Coffee only, groceries are in subcategory
		assert.Equal(t, expectedFoodSpending, foodBudgetStats.SpentAmountMinor)
		assert.Equal(t, money.Minor(80_000)-expectedFoodSpending, foodBudgetStats.RemainingAmountMinor)
		expectedPercentage := expectedFoodSpending.Percent(money.Minor(80_000))
		assert.InDelta(t, expectedPercentage, foodBudgetStats.UsagePercentage, 0.01)
	})

	// Test user permissions and family isolation
	t.Run("VerifyUserAndFamilyIsolation", func(t *testing.T) {
		// Get all users
		familyUsers, err := userRepo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, familyUsers, 2)

		// Get admin users
		adminUsers, err := userRepo.GetUsersByRole(ctx, user.RoleAdmin)
		require.NoError(t, err)
		assert.Len(t, adminUsers, 1)
		assert.Equal(t, adminUser.ID, adminUsers[0].ID)

		// Get family statistics
		familyStats, err := familyRepo.GetFamilyStatistics(ctx)
		require.NoError(t, err)
		assert.Equal(t, family.ID, familyStats.ID)
		assert.Equal(t, 2, familyStats.UserCount)
		assert.Equal(t, 5, familyStats.CategoryCount) // 3 root + 2 subcategories
		assert.Equal(t, 5, familyStats.TransactionCount)
		assert.Equal(t, 2, familyStats.BudgetCount)
		// Expected values: 2 income transactions (3000 + 2000) and 3 expense transactions (120.50 + 85.00 + 45.75)
		assert.Equal(t, money.Minor(500_000), familyStats.TotalIncomeMinor)
		assert.Equal(t, money.Minor(25_125), familyStats.TotalExpensesMinor)
		assert.Equal(t, money.Minor(474_875), familyStats.BalanceMinor)
	})

	// Test complex queries and edge cases
	t.Run("VerifyComplexQueries", func(t *testing.T) {
		// Test categories by type
		expenseCategories, err := categoryRepo.GetByType(ctx, category.TypeExpense)
		require.NoError(t, err)
		assert.Len(t, expenseCategories, 4) // Food, Transport, Groceries, Gas

		incomeCategories, err := categoryRepo.GetByType(ctx, category.TypeIncome)
		require.NoError(t, err)
		assert.Len(t, incomeCategories, 1) // Salary

		// Test pagination
		filter := transaction.Filter{

			Limit:  2,
			Offset: 0,
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

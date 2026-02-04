package budget_test

import (
	"context"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
)

func TestBudgetRepositorySQLite_Create(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ValidData", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Monthly Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testBudget.ID)
		require.NoError(t, err)
		assert.Equal(t, testBudget.ID, retrieved.ID)
		assert.Equal(t, testBudget.Name, retrieved.Name)
		assert.InDelta(t, testBudget.Amount, retrieved.Amount, 0.01)
		assert.Equal(t, testBudget.Period, retrieved.Period)

		_ = familyID // suppress unused warning
	})

	t.Run("Success_WithCategory", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:         uuid.New(),
			Name:       "Food Budget",
			Amount:     500.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  time.Now(),
			EndDate:    time.Now().AddDate(0, 1, 0),
			IsActive:   true,
		}

		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testBudget.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved.CategoryID)
		assert.Equal(t, catUUID, *retrieved.CategoryID)
	})

	t.Run("Error_DuplicateName", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Duplicate Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: startDate,
			EndDate:   endDate,
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		duplicateBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Duplicate Budget",
			Amount:    2000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: startDate,
			EndDate:   endDate,
			IsActive:  true,
		}

		err = repo.Create(ctx, duplicateBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testBudget := &budget.Budget{
			ID:        uuid.Nil,
			Name:      "Invalid ID Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})

	t.Run("Error_InvalidPeriod", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Invalid Period Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.Period("invalid"),
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid period")
	})

	t.Run("Error_InvalidAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Invalid Amount Budget",
			Amount:    -100.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("Error_InvalidDateRange", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Invalid Date Range Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, -1, 0),
			IsActive:  true,
		}

		err = repo.Create(ctx, testBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "end date must be after start date")
	})
}

func TestBudgetRepositorySQLite_GetByID(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ExistingBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, budgetID, retrieved.ID.String())
		assert.Equal(t, "Test Budget", retrieved.Name)
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, err = repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id parameter")
	})
}

func TestBudgetRepositorySQLite_GetAll(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_EmptyList", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, budgets)
	})

	t.Run("Success_MultipleBudgets", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Budget 1", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Budget 2", 2000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Budget 3", 3000.0, "yearly", familyID, nil)
		require.NoError(t, err)

		budgets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, budgets, 3)
	})

	t.Run("Success_OnlyActiveBudgets", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID1, err := helper.CreateTestBudget(ctx, "Active Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		budgetID2, err := helper.CreateTestBudget(ctx, "To Delete Budget", 2000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.Delete(ctx, uuid.MustParse(budgetID2))
		require.NoError(t, err)

		budgets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, budgetID1, budgets[0].ID.String())
	})
}

func TestBudgetRepositorySQLite_Update(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ValidUpdate", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Original Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved.Name = "Updated Budget"
		retrieved.Amount = 2000.0
		retrieved.Spent = 500.0

		err = repo.Update(ctx, retrieved)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, "Updated Budget", updated.Name)
		assert.InDelta(t, 2000.0, updated.Amount, 0.01)
		assert.InDelta(t, 500.0, updated.Spent, 0.01)
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Non Existent",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Update(ctx, nonExistentBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		invalidBudget := &budget.Budget{
			ID:        uuid.Nil,
			Name:      "Invalid",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
			IsActive:  true,
		}

		err = repo.Update(ctx, invalidBudget)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})
}

func TestBudgetRepositorySQLite_Delete(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_SoftDelete", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "To Delete", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.Delete(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		budgets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, budgets)
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.Delete(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.Delete(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id parameter")
	})
}

func TestBudgetRepositorySQLite_GetActiveBudgets(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_OnlyActiveInDateRange", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := time.Now()

		activeBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Active Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 5),
			IsActive:  true,
		}
		err = repo.Create(ctx, activeBudget)
		require.NoError(t, err)

		pastBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Past Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, -2, 0),
			EndDate:   now.AddDate(0, -1, 0),
			IsActive:  true,
		}
		err = repo.Create(ctx, pastBudget)
		require.NoError(t, err)

		futureBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Future Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 1, 0),
			EndDate:   now.AddDate(0, 2, 0),
			IsActive:  true,
		}
		err = repo.Create(ctx, futureBudget)
		require.NoError(t, err)

		budgets, err := repo.GetActiveBudgets(ctx)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, "Active Budget", budgets[0].Name)

		_ = familyID
	})

	t.Run("Success_ExcludesInactive", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := time.Now()

		activeBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Active Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 5),
			IsActive:  true,
		}
		err = repo.Create(ctx, activeBudget)
		require.NoError(t, err)

		err = repo.Delete(ctx, activeBudget.ID)
		require.NoError(t, err)

		budgets, err := repo.GetActiveBudgets(ctx)
		require.NoError(t, err)
		assert.Empty(t, budgets)

		_ = familyID
	})
}

func TestBudgetRepositorySQLite_GetUsageStats(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ZeroPercent", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := time.Now()
		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Zero Usage",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 25),
			IsActive:  true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		stats, err := repo.GetUsageStats(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.InDelta(t, 0.0, stats[0].UsagePercentage, 0.01)
		assert.Equal(t, "safe", stats[0].Status)
	})

	t.Run("Success_FiftyPercent", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		now := time.Now()
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:         uuid.New(),
			Name:       "50% Usage",
			Amount:     1000.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  now.AddDate(0, 0, -5),
			EndDate:    now.AddDate(0, 0, 25),
			IsActive:   true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 500.0, "Test transaction", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		stats, err := repo.GetUsageStats(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.InDelta(t, 50.0, stats[0].UsagePercentage, 0.01)
		assert.Equal(t, "safe", stats[0].Status)
	})

	t.Run("Success_OneHundredPercent", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		now := time.Now()
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:         uuid.New(),
			Name:       "100% Usage",
			Amount:     1000.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  now.AddDate(0, 0, -5),
			EndDate:    now.AddDate(0, 0, 25),
			IsActive:   true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 600.0, "Test transaction 1", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 400.0, "Test transaction 2", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		stats, err := repo.GetUsageStats(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.InDelta(t, 100.0, stats[0].UsagePercentage, 0.01)
		assert.Equal(t, "warning", stats[0].Status)
	})

	t.Run("Success_OverBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		now := time.Now()
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:         uuid.New(),
			Name:       "Over Budget",
			Amount:     1000.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  now.AddDate(0, 0, -5),
			EndDate:    now.AddDate(0, 0, 25),
			IsActive:   true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 800.0, "Test transaction 1", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 500.0, "Test transaction 2", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		stats, err := repo.GetUsageStats(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.InDelta(t, 130.0, stats[0].UsagePercentage, 0.01)
		assert.Equal(t, "over_budget", stats[0].Status)
	})

	t.Run("EdgeCase_ZeroAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := time.Now()
		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Zero Amount",
			Amount:    0.01,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 25),
			IsActive:  true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		stats, err := repo.GetUsageStats(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.InDelta(t, 0.0, stats[0].UsagePercentage, 0.01)
	})
}

func TestBudgetRepositorySQLite_UpdateSpentAmount(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_PositiveAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.MustParse(budgetID), 500.0)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.InDelta(t, 500.0, retrieved.Spent, 0.01)
	})

	t.Run("Error_NegativeAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.MustParse(budgetID), -100.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.Nil, 500.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.New(), 500.0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestBudgetRepositorySQLite_RecalculateSpent(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_OneTransaction", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 300.0, "Test transaction", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.InDelta(t, 300.0, retrieved.Spent, 0.01)
	})

	t.Run("Success_MultipleTransactions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 200.0, "Transaction 1", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 300.0, "Transaction 2", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 150.0, "Transaction 3", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.InDelta(t, 650.0, retrieved.Spent, 0.01)
	})

	t.Run("Success_NoTransactions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.InDelta(t, 0.0, retrieved.Spent, 0.01)
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestBudgetRepositorySQLite_FindBudgetsAffectedByTransaction(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_OneBudgetAffected", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		now := time.Now()
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:         uuid.New(),
			Name:       "Food Budget",
			Amount:     1000.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  now.AddDate(0, 0, -5),
			EndDate:    now.AddDate(0, 0, 25),
			IsActive:   true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		affectedBudgets, err := repo.FindBudgetsAffectedByTransaction(
			ctx,
			uuid.MustParse(familyID),
			catUUID,
			now,
		)
		require.NoError(t, err)
		assert.Len(t, affectedBudgets, 1)
		assert.Equal(t, testBudget.ID, affectedBudgets[0])
	})

	t.Run("Success_MultipleBudgetsAffected", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		now := time.Now()
		catUUID := uuid.MustParse(categoryID)

		budget1 := &budget.Budget{
			ID:         uuid.New(),
			Name:       "Food Budget 1",
			Amount:     1000.0,
			Spent:      0,
			Period:     budget.PeriodMonthly,
			CategoryID: &catUUID,
			StartDate:  now.AddDate(0, 0, -5),
			EndDate:    now.AddDate(0, 0, 25),
			IsActive:   true,
		}
		err = repo.Create(ctx, budget1)
		require.NoError(t, err)

		budget2 := &budget.Budget{
			ID:        uuid.New(),
			Name:      "General Budget",
			Amount:    5000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 25),
			IsActive:  true,
		}
		err = repo.Create(ctx, budget2)
		require.NoError(t, err)

		affectedBudgets, err := repo.FindBudgetsAffectedByTransaction(
			ctx,
			uuid.MustParse(familyID),
			catUUID,
			now,
		)
		require.NoError(t, err)
		assert.Len(t, affectedBudgets, 2)
	})

	t.Run("Success_NoBudgetsAffected", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		affectedBudgets, err := repo.FindBudgetsAffectedByTransaction(
			ctx,
			uuid.MustParse(familyID),
			uuid.MustParse(categoryID),
			time.Now(),
		)
		require.NoError(t, err)
		assert.Empty(t, affectedBudgets)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = repo.FindBudgetsAffectedByTransaction(
			ctx,
			uuid.Nil,
			uuid.New(),
			time.Now(),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})

	t.Run("Error_InvalidCategoryID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = repo.FindBudgetsAffectedByTransaction(
			ctx,
			uuid.MustParse(familyID),
			uuid.Nil,
			time.Now(),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid category ID")
	})
}

func TestBudgetRepositorySQLite_GetByCategory(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ValidCategory", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Food Budget", 1000.0, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		catUUID := uuid.MustParse(categoryID)
		budgets, err := repo.GetByCategory(ctx, &catUUID)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, "Food Budget", budgets[0].Name)
	})

	t.Run("Success_NilCategory", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "General Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		budgets, err := repo.GetByCategory(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
	})

	t.Run("Success_NonExistentCategory", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		budgets, err := repo.GetByCategory(ctx, &nonExistentID)
		require.NoError(t, err)
		assert.Empty(t, budgets)
	})

	t.Run("Error_InvalidCategoryID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nilUUID := uuid.Nil
		_, err = repo.GetByCategory(ctx, &nilUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid category ID")
	})
}

func TestBudgetRepositorySQLite_GetByPeriod(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_MonthPeriod", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := time.Now()
		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Monthly Budget",
			Amount:    1000.0,
			Spent:     0,
			Period:    budget.PeriodMonthly,
			StartDate: now.AddDate(0, 0, -5),
			EndDate:   now.AddDate(0, 0, 25),
			IsActive:  true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		budgets, err := repo.GetByPeriod(ctx, now.AddDate(0, 0, -10), now.AddDate(0, 0, 30))
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, "Monthly Budget", budgets[0].Name)

		_ = familyID
	})

	t.Run("Success_CustomPeriod", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC)

		testBudget := &budget.Budget{
			ID:        uuid.New(),
			Name:      "Q1 Budget",
			Amount:    5000.0,
			Spent:     0,
			Period:    budget.PeriodCustom,
			StartDate: start,
			EndDate:   end,
			IsActive:  true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		budgets, err := repo.GetByPeriod(ctx, start, end)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, "Q1 Budget", budgets[0].Name)

		_ = familyID
	})

	t.Run("Success_NoBudgetsInPeriod", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		pastStart := time.Now().AddDate(-2, 0, 0)
		pastEnd := time.Now().AddDate(-1, 0, 0)

		budgets, err := repo.GetByPeriod(ctx, pastStart, pastEnd)
		require.NoError(t, err)
		assert.Empty(t, budgets)
	})
}

func TestBudgetRepositorySQLite_GetAlerts(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_NoAlerts", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		alerts, err := repo.GetAlerts(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Empty(t, alerts)
	})

	t.Run("Success_MultipleAlerts", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		budgetUUID := uuid.MustParse(budgetID)

		alert1 := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            budgetUUID,
			ThresholdPercentage: 50,
			IsTriggered:         false,
		}
		err = repo.CreateAlert(ctx, alert1)
		require.NoError(t, err)

		alert2 := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            budgetUUID,
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}
		err = repo.CreateAlert(ctx, alert2)
		require.NoError(t, err)

		alerts, err := repo.GetAlerts(ctx, budgetUUID)
		require.NoError(t, err)
		assert.Len(t, alerts, 2)
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		_, err = repo.GetAlerts(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID parameter")
	})
}

func TestBudgetRepositorySQLite_CreateAlert(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ValidAlert", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		alert := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            uuid.MustParse(budgetID),
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}

		err = repo.CreateAlert(ctx, alert)
		require.NoError(t, err)

		alerts, err := repo.GetAlerts(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Len(t, alerts, 1)
		assert.Equal(t, 80, alerts[0].ThresholdPercentage)
	})

	t.Run("Error_DuplicateThreshold", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		budgetUUID := uuid.MustParse(budgetID)

		alert1 := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            budgetUUID,
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}
		err = repo.CreateAlert(ctx, alert1)
		require.NoError(t, err)

		alert2 := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            budgetUUID,
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}
		err = repo.CreateAlert(ctx, alert2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("Error_InvalidAlertID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		alert := &budgetrepo.Alert{
			ID:                  uuid.Nil,
			BudgetID:            uuid.MustParse(budgetID),
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}

		err = repo.CreateAlert(ctx, alert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid alert ID")
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		alert := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            uuid.Nil,
			ThresholdPercentage: 80,
			IsTriggered:         false,
		}

		err = repo.CreateAlert(ctx, alert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})

	t.Run("Error_InvalidThresholdZero", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		alert := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            uuid.MustParse(budgetID),
			ThresholdPercentage: 0,
			IsTriggered:         false,
		}

		err = repo.CreateAlert(ctx, alert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 100")
	})

	t.Run("Error_InvalidThresholdOver100", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 1000.0, "monthly", familyID, nil)
		require.NoError(t, err)

		alert := &budgetrepo.Alert{
			ID:                  uuid.New(),
			BudgetID:            uuid.MustParse(budgetID),
			ThresholdPercentage: 150,
			IsTriggered:         false,
		}

		err = repo.CreateAlert(ctx, alert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be between 1 and 100")
	})
}

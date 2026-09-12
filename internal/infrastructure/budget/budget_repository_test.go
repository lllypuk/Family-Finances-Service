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
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
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
			ID:          uuid.New(),
			Name:        "Monthly Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
		}

		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testBudget.ID)
		require.NoError(t, err)
		assert.Equal(t, testBudget.ID, retrieved.ID)
		assert.Equal(t, testBudget.Name, retrieved.Name)
		assert.Equal(t, testBudget.AmountMinor, retrieved.AmountMinor)
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
			ID:          uuid.New(),
			Name:        "Food Budget",
			AmountMinor: 50_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Duplicate Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   startDate,
			EndDate:     endDate,
			IsActive:    true,
		}

		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		duplicateBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Duplicate Budget",
			AmountMinor: 200_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   startDate,
			EndDate:     endDate,
			IsActive:    true,
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
			ID:          uuid.Nil,
			Name:        "Invalid ID Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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
			ID:          uuid.New(),
			Name:        "Invalid Period Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.Period("invalid"),
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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
			ID:          uuid.New(),
			Name:        "Invalid Amount Budget",
			AmountMinor: -10_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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
			ID:          uuid.New(),
			Name:        "Invalid Date Range Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(-30),
			IsActive:    true,
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

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, nil)
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

		_, err = helper.CreateTestBudget(ctx, "Budget 1", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Budget 2", 200_000, "monthly", familyID, nil)
		require.NoError(t, err)

		_, err = helper.CreateTestBudget(ctx, "Budget 3", 300_000, "yearly", familyID, nil)
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

		budgetID1, err := helper.CreateTestBudget(ctx, "Active Budget", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		budgetID2, err := helper.CreateTestBudget(ctx, "To Delete Budget", 200_000, "monthly", familyID, nil)
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

		budgetID, err := helper.CreateTestBudget(ctx, "Original Budget", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved.Name = "Updated Budget"
		retrieved.AmountMinor = 200_000
		retrieved.SpentMinor = 50_000

		err = repo.Update(ctx, retrieved)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, "Updated Budget", updated.Name)
		assert.Equal(t, money.Minor(200_000), updated.AmountMinor)
		assert.Equal(t, money.Minor(50_000), updated.SpentMinor)
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Non Existent",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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
			ID:          uuid.Nil,
			Name:        "Invalid",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   date.Today(time.UTC),
			EndDate:     date.Today(time.UTC).AddDays(30),
			IsActive:    true,
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

		budgetID, err := helper.CreateTestBudget(ctx, "To Delete", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.Delete(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		budgets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, budgets)

		// Удалённый бюджет не читается и по известному id
		_, err = repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
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

		now := date.Today(time.UTC)

		activeBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Active Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(5),
			IsActive:    true,
		}
		err = repo.Create(ctx, activeBudget)
		require.NoError(t, err)

		pastBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Past Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-60),
			EndDate:     now.AddDays(-30),
			IsActive:    true,
		}
		err = repo.Create(ctx, pastBudget)
		require.NoError(t, err)

		futureBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Future Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(30),
			EndDate:     now.AddDays(60),
			IsActive:    true,
		}
		err = repo.Create(ctx, futureBudget)
		require.NoError(t, err)

		budgets, err := repo.GetActiveBudgets(ctx, now)
		require.NoError(t, err)
		assert.Len(t, budgets, 1)
		assert.Equal(t, "Active Budget", budgets[0].Name)

		// Дата, а не «сегодня»: отчёт за прошлый период должен видеть бюджет того периода.
		past, err := repo.GetActiveBudgets(ctx, now.AddDays(-45))
		require.NoError(t, err)
		assert.Len(t, past, 1)
		assert.Equal(t, "Past Budget", past[0].Name)

		_ = familyID
	})

	t.Run("Success_ExcludesInactive", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		now := date.Today(time.UTC)

		activeBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Active Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(5),
			IsActive:    true,
		}
		err = repo.Create(ctx, activeBudget)
		require.NoError(t, err)

		err = repo.Delete(ctx, activeBudget.ID)
		require.NoError(t, err)

		budgets, err := repo.GetActiveBudgets(ctx, now)
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

		now := date.Today(time.UTC)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Zero Usage",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
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

		now := date.Today(time.UTC)
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "50% Usage",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 50_000, "Test transaction", "expense", categoryID, userID, familyID)
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

		now := date.Today(time.UTC)
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "100% Usage",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(
			ctx,
			60_000,
			"Test transaction 1",
			"expense",
			categoryID,
			userID,
			familyID,
		)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(
			ctx,
			40_000,
			"Test transaction 2",
			"expense",
			categoryID,
			userID,
			familyID,
		)
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

		now := date.Today(time.UTC)
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Over Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(
			ctx,
			80_000,
			"Test transaction 1",
			"expense",
			categoryID,
			userID,
			familyID,
		)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(
			ctx,
			50_000,
			"Test transaction 2",
			"expense",
			categoryID,
			userID,
			familyID,
		)
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

		now := date.Today(time.UTC)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Zero Amount",
			AmountMinor: 1,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
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

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.MustParse(budgetID), 50000)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, money.Minor(50_000), retrieved.SpentMinor)
	})

	t.Run("Error_NegativeAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.MustParse(budgetID), -10000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be negative")
	})

	t.Run("Error_InvalidBudgetID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.Nil, 50000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid budget ID")
	})

	t.Run("Error_NonExistentBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		err = repo.UpdateSpentAmount(ctx, uuid.New(), 50000)
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

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 30_000, "Test transaction", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, money.Minor(30_000), retrieved.SpentMinor)
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

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 20_000, "Transaction 1", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 30_000, "Transaction 2", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		_, err = helper.CreateTestTransaction(ctx, 15_000, "Transaction 3", "expense", categoryID, userID, familyID)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, money.Minor(65_000), retrieved.SpentMinor)
	})

	t.Run("Success_NoTransactions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := budgetrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Test Budget", 100_000, "monthly", familyID, nil)
		require.NoError(t, err)

		err = repo.RecalculateSpent(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, uuid.MustParse(budgetID))
		require.NoError(t, err)
		assert.Equal(t, money.Minor(0), retrieved.SpentMinor)
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

		now := date.Today(time.UTC)
		catUUID := uuid.MustParse(categoryID)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Food Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
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

		now := date.Today(time.UTC)
		catUUID := uuid.MustParse(categoryID)

		budget1 := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Food Budget 1",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			CategoryID:  &catUUID,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
		}
		err = repo.Create(ctx, budget1)
		require.NoError(t, err)

		budget2 := &budget.Budget{
			ID:          uuid.New(),
			Name:        "General Budget",
			AmountMinor: 500_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
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
			date.Today(time.UTC),
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
			date.Today(time.UTC),
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
			date.Today(time.UTC),
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

		_, err = helper.CreateTestBudget(ctx, "Food Budget", 100_000, "monthly", familyID, &categoryID)
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

		_, err = helper.CreateTestBudget(ctx, "General Budget", 100_000, "monthly", familyID, nil)
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

		now := date.Today(time.UTC)
		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Monthly Budget",
			AmountMinor: 100_000,
			SpentMinor:  0,
			Period:      budget.PeriodMonthly,
			StartDate:   now.AddDays(-5),
			EndDate:     now.AddDays(25),
			IsActive:    true,
		}
		err = repo.Create(ctx, testBudget)
		require.NoError(t, err)

		budgets, err := repo.GetByPeriod(ctx, now.AddDays(-10), now.AddDays(30))
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

		start := date.New(2024, time.January, 1)
		end := date.New(2024, time.March, 31)

		testBudget := &budget.Budget{
			ID:          uuid.New(),
			Name:        "Q1 Budget",
			AmountMinor: 500_000,
			SpentMinor:  0,
			Period:      budget.PeriodCustom,
			StartDate:   start,
			EndDate:     end,
			IsActive:    true,
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

		pastStart := date.Today(time.UTC).AddDays(-730)
		pastEnd := date.Today(time.UTC).AddDays(-365)

		budgets, err := repo.GetByPeriod(ctx, pastStart, pastEnd)
		require.NoError(t, err)
		assert.Empty(t, budgets)
	})
}

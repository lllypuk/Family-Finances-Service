package report_test

import (
	"context"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
	reportrepo "family-budget-service/internal/infrastructure/report"
)

func TestReportRepositorySQLite_Create(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ValidData", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Expense Report",
			Type:      report.TypeExpenses,
			Period:    report.PeriodMonthly,
			UserID:    uuid.MustParse(userID),
			StartDate: date.New(2024, time.Month(1), 1),
			EndDate:   date.New(2024, time.Month(1), 31),
			Data: report.Data{
				TotalExpensesMinor: 100_000,
			},
		}

		err = repo.Create(ctx, testReport)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testReport.ID)
		require.NoError(t, err)
		assert.Equal(t, testReport.ID, retrieved.ID)
		assert.Equal(t, testReport.Name, retrieved.Name)
		assert.Equal(t, testReport.Type, retrieved.Type)
		assert.Equal(t, testReport.Period, retrieved.Period)
		assert.Equal(t, testReport.Data.TotalExpensesMinor, retrieved.Data.TotalExpensesMinor)
	})

	t.Run("Error_InvalidReportID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.Nil,
			Name:      "Invalid Report",
			Type:      report.TypeExpenses,
			Period:    report.PeriodMonthly,
			UserID:    uuid.New(),
			StartDate: date.Today(time.UTC),
			EndDate:   date.Today(time.UTC).AddDays(30),
		}

		err = repo.Create(ctx, testReport)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid report ID")
	})

	t.Run("Error_InvalidUserID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Invalid User Report",
			Type:      report.TypeExpenses,
			Period:    report.PeriodMonthly,
			UserID:    uuid.Nil,
			StartDate: date.Today(time.UTC),
			EndDate:   date.Today(time.UTC).AddDays(30),
		}

		err = repo.Create(ctx, testReport)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})

	t.Run("Error_InvalidReportType", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Invalid Type Report",
			Type:      report.Type("invalid"),
			Period:    report.PeriodMonthly,
			UserID:    uuid.MustParse(userID),
			StartDate: date.Today(time.UTC),
			EndDate:   date.Today(time.UTC).AddDays(30),
		}

		err = repo.Create(ctx, testReport)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid report type")
	})

	t.Run("Error_InvalidDateRange", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Invalid Date Range",
			Type:      report.TypeExpenses,
			Period:    report.PeriodMonthly,
			UserID:    uuid.MustParse(userID),
			StartDate: date.New(2024, time.Month(2), 1),
			EndDate:   date.New(2024, time.Month(1), 1),
		}

		err = repo.Create(ctx, testReport)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "end date must be after or equal to start date")
	})
}

func TestReportRepositorySQLite_GetByID(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ExistingReport", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Test Report",
			Type:      report.TypeIncome,
			Period:    report.PeriodMonthly,
			UserID:    uuid.MustParse(userID),
			StartDate: date.New(2024, time.Month(1), 1),
			EndDate:   date.New(2024, time.Month(1), 31),
			Data: report.Data{
				TotalIncomeMinor: 500_000,
			},
		}

		err = repo.Create(ctx, testReport)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testReport.ID)
		require.NoError(t, err)
		assert.Equal(t, testReport.ID, retrieved.ID)
		assert.Equal(t, testReport.Name, retrieved.Name)
		assert.Equal(t, testReport.Data.TotalIncomeMinor, retrieved.Data.TotalIncomeMinor)
	})

	t.Run("Error_NonExistentReport", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		_, err = repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := repo.GetByID(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id parameter")
	})
}

func TestReportRepositorySQLite_GetAll(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_EmptyList", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		reports, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("Success_MultipleReports", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create multiple reports
		for i := range 3 {
			testReport := &report.Report{
				ID:        uuid.New(),
				Name:      "Report " + string(rune(i+'1')),
				Type:      report.TypeExpenses,
				Period:    report.PeriodMonthly,
				UserID:    uuid.MustParse(userID),
				StartDate: date.New(2024, time.Month(1), 1),
				EndDate:   date.New(2024, time.Month(1), 31),
				Data:      report.Data{},
			}
			err = repo.Create(ctx, testReport)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		reports, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, reports, 3)
	})
}

func TestReportRepositorySQLite_GetByFamilyIDWithPagination(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_FirstPage", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create 5 reports
		for i := range 5 {
			testReport := &report.Report{
				ID:        uuid.New(),
				Name:      "Report " + string(rune(i+'1')),
				Type:      report.TypeExpenses,
				Period:    report.PeriodMonthly,
				UserID:    uuid.MustParse(userID),
				StartDate: date.New(2024, time.Month(1), 1),
				EndDate:   date.New(2024, time.Month(1), 31),
				Data:      report.Data{},
			}
			err = repo.Create(ctx, testReport)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		reports, err := repo.GetByFamilyIDWithPagination(ctx, uuid.MustParse(familyID), 2, 0)
		require.NoError(t, err)
		assert.Len(t, reports, 2)
	})

	t.Run("Success_SecondPage", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create 5 reports
		for i := range 5 {
			testReport := &report.Report{
				ID:        uuid.New(),
				Name:      "Report " + string(rune(i+'1')),
				Type:      report.TypeExpenses,
				Period:    report.PeriodMonthly,
				UserID:    uuid.MustParse(userID),
				StartDate: date.New(2024, time.Month(1), 1),
				EndDate:   date.New(2024, time.Month(1), 31),
				Data:      report.Data{},
			}
			err = repo.Create(ctx, testReport)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)
		}

		reports, err := repo.GetByFamilyIDWithPagination(ctx, uuid.MustParse(familyID), 2, 2)
		require.NoError(t, err)
		assert.Len(t, reports, 2)
	})

	t.Run("Success_EmptyPage", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		reports, err := repo.GetByFamilyIDWithPagination(ctx, uuid.MustParse(familyID), 10, 100)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := repo.GetByFamilyIDWithPagination(ctx, uuid.Nil, 10, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid familyID parameter")
	})
}

func TestReportRepositorySQLite_GetByUserID(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_WithReports", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create reports for this user
		for i := range 3 {
			testReport := &report.Report{
				ID:        uuid.New(),
				Name:      "Report " + string(rune(i+'1')),
				Type:      report.TypeExpenses,
				Period:    report.PeriodMonthly,
				UserID:    uuid.MustParse(userID),
				StartDate: date.New(2024, time.Month(1), 1),
				EndDate:   date.New(2024, time.Month(1), 31),
				Data:      report.Data{},
			}
			err = repo.Create(ctx, testReport)
			require.NoError(t, err)
		}

		reports, err := repo.GetByUserID(ctx, uuid.MustParse(userID))
		require.NoError(t, err)
		assert.Len(t, reports, 3)
	})

	t.Run("Success_NoReports", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentUserID := uuid.New()
		reports, err := repo.GetByUserID(ctx, nonExistentUserID)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("Error_InvalidUserID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := repo.GetByUserID(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}

func TestReportRepositorySQLite_Delete(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_ExistingReport", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		testReport := &report.Report{
			ID:        uuid.New(),
			Name:      "Report to Delete",
			Type:      report.TypeExpenses,
			Period:    report.PeriodMonthly,
			UserID:    uuid.MustParse(userID),
			StartDate: date.New(2024, time.Month(1), 1),
			EndDate:   date.New(2024, time.Month(1), 31),
			Data:      report.Data{},
		}

		err = repo.Create(ctx, testReport)
		require.NoError(t, err)

		err = repo.Delete(ctx, testReport.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, testReport.ID)
		require.Error(t, err)
	})

	t.Run("Error_NonExistentReport", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		nonExistentID := uuid.New()
		err = repo.Delete(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error_InvalidID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		err := repo.Delete(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid id parameter")
	})
}

func TestReportRepositorySQLite_GetSummary(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_WithReports", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create different types of reports
		reportTypes := []report.Type{
			report.TypeExpenses,
			report.TypeIncome,
			report.TypeBudget,
			report.TypeCashFlow,
		}

		for _, reportType := range reportTypes {
			testReport := &report.Report{
				ID:        uuid.New(),
				Name:      string(reportType) + " Report",
				Type:      reportType,
				Period:    report.PeriodMonthly,
				UserID:    uuid.MustParse(userID),
				StartDate: date.New(2024, time.Month(1), 1),
				EndDate:   date.New(2024, time.Month(1), 31),
				Data:      report.Data{},
			}
			err = repo.Create(ctx, testReport)
			require.NoError(t, err)
		}

		summary, err := repo.GetSummary(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Equal(t, uuid.MustParse(familyID), summary.FamilyID)
		assert.Equal(t, 4, summary.TotalReports)
		assert.Equal(t, 1, summary.ExpenseReports)
		assert.Equal(t, 1, summary.IncomeReports)
		assert.Equal(t, 1, summary.BudgetReports)
		assert.Equal(t, 1, summary.CashFlowReports)
		assert.NotNil(t, summary.LastGenerated)
	})

	t.Run("Success_EmptySummary", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		summary, err := repo.GetSummary(ctx, uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Equal(t, uuid.MustParse(familyID), summary.FamilyID)
		assert.Equal(t, 0, summary.TotalReports)
		assert.Nil(t, summary.LastGenerated)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		_, err := repo.GetSummary(ctx, uuid.Nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

func TestReportRepositorySQLite_GenerateExpenseReport(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_MultipleCategories", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create categories
		foodCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		transportCatID, err := helper.CreateTestCategory(ctx, "Transport", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions
		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Food: 800 (500 + 300)
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			50_000,
			"Groceries",
			"expense",
			foodCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			30_000,
			"Restaurant",
			"expense",
			foodCatID,
			userID,
			familyID,
			"2024-01-10",
		)
		require.NoError(t, err)

		// Transport: 200
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			20_000,
			"Gas",
			"expense",
			transportCatID,
			userID,
			familyID,
			"2024-01-15",
		)
		require.NoError(t, err)

		data, err := repo.GenerateExpenseReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(100_000), data.TotalExpensesMinor)
		assert.Len(t, data.CategoryBreakdown, 2)

		// Verify Food category
		var foodItem *report.CategoryReportItem
		for i := range data.CategoryBreakdown {
			if data.CategoryBreakdown[i].CategoryName == "Food" {
				foodItem = &data.CategoryBreakdown[i]
				break
			}
		}
		require.NotNil(t, foodItem)
		assert.Equal(t, money.Minor(80_000), foodItem.AmountMinor)
		assert.InDelta(t, 80.0, foodItem.Percentage, 0.01)
		assert.Equal(t, 2, foodItem.Count)

		// Verify top expenses
		assert.NotEmpty(t, data.TopExpenses)
		assert.LessOrEqual(t, len(data.TopExpenses), 10)
	})

	t.Run("Success_NoExpenses", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateExpenseReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(0), data.TotalExpensesMinor)
		assert.Empty(t, data.CategoryBreakdown)
		assert.Empty(t, data.TopExpenses)
	})

	t.Run("Success_DateRangeFilter", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		foodCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		// Transaction in January
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			50_000,
			"January",
			"expense",
			foodCatID,
			userID,
			familyID,
			"2024-01-15",
		)
		require.NoError(t, err)

		// Transaction in February (should be excluded)
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			30_000,
			"February",
			"expense",
			foodCatID,
			userID,
			familyID,
			"2024-02-15",
		)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateExpenseReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(50_000), data.TotalExpensesMinor)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		_, err := repo.GenerateExpenseReport(ctx, uuid.Nil, startDate, endDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

func TestReportRepositorySQLite_GenerateIncomeReport(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_MultipleSources", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		// Create income categories
		salaryCatID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
		require.NoError(t, err)

		freelanceCatID, err := helper.CreateTestCategory(ctx, "Freelance", "income", familyID, nil)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Salary: 5000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			500_000,
			"Monthly salary",
			"income",
			salaryCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)

		// Freelance: 1000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			100_000,
			"Project payment",
			"income",
			freelanceCatID,
			userID,
			familyID,
			"2024-01-15",
		)
		require.NoError(t, err)

		data, err := repo.GenerateIncomeReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(600_000), data.TotalIncomeMinor)
		assert.Len(t, data.CategoryBreakdown, 2)

		// Verify Salary category
		var salaryItem *report.CategoryReportItem
		for i := range data.CategoryBreakdown {
			if data.CategoryBreakdown[i].CategoryName == "Salary" {
				salaryItem = &data.CategoryBreakdown[i]
				break
			}
		}
		require.NotNil(t, salaryItem)
		assert.Equal(t, money.Minor(500_000), salaryItem.AmountMinor)
		assert.InDelta(t, 83.33, salaryItem.Percentage, 0.01)
		assert.Equal(t, 1, salaryItem.Count)
	})

	t.Run("Success_NoIncome", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateIncomeReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(0), data.TotalIncomeMinor)
		assert.Empty(t, data.CategoryBreakdown)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		_, err := repo.GenerateIncomeReport(ctx, uuid.Nil, startDate, endDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

func TestReportRepositorySQLite_GenerateCashFlowReport(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_PositiveFlow", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		incomeCatID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
		require.NoError(t, err)

		expenseCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Income: 5000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			500_000,
			"Salary",
			"income",
			incomeCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)

		// Expenses: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			300_000,
			"Food",
			"expense",
			expenseCatID,
			userID,
			familyID,
			"2024-01-10",
		)
		require.NoError(t, err)

		data, err := repo.GenerateCashFlowReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(500_000), data.TotalIncomeMinor)
		assert.Equal(t, money.Minor(300_000), data.TotalExpensesMinor)
		assert.Equal(t, money.Minor(200_000), data.NetIncomeMinor)
	})

	t.Run("Success_NegativeFlow", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		incomeCatID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
		require.NoError(t, err)

		expenseCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Income: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			300_000,
			"Salary",
			"income",
			incomeCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)

		// Expenses: 5000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			500_000,
			"Food",
			"expense",
			expenseCatID,
			userID,
			familyID,
			"2024-01-10",
		)
		require.NoError(t, err)

		data, err := repo.GenerateCashFlowReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(300_000), data.TotalIncomeMinor)
		assert.Equal(t, money.Minor(500_000), data.TotalExpensesMinor)
		assert.Equal(t, money.Minor(-200_000), data.NetIncomeMinor)
	})

	t.Run("Success_ZeroFlow", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		incomeCatID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
		require.NoError(t, err)

		expenseCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Income: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			300_000,
			"Salary",
			"income",
			incomeCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)

		// Expenses: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			300_000,
			"Food",
			"expense",
			expenseCatID,
			userID,
			familyID,
			"2024-01-10",
		)
		require.NoError(t, err)

		data, err := repo.GenerateCashFlowReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(300_000), data.TotalIncomeMinor)
		assert.Equal(t, money.Minor(300_000), data.TotalExpensesMinor)
		assert.Equal(t, money.Minor(0), data.NetIncomeMinor)
	})

	t.Run("Success_DailyBreakdown", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "password", "Test User", "admin", familyID)
		require.NoError(t, err)

		incomeCatID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
		require.NoError(t, err)

		expenseCatID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		// Day 1: Income 1000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			100_000,
			"Income Day 1",
			"income",
			incomeCatID,
			userID,
			familyID,
			"2024-01-05",
		)
		require.NoError(t, err)

		// Day 2: Expense 500
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			50_000,
			"Expense Day 2",
			"expense",
			expenseCatID,
			userID,
			familyID,
			"2024-01-10",
		)
		require.NoError(t, err)

		data, err := repo.GenerateCashFlowReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.DailyBreakdown)
		assert.Len(t, data.DailyBreakdown, 2)

		// Verify daily balances
		for _, daily := range data.DailyBreakdown {
			assert.Equal(t, daily.IncomeMinor-daily.ExpensesMinor, daily.BalanceMinor)
		}
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		_, err := repo.GenerateCashFlowReport(ctx, uuid.Nil, startDate, endDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

func TestReportRepositorySQLite_GenerateBudgetComparisonReport(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Success_UnderBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create budget
		budgetID, err := helper.CreateTestBudgetWithDates(
			ctx,
			"Food Budget",
			100_000,
			80_000,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.BudgetComparison)

		// Find the budget in comparison
		var budgetItem *report.BudgetComparisonItem
		for i := range data.BudgetComparison {
			if data.BudgetComparison[i].BudgetID == uuid.MustParse(budgetID) {
				budgetItem = &data.BudgetComparison[i]
				break
			}
		}
		require.NotNil(t, budgetItem)
		assert.Equal(t, money.Minor(100_000), budgetItem.PlannedMinor)
		assert.Equal(t, money.Minor(80_000), budgetItem.ActualMinor)
		assert.Equal(t, money.Minor(-20_000), budgetItem.DifferenceMinor)
		assert.InDelta(t, 80.0, budgetItem.Percentage, 0.01)
	})

	t.Run("Success_OverBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create budget with overspending
		_, err = helper.CreateTestBudgetWithDates(
			ctx,
			"Transport Budget",
			50_000,
			70_000,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.BudgetComparison)

		item := data.BudgetComparison[0]
		assert.Equal(t, money.Minor(50_000), item.PlannedMinor)
		assert.Equal(t, money.Minor(70_000), item.ActualMinor)
		assert.Equal(t, money.Minor(20_000), item.DifferenceMinor)
		assert.InDelta(t, 140.0, item.Percentage, 0.01)
	})

	t.Run("Success_OnBudget", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create budget exactly on target
		_, err = helper.CreateTestBudgetWithDates(
			ctx,
			"Entertainment Budget",
			30_000,
			30_000,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.BudgetComparison)

		item := data.BudgetComparison[0]
		assert.Equal(t, money.Minor(30_000), item.PlannedMinor)
		assert.Equal(t, money.Minor(30_000), item.ActualMinor)
		assert.Equal(t, money.Minor(0), item.DifferenceMinor)
		assert.InDelta(t, 100.0, item.Percentage, 0.01)
	})

	t.Run("Success_NoBudgets", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Empty(t, data.BudgetComparison)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := date.New(2024, time.January, 1)
		endDate := date.New(2024, time.January, 31)

		_, err := repo.GenerateBudgetComparisonReport(ctx, uuid.Nil, startDate, endDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

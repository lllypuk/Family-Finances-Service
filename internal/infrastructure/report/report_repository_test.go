package report_test

import (
	"context"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			Data: report.Data{
				TotalExpenses: 1000.0,
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
		assert.InDelta(t, testReport.Data.TotalExpenses, retrieved.Data.TotalExpenses, 0.01)
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
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
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
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
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
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 1, 0),
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
			StartDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
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
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			Data: report.Data{
				TotalIncome: 5000.0,
			},
		}

		err = repo.Create(ctx, testReport)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, testReport.ID)
		require.NoError(t, err)
		assert.Equal(t, testReport.ID, retrieved.ID)
		assert.Equal(t, testReport.Name, retrieved.Name)
		assert.InDelta(t, testReport.Data.TotalIncome, retrieved.Data.TotalIncome, 0.01)
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
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
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
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Food: 800 (500 + 300)
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			500.0,
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
			300.0,
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
			200.0,
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
		assert.InDelta(t, 1000.0, data.TotalExpenses, 0.01)
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
		assert.InDelta(t, 800.0, foodItem.Amount, 0.01)
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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateExpenseReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, data.TotalExpenses, 0.01)
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
			500.0,
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
			300.0,
			"February",
			"expense",
			foodCatID,
			userID,
			familyID,
			"2024-02-15",
		)
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateExpenseReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.InDelta(t, 500.0, data.TotalExpenses, 0.01)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Salary: 5000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			5000.0,
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
			1000.0,
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
		assert.InDelta(t, 6000.0, data.TotalIncome, 0.01)
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
		assert.InDelta(t, 5000.0, salaryItem.Amount, 0.01)
		assert.InDelta(t, 83.33, salaryItem.Percentage, 0.01)
		assert.Equal(t, 1, salaryItem.Count)
	})

	t.Run("Success_NoIncome", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateIncomeReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.InDelta(t, 0.0, data.TotalIncome, 0.01)
		assert.Empty(t, data.CategoryBreakdown)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Income: 5000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			5000.0,
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
			3000.0,
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
		assert.InDelta(t, 5000.0, data.TotalIncome, 0.01)
		assert.InDelta(t, 3000.0, data.TotalExpenses, 0.01)
		assert.InDelta(t, 2000.0, data.NetIncome, 0.01)
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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Income: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			3000.0,
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
			5000.0,
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
		assert.InDelta(t, 3000.0, data.TotalIncome, 0.01)
		assert.InDelta(t, 5000.0, data.TotalExpenses, 0.01)
		assert.InDelta(t, -2000.0, data.NetIncome, 0.01)
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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Income: 3000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			3000.0,
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
			3000.0,
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
		assert.InDelta(t, 3000.0, data.TotalIncome, 0.01)
		assert.InDelta(t, 3000.0, data.TotalExpenses, 0.01)
		assert.InDelta(t, 0.0, data.NetIncome, 0.01)
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

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		// Day 1: Income 1000
		_, err = helper.CreateTestTransactionWithDate(
			ctx,
			1000.0,
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
			500.0,
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
			assert.InDelta(t, daily.Income-daily.Expenses, daily.Balance, 0.01)
		}
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

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
			1000.0,
			800.0,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

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
		assert.InDelta(t, 1000.0, budgetItem.Planned, 0.01)
		assert.InDelta(t, 800.0, budgetItem.Actual, 0.01)
		assert.InDelta(t, -200.0, budgetItem.Difference, 0.01)
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
			500.0,
			700.0,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.BudgetComparison)

		item := data.BudgetComparison[0]
		assert.InDelta(t, 500.0, item.Planned, 0.01)
		assert.InDelta(t, 700.0, item.Actual, 0.01)
		assert.InDelta(t, 200.0, item.Difference, 0.01)
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
			300.0,
			300.0,
			"monthly",
			familyID,
			"2024-01-01",
			"2024-01-31",
			nil,
		)
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, data.BudgetComparison)

		item := data.BudgetComparison[0]
		assert.InDelta(t, 300.0, item.Planned, 0.01)
		assert.InDelta(t, 300.0, item.Actual, 0.01)
		assert.InDelta(t, 0.0, item.Difference, 0.01)
		assert.InDelta(t, 100.0, item.Percentage, 0.01)
	})

	t.Run("Success_NoBudgets", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		data, err := repo.GenerateBudgetComparisonReport(ctx, uuid.MustParse(familyID), startDate, endDate)
		require.NoError(t, err)
		assert.Empty(t, data.BudgetComparison)
	})

	t.Run("Error_InvalidFamilyID", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := reportrepo.NewSQLiteRepository(db)

		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

		_, err := repo.GenerateBudgetComparisonReport(ctx, uuid.Nil, startDate, endDate)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid family ID")
	})
}

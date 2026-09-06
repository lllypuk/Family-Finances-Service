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

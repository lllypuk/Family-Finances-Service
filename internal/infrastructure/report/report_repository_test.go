package report_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	reportrepo "family-budget-service/internal/infrastructure/report"
	"family-budget-service/internal/testhelpers"
)

func TestReportRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := reportrepo.NewRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testReport := testhelpers.CreateTestReport(family.ID, user.ID)

		err := repo.Create(context.Background(), testReport)
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testReport := testhelpers.CreateTestReport(family.ID, user.ID)

		err := repo.Create(context.Background(), testReport)
		require.NoError(t, err)

		retrievedReport, err := repo.GetByID(context.Background(), testReport.ID)
		require.NoError(t, err)
		assert.Equal(t, testReport.ID, retrievedReport.ID)
		assert.Equal(t, testReport.Name, retrievedReport.Name)
		assert.Equal(t, testReport.Type, retrievedReport.Type)
		assert.Equal(t, testReport.Period, retrievedReport.Period)
		assert.Equal(t, testReport.FamilyID, retrievedReport.FamilyID)
		assert.Equal(t, testReport.UserID, retrievedReport.UserID)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByFamilyID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user1 := testhelpers.CreateTestUser(family.ID)
		user1.Email = "user1@example.com"
		user2 := testhelpers.CreateTestUser(family.ID)
		user2.Email = "user2@example.com"

		report1 := testhelpers.CreateTestReport(family.ID, user1.ID)
		report1.Name = "Monthly Expenses Report"
		report2 := testhelpers.CreateTestReport(family.ID, user2.ID)
		report2.Name = "Weekly Income Report"

		err := repo.Create(context.Background(), report1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), report2)
		require.NoError(t, err)

		reports, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, reports, 2)

		reportIDs := make([]uuid.UUID, len(reports))
		for i, r := range reports {
			reportIDs[i] = r.ID
		}
		assert.Contains(t, reportIDs, report1.ID)
		assert.Contains(t, reportIDs, report2.ID)
	})

	t.Run("GetByFamilyID_EmptyResult", func(t *testing.T) {
		nonExistentFamilyID := uuid.New()
		reports, err := repo.GetByFamilyID(context.Background(), nonExistentFamilyID)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("GetByUserID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user1 := testhelpers.CreateTestUser(family.ID)
		user1.Email = "user1@example.com"
		user2 := testhelpers.CreateTestUser(family.ID)
		user2.Email = "user2@example.com"

		// Create reports for user1
		report1 := testhelpers.CreateTestReport(family.ID, user1.ID)
		report1.Name = "User1 Report 1"
		report2 := testhelpers.CreateTestReport(family.ID, user1.ID)
		report2.Name = "User1 Report 2"

		// Create report for user2
		report3 := testhelpers.CreateTestReport(family.ID, user2.ID)
		report3.Name = "User2 Report"

		err := repo.Create(context.Background(), report1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), report2)
		require.NoError(t, err)
		err = repo.Create(context.Background(), report3)
		require.NoError(t, err)

		// Get reports for user1
		user1Reports, err := repo.GetByUserID(context.Background(), user1.ID)
		require.NoError(t, err)
		assert.Len(t, user1Reports, 2)

		user1ReportIDs := make([]uuid.UUID, len(user1Reports))
		for i, r := range user1Reports {
			user1ReportIDs[i] = r.ID
		}
		assert.Contains(t, user1ReportIDs, report1.ID)
		assert.Contains(t, user1ReportIDs, report2.ID)
		assert.NotContains(t, user1ReportIDs, report3.ID)

		// Get reports for user2
		user2Reports, err := repo.GetByUserID(context.Background(), user2.ID)
		require.NoError(t, err)
		assert.Len(t, user2Reports, 1)
		assert.Equal(t, report3.ID, user2Reports[0].ID)
	})

	t.Run("GetByUserID_EmptyResult", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		reports, err := repo.GetByUserID(context.Background(), nonExistentUserID)
		require.NoError(t, err)
		assert.Empty(t, reports)
	})

	t.Run("Delete_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testReport := testhelpers.CreateTestReport(family.ID, user.ID)

		err := repo.Create(context.Background(), testReport)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testReport.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), testReport.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	// Sorting by generated_at should be descending
	t.Run("GetByFamilyID_SortedByGeneratedAt", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)

		older := testhelpers.CreateTestReport(family.ID, user.ID)
		older.GeneratedAt = time.Now().Add(-2 * time.Hour)
		middle := testhelpers.CreateTestReport(family.ID, user.ID)
		middle.GeneratedAt = time.Now().Add(-1 * time.Hour)
		newest := testhelpers.CreateTestReport(family.ID, user.ID)
		newest.GeneratedAt = time.Now()

		require.NoError(t, repo.Create(context.Background(), older))
		require.NoError(t, repo.Create(context.Background(), middle))
		require.NoError(t, repo.Create(context.Background(), newest))

		reports, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		require.Len(t, reports, 3)
		assert.Equal(t, newest.ID, reports[0].ID)
		assert.Equal(t, middle.ID, reports[1].ID)
		assert.Equal(t, older.ID, reports[2].ID)
	})

	t.Run("GetByUserID_SortedByGeneratedAt", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)

		older := testhelpers.CreateTestReport(family.ID, user.ID)
		older.GeneratedAt = time.Now().Add(-3 * time.Hour)
		newer := testhelpers.CreateTestReport(family.ID, user.ID)
		newer.GeneratedAt = time.Now().Add(-1 * time.Hour)

		require.NoError(t, repo.Create(context.Background(), older))
		require.NoError(t, repo.Create(context.Background(), newer))

		userReports, err := repo.GetByUserID(context.Background(), user.ID)
		require.NoError(t, err)
		require.Len(t, userReports, 2)
		assert.Equal(t, newer.ID, userReports[0].ID)
		assert.Equal(t, older.ID, userReports[1].ID)
	})
}

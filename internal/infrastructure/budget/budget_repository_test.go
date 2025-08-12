package budget_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	"family-budget-service/internal/testhelpers"
)

func TestBudgetRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := budgetrepo.NewRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		err := repo.Create(context.Background(), testBudget)
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		err := repo.Create(context.Background(), testBudget)
		require.NoError(t, err)

		retrievedBudget, err := repo.GetByID(context.Background(), testBudget.ID)
		require.NoError(t, err)
		assert.Equal(t, testBudget.ID, retrievedBudget.ID)
		assert.Equal(t, testBudget.Name, retrievedBudget.Name)
		assert.Equal(t, testBudget.Amount, retrievedBudget.Amount)
		assert.Equal(t, testBudget.FamilyID, retrievedBudget.FamilyID)
		assert.Equal(t, testBudget.CategoryID, retrievedBudget.CategoryID)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(context.Background(), nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByFamilyID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory1 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testCategory2 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		budget1 := testhelpers.CreateTestBudget(family.ID, testCategory1.ID)
		budget1.Name = "Food Budget"
		budget2 := testhelpers.CreateTestBudget(family.ID, testCategory2.ID)
		budget2.Name = "Transport Budget"

		err := repo.Create(context.Background(), budget1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), budget2)
		require.NoError(t, err)

		budgets, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, budgets, 2)

		budgetIDs := make([]uuid.UUID, len(budgets))
		for i, b := range budgets {
			budgetIDs[i] = b.ID
		}
		assert.Contains(t, budgetIDs, budget1.ID)
		assert.Contains(t, budgetIDs, budget2.ID)
	})

	t.Run("GetByFamilyID_EmptyResult", func(t *testing.T) {
		nonExistentFamilyID := uuid.New()
		budgets, err := repo.GetByFamilyID(context.Background(), nonExistentFamilyID)
		require.NoError(t, err)
		assert.Len(t, budgets, 0)
	})

	t.Run("GetActiveBudgets_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		now := time.Now()

		// Active budget (current period)
		activeBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		activeBudget.IsActive = true
		activeBudget.StartDate = now.AddDate(0, 0, -10) // 10 days ago
		activeBudget.EndDate = now.AddDate(0, 0, 10)    // 10 days from now

		// Inactive budget
		inactiveBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		inactiveBudget.IsActive = false
		inactiveBudget.StartDate = now.AddDate(0, 0, -10)
		inactiveBudget.EndDate = now.AddDate(0, 0, 10)

		// Expired budget
		expiredBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		expiredBudget.IsActive = true
		expiredBudget.StartDate = now.AddDate(0, 0, -30) // 30 days ago
		expiredBudget.EndDate = now.AddDate(0, 0, -10)   // 10 days ago

		// Future budget
		futureBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		futureBudget.IsActive = true
		futureBudget.StartDate = now.AddDate(0, 0, 10) // 10 days from now
		futureBudget.EndDate = now.AddDate(0, 0, 30)   // 30 days from now

		err := repo.Create(context.Background(), activeBudget)
		require.NoError(t, err)
		err = repo.Create(context.Background(), inactiveBudget)
		require.NoError(t, err)
		err = repo.Create(context.Background(), expiredBudget)
		require.NoError(t, err)
		err = repo.Create(context.Background(), futureBudget)
		require.NoError(t, err)

		activeBudgets, err := repo.GetActiveBudgets(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, activeBudgets, 1)
		assert.Equal(t, activeBudget.ID, activeBudgets[0].ID)
	})

	t.Run("GetActiveBudgets_EmptyResult", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		activeBudgets, err := repo.GetActiveBudgets(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, activeBudgets, 0)
	})

	t.Run("Update_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		err := repo.Create(context.Background(), testBudget)
		require.NoError(t, err)

		testBudget.Name = "Updated Budget Name"
		testBudget.Amount = 2000.0
		err = repo.Update(context.Background(), testBudget)
		require.NoError(t, err)

		retrievedBudget, err := repo.GetByID(context.Background(), testBudget.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Budget Name", retrievedBudget.Name)
		assert.Equal(t, 2000.0, retrievedBudget.Amount)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		nonExistentBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		err := repo.Update(context.Background(), nonExistentBudget)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testBudget := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		err := repo.Create(context.Background(), testBudget)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testBudget.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), testBudget.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetActiveBudgets_InclusiveEdges", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		now := time.Now()

		// start == now, end > now
		activeStartEqNow := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		activeStartEqNow.IsActive = true
		activeStartEqNow.StartDate = now
		activeStartEqNow.EndDate = now.Add(2 * time.Hour)

		// start < now, end == now
		activeEndEqNow := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		activeEndEqNow.IsActive = true
		activeEndEqNow.StartDate = now.Add(-2 * time.Hour)
		activeEndEqNow.EndDate = now

		// start == now, end == now
		activeBothEqNow := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		activeBothEqNow.IsActive = true
		activeBothEqNow.StartDate = now
		activeBothEqNow.EndDate = now

		// Inactive examples (should not be returned)
		inactiveFuture := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		inactiveFuture.IsActive = true
		inactiveFuture.StartDate = now.Add(1 * time.Hour)
		inactiveFuture.EndDate = now.Add(3 * time.Hour)

		inactivePast := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		inactivePast.IsActive = true
		inactivePast.StartDate = now.Add(-3 * time.Hour)
		inactivePast.EndDate = now.Add(-1 * time.Hour)

		require.NoError(t, repo.Create(context.Background(), activeStartEqNow))
		require.NoError(t, repo.Create(context.Background(), activeEndEqNow))
		require.NoError(t, repo.Create(context.Background(), activeBothEqNow))
		require.NoError(t, repo.Create(context.Background(), inactiveFuture))
		require.NoError(t, repo.Create(context.Background(), inactivePast))

		activeBudgets, err := repo.GetActiveBudgets(context.Background(), family.ID)
		require.NoError(t, err)

		var ids []uuid.UUID
		for _, b := range activeBudgets {
			ids = append(ids, b.ID)
		}

		assert.Contains(t, ids, activeStartEqNow.ID)
		assert.NotContains(t, ids, inactiveFuture.ID)
		assert.NotContains(t, ids, inactivePast.ID)
	})

	t.Run("GetByFamilyID_SortedByCreatedAt", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		older := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		middle := testhelpers.CreateTestBudget(family.ID, testCategory.ID)
		newest := testhelpers.CreateTestBudget(family.ID, testCategory.ID)

		now := time.Now()
		older.CreatedAt = now.Add(-3 * time.Hour)
		middle.CreatedAt = now.Add(-2 * time.Hour)
		newest.CreatedAt = now.Add(-1 * time.Hour)

		require.NoError(t, repo.Create(context.Background(), older))
		require.NoError(t, repo.Create(context.Background(), middle))
		require.NoError(t, repo.Create(context.Background(), newest))

		budgets, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(budgets), 3)

		// Expect newest first (descending by created_at)
		assert.Equal(t, newest.ID, budgets[0].ID)
		assert.Equal(t, middle.ID, budgets[1].ID)
		assert.Equal(t, older.ID, budgets[2].ID)
	})
}

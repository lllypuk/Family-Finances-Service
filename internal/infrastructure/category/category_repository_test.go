package category_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	"family-budget-service/internal/testhelpers"
)

func TestCategoryRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := categoryrepo.NewRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		err := repo.Create(context.Background(), testCategory)
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		err := repo.Create(context.Background(), testCategory)
		require.NoError(t, err)

		retrievedCategory, err := repo.GetByID(context.Background(), testCategory.ID)
		require.NoError(t, err)
		assert.Equal(t, testCategory.ID, retrievedCategory.ID)
		assert.Equal(t, testCategory.Name, retrievedCategory.Name)
		assert.Equal(t, testCategory.Type, retrievedCategory.Type)
		assert.Equal(t, testCategory.FamilyID, retrievedCategory.FamilyID)
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
		testCategory1.Name = "Food"
		testCategory2 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeIncome)
		testCategory2.Name = "Salary"

		err := repo.Create(context.Background(), testCategory1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), testCategory2)
		require.NoError(t, err)

		categories, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, categories, 2)

		categoryIDs := make([]uuid.UUID, len(categories))
		for i, c := range categories {
			categoryIDs[i] = c.ID
		}
		assert.Contains(t, categoryIDs, testCategory1.ID)
		assert.Contains(t, categoryIDs, testCategory2.ID)
	})

	t.Run("GetByFamilyID_OnlyActiveCategories", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory1 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testCategory1.IsActive = true
		testCategory2 := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testCategory2.IsActive = false

		err := repo.Create(context.Background(), testCategory1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), testCategory2)
		require.NoError(t, err)

		categories, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, categories, 1)
		assert.Equal(t, testCategory1.ID, categories[0].ID)
	})

	t.Run("GetByType_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		expenseCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		incomeCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeIncome)

		err := repo.Create(context.Background(), expenseCategory)
		require.NoError(t, err)
		err = repo.Create(context.Background(), incomeCategory)
		require.NoError(t, err)

		expenseCategories, err := repo.GetByType(context.Background(), family.ID, category.CategoryTypeExpense)
		require.NoError(t, err)
		assert.Len(t, expenseCategories, 1)
		assert.Equal(t, expenseCategory.ID, expenseCategories[0].ID)

		incomeCategories, err := repo.GetByType(context.Background(), family.ID, category.CategoryTypeIncome)
		require.NoError(t, err)
		assert.Len(t, incomeCategories, 1)
		assert.Equal(t, incomeCategory.ID, incomeCategories[0].ID)
	})

	t.Run("Update_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		err := repo.Create(context.Background(), testCategory)
		require.NoError(t, err)

		testCategory.Name = "Updated Category Name"
		testCategory.Color = "#FF5733"
		err = repo.Update(context.Background(), testCategory)
		require.NoError(t, err)

		retrievedCategory, err := repo.GetByID(context.Background(), testCategory.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Category Name", retrievedCategory.Name)
		assert.Equal(t, "#FF5733", retrievedCategory.Color)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		nonExistentCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		err := repo.Update(context.Background(), nonExistentCategory)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_SoftDelete", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		err := repo.Create(context.Background(), testCategory)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testCategory.ID)
		require.NoError(t, err)

		// Category should still exist but be inactive
		retrievedCategory, err := repo.GetByID(context.Background(), testCategory.ID)
		require.NoError(t, err)
		assert.False(t, retrievedCategory.IsActive)

		// Should not appear in family categories list
		categories, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Empty(t, categories)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

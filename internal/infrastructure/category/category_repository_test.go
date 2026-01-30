package category_test

import (
	"context"
	"testing"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	categoryrepo "family-budget-service/internal/infrastructure/category"
)

func TestCategoryRepositorySQLite_Integration(t *testing.T) {
	// Setup SQLite in-memory database
	container := testutils.SetupSQLiteTestDB(t)

	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create category
		testCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Groceries",
			Type: category.TypeExpense,

			IsActive: true,
		}

		err = repo.Create(ctx, testCategory)
		require.NoError(t, err)

		// Verify category was created
		retrievedCategory, err := repo.GetByID(ctx, testCategory.ID)
		require.NoError(t, err)
		assert.Equal(t, testCategory.ID, retrievedCategory.ID)
		assert.Equal(t, testCategory.Name, retrievedCategory.Name)
		assert.Equal(t, testCategory.Type, retrievedCategory.Type)
		assert.Equal(t, testCategory.IsActive, retrievedCategory.IsActive)
	})

	t.Run("Create_WithParent_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Hierarchy Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create parent category
		parentCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Food & Beverages",
			Type: category.TypeExpense,

			IsActive: true,
		}

		err = repo.Create(ctx, parentCategory)
		require.NoError(t, err)

		// Create child category
		childCategory := &category.Category{
			ID:       uuid.New(),
			Name:     "Groceries",
			Type:     category.TypeExpense,
			ParentID: &parentCategory.ID,

			IsActive: true,
		}

		err = repo.Create(ctx, childCategory)
		require.NoError(t, err)

		// Verify child category
		retrievedChild, err := repo.GetByID(ctx, childCategory.ID)
		require.NoError(t, err)
		assert.Equal(t, childCategory.ParentID, retrievedChild.ParentID)
		assert.True(t, retrievedChild.IsSubcategory())
	})

	t.Run("GetCategoryChildren_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Children Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create parent category
		parentCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Transportation",
			Type: category.TypeExpense,

			IsActive: true,
		}

		err = repo.Create(ctx, parentCategory)
		require.NoError(t, err)

		// Create child categories
		childCategories := []*category.Category{
			{
				ID:       uuid.New(),
				Name:     "Public Transport",
				Type:     category.TypeExpense,
				ParentID: &parentCategory.ID,

				IsActive: true,
			},
			{
				ID:       uuid.New(),
				Name:     "Car Expenses",
				Type:     category.TypeExpense,
				ParentID: &parentCategory.ID,

				IsActive: true,
			},
		}

		for _, child := range childCategories {
			err = repo.Create(ctx, child)
			require.NoError(t, err)
		}

		// Create grandchild category
		grandChild := &category.Category{
			ID:       uuid.New(),
			Name:     "Gas",
			Type:     category.TypeExpense,
			ParentID: &childCategories[1].ID, // Under "Car Expenses"

			IsActive: true,
		}

		err = repo.Create(ctx, grandChild)
		require.NoError(t, err)

		// Get all children of parent category
		children, err := repo.GetCategoryChildren(ctx, parentCategory.ID)
		require.NoError(t, err)
		assert.Len(t, children, 4) // Parent + 2 children + 1 grandchild

		// Verify hierarchy levels
		levelCounts := make(map[int]int)
		for _, child := range children {
			levelCounts[child.Level]++
		}

		assert.Equal(t, 1, levelCounts[0]) // Parent (level 0)
		assert.Equal(t, 2, levelCounts[1]) // Direct children (level 1)
		assert.Equal(t, 1, levelCounts[2]) // Grandchild (level 2)
	})

	t.Run("GetCategoryPath_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Path Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create category hierarchy: Root -> Level1 -> Level2
		rootCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Housing",
			Type: category.TypeExpense,

			IsActive: true,
		}

		level1Category := &category.Category{
			ID:       uuid.New(),
			Name:     "Utilities",
			Type:     category.TypeExpense,
			ParentID: &rootCategory.ID,

			IsActive: true,
		}

		level2Category := &category.Category{
			ID:       uuid.New(),
			Name:     "Electricity",
			Type:     category.TypeExpense,
			ParentID: &level1Category.ID,

			IsActive: true,
		}

		// Create categories in order
		err = repo.Create(ctx, rootCategory)
		require.NoError(t, err)
		err = repo.Create(ctx, level1Category)
		require.NoError(t, err)
		err = repo.Create(ctx, level2Category)
		require.NoError(t, err)

		// Get path from deepest category to root
		path, err := repo.GetCategoryPath(ctx, level2Category.ID)
		require.NoError(t, err)
		assert.Len(t, path, 3) // Should include all 3 levels

		// Verify path order (root to leaf)
		assert.Equal(t, "Housing", path[0].Name)     // Root (level 0, but ordered first)
		assert.Equal(t, "Utilities", path[1].Name)   // Level 1
		assert.Equal(t, "Electricity", path[2].Name) // Level 2 (deepest)
	})

	t.Run("GetByFamilyIDAndType_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Type Filter Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create categories of different types
		expenseCategories := []*category.Category{
			{
				ID:   uuid.New(),
				Name: "Food",
				Type: category.TypeExpense,

				IsActive: true,
			},
			{
				ID:   uuid.New(),
				Name: "Transport",
				Type: category.TypeExpense,

				IsActive: true,
			},
		}

		incomeCategories := []*category.Category{
			{
				ID:   uuid.New(),
				Name: "Salary",
				Type: category.TypeIncome,

				IsActive: true,
			},
			{
				ID:   uuid.New(),
				Name: "Freelance",
				Type: category.TypeIncome,

				IsActive: true,
			},
		}

		// Create all categories
		expenseCategories = append(expenseCategories, incomeCategories...)
		for _, cat := range expenseCategories {
			err = repo.Create(ctx, cat)
			require.NoError(t, err)
		}

		// Get expense categories only
		expenseResults, err := repo.GetByFamilyIDAndType(ctx, familyUUID, category.TypeExpense)
		require.NoError(t, err)
		assert.Len(t, expenseResults, 2)
		for _, cat := range expenseResults {
			assert.Equal(t, category.TypeExpense, cat.Type)
		}

		// Get income categories only
		incomeResults, err := repo.GetByFamilyIDAndType(ctx, familyUUID, category.TypeIncome)
		require.NoError(t, err)
		assert.Len(t, incomeResults, 2)
		for _, cat := range incomeResults {
			assert.Equal(t, category.TypeIncome, cat.Type)
		}
	})

	t.Run("Update_PreventCircularReference", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Circular Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create parent and child categories
		parentCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Parent",
			Type: category.TypeExpense,

			IsActive: true,
		}

		childCategory := &category.Category{
			ID:       uuid.New(),
			Name:     "Child",
			Type:     category.TypeExpense,
			ParentID: &parentCategory.ID,

			IsActive: true,
		}

		err = repo.Create(ctx, parentCategory)
		require.NoError(t, err)
		err = repo.Create(ctx, childCategory)
		require.NoError(t, err)

		// Try to create circular reference (set parent's parent to child)
		parentCategory.ParentID = &childCategory.ID

		err = repo.Update(ctx, parentCategory)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular reference")
	})

	t.Run("Delete_WithChildren_ShouldFail", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Delete Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create parent category
		parentCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Parent with Children",
			Type: category.TypeExpense,

			IsActive: true,
		}

		err = repo.Create(ctx, parentCategory)
		require.NoError(t, err)

		// Create child category
		childCategory := &category.Category{
			ID:       uuid.New(),
			Name:     "Child",
			Type:     category.TypeExpense,
			ParentID: &parentCategory.ID,

			IsActive: true,
		}

		err = repo.Create(ctx, childCategory)
		require.NoError(t, err)

		// Try to delete parent with children - should fail
		err = repo.Delete(ctx, parentCategory.ID, familyUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete category with subcategories")
	})

	t.Run("Delete_LeafCategory_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := categoryrepo.NewSQLiteRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Delete Leaf Family", "USD")
		require.NoError(t, err)

		// Create leaf category (no children)
		leafCategory := &category.Category{
			ID:   uuid.New(),
			Name: "Leaf Category",
			Type: category.TypeExpense,

			IsActive: true,
		}

		err = repo.Create(ctx, leafCategory)
		require.NoError(t, err)

		// Delete leaf category - should succeed
		err = repo.Delete(ctx, leafCategory.ID, leafCategory.FamilyID)
		require.NoError(t, err)

		// Verify category is soft deleted (should not be found)
		deletedCategory, err := repo.GetByID(ctx, leafCategory.ID)
		if err != nil {
			assert.Contains(t, err.Error(), "not found")
		} else {
			// If no error, category should be marked as inactive/deleted
			assert.NotNil(t, deletedCategory)
			assert.False(t, deletedCategory.IsActive, "Deleted category should be inactive")
		}
	})
}

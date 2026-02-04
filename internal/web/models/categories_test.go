package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/web/models"
)

func TestCategoryForm_ToCategoryType(t *testing.T) {
	tests := []struct {
		name     string
		formType string
		expected category.Type
	}{
		{"income", "income", category.TypeIncome},
		{"expense", "expense", category.TypeExpense},
		{"default expense", "invalid", category.TypeExpense},
		{"empty defaults to expense", "", category.TypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.CategoryForm{Type: tt.formType}
			result := form.ToCategoryType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategoryForm_GetParentID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name      string
		parentID  string
		expectNil bool
	}{
		{
			name:      "valid UUID",
			parentID:  validUUID.String(),
			expectNil: false,
		},
		{
			name:      "empty string",
			parentID:  "",
			expectNil: true,
		},
		{
			name:      "invalid UUID",
			parentID:  "invalid-uuid",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.CategoryForm{ParentID: tt.parentID}
			result := form.GetParentID()

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, validUUID, *result)
			}
		})
	}
}

func TestCategoryViewModel_FromDomain(t *testing.T) {
	parentID := uuid.New()
	cat := &category.Category{
		ID:       uuid.New(),
		Name:     "Groceries",
		Type:     category.TypeExpense,
		Color:    "#FF5733",
		Icon:     "🛒",
		ParentID: &parentID,
		IsActive: true,
	}

	vm := &models.CategoryViewModel{}
	vm.FromDomain(cat)

	assert.Equal(t, cat.ID, vm.ID)
	assert.Equal(t, cat.Name, vm.Name)
	assert.Equal(t, cat.Type, vm.Type)
	assert.Equal(t, cat.Color, vm.Color)
	assert.Equal(t, cat.Icon, vm.Icon)
	assert.Equal(t, cat.ParentID, vm.ParentID)
	assert.Equal(t, cat.IsActive, vm.IsActive)
	assert.NotNil(t, vm.SubCategories)
	assert.Empty(t, vm.SubCategories)
	assert.True(t, vm.CanDelete)
}

func TestCategoryViewModel_FromDomain_NoParent(t *testing.T) {
	cat := &category.Category{
		ID:       uuid.New(),
		Name:     "Food",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "🍔",
		ParentID: nil,
		IsActive: true,
	}

	vm := &models.CategoryViewModel{}
	vm.FromDomain(cat)

	assert.Nil(t, vm.ParentID)
	assert.True(t, vm.CanDelete)
}

func TestBuildCategoryTree(t *testing.T) {
	// Create root categories
	rootID1 := uuid.New()
	rootID2 := uuid.New()

	// Create subcategories
	subID1 := uuid.New()
	subID2 := uuid.New()

	categories := []models.CategoryViewModel{
		{
			ID:       rootID1,
			Name:     "Food",
			Type:     category.TypeExpense,
			ParentID: nil,
		},
		{
			ID:       rootID2,
			Name:     "Income",
			Type:     category.TypeIncome,
			ParentID: nil,
		},
		{
			ID:       subID1,
			Name:     "Groceries",
			Type:     category.TypeExpense,
			ParentID: &rootID1,
		},
		{
			ID:       subID2,
			Name:     "Restaurants",
			Type:     category.TypeExpense,
			ParentID: &rootID1,
		},
	}

	tree := models.BuildCategoryTree(categories)

	// Should have 2 root categories
	assert.Len(t, tree, 2)

	// Find the Food category in the tree
	var foodCategory *models.CategoryViewModel
	for i := range tree {
		if tree[i].ID == rootID1 {
			foodCategory = &tree[i]
			break
		}
	}

	require.NotNil(t, foodCategory)
	assert.Equal(t, "Food", foodCategory.Name)
}

func TestBuildCategoryTree_EmptyList(t *testing.T) {
	categories := []models.CategoryViewModel{}
	tree := models.BuildCategoryTree(categories)

	assert.Empty(t, tree)
}

func TestBuildCategoryTree_OnlyRootCategories(t *testing.T) {
	categories := []models.CategoryViewModel{
		{
			ID:       uuid.New(),
			Name:     "Category 1",
			ParentID: nil,
		},
		{
			ID:       uuid.New(),
			Name:     "Category 2",
			ParentID: nil,
		},
		{
			ID:       uuid.New(),
			Name:     "Category 3",
			ParentID: nil,
		},
	}

	tree := models.BuildCategoryTree(categories)

	assert.Len(t, tree, 3)
	for _, cat := range tree {
		assert.Nil(t, cat.ParentID)
	}
}

func TestBuildCategoryTree_OrphanSubcategories(t *testing.T) {
	// Create subcategory with non-existent parent
	nonExistentParentID := uuid.New()

	categories := []models.CategoryViewModel{
		{
			ID:       uuid.New(),
			Name:     "Root Category",
			ParentID: nil,
		},
		{
			ID:       uuid.New(),
			Name:     "Orphan Subcategory",
			ParentID: &nonExistentParentID,
		},
	}

	tree := models.BuildCategoryTree(categories)

	// Should only have the root category
	assert.Len(t, tree, 1)
	assert.Equal(t, "Root Category", tree[0].Name)
}

func TestBuildCategoryTree_DeepHierarchy(t *testing.T) {
	// Create a 3-level hierarchy
	level1ID := uuid.New()
	level2ID := uuid.New()

	categories := []models.CategoryViewModel{
		{
			ID:       level1ID,
			Name:     "Level 1",
			ParentID: nil,
		},
		{
			ID:       level2ID,
			Name:     "Level 2",
			ParentID: &level1ID,
		},
		{
			ID:       uuid.New(),
			Name:     "Level 3",
			ParentID: &level2ID,
		},
	}

	tree := models.BuildCategoryTree(categories)

	// Should have 1 root category
	assert.Len(t, tree, 1)
	assert.Equal(t, "Level 1", tree[0].Name)
}

func TestCategoryViewModel_EdgeCases(t *testing.T) {
	t.Run("unicode in name", func(t *testing.T) {
		cat := &category.Category{
			ID:       uuid.New(),
			Name:     "Продукты 🛒",
			Type:     category.TypeExpense,
			Color:    "#FF5733",
			Icon:     "🛒",
			IsActive: true,
		}

		vm := &models.CategoryViewModel{}
		vm.FromDomain(cat)

		assert.Equal(t, "Продукты 🛒", vm.Name)
		assert.Equal(t, "🛒", vm.Icon)
	})

	t.Run("empty subcategories list", func(t *testing.T) {
		cat := &category.Category{
			ID:       uuid.New(),
			Name:     "Empty Category",
			Type:     category.TypeExpense,
			Color:    "#000000",
			Icon:     "📁",
			IsActive: true,
		}

		vm := &models.CategoryViewModel{}
		vm.FromDomain(cat)

		assert.NotNil(t, vm.SubCategories)
		assert.Empty(t, vm.SubCategories)
	})

	t.Run("inactive category", func(t *testing.T) {
		cat := &category.Category{
			ID:       uuid.New(),
			Name:     "Inactive",
			Type:     category.TypeExpense,
			Color:    "#999999",
			Icon:     "⛔",
			IsActive: false,
		}

		vm := &models.CategoryViewModel{}
		vm.FromDomain(cat)

		assert.False(t, vm.IsActive)
		assert.True(t, vm.CanDelete) // Default is true
	})
}

func TestCategoryFilter_EdgeCases(t *testing.T) {
	t.Run("filter with all fields set", func(t *testing.T) {
		isActive := true
		filter := models.CategoryFilter{
			Name:       "Test",
			Type:       "expense",
			IsActive:   &isActive,
			ParentOnly: true,
		}

		assert.Equal(t, "Test", filter.Name)
		assert.Equal(t, "expense", filter.Type)
		assert.NotNil(t, filter.IsActive)
		assert.True(t, *filter.IsActive)
		assert.True(t, filter.ParentOnly)
	})

	t.Run("filter with nil is_active", func(t *testing.T) {
		filter := models.CategoryFilter{
			IsActive: nil,
		}

		assert.Nil(t, filter.IsActive)
	})

	t.Run("filter with false is_active", func(t *testing.T) {
		isActive := false
		filter := models.CategoryFilter{
			IsActive: &isActive,
		}

		assert.NotNil(t, filter.IsActive)
		assert.False(t, *filter.IsActive)
	})
}

func TestCategorySelectOption(t *testing.T) {
	option := models.CategorySelectOption{
		ID:       uuid.New(),
		Name:     "Food",
		Type:     "expense",
		Color:    "#FF0000",
		Icon:     "🍔",
		IsParent: true,
		Level:    0,
	}

	assert.NotEqual(t, uuid.Nil, option.ID)
	assert.Equal(t, "Food", option.Name)
	assert.Equal(t, "expense", option.Type)
	assert.Equal(t, "#FF0000", option.Color)
	assert.Equal(t, "🍔", option.Icon)
	assert.True(t, option.IsParent)
	assert.Equal(t, 0, option.Level)
}

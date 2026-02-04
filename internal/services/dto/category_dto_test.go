package dto

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/category"
)

func TestCreateCategoryDTO_AllFields(t *testing.T) {
	parentID := uuid.New()

	dto := CreateCategoryDTO{
		Name:     "Food",
		Type:     category.TypeExpense,
		Color:    "#FF0000",
		Icon:     "food",
		ParentID: &parentID,
	}

	assert.Equal(t, "Food", dto.Name)
	assert.Equal(t, category.TypeExpense, dto.Type)
	assert.Equal(t, "#FF0000", dto.Color)
	assert.Equal(t, "food", dto.Icon)
	assert.NotNil(t, dto.ParentID)
	assert.Equal(t, parentID, *dto.ParentID)
}

func TestCreateCategoryDTO_WithoutParent(t *testing.T) {
	dto := CreateCategoryDTO{
		Name:  "Salary",
		Type:  category.TypeIncome,
		Color: "#00FF00",
		Icon:  "money",
	}

	assert.Equal(t, "Salary", dto.Name)
	assert.Equal(t, category.TypeIncome, dto.Type)
	assert.Equal(t, "#00FF00", dto.Color)
	assert.Equal(t, "money", dto.Icon)
	assert.Nil(t, dto.ParentID)
}

func TestUpdateCategoryDTO_AllFields(t *testing.T) {
	name := "Updated Category"
	color := "#0000FF"
	icon := "new-icon"

	dto := UpdateCategoryDTO{
		Name:  &name,
		Color: &color,
		Icon:  &icon,
	}

	assert.NotNil(t, dto.Name)
	assert.Equal(t, "Updated Category", *dto.Name)
	assert.NotNil(t, dto.Color)
	assert.Equal(t, "#0000FF", *dto.Color)
	assert.NotNil(t, dto.Icon)
	assert.Equal(t, "new-icon", *dto.Icon)
}

func TestUpdateCategoryDTO_PartialUpdate(t *testing.T) {
	name := "New Name"

	dto := UpdateCategoryDTO{
		Name: &name,
	}

	assert.NotNil(t, dto.Name)
	assert.Equal(t, "New Name", *dto.Name)
	assert.Nil(t, dto.Color)
	assert.Nil(t, dto.Icon)
}

func TestCategoryFilterDTO_AllFilters(t *testing.T) {
	catType := category.TypeExpense
	parentID := uuid.New()
	isActive := true

	filter := CategoryFilterDTO{
		Type:     &catType,
		ParentID: &parentID,
		IsActive: &isActive,
	}

	assert.NotNil(t, filter.Type)
	assert.Equal(t, category.TypeExpense, *filter.Type)
	assert.NotNil(t, filter.ParentID)
	assert.Equal(t, parentID, *filter.ParentID)
	assert.NotNil(t, filter.IsActive)
	assert.True(t, *filter.IsActive)
}

func TestCategoryFilterDTO_EmptyFilter(t *testing.T) {
	filter := CategoryFilterDTO{}

	assert.Nil(t, filter.Type)
	assert.Nil(t, filter.ParentID)
	assert.Nil(t, filter.IsActive)
}

func TestCategoryFilterDTO_TypeOnly(t *testing.T) {
	catType := category.TypeIncome

	filter := CategoryFilterDTO{
		Type: &catType,
	}

	assert.NotNil(t, filter.Type)
	assert.Equal(t, category.TypeIncome, *filter.Type)
	assert.Nil(t, filter.ParentID)
	assert.Nil(t, filter.IsActive)
}

func TestCategoryResponseDTO_ParentCategory(t *testing.T) {
	now := testTime()
	categoryID := uuid.New()

	response := CategoryResponseDTO{
		ID:        categoryID,
		Name:      "Food",
		Type:      "expense",
		Color:     "#FF0000",
		Icon:      "food",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, categoryID, response.ID)
	assert.Equal(t, "Food", response.Name)
	assert.Equal(t, "expense", response.Type)
	assert.Equal(t, "#FF0000", response.Color)
	assert.Equal(t, "food", response.Icon)
	assert.Nil(t, response.ParentID)
	assert.True(t, response.IsActive)
}

func TestCategoryResponseDTO_ChildCategory(t *testing.T) {
	now := testTime()
	categoryID := uuid.New()
	parentID := uuid.New()

	response := CategoryResponseDTO{
		ID:        categoryID,
		Name:      "Groceries",
		Type:      "expense",
		Color:     "#00FF00",
		Icon:      "cart",
		ParentID:  &parentID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, categoryID, response.ID)
	assert.Equal(t, "Groceries", response.Name)
	assert.NotNil(t, response.ParentID)
	assert.Equal(t, parentID, *response.ParentID)
	assert.True(t, response.IsActive)
}

func TestCategoryResponseDTO_InactiveCategory(t *testing.T) {
	now := testTime()
	categoryID := uuid.New()

	response := CategoryResponseDTO{
		ID:        categoryID,
		Name:      "Old Category",
		Type:      "expense",
		Color:     "#000000",
		Icon:      "archive",
		IsActive:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, categoryID, response.ID)
	assert.False(t, response.IsActive)
}

func TestCategoryHierarchyDTO_WithSubcategories(t *testing.T) {
	parentID := uuid.New()
	childID1 := uuid.New()
	childID2 := uuid.New()

	parent := CategoryHierarchyDTO{
		Category: CategoryResponseDTO{
			ID:       parentID,
			Name:     "Food",
			Type:     "expense",
			Color:    "#FF0000",
			Icon:     "food",
			IsActive: true,
		},
		Subcategories: []*CategoryHierarchyDTO{
			{
				Category: CategoryResponseDTO{
					ID:       childID1,
					Name:     "Groceries",
					Type:     "expense",
					ParentID: &parentID,
				},
			},
			{
				Category: CategoryResponseDTO{
					ID:       childID2,
					Name:     "Restaurants",
					Type:     "expense",
					ParentID: &parentID,
				},
			},
		},
	}

	assert.Equal(t, "Food", parent.Category.Name)
	assert.Len(t, parent.Subcategories, 2)
	assert.Equal(t, "Groceries", parent.Subcategories[0].Category.Name)
	assert.Equal(t, "Restaurants", parent.Subcategories[1].Category.Name)
}

func TestCategoryHierarchyDTO_WithoutSubcategories(t *testing.T) {
	categoryID := uuid.New()

	hierarchy := CategoryHierarchyDTO{
		Category: CategoryResponseDTO{
			ID:       categoryID,
			Name:     "Simple Category",
			Type:     "expense",
			IsActive: true,
		},
	}

	assert.Equal(t, "Simple Category", hierarchy.Category.Name)
	assert.Nil(t, hierarchy.Subcategories)
}

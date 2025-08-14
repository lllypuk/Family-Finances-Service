package category_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/category"
)

func TestNewCategory(t *testing.T) {
	// Test data
	name := "Groceries"
	categoryType := category.CategoryTypeExpense
	familyID := uuid.New()

	// Execute
	cat := category.NewCategory(name, categoryType, familyID)

	// Assert
	assert.NotEqual(t, uuid.Nil, cat.ID)
	assert.Equal(t, name, cat.Name)
	assert.Equal(t, categoryType, cat.Type)
	assert.Equal(t, familyID, cat.FamilyID)
	assert.Equal(t, "#007BFF", cat.Color)
	assert.Equal(t, "default", cat.Icon)
	assert.True(t, cat.IsActive)
	assert.Nil(t, cat.ParentID)
	assert.False(t, cat.CreatedAt.IsZero())
	assert.False(t, cat.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), cat.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), cat.UpdatedAt, time.Second)
}

func TestCategoryType_Constants(t *testing.T) {
	// Test that category type constants have expected values
	assert.Equal(t, "income", string(category.CategoryTypeIncome))
	assert.Equal(t, "expense", string(category.CategoryTypeExpense))
}

func TestCategory_IsSubcategory(t *testing.T) {
	familyID := uuid.New()
	parentID := uuid.New()

	tests := []struct {
		name           string
		parentID       *uuid.UUID
		expectedResult bool
	}{
		{
			name:           "Main category (no parent)",
			parentID:       nil,
			expectedResult: false,
		},
		{
			name:           "Subcategory (has parent)",
			parentID:       &parentID,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := &category.Category{
				ID:        uuid.New(),
				Name:      "Test Category",
				Type:      category.CategoryTypeExpense,
				FamilyID:  familyID,
				ParentID:  tt.parentID,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			result := cat.IsSubcategory()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestDefaultExpenseCategories(t *testing.T) {
	// Test that default expense categories are defined
	assert.NotEmpty(t, category.DefaultExpenseCategories)
	assert.Contains(t, category.DefaultExpenseCategories, "Продукты")
	assert.Contains(t, category.DefaultExpenseCategories, "Транспорт")
	assert.Contains(t, category.DefaultExpenseCategories, "Жилье и ЖКХ")
	assert.Contains(t, category.DefaultExpenseCategories, "Здоровье")
	assert.Contains(t, category.DefaultExpenseCategories, "Образование")
	assert.Contains(t, category.DefaultExpenseCategories, "Развлечения")
	assert.Contains(t, category.DefaultExpenseCategories, "Одежда")
	assert.Contains(t, category.DefaultExpenseCategories, "Ресторан и кафе")
	assert.Contains(t, category.DefaultExpenseCategories, "Спорт")
	assert.Contains(t, category.DefaultExpenseCategories, "Подарки")
	assert.Contains(t, category.DefaultExpenseCategories, "Разное")
}

func TestDefaultIncomeCategories(t *testing.T) {
	// Test that default income categories are defined
	assert.NotEmpty(t, category.DefaultIncomeCategories)
	assert.Contains(t, category.DefaultIncomeCategories, "Зарплата")
	assert.Contains(t, category.DefaultIncomeCategories, "Фриланс")
	assert.Contains(t, category.DefaultIncomeCategories, "Инвестиции")
	assert.Contains(t, category.DefaultIncomeCategories, "Подарки")
	assert.Contains(t, category.DefaultIncomeCategories, "Продажи")
	assert.Contains(t, category.DefaultIncomeCategories, "Разное")
}

func TestNewCategory_DifferentTypes(t *testing.T) {
	familyID := uuid.New()

	tests := []struct {
		name         string
		categoryType category.CategoryType
	}{
		{"Income Category", category.CategoryTypeIncome},
		{"Expense Category", category.CategoryTypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat := category.NewCategory("Test", tt.categoryType, familyID)
			assert.Equal(t, tt.categoryType, cat.Type)
		})
	}
}

func TestCategory_StructFields(t *testing.T) {
	// Test that Category struct has all required fields
	familyID := uuid.New()
	parentID := uuid.New()

	cat := &category.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Type:      category.CategoryTypeExpense,
		Color:     "#FF5733",
		Icon:      "food",
		ParentID:  &parentID,
		FamilyID:  familyID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, cat.ID)
	assert.NotEmpty(t, cat.Name)
	assert.NotEmpty(t, cat.Type)
	assert.NotEmpty(t, cat.Color)
	assert.NotEmpty(t, cat.Icon)
	assert.NotNil(t, cat.ParentID)
	assert.Equal(t, familyID, cat.FamilyID)
	assert.True(t, cat.IsActive)
	assert.False(t, cat.CreatedAt.IsZero())
	assert.False(t, cat.UpdatedAt.IsZero())
}

func TestCategory_TimestampGeneration(t *testing.T) {
	// Record time before creating category
	beforeTime := time.Now()

	// Create category
	cat := category.NewCategory("Test Category", category.CategoryTypeExpense, uuid.New())

	// Record time after creating category
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, cat.CreatedAt.After(beforeTime) || cat.CreatedAt.Equal(beforeTime))
	assert.True(t, cat.CreatedAt.Before(afterTime) || cat.CreatedAt.Equal(afterTime))
	assert.True(t, cat.UpdatedAt.After(beforeTime) || cat.UpdatedAt.Equal(beforeTime))
	assert.True(t, cat.UpdatedAt.Before(afterTime) || cat.UpdatedAt.Equal(afterTime))
}

func TestDefaultCategories_NonEmpty(t *testing.T) {
	// Test that we have default categories defined
	assert.NotEmpty(t, category.DefaultExpenseCategories, "Should have default expense categories")
	assert.NotEmpty(t, category.DefaultIncomeCategories, "Should have default income categories")
}

func TestDefaultCategories_UniqueValues(t *testing.T) {
	// Test that default expense categories don't have duplicates
	expenseSet := make(map[string]bool)
	for _, cat := range category.DefaultExpenseCategories {
		assert.False(t, expenseSet[cat], "Duplicate expense category found: %s", cat)
		expenseSet[cat] = true
	}

	// Test that default income categories don't have duplicates
	incomeSet := make(map[string]bool)
	for _, cat := range category.DefaultIncomeCategories {
		assert.False(t, incomeSet[cat], "Duplicate income category found: %s", cat)
		incomeSet[cat] = true
	}
}

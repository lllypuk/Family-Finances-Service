package category

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewCategory(t *testing.T) {
	// Test data
	name := "Food"
	categoryType := CategoryTypeExpense
	familyID := uuid.New()

	// Execute
	category := NewCategory(name, categoryType, familyID)

	// Assert
	assert.NotEqual(t, uuid.Nil, category.ID)
	assert.Equal(t, name, category.Name)
	assert.Equal(t, categoryType, category.Type)
	assert.Equal(t, familyID, category.FamilyID)
	assert.Equal(t, "#007BFF", category.Color)
	assert.Equal(t, "default", category.Icon)
	assert.True(t, category.IsActive)
	assert.Nil(t, category.ParentID)
	assert.False(t, category.CreatedAt.IsZero())
	assert.False(t, category.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), category.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), category.UpdatedAt, time.Second)
}

func TestCategoryType_Constants(t *testing.T) {
	// Test that category type constants have expected values
	assert.Equal(t, "income", string(CategoryTypeIncome))
	assert.Equal(t, "expense", string(CategoryTypeExpense))
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
			category := &Category{
				ID:        uuid.New(),
				Name:      "Test Category",
				Type:      CategoryTypeExpense,
				FamilyID:  familyID,
				ParentID:  tt.parentID,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			result := category.IsSubcategory()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestDefaultExpenseCategories(t *testing.T) {
	// Test that default expense categories are defined
	assert.NotEmpty(t, DefaultExpenseCategories)
	assert.Contains(t, DefaultExpenseCategories, "Продукты")
	assert.Contains(t, DefaultExpenseCategories, "Транспорт")
	assert.Contains(t, DefaultExpenseCategories, "Жилье и ЖКХ")
	assert.Contains(t, DefaultExpenseCategories, "Здоровье")
	assert.Contains(t, DefaultExpenseCategories, "Образование")
	assert.Contains(t, DefaultExpenseCategories, "Развлечения")
	assert.Contains(t, DefaultExpenseCategories, "Одежда")
	assert.Contains(t, DefaultExpenseCategories, "Ресторан и кафе")
	assert.Contains(t, DefaultExpenseCategories, "Спорт")
	assert.Contains(t, DefaultExpenseCategories, "Подарки")
	assert.Contains(t, DefaultExpenseCategories, "Разное")
}

func TestDefaultIncomeCategories(t *testing.T) {
	// Test that default income categories are defined
	assert.NotEmpty(t, DefaultIncomeCategories)
	assert.Contains(t, DefaultIncomeCategories, "Зарплата")
	assert.Contains(t, DefaultIncomeCategories, "Фриланс")
	assert.Contains(t, DefaultIncomeCategories, "Инвестиции")
	assert.Contains(t, DefaultIncomeCategories, "Подарки")
	assert.Contains(t, DefaultIncomeCategories, "Продажи")
	assert.Contains(t, DefaultIncomeCategories, "Разное")
}

func TestNewCategory_DifferentTypes(t *testing.T) {
	familyID := uuid.New()
	
	tests := []struct {
		name         string
		categoryType CategoryType
	}{
		{"Income Category", CategoryTypeIncome},
		{"Expense Category", CategoryTypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := NewCategory("Test Category", tt.categoryType, familyID)
			assert.Equal(t, tt.categoryType, category.Type)
		})
	}
}

func TestCategory_StructFields(t *testing.T) {
	// Test that Category struct has all required fields
	familyID := uuid.New()
	parentID := uuid.New()
	
	category := &Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Type:      CategoryTypeExpense,
		Color:     "#FF5733",
		Icon:      "food",
		ParentID:  &parentID,
		FamilyID:  familyID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, category.ID)
	assert.NotEmpty(t, category.Name)
	assert.NotEmpty(t, category.Type)
	assert.NotEmpty(t, category.Color)
	assert.NotEmpty(t, category.Icon)
	assert.NotNil(t, category.ParentID)
	assert.Equal(t, familyID, category.FamilyID)
	assert.True(t, category.IsActive)
	assert.False(t, category.CreatedAt.IsZero())
	assert.False(t, category.UpdatedAt.IsZero())
}

func TestCategory_TimestampGeneration(t *testing.T) {
	familyID := uuid.New()
	
	// Record time before creating category
	beforeTime := time.Now()
	
	// Create category
	category := NewCategory("Test Category", CategoryTypeExpense, familyID)
	
	// Record time after creating category
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, category.CreatedAt.After(beforeTime) || category.CreatedAt.Equal(beforeTime))
	assert.True(t, category.CreatedAt.Before(afterTime) || category.CreatedAt.Equal(afterTime))
	assert.True(t, category.UpdatedAt.After(beforeTime) || category.UpdatedAt.Equal(beforeTime))
	assert.True(t, category.UpdatedAt.Before(afterTime) || category.UpdatedAt.Equal(afterTime))
}

func TestDefaultCategories_NonEmpty(t *testing.T) {
	// Test that we have default categories defined
	assert.Greater(t, len(DefaultExpenseCategories), 0, "Should have default expense categories")
	assert.Greater(t, len(DefaultIncomeCategories), 0, "Should have default income categories")
}

func TestDefaultCategories_UniqueValues(t *testing.T) {
	// Test that default expense categories don't have duplicates
	expenseSet := make(map[string]bool)
	for _, cat := range DefaultExpenseCategories {
		assert.False(t, expenseSet[cat], "Duplicate expense category found: %s", cat)
		expenseSet[cat] = true
	}

	// Test that default income categories don't have duplicates
	incomeSet := make(map[string]bool)
	for _, cat := range DefaultIncomeCategories {
		assert.False(t, incomeSet[cat], "Duplicate income category found: %s", cat)
		incomeSet[cat] = true
	}
}
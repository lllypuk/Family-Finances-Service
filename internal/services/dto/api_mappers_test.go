package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/category"
)

func TestToCategoryAPIResponse(t *testing.T) {
	now := time.Now()
	categoryID := uuid.New()
	parentID := uuid.New()

	tests := []struct {
		name     string
		category *category.Category
		expected CategoryAPIResponse
	}{
		{
			name: "parent category",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Food",
				Type:      category.TypeExpense,
				Color:     "#FF0000",
				Icon:      "food",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Food",
				Type:      "expense",
				Color:     "#FF0000",
				Icon:      "food",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "child category with parent",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Groceries",
				Type:      category.TypeExpense,
				Color:     "#00FF00",
				Icon:      "cart",
				ParentID:  &parentID,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Groceries",
				Type:      "expense",
				Color:     "#00FF00",
				Icon:      "cart",
				ParentID:  &parentID,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "inactive category",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Old Category",
				Type:      category.TypeExpense,
				Color:     "#000000",
				Icon:      "archive",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Old Category",
				Type:      "expense",
				Color:     "#000000",
				Icon:      "archive",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCategoryAPIResponse(tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

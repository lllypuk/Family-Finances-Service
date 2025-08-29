package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
)

// CreateCategoryDTO represents the data transfer object for creating a category
type CreateCategoryDTO struct {
	Name     string        `validate:"required,min=2,max=50"`
	Type     category.Type `validate:"required"`
	Color    string        `validate:"required,hexcolor"`
	Icon     string        `validate:"required,min=1,max=50"`
	ParentID *uuid.UUID    `validate:"omitempty"`
	FamilyID uuid.UUID     `validate:"required"`
}

// UpdateCategoryDTO represents the data transfer object for updating a category
type UpdateCategoryDTO struct {
	Name  *string `validate:"omitempty,min=2,max=50"`
	Color *string `validate:"omitempty,hexcolor"`
	Icon  *string `validate:"omitempty,min=1,max=50"`
}

// CategoryFilterDTO represents the data transfer object for filtering categories
type CategoryFilterDTO struct {
	FamilyID uuid.UUID
	Type     *category.Type
	ParentID *uuid.UUID
	IsActive *bool
}

// CategoryResponseDTO represents the data transfer object for category responses
type CategoryResponseDTO struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Color     string     `json:"color"`
	Icon      string     `json:"icon"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	FamilyID  uuid.UUID  `json:"family_id"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CategoryHierarchyDTO represents a category with its subcategories
type CategoryHierarchyDTO struct {
	Category      CategoryResponseDTO     `json:"category"`
	Subcategories []*CategoryHierarchyDTO `json:"subcategories,omitempty"`
}

package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
)

type CategoryAPIResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Color     string     `json:"color"`
	Icon      string     `json:"icon"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ToCategoryAPIResponse converts domain Category to API CategoryResponse
func ToCategoryAPIResponse(c *category.Category) CategoryAPIResponse {
	return CategoryAPIResponse{
		ID:        c.ID,
		Name:      c.Name,
		Type:      string(c.Type),
		Color:     c.Color,
		Icon:      c.Icon,
		ParentID:  c.ParentID,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/user"
)

// CreateUserAPIRequest represents API request for user creation
type CreateUserAPIRequest struct {
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	FamilyID  uuid.UUID `json:"family_id"`
	Role      string    `json:"role"`
}

type UpdateUserAPIRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
}

type UserAPIResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	FamilyID  uuid.UUID `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromCreateUserAPIRequest converts API CreateUserRequest to CreateUserDTO
func FromCreateUserAPIRequest(req CreateUserAPIRequest) CreateUserDTO {
	return CreateUserDTO{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      user.Role(req.Role),
		FamilyID:  req.FamilyID,
	}
}

// FromUpdateUserAPIRequest converts API UpdateUserRequest to UpdateUserDTO
func FromUpdateUserAPIRequest(req UpdateUserAPIRequest) UpdateUserDTO {
	return UpdateUserDTO(req)
}

// ToUserAPIResponse converts domain User to API UserResponse
func ToUserAPIResponse(u *user.User) UserAPIResponse {
	return UserAPIResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		FamilyID:  u.FamilyID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// CreateCategoryAPIRequest represents API request for category creation
type CreateCategoryAPIRequest struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Color    string     `json:"color"`
	Icon     string     `json:"icon"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	FamilyID uuid.UUID  `json:"family_id"`
}

type UpdateCategoryAPIRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}

type CategoryAPIResponse struct {
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

// FromCreateCategoryAPIRequest converts API CreateCategoryRequest to CreateCategoryDTO
func FromCreateCategoryAPIRequest(req CreateCategoryAPIRequest) CreateCategoryDTO {
	return CreateCategoryDTO{
		Name:     req.Name,
		Type:     category.Type(req.Type),
		Color:    req.Color,
		Icon:     req.Icon,
		ParentID: req.ParentID,
		FamilyID: req.FamilyID,
	}
}

// FromUpdateCategoryAPIRequest converts API UpdateCategoryRequest to UpdateCategoryDTO
func FromUpdateCategoryAPIRequest(req UpdateCategoryAPIRequest) UpdateCategoryDTO {
	return UpdateCategoryDTO(req)
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
		FamilyID:  c.FamilyID,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

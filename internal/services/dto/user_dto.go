package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

// CreateUserDTO represents the data transfer object for creating a user
type CreateUserDTO struct {
	Email     string    `validate:"required,email,max=254"`
	FirstName string    `validate:"required,min=2,max=50"`
	LastName  string    `validate:"required,min=2,max=50"`
	Password  string    `validate:"required,min=6"`
	Role      user.Role `validate:"required"`
	FamilyID  uuid.UUID `validate:"required"`
}

// UpdateUserDTO represents the data transfer object for updating a user
type UpdateUserDTO struct {
	FirstName *string `validate:"omitempty,min=2,max=50"`
	LastName  *string `validate:"omitempty,min=2,max=50"`
	Email     *string `validate:"omitempty,email,max=254"`
}

// UserFilterDTO represents the data transfer object for filtering users
type UserFilterDTO struct {
	FamilyID uuid.UUID
	Role     *user.Role
	Email    *string
}

// UserResponseDTO represents the data transfer object for user responses
type UserResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      user.Role `json:"role"`
	FamilyID  uuid.UUID `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateFamilyDTO represents the data transfer object for creating a family
type CreateFamilyDTO struct {
	Name     string `validate:"required,min=2,max=100"`
	Currency string `validate:"required,len=3"`
}

// UpdateFamilyDTO represents the data transfer object for updating a family
type UpdateFamilyDTO struct {
	Name     *string `validate:"omitempty,min=2,max=100"`
	Currency *string `validate:"omitempty,len=3"`
}

// FamilyResponseDTO represents the data transfer object for family responses
type FamilyResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

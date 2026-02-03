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
}

// UpdateUserDTO represents the data transfer object for updating a user
type UpdateUserDTO struct {
	FirstName *string `validate:"omitempty,min=2,max=50"`
	LastName  *string `validate:"omitempty,min=2,max=50"`
	Email     *string `validate:"omitempty,email,max=254"`
}

// UserFilterDTO represents the data transfer object for filtering users
type UserFilterDTO struct {
	Role  *user.Role
	Email *string
}

// UserResponseDTO represents the data transfer object for user responses
type UserResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      user.Role `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SetupFamilyDTO represents the data for initial family setup (bootstrap)
type SetupFamilyDTO struct {
	// Family data
	FamilyName string `validate:"required,min=2,max=100"`
	// Currency is a 3-character ISO currency code, for example "USD"
	Currency string `validate:"required,len=3"`
	// First user (admin) data
	Email     string `validate:"required,email,max=254"`
	FirstName string `validate:"required,min=2,max=50"`
	LastName  string `validate:"required,min=2,max=50"`
	// Password must be at least 6 characters long
	Password string `validate:"required,min=6"`
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

// CreateInviteDTO represents the data for creating a new invite
type CreateInviteDTO struct {
	Email string `validate:"required,email,max=254"`
	Role  string `validate:"required,oneof=admin member child"`
}

// AcceptInviteDTO represents the data for accepting an invite
type AcceptInviteDTO struct {
	Email    string `validate:"required,email,max=254"`
	Name     string `validate:"required,min=2,max=100"`
	Password string `validate:"required,min=6"`
}

// InviteResponseDTO represents the data transfer object for invite responses
type InviteResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
}

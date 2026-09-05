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
	Password  string    `validate:"required,password"`
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

// SetupFamilyDTO represents the data for initial family setup (bootstrap)
type SetupFamilyDTO struct {
	// Family data
	FamilyName string `validate:"required,min=2,max=100"`
	// Currency is a 3-character ISO currency code, for example "USD"
	Currency string `validate:"required,len=3"`
	// Timezone — IANA-имя; проверяется здесь, в Family попадает с планом 04 (A-06).
	Timezone string `validate:"required,timezone"`
	// First user (admin) data
	Email     string `validate:"required,email,max=254"`
	FirstName string `validate:"required,min=2,max=50"`
	LastName  string `validate:"required,min=2,max=50"`
	// Password policy: auth.ValidatePassword (10…72 bytes)
	Password string `validate:"required,password"`
}

// UpdateFamilyDTO represents the data transfer object for updating a family
type UpdateFamilyDTO struct {
	Name     *string `validate:"omitempty,min=2,max=100"`
	Currency *string `validate:"omitempty,len=3"`
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
	Password string `validate:"required,password"`
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

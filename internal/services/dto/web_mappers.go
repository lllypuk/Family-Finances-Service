package dto

import (
	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

// CreateUserWebRequest represents web form request for user creation
type CreateUserWebRequest struct {
	Email     string `form:"email"`
	Password  string `form:"password"`
	FirstName string `form:"first_name"`
	LastName  string `form:"last_name"`
	Role      string `form:"role"`
}

// FromCreateUserWebRequest converts web form CreateUserRequest to CreateUserDTO
func FromCreateUserWebRequest(req CreateUserWebRequest, familyID uuid.UUID) CreateUserDTO {
	return CreateUserDTO{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      user.Role(req.Role),
		FamilyID:  familyID,
	}
}

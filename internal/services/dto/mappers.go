package dto

import (
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/models"

	"github.com/google/uuid"
)

// ToUserResponseDTO converts a domain User to UserResponseDTO
func ToUserResponseDTO(u *user.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		FamilyID:  uuid.Nil, // Single family model - FamilyID no longer stored in User
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// ToFamilyResponseDTO converts a domain Family to FamilyResponseDTO
func ToFamilyResponseDTO(f *user.Family) FamilyResponseDTO {
	return FamilyResponseDTO{
		ID:        f.ID,
		Name:      f.Name,
		Currency:  f.Currency,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

// FromCreateUserForm converts web CreateUserForm to CreateUserDTO
func FromCreateUserForm(form *models.CreateUserForm) CreateUserDTO {
	return CreateUserDTO{
		Email:     form.Email,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Password:  form.Password,
		Role:      user.Role(form.Role),
	}
}

// FromSetupForm converts web SetupForm to SetupFamilyDTO
func FromSetupForm(form *models.SetupForm) SetupFamilyDTO {
	return SetupFamilyDTO{
		FamilyName: form.FamilyName,
		Currency:   form.Currency,
		Email:      form.Email,
		FirstName:  form.FirstName,
		LastName:   form.LastName,
		Password:   form.Password,
	}
}

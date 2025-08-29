package dto

import (
	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/models"
)

// ToUserResponseDTO converts a domain User to UserResponseDTO
func ToUserResponseDTO(u *user.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
		FamilyID:  u.FamilyID,
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
func FromCreateUserForm(form *models.CreateUserForm, familyID uuid.UUID) CreateUserDTO {
	return CreateUserDTO{
		Email:     form.Email,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Password:  form.Password,
		Role:      user.Role(form.Role),
		FamilyID:  familyID,
	}
}

// FromRegisterForm converts web RegisterForm to CreateFamilyDTO and CreateUserDTO
func FromRegisterForm(form *models.RegisterForm) (CreateFamilyDTO, CreateUserDTO) {
	familyDTO := CreateFamilyDTO{
		Name:     form.FamilyName,
		Currency: form.Currency,
	}

	userDTO := CreateUserDTO{
		Email:     form.Email,
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Password:  form.Password,
		Role:      user.RoleAdmin, // Registration creates admin user
		// FamilyID will be set after family creation
	}

	return familyDTO, userDTO
}

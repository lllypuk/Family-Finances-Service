package dto

import (
	"family-budget-service/internal/domain/user"
)

// ToUserResponseDTO converts a domain User to UserResponseDTO
func ToUserResponseDTO(u *user.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      u.Role,
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

package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
)

func TestCreateUserDTO_AllFields(t *testing.T) {
	dto := CreateUserDTO{
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Password:  "password123",
		Role:      user.RoleAdmin,
	}

	assert.Equal(t, "test@example.com", dto.Email)
	assert.Equal(t, "John", dto.FirstName)
	assert.Equal(t, "Doe", dto.LastName)
	assert.Equal(t, "password123", dto.Password)
	assert.Equal(t, user.RoleAdmin, dto.Role)
}

func TestUpdateUserDTO_AllFields(t *testing.T) {
	firstName := "NewFirst"
	lastName := "NewLast"
	email := "new@example.com"

	dto := UpdateUserDTO{
		FirstName: &firstName,
		LastName:  &lastName,
		Email:     &email,
	}

	assert.NotNil(t, dto.FirstName)
	assert.Equal(t, "NewFirst", *dto.FirstName)
	assert.NotNil(t, dto.LastName)
	assert.Equal(t, "NewLast", *dto.LastName)
	assert.NotNil(t, dto.Email)
	assert.Equal(t, "new@example.com", *dto.Email)
}

func TestUpdateUserDTO_PartialUpdate(t *testing.T) {
	firstName := "UpdatedFirst"

	dto := UpdateUserDTO{
		FirstName: &firstName,
	}

	assert.NotNil(t, dto.FirstName)
	assert.Equal(t, "UpdatedFirst", *dto.FirstName)
	assert.Nil(t, dto.LastName)
	assert.Nil(t, dto.Email)
}

func TestUserFilterDTO_AllFilters(t *testing.T) {
	role := user.RoleMember
	email := "test@example.com"

	filter := UserFilterDTO{
		Role:  &role,
		Email: &email,
	}

	assert.NotNil(t, filter.Role)
	assert.Equal(t, user.RoleMember, *filter.Role)
	assert.NotNil(t, filter.Email)
	assert.Equal(t, "test@example.com", *filter.Email)
}

func TestSetupFamilyDTO_AllFields(t *testing.T) {
	dto := SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@example.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "password123",
	}

	assert.Equal(t, "Test Family", dto.FamilyName)
	assert.Equal(t, "USD", dto.Currency)
	assert.Equal(t, "admin@example.com", dto.Email)
	assert.Equal(t, "Admin", dto.FirstName)
	assert.Equal(t, "User", dto.LastName)
	assert.Equal(t, "password123", dto.Password)
}

func TestUpdateFamilyDTO_AllFields(t *testing.T) {
	name := "Updated Family"
	currency := "EUR"

	dto := UpdateFamilyDTO{
		Name:     &name,
		Currency: &currency,
	}

	assert.NotNil(t, dto.Name)
	assert.Equal(t, "Updated Family", *dto.Name)
	assert.NotNil(t, dto.Currency)
	assert.Equal(t, "EUR", *dto.Currency)
}

func TestUpdateFamilyDTO_PartialUpdate(t *testing.T) {
	name := "New Name"

	dto := UpdateFamilyDTO{
		Name: &name,
	}

	assert.NotNil(t, dto.Name)
	assert.Equal(t, "New Name", *dto.Name)
	assert.Nil(t, dto.Currency)
}

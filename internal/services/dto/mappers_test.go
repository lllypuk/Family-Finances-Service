package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/models"
)

func TestToUserResponseDTO(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	tests := []struct {
		name     string
		user     *user.User
		expected UserResponseDTO
	}{
		{
			name: "admin user",
			user: &user.User{
				ID:        userID,
				Email:     "admin@example.com",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: UserResponseDTO{
				ID:        userID,
				Email:     "admin@example.com",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "member user",
			user: &user.User{
				ID:        userID,
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Role:      user.RoleMember,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: UserResponseDTO{
				ID:        userID,
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Role:      user.RoleMember,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "child user",
			user: &user.User{
				ID:        userID,
				Email:     "child@example.com",
				FirstName: "Child",
				LastName:  "User",
				Role:      user.RoleChild,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: UserResponseDTO{
				ID:        userID,
				Email:     "child@example.com",
				FirstName: "Child",
				LastName:  "User",
				Role:      user.RoleChild,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUserResponseDTO(tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToFamilyResponseDTO(t *testing.T) {
	now := time.Now()
	familyID := uuid.New()

	tests := []struct {
		name     string
		family   *user.Family
		expected FamilyResponseDTO
	}{
		{
			name: "USD family",
			family: &user.Family{
				ID:        familyID,
				Name:      "Test Family",
				Currency:  "USD",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: FamilyResponseDTO{
				ID:        familyID,
				Name:      "Test Family",
				Currency:  "USD",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "EUR family",
			family: &user.Family{
				ID:        familyID,
				Name:      "European Family",
				Currency:  "EUR",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: FamilyResponseDTO{
				ID:        familyID,
				Name:      "European Family",
				Currency:  "EUR",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToFamilyResponseDTO(tt.family)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromCreateUserForm(t *testing.T) {
	tests := []struct {
		name     string
		form     *models.CreateUserForm
		expected CreateUserDTO
	}{
		{
			name: "complete form",
			form: &models.CreateUserForm{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
				Role:      "admin",
			},
			expected: CreateUserDTO{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
				Role:      user.RoleAdmin,
			},
		},
		{
			name: "member role",
			form: &models.CreateUserForm{
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Password:  "password123",
				Role:      "member",
			},
			expected: CreateUserDTO{
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Password:  "password123",
				Role:      user.RoleMember,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromCreateUserForm(tt.form)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromSetupForm(t *testing.T) {
	tests := []struct {
		name     string
		form     *models.SetupForm
		expected SetupFamilyDTO
	}{
		{
			name: "complete setup form",
			form: &models.SetupForm{
				FamilyName: "Test Family",
				Currency:   "USD",
				Email:      "admin@example.com",
				FirstName:  "Admin",
				LastName:   "User",
				Password:   "password123",
			},
			expected: SetupFamilyDTO{
				FamilyName: "Test Family",
				Currency:   "USD",
				Email:      "admin@example.com",
				FirstName:  "Admin",
				LastName:   "User",
				Password:   "password123",
			},
		},
		{
			name: "EUR currency",
			form: &models.SetupForm{
				FamilyName: "European Family",
				Currency:   "EUR",
				Email:      "admin@example.com",
				FirstName:  "Admin",
				LastName:   "User",
				Password:   "password123",
			},
			expected: SetupFamilyDTO{
				FamilyName: "European Family",
				Currency:   "EUR",
				Email:      "admin@example.com",
				FirstName:  "Admin",
				LastName:   "User",
				Password:   "password123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromSetupForm(tt.form)
			assert.Equal(t, tt.expected, result)
		})
	}
}

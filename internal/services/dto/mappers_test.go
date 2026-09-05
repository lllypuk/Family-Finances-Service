package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
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

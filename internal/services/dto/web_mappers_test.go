package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
)

func TestFromCreateUserWebRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  CreateUserWebRequest
		expected CreateUserDTO
	}{
		{
			name: "admin user",
			request: CreateUserWebRequest{
				Email:     "admin@example.com",
				Password:  "password123",
				FirstName: "Admin",
				LastName:  "User",
				Role:      "admin",
			},
			expected: CreateUserDTO{
				Email:     "admin@example.com",
				Password:  "password123",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
			},
		},
		{
			name: "member user",
			request: CreateUserWebRequest{
				Email:     "member@example.com",
				Password:  "password123",
				FirstName: "Member",
				LastName:  "User",
				Role:      "member",
			},
			expected: CreateUserDTO{
				Email:     "member@example.com",
				Password:  "password123",
				FirstName: "Member",
				LastName:  "User",
				Role:      user.RoleMember,
			},
		},
		{
			name: "child user",
			request: CreateUserWebRequest{
				Email:     "child@example.com",
				Password:  "password123",
				FirstName: "Child",
				LastName:  "User",
				Role:      "child",
			},
			expected: CreateUserDTO{
				Email:     "child@example.com",
				Password:  "password123",
				FirstName: "Child",
				LastName:  "User",
				Role:      user.RoleChild,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromCreateUserWebRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

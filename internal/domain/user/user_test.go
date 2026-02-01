package user_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
)

func TestNewUser(t *testing.T) {
	// Test data
	email := "test@example.com"
	firstName := "John"
	lastName := "Doe"
	role := user.RoleMember

	// Execute
	u := user.NewUser(email, firstName, lastName, role)

	// Assert
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.Equal(t, email, u.Email)
	assert.Equal(t, firstName, u.FirstName)
	assert.Equal(t, lastName, u.LastName)
	assert.Equal(t, role, u.Role)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), u.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), u.UpdatedAt, time.Second)
}

func TestNewFamily(t *testing.T) {
	// Test data
	name := "Test Family"
	currency := "USD"

	// Execute
	family := user.NewFamily(name, currency)

	// Assert
	assert.NotEqual(t, uuid.Nil, family.ID)
	assert.Equal(t, name, family.Name)
	assert.Equal(t, currency, family.Currency)
	assert.False(t, family.CreatedAt.IsZero())
	assert.False(t, family.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), family.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), family.UpdatedAt, time.Second)
}

func TestRole_Constants(t *testing.T) {
	// Test that role constants have expected values
	assert.Equal(t, "admin", string(user.RoleAdmin))
	assert.Equal(t, "member", string(user.RoleMember))
	assert.Equal(t, "child", string(user.RoleChild))
}

func TestUser_StructFields(t *testing.T) {
	// Test that User struct has all required fields
	u := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, u.ID)
	assert.NotEmpty(t, u.Email)
	assert.NotEmpty(t, u.Password)
	assert.NotEmpty(t, u.FirstName)
	assert.NotEmpty(t, u.LastName)
	assert.NotEmpty(t, u.Role)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
}

func TestFamily_StructFields(t *testing.T) {
	// Test that Family struct has all required fields
	testFamily := &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, testFamily.ID)
	assert.NotEmpty(t, testFamily.Name)
	assert.NotEmpty(t, testFamily.Currency)
	assert.False(t, testFamily.CreatedAt.IsZero())
	assert.False(t, testFamily.UpdatedAt.IsZero())
}

func TestNewUser_DifferentRoles(t *testing.T) {
	tests := []struct {
		name string
		role user.Role
	}{
		{"Admin Role", user.RoleAdmin},
		{"Member Role", user.RoleMember},
		{"Child Role", user.RoleChild},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := user.NewUser("test@example.com", "John", "Doe", tt.role)
			assert.Equal(t, tt.role, u.Role)
		})
	}
}

func TestNewFamily_DifferentCurrencies(t *testing.T) {
	currencies := []string{"USD", "EUR", "RUB", "GBP", "JPY"}

	for _, currency := range currencies {
		t.Run("Currency_"+currency, func(t *testing.T) {
			family := user.NewFamily("Test Family", currency)
			assert.Equal(t, currency, family.Currency)
		})
	}
}

func TestUser_TimestampGeneration(t *testing.T) {
	// Record time before creating user
	beforeTime := time.Now()

	// Create user
	u := user.NewUser("test@example.com", "John", "Doe", user.RoleMember)

	// Record time after creating user
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, u.CreatedAt.After(beforeTime) || u.CreatedAt.Equal(beforeTime))
	assert.True(t, u.CreatedAt.Before(afterTime) || u.CreatedAt.Equal(afterTime))
	assert.True(t, u.UpdatedAt.After(beforeTime) || u.UpdatedAt.Equal(beforeTime))
	assert.True(t, u.UpdatedAt.Before(afterTime) || u.UpdatedAt.Equal(afterTime))
}

func TestFamily_TimestampGeneration(t *testing.T) {
	// Record time before creating family
	beforeTime := time.Now()

	// Create family
	family := user.NewFamily("Test Family", "USD")

	// Record time after creating family
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, family.CreatedAt.After(beforeTime) || family.CreatedAt.Equal(beforeTime))
	assert.True(t, family.CreatedAt.Before(afterTime) || family.CreatedAt.Equal(afterTime))
	assert.True(t, family.UpdatedAt.After(beforeTime) || family.UpdatedAt.Equal(beforeTime))
	assert.True(t, family.UpdatedAt.Before(afterTime) || family.UpdatedAt.Equal(afterTime))
}

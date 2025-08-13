package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	// Test data
	email := "test@example.com"
	firstName := "John"
	lastName := "Doe"
	familyID := uuid.New()
	role := RoleMember

	// Execute
	user := NewUser(email, firstName, lastName, familyID, role)

	// Assert
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, firstName, user.FirstName)
	assert.Equal(t, lastName, user.LastName)
	assert.Equal(t, familyID, user.FamilyID)
	assert.Equal(t, role, user.Role)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), user.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), user.UpdatedAt, time.Second)
}

func TestNewFamily(t *testing.T) {
	// Test data
	name := "Test Family"
	currency := "USD"

	// Execute
	family := NewFamily(name, currency)

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
	assert.Equal(t, "admin", string(RoleAdmin))
	assert.Equal(t, "member", string(RoleMember))
	assert.Equal(t, "child", string(RoleChild))
}

func TestUser_StructFields(t *testing.T) {
	// Test that User struct has all required fields
	familyID := uuid.New()
	user := &User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Password:  "hashed_password",
		FirstName: "John",
		LastName:  "Doe",
		Role:      RoleAdmin,
		FamilyID:  familyID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotEmpty(t, user.Email)
	assert.NotEmpty(t, user.Password)
	assert.NotEmpty(t, user.FirstName)
	assert.NotEmpty(t, user.LastName)
	assert.NotEmpty(t, user.Role)
	assert.Equal(t, familyID, user.FamilyID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestFamily_StructFields(t *testing.T) {
	// Test that Family struct has all required fields
	family := &Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "EUR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, family.ID)
	assert.NotEmpty(t, family.Name)
	assert.NotEmpty(t, family.Currency)
	assert.False(t, family.CreatedAt.IsZero())
	assert.False(t, family.UpdatedAt.IsZero())
}

func TestNewUser_DifferentRoles(t *testing.T) {
	familyID := uuid.New()

	tests := []struct {
		name string
		role Role
	}{
		{"Admin Role", RoleAdmin},
		{"Member Role", RoleMember},
		{"Child Role", RoleChild},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := NewUser("test@example.com", "John", "Doe", familyID, tt.role)
			assert.Equal(t, tt.role, user.Role)
		})
	}
}

func TestNewFamily_DifferentCurrencies(t *testing.T) {
	currencies := []string{"USD", "EUR", "RUB", "GBP", "JPY"}

	for _, currency := range currencies {
		t.Run("Currency_"+currency, func(t *testing.T) {
			family := NewFamily("Test Family", currency)
			assert.Equal(t, currency, family.Currency)
		})
	}
}

func TestUser_TimestampGeneration(t *testing.T) {
	// Record time before creating user
	beforeTime := time.Now()

	// Create user
	user := NewUser("test@example.com", "John", "Doe", uuid.New(), RoleMember)

	// Record time after creating user
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, user.CreatedAt.After(beforeTime) || user.CreatedAt.Equal(beforeTime))
	assert.True(t, user.CreatedAt.Before(afterTime) || user.CreatedAt.Equal(afterTime))
	assert.True(t, user.UpdatedAt.After(beforeTime) || user.UpdatedAt.Equal(beforeTime))
	assert.True(t, user.UpdatedAt.Before(afterTime) || user.UpdatedAt.Equal(afterTime))
}

func TestFamily_TimestampGeneration(t *testing.T) {
	// Record time before creating family
	beforeTime := time.Now()

	// Create family
	family := NewFamily("Test Family", "USD")

	// Record time after creating family
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, family.CreatedAt.After(beforeTime) || family.CreatedAt.Equal(beforeTime))
	assert.True(t, family.CreatedAt.Before(afterTime) || family.CreatedAt.Equal(afterTime))
	assert.True(t, family.UpdatedAt.After(beforeTime) || family.UpdatedAt.Equal(beforeTime))
	assert.True(t, family.UpdatedAt.Before(afterTime) || family.UpdatedAt.Equal(afterTime))
}

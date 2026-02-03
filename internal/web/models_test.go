package web_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web"
)

func TestSessionData_StructFields(t *testing.T) {
	// Test SessionData struct
	userID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)

	session := &web.SessionData{
		UserID:    userID,
		Role:      user.RoleAdmin,
		Email:     "test@example.com",
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, user.RoleAdmin, session.Role)
	assert.Equal(t, "test@example.com", session.Email)
	assert.Equal(t, expiresAt, session.ExpiresAt)
}

func TestDashboardData_StructFields(t *testing.T) {
	// Create test user and family
	testUser := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Role:      user.RoleAdmin,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	testFamily := &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test DashboardData struct
	dashboard := &web.DashboardData{
		User:             testUser,
		Family:           testFamily,
		TotalIncome:      5000.50,
		TotalExpenses:    3500.25,
		NetIncome:        1500.25,
		TransactionCount: 25,
		BudgetCount:      5,
	}

	assert.Equal(t, testUser, dashboard.User)
	assert.Equal(t, testFamily, dashboard.Family)
	assert.InDelta(t, 5000.50, dashboard.TotalIncome, 0.01)
	assert.InDelta(t, 3500.25, dashboard.TotalExpenses, 0.01)
	assert.InDelta(t, 1500.25, dashboard.NetIncome, 0.01)
	assert.Equal(t, 25, dashboard.TransactionCount)
	assert.Equal(t, 5, dashboard.BudgetCount)
}

func TestFormErrors_IsEmpty(t *testing.T) {
	// Test empty errors
	errors := make(web.FormErrors)
	assert.True(t, errors.IsEmpty())

	// Test non-empty errors
	errors.Add("field1", "Error message")
	assert.False(t, errors.IsEmpty())
}

func TestFormErrors_Add(t *testing.T) {
	errors := make(web.FormErrors)

	// Test adding errors
	errors.Add("email", "Invalid email format")
	errors.Add("password", "Password too short")

	assert.Equal(t, "Invalid email format", errors["email"])
	assert.Equal(t, "Password too short", errors["password"])
	assert.Len(t, errors, 2)
}

func TestFormErrors_Get(t *testing.T) {
	errors := make(web.FormErrors)
	errors.Add("username", "Username is required")

	// Test getting existing error
	assert.Equal(t, "Username is required", errors.Get("username"))

	// Test getting non-existent error
	assert.Empty(t, errors.Get("nonexistent"))
}

func TestFormErrors_Has(t *testing.T) {
	errors := make(web.FormErrors)
	errors.Add("name", "Name is required")

	// Test existing field
	assert.True(t, errors.Has("name"))

	// Test non-existent field
	assert.False(t, errors.Has("nonexistent"))
}

func TestPageData_StructFields(t *testing.T) {
	testUser := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
		Role:      user.RoleMember,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	testFamily := &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "EUR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	errors := make(web.FormErrors)
	errors.Add("field1", "Error 1")

	messages := []web.Message{
		{Type: "success", Text: "Success message", Timeout: 5},
		{Type: "error", Text: "Error message", Timeout: 0},
	}

	pageData := &web.PageData{
		Title:       "Test Page",
		CurrentUser: testUser,
		Family:      testFamily,
		Errors:      errors,
		Messages:    messages,
		CSRFToken:   "csrf-token-123",
	}

	assert.Equal(t, "Test Page", pageData.Title)
	assert.Equal(t, testUser, pageData.CurrentUser)
	assert.Equal(t, testFamily, pageData.Family)
	assert.Equal(t, errors, pageData.Errors)
	assert.Equal(t, messages, pageData.Messages)
	assert.Equal(t, "csrf-token-123", pageData.CSRFToken)
}

func TestMessage_StructFields(t *testing.T) {
	message := web.Message{
		Type:    "warning",
		Text:    "This is a warning message",
		Timeout: 10,
	}

	assert.Equal(t, "warning", message.Type)
	assert.Equal(t, "This is a warning message", message.Text)
	assert.Equal(t, 10, message.Timeout)
}

func TestLoginFormData_StructFields(t *testing.T) {
	loginForm := &web.LoginFormData{
		Email:    "user@example.com",
		Password: "securepassword123",
	}

	assert.Equal(t, "user@example.com", loginForm.Email)
	assert.Equal(t, "securepassword123", loginForm.Password)
}

func TestRegisterFormData_StructFields(t *testing.T) {
	registerForm := &web.RegisterFormData{
		FamilyName: "Smith Family",
		Currency:   "USD",
		FirstName:  "John",
		LastName:   "Smith",
		Email:      "john.smith@example.com",
		Password:   "verysecurepassword",
	}

	assert.Equal(t, "Smith Family", registerForm.FamilyName)
	assert.Equal(t, "USD", registerForm.Currency)
	assert.Equal(t, "John", registerForm.FirstName)
	assert.Equal(t, "Smith", registerForm.LastName)
	assert.Equal(t, "john.smith@example.com", registerForm.Email)
	assert.Equal(t, "verysecurepassword", registerForm.Password)
}

func TestFormErrors_NilSafety(t *testing.T) {
	// Test nil safety of methods
	var errors web.FormErrors

	// These should not panic even with nil map
	assert.True(t, errors.IsEmpty())
	assert.Empty(t, errors.Get("nonexistent"))
	assert.False(t, errors.Has("nonexistent"))

	// Note: Add method will panic with nil receiver for map type
	// This is expected Go behavior - maps must be initialized before use
	// We don't test Add with nil receiver as it would cause panic
}

func TestFormErrors_EdgeCases(t *testing.T) {
	errors := make(web.FormErrors)

	// Test empty field name
	errors.Add("", "Empty field error")
	assert.Equal(t, "Empty field error", errors.Get(""))
	assert.True(t, errors.Has(""))

	// Test empty error message
	errors.Add("field", "")
	assert.Empty(t, errors.Get("field"))
	assert.True(t, errors.Has("field"))

	// Test overwriting existing error
	errors.Add("field", "First error")
	errors.Add("field", "Second error")
	assert.Equal(t, "Second error", errors.Get("field"))
}

func TestMessage_Types(t *testing.T) {
	// Test common message types
	types := []string{"success", "error", "warning", "info"}

	for _, msgType := range types {
		message := web.Message{
			Type:    msgType,
			Text:    "Test message for " + msgType,
			Timeout: 5,
		}

		assert.Equal(t, msgType, message.Type)
		assert.Contains(t, message.Text, msgType)
		assert.Equal(t, 5, message.Timeout)
	}
}

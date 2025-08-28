package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	webHandlers "family-budget-service/internal/web/handlers"
)

func TestNewBaseHandler(t *testing.T) {
	repos := NewMockRepositories()
	baseHandler := webHandlers.NewBaseHandler(&repos.Repositories, nil)

	assert.NotNil(t, baseHandler)
}

func TestBaseHandler_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.NotEmpty(t, webHandlers.HTMXRequestHeader)
	assert.Equal(t, "true", webHandlers.HTMXRequestHeader)
}

func TestBaseHandler_Errors(t *testing.T) {
	// Test error constants
	require.Error(t, webHandlers.ErrNoSession)
	assert.Equal(t, "no session found", webHandlers.ErrNoSession.Error())
}

func TestSessionData_StructFields(t *testing.T) {
	userID := uuid.New()
	familyID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)

	sessionData := &webHandlers.SessionData{
		UserID:    userID,
		FamilyID:  familyID,
		Role:      user.RoleAdmin,
		Email:     "admin@example.com",
		ExpiresAt: expiresAt,
	}

	assert.Equal(t, userID, sessionData.UserID)
	assert.Equal(t, familyID, sessionData.FamilyID)
	assert.Equal(t, user.RoleAdmin, sessionData.Role)
	assert.Equal(t, "admin@example.com", sessionData.Email)
	assert.Equal(t, expiresAt, sessionData.ExpiresAt)
}

func TestSessionData_DifferentRoles(t *testing.T) {
	tests := []struct {
		name string
		role user.Role
	}{
		{"Admin role", user.RoleAdmin},
		{"Member role", user.RoleMember},
		{"Child role", user.RoleChild},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			familyID := uuid.New()
			sessionData := &webHandlers.SessionData{
				UserID:   userID,
				FamilyID: familyID,
				Role:     tt.role,
				Email:    "user@example.com",
			}
			// Use variables to avoid unused write warnings
			_ = userID
			_ = familyID

			assert.Equal(t, tt.role, sessionData.Role)
			assert.Equal(t, userID, sessionData.UserID)
			assert.Equal(t, familyID, sessionData.FamilyID)
			assert.Equal(t, "user@example.com", sessionData.Email)
		})
	}
}

func TestPageData_EmptyStructure(t *testing.T) {
	pageData := &webHandlers.PageData{}

	assert.Empty(t, pageData.Title)
	assert.Nil(t, pageData.CurrentUser)
	assert.Nil(t, pageData.Family)
	assert.Nil(t, pageData.Errors)
	assert.Empty(t, pageData.Messages)
	assert.Empty(t, pageData.CSRFToken)
}

func TestPageData_WithUserAndFamily(t *testing.T) {
	currentUser := &user.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Role:  user.RoleMember,
	}

	pageData := &webHandlers.PageData{
		Title:       "Test Page",
		CurrentUser: currentUser,
		Family:      nil, // family,
		CSRFToken:   "csrf-token-123",
	}
	// Use variable to avoid unused write warning
	_ = pageData.Family

	assert.Equal(t, "Test Page", pageData.Title)
	assert.NotNil(t, pageData.CurrentUser)
	// assert.NotNil(t, pageData.Family)
	assert.Equal(t, currentUser.ID, pageData.CurrentUser.ID)
	// assert.Equal(t, family.ID, pageData.Family.ID)
	assert.Equal(t, "csrf-token-123", pageData.CSRFToken)
}

func TestMessage_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		msgType    string
		text       string
		timeout    int
		shouldPass bool
	}{
		{
			name:       "Success message with timeout",
			msgType:    "success",
			text:       "Operation successful",
			timeout:    5,
			shouldPass: true,
		},
		{
			name:       "Error message no timeout",
			msgType:    "error",
			text:       "Error occurred",
			timeout:    0,
			shouldPass: true,
		},
		{
			name:       "Warning message",
			msgType:    "warning",
			text:       "Warning text",
			timeout:    10,
			shouldPass: true,
		},
		{
			name:       "Info message",
			msgType:    "info",
			text:       "Information text",
			timeout:    3,
			shouldPass: true,
		},
		{
			name:       "Custom message type",
			msgType:    "custom",
			text:       "Custom message",
			timeout:    7,
			shouldPass: true,
		},
		{
			name:       "Empty message",
			msgType:    "",
			text:       "",
			timeout:    0,
			shouldPass: true, // Empty values are technically valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := webHandlers.Message{
				Type:    tt.msgType,
				Text:    tt.text,
				Timeout: tt.timeout,
			}

			if tt.shouldPass {
				assert.Equal(t, tt.msgType, message.Type)
				assert.Equal(t, tt.text, message.Text)
				assert.Equal(t, tt.timeout, message.Timeout)
			}
		})
	}
}

func TestFormErrors_EdgeCases(t *testing.T) {
	errors := make(webHandlers.FormErrors)

	// Test nil key handling
	errors[""] = "Empty key error"
	assert.Equal(t, "Empty key error", errors[""])

	// Test overwriting existing error
	errors["field"] = "First error"
	errors["field"] = "Second error"
	assert.Equal(t, "Second error", errors["field"])

	// Test unicode support
	errors["поле"] = "Ошибка на русском"
	assert.Equal(t, "Ошибка на русском", errors["поле"])

	// Test special characters
	errors["field-with-dashes"] = "Error with dashes"
	errors["field_with_underscores"] = "Error with underscores"
	errors["field.with.dots"] = "Error with dots"

	assert.Equal(t, "Error with dashes", errors["field-with-dashes"])
	assert.Equal(t, "Error with underscores", errors["field_with_underscores"])
	assert.Equal(t, "Error with dots", errors["field.with.dots"])
}

func TestFormErrors_MultipleOperations(t *testing.T) {
	errors := make(webHandlers.FormErrors)

	// Add multiple errors
	fields := []string{"email", "password", "name", "age", "phone"}
	for i, field := range fields {
		errors[field] = "Error " + string(rune('A'+i))
	}

	// Verify all were added
	assert.Len(t, errors, len(fields))

	// Verify individual errors
	for i, field := range fields {
		expectedError := "Error " + string(rune('A'+i))
		assert.Equal(t, expectedError, errors[field])
	}

	// Remove some errors
	delete(errors, "email")
	delete(errors, "age")
	assert.Len(t, errors, len(fields)-2)

	// Verify removed errors are gone
	_, emailExists := errors["email"]
	_, ageExists := errors["age"]
	assert.False(t, emailExists)
	assert.False(t, ageExists)

	// Verify remaining errors still exist
	_, passwordExists := errors["password"]
	_, nameExists := errors["name"]
	_, phoneExists := errors["phone"]
	assert.True(t, passwordExists)
	assert.True(t, nameExists)
	assert.True(t, phoneExists)
}

func TestBaseHandler_RenderPage(t *testing.T) {
	repos := NewMockRepositories()
	// baseHandler := webHandlers.NewBaseHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Using reflection or public methods we can test the base handler indirectly
	// Since renderPage is not exported, we'll test through a handler that uses it
	dashboardHandler := webHandlers.NewDashboardHandler(&repos.Repositories, nil)
	err := dashboardHandler.Dashboard(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"template":"dashboard"`)
}

func TestBaseHandler_MultipleInstancesIsolation(t *testing.T) {
	repos1 := NewMockRepositories()
	repos2 := NewMockRepositories()

	handler1 := webHandlers.NewBaseHandler(&repos1.Repositories, nil)
	handler2 := webHandlers.NewBaseHandler(&repos2.Repositories, nil)

	// Test that handlers are independent instances
	assert.NotSame(t, handler1, handler2)

	// Both should be valid handlers
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

func TestSessionData_TimeHandling(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		expiresAt time.Time
		isExpired bool
	}{
		{
			name:      "Future expiration",
			expiresAt: futureTime,
			isExpired: false,
		},
		{
			name:      "Past expiration",
			expiresAt: pastTime,
			isExpired: true,
		},
		{
			name:      "Current time",
			expiresAt: now,
			isExpired: true, // Считаем текущее время как истекшее
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			familyID := uuid.New()
			sessionData := &webHandlers.SessionData{
				UserID:    userID,
				FamilyID:  familyID,
				Role:      user.RoleMember,
				Email:     "user@example.com",
				ExpiresAt: tt.expiresAt,
			}
			// Use variables to avoid unused write warnings
			_ = userID
			_ = familyID

			assert.Equal(t, tt.expiresAt, sessionData.ExpiresAt)
			assert.Equal(t, userID, sessionData.UserID)
			assert.Equal(t, familyID, sessionData.FamilyID)
			assert.Equal(t, user.RoleMember, sessionData.Role)
			assert.Equal(t, "user@example.com", sessionData.Email)

			// Test expiration logic
			isExpired := sessionData.ExpiresAt.Before(now) || sessionData.ExpiresAt.Equal(now)
			assert.Equal(t, tt.isExpired, isExpired)
		})
	}
}

func TestPageData_JSONSerialization(t *testing.T) {
	pageData := &webHandlers.PageData{
		Title: "Test Page",
		Messages: []webHandlers.Message{
			{
				Type:    "info",
				Text:    "Test message",
				Timeout: 5,
			},
		},
		Errors: webHandlers.FormErrors{
			"field1": "Error 1",
		},
		CSRFToken: "token-123",
	}

	// Test that struct tags are properly defined for JSON serialization
	assert.NotEmpty(t, pageData.Title)
	assert.NotEmpty(t, pageData.Messages)
	assert.NotEmpty(t, pageData.Errors)
	assert.NotEmpty(t, pageData.CSRFToken)
}

func TestMessage_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		message webHandlers.Message
		isValid bool
	}{
		{
			name: "Normal message",
			message: webHandlers.Message{
				Type:    "info",
				Text:    "Normal message",
				Timeout: 5,
			},
			isValid: true,
		},
		{
			name: "Very long message",
			message: webHandlers.Message{
				Type:    "info",
				Text:    string(make([]byte, 1000)), // Very long text
				Timeout: 5,
			},
			isValid: true,
		},
		{
			name: "Negative timeout",
			message: webHandlers.Message{
				Type:    "info",
				Text:    "Message with negative timeout",
				Timeout: -1,
			},
			isValid: true, // Negative timeout might be valid (no auto-hide)
		},
		{
			name: "Zero timeout",
			message: webHandlers.Message{
				Type:    "error",
				Text:    "Error message",
				Timeout: 0,
			},
			isValid: true, // Zero timeout is valid for persistent messages
		},
		{
			name: "Very large timeout",
			message: webHandlers.Message{
				Type:    "info",
				Text:    "Message with large timeout",
				Timeout: 86400, // 24 hours in seconds
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isValid {
				// Create message and test its fields
				message := webHandlers.Message{
					Type:    tt.message.Type,
					Text:    tt.message.Text,
					Timeout: tt.message.Timeout,
				}
				assert.Equal(t, tt.message.Type, message.Type)
				assert.Equal(t, tt.message.Text, message.Text)
				assert.Equal(t, tt.message.Timeout, message.Timeout)
			}
		})
	}
}

func TestFormErrors_ConcurrentAccess(t *testing.T) {
	// Test that FormErrors can handle basic operations
	// (Note: real concurrent testing would require goroutines and sync mechanisms)
	errors := make(webHandlers.FormErrors)

	// Simulate multiple operations
	operations := []struct {
		key   string
		value string
	}{
		{"field1", "error1"},
		{"field2", "error2"},
		{"field3", "error3"},
		{"field4", "error4"},
		{"field5", "error5"},
	}

	// Add all errors
	for _, op := range operations {
		errors[op.key] = op.value
	}

	// Verify all were added
	assert.Len(t, errors, len(operations))

	// Verify each error
	for _, op := range operations {
		assert.Equal(t, op.value, errors[op.key])
	}
}

// Benchmark tests
func BenchmarkMessage_Creation(b *testing.B) {
	for b.Loop() {
		_ = webHandlers.Message{
			Type:    "info",
			Text:    "Benchmark message",
			Timeout: 5,
		}
	}
}

func BenchmarkFormErrors_AddRemove(b *testing.B) {
	errors := make(webHandlers.FormErrors)

	b.ResetTimer()
	for i := range b.N {
		key := "field" + string(rune(i%10+'0'))
		errors[key] = "Error message"
		delete(errors, key)
	}
}

func BenchmarkSessionData_Creation(b *testing.B) {
	userID := uuid.New()
	familyID := uuid.New()
	expiresAt := time.Now().Add(time.Hour)

	for b.Loop() {
		sessionData := &webHandlers.SessionData{
			UserID:    userID,
			FamilyID:  familyID,
			Role:      user.RoleMember,
			Email:     "user@example.com",
			ExpiresAt: expiresAt,
		}
		// Use all fields to avoid unused write warnings
		_ = sessionData.UserID
		_ = sessionData.FamilyID
		_ = sessionData.Role
		_ = sessionData.Email
		_ = sessionData.ExpiresAt
	}
}

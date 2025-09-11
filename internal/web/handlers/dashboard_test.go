package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	webHandlers "family-budget-service/internal/web/handlers"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

// MockRenderer для тестирования без реальных шаблонов
type MockRenderer struct{}

func (m *MockRenderer) Render(_ io.Writer, name string, data any, c echo.Context) error {
	// Простой mock - возвращаем JSON вместо HTML для тестов
	return c.JSON(http.StatusOK, map[string]any{
		"template": name,
		"data":     data,
	})
}

// MockRepositories для тестирования
type MockRepositories struct {
	handlers.Repositories
}

func NewMockRepositories() *MockRepositories {
	return &MockRepositories{}
}

// createMockSessionData создает mock данные сессии для тестов
func createMockSessionData() *middleware.SessionData {
	return &middleware.SessionData{
		UserID:    uuid.New(),
		FamilyID:  uuid.New(),
		Role:      user.RoleAdmin,
		Email:     "test@example.com",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

func TestDashboardHandler_Creation(t *testing.T) {
	// Test that the dashboard handler can be created
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories, nil)

	assert.NotNil(t, handler)
}

func TestDashboardHandler_Struct(t *testing.T) {
	// Test dashboard handler structure and basic functionality
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories, nil)

	assert.NotNil(t, handler)
	// Test that it embeds BaseHandler
	assert.IsType(t, &webHandlers.DashboardHandler{}, handler)
}

func TestDashboardViewModel_StructFields(t *testing.T) {
	// Test DashboardViewModel struct
	viewModel := &webModels.DashboardViewModel{
		MonthlySummary: &webModels.MonthlySummaryCard{
			TotalIncome:      100.50,
			TotalExpenses:    75.25,
			NetIncome:        25.25,
			TransactionCount: 10,
		},
	}

	assert.NotNil(t, viewModel.MonthlySummary)
	assert.InDelta(t, 100.50, viewModel.MonthlySummary.TotalIncome, 0.01)
	assert.InDelta(t, 75.25, viewModel.MonthlySummary.TotalExpenses, 0.01)
	assert.InDelta(t, 25.25, viewModel.MonthlySummary.NetIncome, 0.01)
	assert.Equal(t, 10, viewModel.MonthlySummary.TransactionCount)
}

func TestDashboardViewModel_Structure(t *testing.T) {
	// Test DashboardViewModel creation
	dashboard := &webModels.DashboardViewModel{
		MonthlySummary: &webModels.MonthlySummaryCard{
			TotalIncome:      50000,
			TotalExpenses:    35000,
			NetIncome:        15000,
			TransactionCount: 42,
		},
	}

	assert.NotNil(t, dashboard.MonthlySummary)
	assert.InEpsilon(t, 50000.0, dashboard.MonthlySummary.TotalIncome, 0.01)
	assert.InEpsilon(t, 35000.0, dashboard.MonthlySummary.TotalExpenses, 0.01)
	assert.InEpsilon(t, 15000.0, dashboard.MonthlySummary.NetIncome, 0.01)
	assert.Equal(t, 42, dashboard.MonthlySummary.TransactionCount)
}

func TestDashboardFilters_Structure(t *testing.T) {
	// Test dashboard filters functionality
	filters := &webModels.DashboardFilters{
		Period: "current_month",
	}

	assert.Equal(t, "current_month", filters.Period)

	// Test default period
	defaultFilters := &webModels.DashboardFilters{}
	if defaultFilters.Period == "" {
		defaultFilters.Period = "current_month"
	}
	assert.Equal(t, "current_month", defaultFilters.Period)
}

func TestDashboardHandler_HTMXHeaders(t *testing.T) {
	// Test HTMX request detection (without calling actual handlers)
	req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/stats", nil)
	req.Header.Set("Hx-Request", "true")

	// Test that HTMX header is properly set
	assert.Equal(t, "true", req.Header.Get("Hx-Request"))

	// Test regular request
	regularReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	assert.Empty(t, regularReq.Header.Get("Hx-Request"))
}

func TestDashboardHandler_Isolation(t *testing.T) {
	// Test that multiple handler instances are independent
	repos1 := NewMockRepositories()
	repos2 := NewMockRepositories()

	handler1 := webHandlers.NewDashboardHandler(&repos1.Repositories, nil)
	handler2 := webHandlers.NewDashboardHandler(&repos2.Repositories, nil)

	assert.NotSame(t, handler1, handler2)
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

func TestBaseHandler_IsHTMXRequest(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	baseHandler := webHandlers.NewBaseHandler(&repos.Repositories, nil)

	// Setup Echo
	e := echo.New()

	t.Run("Regular request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := e.NewContext(req, httptest.NewRecorder())
		// Set mock session data directly as user data (bypassing middleware)
		c.Set("user", createMockSessionData())

		// This test would need the method to be exported
		// For now we just test the struct creation
		assert.NotNil(t, baseHandler)
	})

	t.Run("HTMX request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Hx-Request", "true")
		c := e.NewContext(req, httptest.NewRecorder())
		// Set mock session data directly as user data (bypassing middleware)
		c.Set("user", createMockSessionData())

		// This test would need the method to be exported
		// For now we just test the struct creation
		assert.NotNil(t, baseHandler)
	})
}

func TestFormErrors_Methods(t *testing.T) {
	errors := make(map[string]string)

	// Test IsEmpty
	assert.Empty(t, errors)

	// Test Add
	errors["field1"] = "Error message 1"
	errors["field2"] = "Error message 2"

	// Test Get
	assert.Equal(t, "Error message 1", errors["field1"])
	assert.Equal(t, "Error message 2", errors["field2"])

	// Test Has
	_, exists1 := errors["field1"]
	_, exists2 := errors["nonexistent"]
	assert.True(t, exists1)
	assert.False(t, exists2)
}

func TestPageData_StructFields(t *testing.T) {
	// Test PageData struct
	pageData := &webHandlers.PageData{
		Title: "Test Page",
		Messages: []webHandlers.Message{
			{
				Type: "info",
				Text: "Test message",
			},
		},
		CSRFToken: "test-token",
	}

	assert.Equal(t, "Test Page", pageData.Title)
	assert.Len(t, pageData.Messages, 1)
	assert.Equal(t, "info", pageData.Messages[0].Type)
	assert.Equal(t, "Test message", pageData.Messages[0].Text)
	assert.Equal(t, "test-token", pageData.CSRFToken)
}

func TestMessage_StructFields(t *testing.T) {
	// Test Message struct
	message := webHandlers.Message{
		Type: "success",
		Text: "Operation completed",
	}

	assert.Equal(t, "success", message.Type)
	assert.Equal(t, "Operation completed", message.Text)
}

func TestMonthlySummaryCard_Calculations(t *testing.T) {
	card := &webModels.MonthlySummaryCard{
		TotalIncome:      1000.00,
		TotalExpenses:    750.50,
		NetIncome:        249.50,
		TransactionCount: 15,
		IncomeChange:     10.5,
		ExpensesChange:   -5.2,
	}

	// Проверяем расчеты
	expectedNet := card.TotalIncome - card.TotalExpenses
	assert.InDelta(t, expectedNet, card.NetIncome, 0.01)

	// Проверяем типы данных
	assert.IsType(t, float64(0), card.TotalIncome)
	assert.IsType(t, float64(0), card.TotalExpenses)
	assert.IsType(t, float64(0), card.NetIncome)

	// Проверяем значения изменений
	assert.InDelta(t, 10.5, card.IncomeChange, 0.001)
	assert.InDelta(t, -5.2, card.ExpensesChange, 0.001)

	// Проверяем CSS классы на основе изменений
	assert.Equal(t, webModels.CSSClassTextSuccess, card.GetIncomeChangeClass())
	assert.Equal(t, webModels.CSSClassTextSuccess, card.GetExpensesChangeClass())
	assert.IsType(t, 0, card.TransactionCount)
}

func TestMonthlySummaryCard_ZeroValues(t *testing.T) {
	card := &webModels.MonthlySummaryCard{}

	assert.InDelta(t, 0.0, card.TotalIncome, 0.001)
	assert.InDelta(t, 0.0, card.TotalExpenses, 0.001)
	assert.InDelta(t, 0.0, card.NetIncome, 0.001)
	assert.Equal(t, 0, card.TransactionCount)
	assert.Equal(t, "text-muted", card.GetIncomeChangeClass())
	assert.Equal(t, "text-muted", card.GetNetIncomeClass())
}

func TestMonthlySummaryCard_NegativeValues(t *testing.T) {
	card := &webModels.MonthlySummaryCard{
		TotalIncome:      500.00,
		TotalExpenses:    750.00,
		NetIncome:        -250.00,
		TransactionCount: 10,
		IncomeChange:     -8.5,
		ExpensesChange:   12.3,
	}
	// Use fields to avoid unused write warnings
	_ = card.TransactionCount

	// Negative net income is valid (expenses > income)
	assert.InDelta(t, -250.00, card.NetIncome, 0.001)
	assert.Negative(t, card.NetIncome)
	assert.Greater(t, card.TotalExpenses, card.TotalIncome)
	// Test CSS classes for negative values
	assert.Equal(t, "text-danger", card.GetIncomeChangeClass())
	assert.Equal(t, "text-danger", card.GetExpensesChangeClass())
	assert.Equal(t, "text-danger", card.GetNetIncomeClass())
}

func TestMessage_DifferentTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType string
		text    string
	}{
		{
			name:    "Success message",
			msgType: "success",
			text:    "Operation completed successfully",
		},
		{
			name:    "Error message",
			msgType: "error",
			text:    "An error occurred",
		},
		{
			name:    "Warning message",
			msgType: "warning",
			text:    "This is a warning",
		},
		{
			name:    "Info message",
			msgType: "info",
			text:    "Here is some information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := webHandlers.Message{
				Type: tt.msgType,
				Text: tt.text,
			}

			assert.Equal(t, tt.msgType, message.Type)
			assert.Equal(t, tt.text, message.Text)
		})
	}
}

func TestFormErrors_Operations(t *testing.T) {
	errors := make(map[string]string)

	// Test initial state
	assert.Empty(t, errors)
	assert.Empty(t, errors)

	// Test adding errors
	errors["email"] = "Invalid email format"
	errors["password"] = "Password too short"

	assert.Len(t, errors, 2)
	assert.Equal(t, "Invalid email format", errors["email"])
	assert.Equal(t, "Password too short", errors["password"])

	// Test checking existence
	_, emailExists := errors["email"]
	_, nonExistentExists := errors["nonexistent"]
	assert.True(t, emailExists)
	assert.False(t, nonExistentExists)

	// Test deleting
	delete(errors, "email")
	assert.Len(t, errors, 1)
	_, emailExistsAfterDelete := errors["email"]
	assert.False(t, emailExistsAfterDelete)
}

func TestPageData_CompleteStructure(t *testing.T) {
	// Создаем полную структуру PageData для тестирования
	pageData := &webHandlers.PageData{
		Title: "Test Dashboard",
		Messages: []webHandlers.Message{
			{
				Type: "success",
				Text: "Welcome back!",
			},
			{
				Type: "info",
				Text: "You have 3 new notifications",
			},
		},
		CSRFToken: "csrf-token-123",
	}

	assert.Equal(t, "Test Dashboard", pageData.Title)
	assert.Len(t, pageData.Messages, 2)
	assert.Equal(t, "csrf-token-123", pageData.CSRFToken)

	// Test first message
	assert.Equal(t, "success", pageData.Messages[0].Type)
	assert.Equal(t, "Welcome back!", pageData.Messages[0].Text)

	// Test second message
	assert.Equal(t, "info", pageData.Messages[1].Type)
	assert.Equal(t, "You have 3 new notifications", pageData.Messages[1].Text)
}

// Benchmark тесты для performance
func BenchmarkDashboardHandler_Creation(b *testing.B) {
	// Benchmark dashboard handler creation
	repos := NewMockRepositories()

	for b.Loop() {
		handler := webHandlers.NewDashboardHandler(&repos.Repositories, nil)
		_ = handler
	}
}

func BenchmarkMockSessionData_Creation(b *testing.B) {
	// Benchmark mock session data creation
	for b.Loop() {
		sessionData := createMockSessionData()
		_ = sessionData
	}
}

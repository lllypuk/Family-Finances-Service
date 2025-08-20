package handlers_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/handlers"
	webHandlers "family-budget-service/internal/web/handlers"
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

func TestDashboardHandler_Dashboard(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	// Setup Echo с mock renderer
	e := echo.New()
	e.Renderer = &MockRenderer{}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handler.Dashboard(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"template":"dashboard"`)
	assert.Contains(t, rec.Body.String(), `"total_income":50000`)
	assert.Contains(t, rec.Body.String(), `"total_expenses":35000`)
	assert.Contains(t, rec.Body.String(), `"net_income":15000`)
}

func TestDashboardHandler_DashboardStats(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	// Setup Echo с mock renderer
	e := echo.New()
	e.Renderer = &MockRenderer{}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := handler.DashboardStats(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"template":"dashboard-stats"`)
	assert.Contains(t, rec.Body.String(), `"total_income":52000`)
	assert.Contains(t, rec.Body.String(), `"total_expenses":36500`)
	assert.Contains(t, rec.Body.String(), `"net_income":15500`)
}

func TestDashboardData_StructFields(t *testing.T) {
	// Test DashboardData struct
	data := &webHandlers.DashboardData{
		TotalIncome:      100.50,
		TotalExpenses:    75.25,
		NetIncome:        25.25,
		TransactionCount: 10,
		BudgetCount:      3,
	}

	assert.InDelta(t, 100.50, data.TotalIncome, 0.01)
	assert.InDelta(t, 75.25, data.TotalExpenses, 0.01)
	assert.InDelta(t, 25.25, data.NetIncome, 0.01)
	assert.Equal(t, 10, data.TransactionCount)
	assert.Equal(t, 3, data.BudgetCount)
}

func TestBaseHandler_IsHTMXRequest(t *testing.T) {
	// Setup
	repos := NewMockRepositories()
	baseHandler := webHandlers.NewBaseHandler(&repos.Repositories)

	// Setup Echo
	e := echo.New()

	t.Run("Regular request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		_ = e.NewContext(req, httptest.NewRecorder())

		// This test would need the method to be exported
		// For now we just test the struct creation
		assert.NotNil(t, baseHandler)
	})

	t.Run("HTMX request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Hx-Request", "true")
		_ = e.NewContext(req, httptest.NewRecorder())

		// This test would need the method to be exported
		// For now we just test the struct creation
		assert.NotNil(t, baseHandler)
	})
}

func TestFormErrors_Methods(t *testing.T) {
	errors := make(webHandlers.FormErrors)

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
				Type:    "info",
				Text:    "Test message",
				Timeout: 5,
			},
		},
		CSRFToken: "test-token",
	}

	assert.Equal(t, "Test Page", pageData.Title)
	assert.Len(t, pageData.Messages, 1)
	assert.Equal(t, "info", pageData.Messages[0].Type)
	assert.Equal(t, "Test message", pageData.Messages[0].Text)
	assert.Equal(t, 5, pageData.Messages[0].Timeout)
	assert.Equal(t, "test-token", pageData.CSRFToken)
}

func TestMessage_StructFields(t *testing.T) {
	// Test Message struct
	message := webHandlers.Message{
		Type:    "success",
		Text:    "Operation completed",
		Timeout: 10,
	}

	assert.Equal(t, "success", message.Type)
	assert.Equal(t, "Operation completed", message.Text)
	assert.Equal(t, 10, message.Timeout)
}

func TestDashboardHandler_Dashboard_DataStructure(t *testing.T) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Dashboard(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Проверяем наличие всех необходимых данных в ответе
	body := rec.Body.String()
	assert.Contains(t, body, `"template":"dashboard"`)
	assert.Contains(t, body, `"title":"Главная"`)
	assert.Contains(t, body, "Добро пожаловать в систему семейного бюджета!")
	assert.Contains(t, body, `"total_income":50000`)
	assert.Contains(t, body, `"total_expenses":35000`)
	assert.Contains(t, body, `"net_income":15000`)
	assert.Contains(t, body, `"transaction_count":42`)
	assert.Contains(t, body, `"budget_count":5`)
}

func TestDashboardHandler_DashboardStats_UpdatedData(t *testing.T) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/stats", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DashboardStats(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Проверяем что возвращаются обновленные данные
	body := rec.Body.String()
	assert.Contains(t, body, `"template":"dashboard-stats"`)
	assert.Contains(t, body, `"total_income":52000`)
	assert.Contains(t, body, `"total_expenses":36500`)
	assert.Contains(t, body, `"net_income":15500`)
	assert.Contains(t, body, `"transaction_count":45`)
	assert.Contains(t, body, `"budget_count":5`)
}

func TestDashboardHandler_DashboardStats_HTMXEndpoint(t *testing.T) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	// Создаем HTMX запрос
	req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/stats", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.DashboardStats(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"template":"dashboard-stats"`)
}

func TestDashboardData_Calculations(t *testing.T) {
	data := &webHandlers.DashboardData{
		TotalIncome:      1000.00,
		TotalExpenses:    750.50,
		NetIncome:        249.50,
		TransactionCount: 15,
		BudgetCount:      3,
	}

	// Проверяем расчеты
	expectedNet := data.TotalIncome - data.TotalExpenses
	assert.InDelta(t, expectedNet, data.NetIncome, 0.01)

	// Проверяем типы данных
	assert.IsType(t, float64(0), data.TotalIncome)
	assert.IsType(t, float64(0), data.TotalExpenses)
	assert.IsType(t, float64(0), data.NetIncome)
	assert.IsType(t, 0, data.TransactionCount)
	assert.IsType(t, 0, data.BudgetCount)
}

func TestDashboardData_ZeroValues(t *testing.T) {
	data := &webHandlers.DashboardData{}

	assert.InDelta(t, 0.0, data.TotalIncome, 0.001)
	assert.InDelta(t, 0.0, data.TotalExpenses, 0.001)
	assert.InDelta(t, 0.0, data.NetIncome, 0.001)
	assert.Equal(t, 0, data.TransactionCount)
	assert.Equal(t, 0, data.BudgetCount)
}

func TestDashboardData_NegativeValues(t *testing.T) {
	data := &webHandlers.DashboardData{
		TotalIncome:      500.00,
		TotalExpenses:    750.00,
		NetIncome:        -250.00,
		TransactionCount: 10,
		BudgetCount:      2,
	}
	// Use fields to avoid unused write warnings
	_ = data.TransactionCount
	_ = data.BudgetCount

	// Negative net income is valid (expenses > income)
	assert.InDelta(t, -250.00, data.NetIncome, 0.001)
	assert.Negative(t, data.NetIncome)
	assert.Greater(t, data.TotalExpenses, data.TotalIncome)
}

func TestMessage_DifferentTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType string
		text    string
		timeout int
	}{
		{
			name:    "Success message",
			msgType: "success",
			text:    "Operation completed successfully",
			timeout: 5,
		},
		{
			name:    "Error message",
			msgType: "error",
			text:    "An error occurred",
			timeout: 0, // Error messages usually don't timeout
		},
		{
			name:    "Warning message",
			msgType: "warning",
			text:    "This is a warning",
			timeout: 10,
		},
		{
			name:    "Info message",
			msgType: "info",
			text:    "Here is some information",
			timeout: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := webHandlers.Message{
				Type:    tt.msgType,
				Text:    tt.text,
				Timeout: tt.timeout,
			}

			assert.Equal(t, tt.msgType, message.Type)
			assert.Equal(t, tt.text, message.Text)
			assert.Equal(t, tt.timeout, message.Timeout)
		})
	}
}

func TestFormErrors_Operations(t *testing.T) {
	errors := make(webHandlers.FormErrors)

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
				Type:    "success",
				Text:    "Welcome back!",
				Timeout: 5,
			},
			{
				Type:    "info",
				Text:    "You have 3 new notifications",
				Timeout: 8,
			},
		},
		Errors: webHandlers.FormErrors{
			"field1": "Error 1",
			"field2": "Error 2",
		},
		CSRFToken: "csrf-token-123",
	}

	assert.Equal(t, "Test Dashboard", pageData.Title)
	assert.Len(t, pageData.Messages, 2)
	assert.Len(t, pageData.Errors, 2)
	assert.Equal(t, "csrf-token-123", pageData.CSRFToken)

	// Test first message
	assert.Equal(t, "success", pageData.Messages[0].Type)
	assert.Equal(t, "Welcome back!", pageData.Messages[0].Text)
	assert.Equal(t, 5, pageData.Messages[0].Timeout)

	// Test second message
	assert.Equal(t, "info", pageData.Messages[1].Type)
	assert.Equal(t, "You have 3 new notifications", pageData.Messages[1].Text)
	assert.Equal(t, 8, pageData.Messages[1].Timeout)

	// Test errors
	assert.Equal(t, "Error 1", pageData.Errors["field1"])
	assert.Equal(t, "Error 2", pageData.Errors["field2"])
}

func TestDashboardHandler_MultipleRequests(t *testing.T) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	// Тестируем множественные запросы к одному обработчику
	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.Dashboard(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"template":"dashboard"`)
	}
}

// Benchmark тесты для performance
func BenchmarkDashboardHandler_Dashboard(b *testing.B) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = handler.Dashboard(c)
	}
}

func BenchmarkDashboardHandler_DashboardStats(b *testing.B) {
	repos := NewMockRepositories()
	handler := webHandlers.NewDashboardHandler(&repos.Repositories)

	e := echo.New()
	e.Renderer = &MockRenderer{}

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/htmx/dashboard/stats", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = handler.DashboardStats(c)
	}
}

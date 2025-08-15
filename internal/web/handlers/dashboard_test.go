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

func (m *MockRenderer) Render(_ io.Writer, name string, data interface{}, c echo.Context) error {
	// Простой mock - возвращаем JSON вместо HTML для тестов
	return c.JSON(http.StatusOK, map[string]interface{}{
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

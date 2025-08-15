package handlers

import (
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/handlers"
)

const (
	// Demo data constants for dashboard
	mockTotalIncome             = 50000.00
	mockTotalExpenses           = 35000.00
	mockNetIncome               = 15000.00
	mockTransactionCount        = 42
	mockBudgetCount             = 5
	mockUpdatedTotalIncome      = 52000.00
	mockUpdatedTotalExpenses    = 36500.00
	mockUpdatedNetIncome        = 15500.00
	mockUpdatedTransactionCount = 45
)

// DashboardHandler обрабатывает главную страницу
type DashboardHandler struct {
	*BaseHandler
}

// NewDashboardHandler создает новый обработчик дашборда
func NewDashboardHandler(repositories *handlers.Repositories) *DashboardHandler {
	return &DashboardHandler{
		BaseHandler: NewBaseHandler(repositories),
	}
}

// DashboardData содержит данные для главной страницы
type DashboardData struct {
	TotalIncome      float64 `json:"total_income"`
	TotalExpenses    float64 `json:"total_expenses"`
	NetIncome        float64 `json:"net_income"`
	TransactionCount int     `json:"transaction_count"`
	BudgetCount      int     `json:"budget_count"`
}

// Dashboard отображает главную страницу
func (h *DashboardHandler) Dashboard(c echo.Context) error {
	// TODO: Получить данные из сессии
	// session, err := h.getCurrentSession(c)
	// if err != nil {
	//     return h.redirect(c, "/login")
	// }

	// Тестовые данные для демонстрации
	pageData := &PageData{
		Title: "Главная",
		// CurrentUser: testUser,
		// Family:      testFamily,
		Messages: []Message{
			{
				Type: "info",
				Text: "Добро пожаловать в систему семейного бюджета!",
			},
		},
	}

	// Данные для дашборда
	dashboardData := &DashboardData{
		TotalIncome:      mockTotalIncome,
		TotalExpenses:    mockTotalExpenses,
		NetIncome:        mockNetIncome,
		TransactionCount: mockTransactionCount,
		BudgetCount:      mockBudgetCount,
	}

	// Объединяем данные
	data := struct {
		*PageData
		*DashboardData
	}{
		PageData:      pageData,
		DashboardData: dashboardData,
	}

	return h.renderPage(c, "dashboard", data)
}

// DashboardStats возвращает обновленную статистику (HTMX endpoint)
func (h *DashboardHandler) DashboardStats(c echo.Context) error {
	// Тестовые обновленные данные
	data := &DashboardData{
		TotalIncome:      mockUpdatedTotalIncome,
		TotalExpenses:    mockUpdatedTotalExpenses,
		NetIncome:        mockUpdatedNetIncome,
		TransactionCount: mockUpdatedTransactionCount,
		BudgetCount:      mockBudgetCount,
	}

	return h.renderPartial(c, "dashboard-stats", data)
}

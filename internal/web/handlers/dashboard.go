package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

// Alert level константы
const (
	AlertLevelSuccess = "success"
	AlertLevelDanger  = "danger"
	AlertLevelWarning = "warning"
)

// Dashboard constants
const (
	// DefaultPeriod — период дашборда, если фильтр не задан
	DefaultPeriod = "current_month"

	uncategorizedName = "Без категории"
	generalBudgetName = "Общий бюджет"
)

// DashboardHandler обрабатывает главную страницу
type DashboardHandler struct {
	*BaseHandler
}

// NewDashboardHandler создает новый обработчик дашборда
func NewDashboardHandler(
	repositories *handlers.Repositories,
	services *services.Services,
	cookieSecure bool,
) *DashboardHandler {
	return &DashboardHandler{
		BaseHandler: NewBaseHandler(repositories, services, cookieSecure),
	}
}

// Dashboard отображает главную страницу
func (h *DashboardHandler) Dashboard(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Session error occurred")
	}

	filters := dashboardFilters(c)
	summary, err := h.summary(c, filters)
	if err != nil {
		return err
	}

	// Общие данные страницы собирает buildPageData: он же кладёт CSRF-токен,
	// без которого форма выхода на дашборде уходила с пустым _token и
	// получала 403.
	pageData := h.buildPageData(c, titleDashboard)

	data := struct {
		*PageData
		*webModels.DashboardViewModel

		Filters *webModels.DashboardFilters
	}{
		PageData:           pageData,
		DashboardViewModel: dashboardViewModel(summary),
		Filters:            filters,
	}

	if renderErr := h.renderPage(c, "dashboard", data); renderErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Render error occurred")
	}
	return nil
}

// DashboardFilter обновляет весь dashboard с новыми фильтрами (HTMX endpoint)
func (h *DashboardHandler) DashboardFilter(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	filters := dashboardFilters(c)
	if validationErr := filters.ValidateCustomDateRange(); validationErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid date range provided")
	}

	summary, err := h.summary(c, filters)
	if err != nil {
		return err
	}

	return h.renderPartial(c, "dashboard-content", map[string]any{
		"DashboardViewModel": dashboardViewModel(summary),
		tplKeyFilters:        filters,
	})
}

// DashboardStats возвращает обновленную статистику (HTMX endpoint)
func (h *DashboardHandler) DashboardStats(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	summary, err := h.summary(c, dashboardFilters(c))
	if err != nil {
		return err
	}

	return h.renderPartial(c, "dashboard-stats", map[string]any{
		"MonthlySummary": monthlySummaryCard(summary),
	})
}

// RecentTransactions возвращает последние транзакции (HTMX endpoint)
func (h *DashboardHandler) RecentTransactions(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	summary, err := h.summary(c, dashboardFilters(c))
	if err != nil {
		return err
	}

	return h.renderPartial(c, "recent-transactions", map[string]any{
		"RecentActivity": recentActivityCard(summary),
	})
}

// BudgetOverview возвращает обзор бюджетов (HTMX endpoint)
func (h *DashboardHandler) BudgetOverview(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	summary, err := h.summary(c, dashboardFilters(c))
	if err != nil {
		return err
	}

	return h.renderPartial(c, "budget-overview", map[string]any{
		"BudgetOverview": budgetOverviewCard(summary),
	})
}

// CategoryInsights возвращает аналитику по категориям с фильтрацией (HTMX endpoint)
func (h *DashboardHandler) CategoryInsights(c echo.Context) error {
	if _, err := middleware.GetUserFromContext(c); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	summary, err := h.summary(c, dashboardFilters(c))
	if err != nil {
		return err
	}

	insights := categoryInsightsCard(summary)
	switch c.QueryParam("type") {
	case "expense":
		insights.TopIncomeCategories = nil
		insights.TotalIncome = 0
	case "income":
		insights.TopExpenseCategories = nil
		insights.TotalExpenses = 0
	}

	return h.renderPartial(c, "category-insights-enhanced", map[string]any{
		"CategoryInsights": insights,
	})
}

// summary считает сводку за период фильтра; ошибка сервиса становится 500.
func (h *DashboardHandler) summary(
	c echo.Context,
	filters *webModels.DashboardFilters,
) (*dto.StatsSummary, error) {
	from, to := filters.GetPeriodDates()
	// Негодный пользовательский диапазон не роняет страницу в 500: возвращаемся к периоду.
	if filters.StartDate != nil && filters.EndDate != nil && filters.ValidateCustomDateRange() == nil {
		from, to = *filters.StartDate, *filters.EndDate
	}

	summary, err := h.services.Stats.Summary(c.Request().Context(), from, to)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to load dashboard data")
	}
	return summary, nil
}

func dashboardFilters(c echo.Context) *webModels.DashboardFilters {
	filters := &webModels.DashboardFilters{Period: DefaultPeriod}
	if bindErr := c.Bind(filters); bindErr != nil {
		filters.Period = DefaultPeriod
	}
	if filters.Period == "" {
		filters.Period = DefaultPeriod
	}
	return filters
}

func dashboardViewModel(s *dto.StatsSummary) *webModels.DashboardViewModel {
	return &webModels.DashboardViewModel{
		MonthlySummary:   monthlySummaryCard(s),
		BudgetOverview:   budgetOverviewCard(s),
		RecentActivity:   recentActivityCard(s),
		CategoryInsights: categoryInsightsCard(s),
	}
}

func monthlySummaryCard(s *dto.StatsSummary) *webModels.MonthlySummaryCard {
	const monthLayout = "January 2006"

	return &webModels.MonthlySummaryCard{
		TotalIncome:      s.Current.Income,
		TotalExpenses:    s.Current.Expenses,
		NetIncome:        s.Current.Net,
		TransactionCount: s.Current.TransactionCount,
		PreviousIncome:   s.Previous.Income,
		PreviousExpenses: s.Previous.Expenses,
		IncomeChange:     s.IncomeDelta * webModels.PercentageMultiplier,
		ExpensesChange:   s.ExpensesDelta * webModels.PercentageMultiplier,
		CurrentMonth:     s.From.Format(monthLayout),
		PreviousMonth:    s.Previous.From.Format(monthLayout),
		HasPreviousData:  s.HasPreviousData,
	}
}

func budgetOverviewCard(s *dto.StatsSummary) *webModels.BudgetOverviewCard {
	card := &webModels.BudgetOverviewCard{
		TotalBudgets: len(s.Budgets),
		TopBudgets:   make([]*webModels.BudgetProgressItem, 0, len(s.Budgets)),
	}

	for _, b := range s.Budgets {
		if b.IsActive {
			card.ActiveBudgets++
		}
		switch {
		case b.IsOverBudget:
			card.OverBudget++
		case b.IsNearLimit:
			card.NearLimit++
		}

		if len(card.TopBudgets) < webModels.MaxTopBudgets {
			card.TopBudgets = append(card.TopBudgets, budgetProgressItem(b))
		}
	}

	card.AlertsSummary = &webModels.BudgetAlertsSummary{
		CriticalAlerts: card.OverBudget,
		WarningAlerts:  card.NearLimit,
		TotalAlerts:    card.OverBudget + card.NearLimit,
	}

	return card
}

func budgetProgressItem(b dto.BudgetProgress) *webModels.BudgetProgressItem {
	categoryName := b.CategoryName
	if categoryName == "" {
		categoryName = generalBudgetName
	}

	return &webModels.BudgetProgressItem{
		ID:            b.ID,
		Name:          b.Name,
		CategoryName:  categoryName,
		Amount:        b.Amount,
		Spent:         b.Spent,
		Remaining:     b.Remaining,
		Percentage:    b.Utilization * webModels.PercentageMultiplier,
		Period:        b.Period,
		StartDate:     b.StartDate,
		EndDate:       b.EndDate,
		DaysRemaining: b.DaysRemaining,
		IsOverBudget:  b.IsOverBudget,
		IsNearLimit:   b.IsNearLimit,
		AlertLevel:    budgetAlertLevel(b),
	}
}

func budgetAlertLevel(b dto.BudgetProgress) string {
	switch {
	case b.IsOverBudget:
		return AlertLevelDanger
	case b.IsNearLimit:
		return AlertLevelWarning
	default:
		return AlertLevelSuccess
	}
}

func recentActivityCard(s *dto.StatsSummary) *webModels.RecentActivityCard {
	items := make([]*webModels.RecentTransactionItem, 0, len(s.Recent))
	for _, tx := range s.Recent {
		categoryName := tx.CategoryName
		if categoryName == "" {
			categoryName = uncategorizedName
		}

		items = append(items, &webModels.RecentTransactionItem{
			ID:           tx.ID,
			Description:  tx.Description,
			Amount:       tx.Amount,
			Type:         tx.Type,
			CategoryName: categoryName,
			Date:         tx.Date,
			CreatedAt:    tx.CreatedAt,
			RelativeTime: formatRelativeTime(tx.CreatedAt),
		})
	}

	return &webModels.RecentActivityCard{
		Transactions: items,
		TotalCount:   s.TransactionsTotal,
		ShowingCount: len(items),
		HasMoreData:  s.TransactionsTotal > len(items),
		LastUpdated:  time.Now(),
	}
}

func categoryInsightsCard(s *dto.StatsSummary) *webModels.CategoryInsightsCard {
	return &webModels.CategoryInsightsCard{
		TopExpenseCategories: categoryInsightItems(s.ExpenseCategories),
		TopIncomeCategories:  categoryInsightItems(s.IncomeCategories),
		PeriodStart:          s.From,
		PeriodEnd:            s.To,
		TotalExpenses:        s.Current.Expenses,
		TotalIncome:          s.Current.Income,
	}
}

func categoryInsightItems(shares []dto.CategoryShare) []*webModels.CategoryInsightItem {
	if len(shares) > webModels.MaxTopCategories {
		shares = shares[:webModels.MaxTopCategories]
	}

	items := make([]*webModels.CategoryInsightItem, 0, len(shares))
	for _, share := range shares {
		items = append(items, &webModels.CategoryInsightItem{
			CategoryID:       share.CategoryID,
			CategoryName:     share.Name,
			CategoryColor:    share.Color,
			CategoryIcon:     share.Icon,
			Amount:           share.Amount,
			TransactionCount: share.TransactionCount,
			Percentage:       share.Share * webModels.PercentageMultiplier,
		})
	}
	return items
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "только что"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 минуту назад"
		}
		return fmt.Sprintf("%d минут назад", minutes)
	case diff < webModels.HoursInDay*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 час назад"
		}
		return fmt.Sprintf("%d часов назад", hours)
	case diff < webModels.DaysInWeek*webModels.HoursInDay*time.Hour:
		days := int(diff.Hours() / webModels.HoursInDay)
		if days == 1 {
			return "вчера"
		}
		return fmt.Sprintf("%d дней назад", days)
	case diff < webModels.DaysInMonth*webModels.HoursInDay*time.Hour:
		weeks := int(diff.Hours() / (webModels.HoursInDay * webModels.DaysInWeek))
		if weeks == 1 {
			return "1 неделю назад"
		}
		return fmt.Sprintf("%d недель назад", weeks)
	default:
		return t.Format("02.01.2006")
	}
}

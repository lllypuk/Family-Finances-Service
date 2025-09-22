package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/transaction"
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
	// HTTP status codes
	HTTPStatusInternalServerError = 500

	// Default values for dashboard data loading
	DefaultPeriod = "current_month"

	// Query limits for data fetching
	DefaultQueryLimit = 1000
)

// DashboardHandler обрабатывает главную страницу
type DashboardHandler struct {
	*BaseHandler
}

// NewDashboardHandler создает новый обработчик дашборда
func NewDashboardHandler(repositories *handlers.Repositories, services *services.Services) *DashboardHandler {
	return &DashboardHandler{
		BaseHandler: NewBaseHandler(repositories, services),
	}
}

// Dashboard отображает главную страницу
func (h *DashboardHandler) Dashboard(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(HTTPStatusInternalServerError, "Session error occurred")
	}

	// Парсим фильтры
	filters := &webModels.DashboardFilters{
		Period: DefaultPeriod,
	}
	if bindErr := c.Bind(filters); bindErr != nil {
		// Игнорируем ошибки привязки и используем значения по умолчанию
		filters.Period = DefaultPeriod
	}

	// Получаем реальные данные для всех компонентов
	monthlySummary, err := h.buildMonthlySummary(c.Request().Context(), sessionData.FamilyID, filters)
	if err != nil {
		return echo.NewHTTPError(HTTPStatusInternalServerError, "Failed to load monthly summary")
	}

	// Получаем расширенную статистику
	enhancedStats, _ := h.buildEnhancedStats(c.Request().Context(), sessionData.FamilyID, filters, monthlySummary)

	// Создаем полные данные dashboard
	dashboardData := h.buildDashboardViewModel(
		c.Request().Context(),
		sessionData.FamilyID,
		monthlySummary,
		enhancedStats,
		filters,
	)

	// Получаем данные пользователя для персонализации
	currentUser, userErr := h.services.User.GetUserByID(c.Request().Context(), sessionData.UserID)
	var firstName, lastName string
	if userErr == nil && currentUser != nil {
		firstName = currentUser.FirstName
		lastName = currentUser.LastName
	}

	// Подготавливаем данные для страницы
	pageData := &PageData{
		Title:    "Главная",
		Messages: h.getFlashMessages(c),
		CurrentUser: &SessionData{
			UserID:    sessionData.UserID,
			FamilyID:  sessionData.FamilyID,
			Role:      sessionData.Role,
			Email:     sessionData.Email,
			FirstName: firstName,
			LastName:  lastName,
		},
	}

	// Объединяем данные
	data := struct {
		*PageData
		*webModels.DashboardViewModel

		Filters *webModels.DashboardFilters
	}{
		PageData:           pageData,
		DashboardViewModel: dashboardData,
		Filters:            filters,
	}

	// Пробуем рендерить
	err = h.renderPage(c, "dashboard", data)
	if err != nil {
		return echo.NewHTTPError(HTTPStatusInternalServerError, "Render error occurred")
	}
	return nil
}

// DashboardFilter обновляет весь dashboard с новыми фильтрами (HTMX endpoint)
func (h *DashboardHandler) DashboardFilter(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим фильтры
	filters := &webModels.DashboardFilters{
		Period: DefaultPeriod,
	}
	if bindErr := c.Bind(filters); bindErr != nil {
		filters.Period = DefaultPeriod
	}

	// Валидируем пользовательский диапазон дат
	if validationErr := filters.ValidateCustomDateRange(); validationErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid date range provided")
	}

	// Получаем все данные dashboard с новыми фильтрами
	monthlySummary, err := h.buildMonthlySummary(c.Request().Context(), sessionData.FamilyID, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load monthly summary")
	}

	budgetOverview, err := h.buildBudgetOverview(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load budget overview")
	}

	recentActivity, err := h.buildRecentActivity(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load recent activity")
	}

	categoryInsights, err := h.buildCategoryInsights(c.Request().Context(), sessionData.FamilyID, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load category insights")
	}

	enhancedStats, err := h.buildEnhancedStats(c.Request().Context(), sessionData.FamilyID, filters, monthlySummary)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load enhanced stats")
	}

	dashboardData := &webModels.DashboardViewModel{
		MonthlySummary:   monthlySummary,
		BudgetOverview:   budgetOverview,
		RecentActivity:   recentActivity,
		CategoryInsights: categoryInsights,
		EnhancedStats:    enhancedStats,
	}

	return h.renderPartial(c, "dashboard-content", map[string]any{
		"DashboardViewModel": dashboardData,
		"Filters":            filters,
	})
}

// DashboardStats возвращает обновленную статистику (HTMX endpoint)
func (h *DashboardHandler) DashboardStats(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим фильтры
	filters := &webModels.DashboardFilters{
		Period: "current_month",
	}
	if bindErr := c.Bind(filters); bindErr != nil {
		filters.Period = "current_month"
	}

	// Получаем только monthly summary
	monthlySummary, err := h.buildMonthlySummary(c.Request().Context(), sessionData.FamilyID, filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load monthly summary")
	}

	return h.renderPartial(c, "dashboard-stats", map[string]any{
		"MonthlySummary": monthlySummary,
	})
}

// RecentTransactions возвращает последние транзакции (HTMX endpoint)
func (h *DashboardHandler) RecentTransactions(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем последние транзакции
	recentActivity, err := h.buildRecentActivity(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load recent transactions")
	}

	return h.renderPartial(c, "recent-transactions", map[string]any{
		"RecentActivity": recentActivity,
	})
}

// BudgetOverview возвращает обзор бюджетов (HTMX endpoint)
func (h *DashboardHandler) BudgetOverview(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем обзор бюджетов
	budgetOverview, err := h.buildBudgetOverview(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load budget overview")
	}

	return h.renderPartial(c, "budget-overview", map[string]any{
		"BudgetOverview": budgetOverview,
	})
}

// buildMonthlySummary создает сводку за месяц
func (h *DashboardHandler) buildMonthlySummary(
	ctx context.Context,
	familyID uuid.UUID,
	filters *webModels.DashboardFilters,
) (*webModels.MonthlySummaryCard, error) {
	startDate, endDate := filters.GetPeriodDates()

	// Получаем транзакции за текущий период
	currentFilter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    DefaultQueryLimit, // Достаточно для подсчета
	}

	transactions, err := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, currentFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get current transactions: %w", err)
	}

	// Подсчитываем текущие суммы
	var totalIncome, totalExpenses float64
	transactionCount := len(transactions)

	for _, tx := range transactions {
		switch tx.Type {
		case transaction.TypeIncome:
			totalIncome += tx.Amount
		case transaction.TypeExpense:
			totalExpenses += tx.Amount
		}
	}

	netIncome := totalIncome - totalExpenses

	// Получаем данные за предыдущий период для сравнения
	previousStart, previousEnd := h.getPreviousPeriodDates(startDate, endDate)
	previousIncome, previousExpenses, hasPreviousData := h.calculatePreviousData(
		ctx, familyID, previousStart, previousEnd)
	incomeChange, expensesChange := h.calculateChanges(
		totalIncome,
		totalExpenses,
		previousIncome,
		previousExpenses,
		hasPreviousData,
	)

	return &webModels.MonthlySummaryCard{
		TotalIncome:      totalIncome,
		TotalExpenses:    totalExpenses,
		NetIncome:        netIncome,
		TransactionCount: transactionCount,
		PreviousIncome:   previousIncome,
		PreviousExpenses: previousExpenses,
		IncomeChange:     incomeChange,
		ExpensesChange:   expensesChange,
		CurrentMonth:     startDate.Format("January 2006"),
		PreviousMonth:    previousStart.Format("January 2006"),
		HasPreviousData:  hasPreviousData,
	}, nil
}

// calculatePreviousData получает и вычисляет данные за предыдущий период
func (h *DashboardHandler) calculatePreviousData(
	ctx context.Context,
	familyID uuid.UUID,
	previousStart, previousEnd time.Time,
) (float64, float64, bool) {
	if previousStart.IsZero() || previousEnd.IsZero() {
		return 0, 0, false
	}

	previousFilter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		DateFrom: &previousStart,
		DateTo:   &previousEnd,
		Limit:    DefaultQueryLimit,
	}

	previousTransactions, prevErr := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, previousFilter)
	if prevErr != nil || len(previousTransactions) == 0 {
		return 0, 0, false
	}

	var previousIncome float64
	var previousExpenses float64
	hasPreviousData := true
	for _, tx := range previousTransactions {
		switch tx.Type {
		case transaction.TypeIncome:
			previousIncome += tx.Amount
		case transaction.TypeExpense:
			previousExpenses += tx.Amount
		}
	}

	return previousIncome, previousExpenses, hasPreviousData
}

// calculateChanges вычисляет процентные изменения относительно предыдущего периода
func (h *DashboardHandler) calculateChanges(
	currentIncome, currentExpenses, previousIncome, previousExpenses float64,
	hasPreviousData bool,
) (float64, float64) {
	if !hasPreviousData {
		return 0, 0
	}

	var incomeChange float64
	var expensesChange float64

	if previousIncome > 0 {
		incomeChange = ((currentIncome - previousIncome) / previousIncome) * webModels.PercentageMultiplier
	}
	if previousExpenses > 0 {
		expensesChange = ((currentExpenses - previousExpenses) / previousExpenses) * webModels.PercentageMultiplier
	}
	return incomeChange, expensesChange
}

// buildBudgetOverview создает обзор бюджетов
func (h *DashboardHandler) buildBudgetOverview(
	ctx context.Context,
	familyID uuid.UUID,
) (*webModels.BudgetOverviewCard, error) {
	now := time.Now()

	// Получаем все активные бюджеты
	activeBudgets, err := h.services.Budget.GetActiveBudgets(ctx, familyID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	// Обрабатываем каждый бюджет
	stats, topBudgets := h.processBudgets(ctx, convertBudgetSlice(activeBudgets), now)

	// Сортируем и ограничиваем топ бюджеты
	h.sortAndLimitBudgets(&topBudgets)

	return &webModels.BudgetOverviewCard{
		TotalBudgets:  len(activeBudgets),
		ActiveBudgets: stats.activeBudgetsCount,
		OverBudget:    stats.overBudgetCount,
		NearLimit:     stats.nearLimitCount,
		TopBudgets:    topBudgets,
		AlertsSummary: &webModels.BudgetAlertsSummary{
			CriticalAlerts: stats.overBudgetCount,
			WarningAlerts:  stats.nearLimitCount,
			TotalAlerts:    stats.overBudgetCount + stats.nearLimitCount,
		},
	}, nil
}

// budgetStats содержит статистику по бюджетам
type budgetStats struct {
	activeBudgetsCount int
	overBudgetCount    int
	nearLimitCount     int
}

// processBudgets обрабатывает список бюджетов и возвращает статистику
func (h *DashboardHandler) processBudgets(
	ctx context.Context,
	budgets []*budget.Budget,
	now time.Time,
) (budgetStats, []*webModels.BudgetProgressItem) {
	stats := budgetStats{}
	var topBudgets []*webModels.BudgetProgressItem

	for _, b := range budgets {
		if b.IsActive {
			stats.activeBudgetsCount++
		}

		budgetItem, isOverBudget, isNearLimit := h.createBudgetItem(ctx, *b, now)

		if isOverBudget {
			stats.overBudgetCount++
		} else if isNearLimit {
			stats.nearLimitCount++
		}

		topBudgets = append(topBudgets, budgetItem)
	}

	return stats, topBudgets
}

// createBudgetItem создает элемент прогресса бюджета
func (h *DashboardHandler) createBudgetItem(
	ctx context.Context,
	b budget.Budget,
	now time.Time,
) (*webModels.BudgetProgressItem, bool, bool) {
	// Рассчитываем прогресс
	percentage := 0.0
	if b.Amount > 0 {
		percentage = (b.Spent / b.Amount) * webModels.PercentageMultiplier
	}

	remaining := b.Amount - b.Spent
	daysRemaining := max(int(b.EndDate.Sub(now).Hours()/webModels.HoursInDay), 0)

	isOverBudget := percentage >= webModels.BudgetOverLimitThreshold
	isNearLimit := percentage >= webModels.BudgetNearLimitThreshold && !isOverBudget

	// Определяем уровень алерта
	alertLevel := h.getBudgetAlertLevel(isOverBudget, isNearLimit)

	// Получаем название категории
	categoryName := h.getBudgetCategoryName(ctx, b.CategoryID)

	budgetItem := &webModels.BudgetProgressItem{
		ID:            b.ID,
		Name:          b.Name,
		CategoryName:  categoryName,
		Amount:        b.Amount,
		Spent:         b.Spent,
		Remaining:     remaining,
		Percentage:    percentage,
		Period:        b.Period,
		StartDate:     b.StartDate,
		EndDate:       b.EndDate,
		DaysRemaining: daysRemaining,
		IsOverBudget:  isOverBudget,
		IsNearLimit:   isNearLimit,
		AlertLevel:    alertLevel,
	}

	return budgetItem, isOverBudget, isNearLimit
}

// getBudgetAlertLevel определяет уровень алерта для бюджета
func (h *DashboardHandler) getBudgetAlertLevel(isOverBudget, isNearLimit bool) string {
	if isOverBudget {
		return AlertLevelDanger
	}
	if isNearLimit {
		return AlertLevelWarning
	}
	return AlertLevelSuccess
}

// getBudgetCategoryName получает название категории бюджета
func (h *DashboardHandler) getBudgetCategoryName(ctx context.Context, categoryID *uuid.UUID) string {
	if categoryID == nil {
		return "Общий бюджет"
	}

	if category, err := h.services.Category.GetCategoryByID(ctx, *categoryID); err == nil && category != nil {
		return category.Name
	}

	return "Общий бюджет"
}

// sortAndLimitBudgets сортирует бюджеты по проценту использования и ограничивает количество
func (h *DashboardHandler) sortAndLimitBudgets(topBudgets *[]*webModels.BudgetProgressItem) {
	if len(*topBudgets) <= webModels.MaxTopBudgets {
		return
	}

	// Сортировка по percentage в убывающем порядке
	sort.Slice(*topBudgets, func(i, j int) bool {
		return (*topBudgets)[i].Percentage > (*topBudgets)[j].Percentage
	})

	// Ограничиваем до MaxTopBudgets элементов
	*topBudgets = (*topBudgets)[:webModels.MaxTopBudgets]
}

// buildRecentActivity создает список последних транзакций
func (h *DashboardHandler) buildRecentActivity(
	ctx context.Context,
	familyID uuid.UUID,
) (*webModels.RecentActivityCard, error) {
	// Получаем последние транзакции
	filter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		Limit:    webModels.MaxRecentTransactions,
	}

	transactions, err := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	var recentItems []*webModels.RecentTransactionItem

	for _, tx := range transactions {
		// Получаем название категории
		categoryName := "Без категории"
		if category, catErr := h.services.Category.GetCategoryByID(ctx, tx.CategoryID); catErr == nil {
			categoryName = category.Name
		}

		// Вычисляем относительное время
		relativeTime := h.formatRelativeTime(tx.CreatedAt)

		item := &webModels.RecentTransactionItem{
			ID:           tx.ID,
			Description:  tx.Description,
			Amount:       tx.Amount,
			Type:         tx.Type,
			CategoryName: categoryName,
			Date:         tx.Date,
			CreatedAt:    tx.CreatedAt,
			RelativeTime: relativeTime,
		}

		recentItems = append(recentItems, item)
	}

	// Получаем общее количество транзакций для показа статистики
	totalFilter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		Limit:    DefaultQueryLimit, // Достаточно для подсчета
	}
	allTransactions, totalErr := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, totalFilter)
	totalCount := 0
	if totalErr == nil {
		totalCount = len(allTransactions)
	}

	return &webModels.RecentActivityCard{
		Transactions: recentItems,
		TotalCount:   totalCount,
		ShowingCount: len(recentItems),
		HasMoreData:  totalCount > len(recentItems),
		LastUpdated:  time.Now(),
	}, nil
}

// buildCategoryInsights создает аналитику по категориям
func (h *DashboardHandler) buildCategoryInsights(
	ctx context.Context,
	familyID uuid.UUID,
	filters *webModels.DashboardFilters,
) (*webModels.CategoryInsightsCard, error) {
	startDate, endDate := filters.GetPeriodDates()

	// Получаем транзакции за период
	transactions, err := h.getTransactionsForPeriod(ctx, familyID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Группируем транзакции по категориям
	categoryStats, totalIncome, totalExpenses := h.groupTransactionsByCategory(ctx, transactions)

	// Создаем списки топ категорий
	topExpenseCategories := h.createCategoryInsights(categoryStats, totalExpenses, transaction.TypeExpense)
	topIncomeCategories := h.createCategoryInsights(categoryStats, totalIncome, transaction.TypeIncome)

	// Сортируем и ограничиваем
	h.sortAndLimitCategoryInsights(&topExpenseCategories)
	h.sortAndLimitCategoryInsights(&topIncomeCategories)

	return &webModels.CategoryInsightsCard{
		TopExpenseCategories: topExpenseCategories,
		TopIncomeCategories:  topIncomeCategories,
		PeriodStart:          startDate,
		PeriodEnd:            endDate,
		TotalExpenses:        totalExpenses,
		TotalIncome:          totalIncome,
	}, nil
}

// categoryStatsData содержит статистику по категории
type categoryStatsData struct {
	name     string
	color    string
	icon     string
	income   float64
	expenses float64
	count    int
}

// getTransactionsForPeriod получает транзакции за указанный период
func (h *DashboardHandler) getTransactionsForPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*transaction.Transaction, error) {
	filter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    DefaultQueryLimit,
	}

	transactions, err := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for insights: %w", err)
	}
	return transactions, nil
}

// groupTransactionsByCategory группирует транзакции по категориям
func (h *DashboardHandler) groupTransactionsByCategory(
	ctx context.Context,
	transactions []*transaction.Transaction,
) (map[uuid.UUID]*categoryStatsData, float64, float64) {
	categoryStats := make(map[uuid.UUID]*categoryStatsData)
	var totalIncome, totalExpenses float64

	for _, tx := range transactions {
		categoryID := tx.CategoryID

		// Создаем статистику для категории если её ещё нет
		if _, exists := categoryStats[categoryID]; !exists {
			stats := h.createCategoryStats(ctx, categoryID)
			if stats == nil {
				continue
			}
			categoryStats[categoryID] = stats
		}

		stats := categoryStats[categoryID]
		stats.count++

		// Обновляем суммы
		h.updateCategoryAmounts(stats, *tx, &totalIncome, &totalExpenses)
	}

	return categoryStats, totalIncome, totalExpenses
}

// createCategoryStats создает статистику для категории
func (h *DashboardHandler) createCategoryStats(ctx context.Context, categoryID uuid.UUID) *categoryStatsData {
	category, err := h.services.Category.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return nil
	}

	return &categoryStatsData{
		name:  category.Name,
		color: category.Color,
		icon:  category.Icon,
	}
}

// updateCategoryAmounts обновляет суммы по категории
func (h *DashboardHandler) updateCategoryAmounts(
	stats *categoryStatsData,
	tx transaction.Transaction,
	totalIncome, totalExpenses *float64,
) {
	switch tx.Type {
	case transaction.TypeIncome:
		stats.income += tx.Amount
		*totalIncome += tx.Amount
	case transaction.TypeExpense:
		stats.expenses += tx.Amount
		*totalExpenses += tx.Amount
	}
}

// createCategoryInsights создает список аналитики по категориям для определенного типа
func (h *DashboardHandler) createCategoryInsights(
	categoryStats map[uuid.UUID]*categoryStatsData,
	total float64,
	txType transaction.Type,
) []*webModels.CategoryInsightItem {
	var insights []*webModels.CategoryInsightItem

	for categoryID, stats := range categoryStats {
		amount := h.getAmountByType(stats, txType)
		if amount <= 0 {
			continue
		}

		percentage := h.calculatePercentage(amount, total)

		item := &webModels.CategoryInsightItem{
			CategoryID:       categoryID,
			CategoryName:     stats.name,
			CategoryColor:    stats.color,
			CategoryIcon:     stats.icon,
			Amount:           amount,
			TransactionCount: stats.count,
			Percentage:       percentage,
		}
		insights = append(insights, item)
	}

	return insights
}

// getAmountByType возвращает сумму по типу транзакции
func (h *DashboardHandler) getAmountByType(stats *categoryStatsData, txType transaction.Type) float64 {
	switch txType {
	case transaction.TypeIncome:
		return stats.income
	case transaction.TypeExpense:
		return stats.expenses
	default:
		return 0
	}
}

// calculatePercentage вычисляет процент от общей суммы
func (h *DashboardHandler) calculatePercentage(amount, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return (amount / total) * webModels.PercentageMultiplier
}

// sortAndLimitCategoryInsights сортирует и ограничивает список аналитики по категориям
func (h *DashboardHandler) sortAndLimitCategoryInsights(insights *[]*webModels.CategoryInsightItem) {
	sortCategoryInsights(*insights)

	if len(*insights) > webModels.MaxTopCategories {
		*insights = (*insights)[:webModels.MaxTopCategories]
	}
}

// Helper methods

// convertBudgetSlice converts []*budget.Budget to []*budget.Budget (no-op for type compatibility)
func convertBudgetSlice(budgets []*budget.Budget) []*budget.Budget {
	return budgets
}

func (h *DashboardHandler) getPreviousPeriodDates(currentStart, currentEnd time.Time) (time.Time, time.Time) {
	duration := currentEnd.Sub(currentStart)
	previousEnd := currentStart.Add(-time.Second)
	previousStart := previousEnd.Add(-duration)
	return previousStart, previousEnd
}

func (h *DashboardHandler) formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

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

func sortCategoryInsights(insights []*webModels.CategoryInsightItem) {
	// Простая сортировка по убыванию суммы
	for i := range len(insights) - 1 {
		for j := i + 1; j < len(insights); j++ {
			if insights[j].Amount > insights[i].Amount {
				insights[i], insights[j] = insights[j], insights[i]
			}
		}
	}
}

// buildEnhancedStats создает расширенную статистику
func (h *DashboardHandler) buildEnhancedStats(
	ctx context.Context,
	familyID uuid.UUID,
	filters *webModels.DashboardFilters,
	_ *webModels.MonthlySummaryCard,
) (*webModels.EnhancedStatsCard, error) {
	startDate, endDate := filters.GetPeriodDates()

	// Получаем транзакции за период
	transactions, err := h.getTransactionsForPeriod(ctx, familyID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Подсчитываем базовую статистику
	var incomeTransactions, expenseTransactions int
	var totalIncomeAmount, totalExpenseAmount float64

	for _, tx := range transactions {
		switch tx.Type {
		case transaction.TypeIncome:
			incomeTransactions++
			totalIncomeAmount += tx.Amount
		case transaction.TypeExpense:
			expenseTransactions++
			totalExpenseAmount += tx.Amount
		}
	}

	// Вычисляем период в днях
	const hoursInDay = 24
	periodDays := int(endDate.Sub(startDate).Hours()/hoursInDay) + 1
	if periodDays <= 0 {
		periodDays = 1
	}

	// Средние значения за день
	avgIncomePerDay := totalIncomeAmount / float64(periodDays)
	avgExpensePerDay := totalExpenseAmount / float64(periodDays)

	// Средний размер транзакции
	avgTransactionAmount := 0.0
	if len(transactions) > 0 {
		avgTransactionAmount = (totalIncomeAmount + totalExpenseAmount) / float64(len(transactions))
	}

	// Норма сбережений (% от доходов)
	savingsRate := 0.0
	if totalIncomeAmount > 0 {
		savingsRate = ((totalIncomeAmount - totalExpenseAmount) / totalIncomeAmount) * webModels.PercentageMultiplier
	}

	// Прогноз (если это текущий месяц)
	forecast := h.buildForecast(ctx, familyID, startDate, endDate, avgIncomePerDay, avgExpensePerDay)

	// TODO: Implement user preferences system for financial goals
	// For now, goals are disabled to avoid misleading hardcoded values
	var incomeGoal float64
	var incomeGoalProgress float64

	var expenseBudget float64
	var expenseBudgetProgress float64

	return &webModels.EnhancedStatsCard{
		AvgIncomePerDay:          avgIncomePerDay,
		IncomeTransactionsCount:  incomeTransactions,
		IncomeGoal:               incomeGoal,
		IncomeGoalProgress:       incomeGoalProgress,
		AvgExpensePerDay:         avgExpensePerDay,
		ExpenseTransactionsCount: expenseTransactions,
		ExpenseBudget:            expenseBudget,
		ExpenseBudgetProgress:    expenseBudgetProgress,
		AvgTransactionAmount:     avgTransactionAmount,
		SavingsRate:              savingsRate,
		Forecast:                 forecast,
	}, nil
}

// buildForecast создает прогноз на основе текущих трендов
func (h *DashboardHandler) buildForecast(
	_ context.Context,
	_ uuid.UUID,
	startDate, endDate time.Time,
	avgIncomePerDay, avgExpensePerDay float64,
) *webModels.ForecastData {
	now := time.Now()

	// Проверяем, является ли это текущим месяцем
	if now.Before(startDate) || now.After(endDate) {
		return nil
	}

	// Дни до конца периода
	const hoursInDay = 24
	daysRemaining := int(endDate.Sub(now).Hours() / hoursInDay)
	if daysRemaining <= 0 {
		return nil
	}

	// Прогнозируемые доходы и расходы до конца месяца
	expectedIncome := avgIncomePerDay * float64(daysRemaining)
	expectedExpenses := avgExpensePerDay * float64(daysRemaining)
	monthEndBalance := expectedIncome - expectedExpenses

	return &webModels.ForecastData{
		ExpectedIncome:   expectedIncome,
		ExpectedExpenses: expectedExpenses,
		MonthEndBalance:  monthEndBalance,
		DaysRemaining:    daysRemaining,
	}
}

// CategoryInsights возвращает аналитику по категориям с фильтрацией (HTMX endpoint)
func (h *DashboardHandler) CategoryInsights(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим фильтры
	filters := &webModels.DashboardFilters{
		Period: "current_month",
	}
	if bindErr := c.Bind(filters); bindErr != nil {
		filters.Period = "current_month"
	}

	// Получаем тип фильтра из query параметра
	filterType := c.QueryParam("type")
	if filterType == "" {
		filterType = "all"
	}

	// Получаем даты периода
	startDate, endDate := filters.GetPeriodDates()
	if filters.StartDate != nil && filters.EndDate != nil {
		startDate = *filters.StartDate
		endDate = *filters.EndDate
	}

	// Получаем аналитику по категориям
	categoryInsights, err := h.buildCategoryInsightsWithFilter(
		c.Request().Context(),
		sessionData.FamilyID,
		startDate,
		endDate,
		filterType,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load category insights")
	}

	return h.renderPartial(c, "category-insights-enhanced", map[string]any{
		"CategoryInsights": categoryInsights,
	})
}

// buildCategoryInsightsWithFilter создает аналитику по категориям с поддержкой фильтрации
func (h *DashboardHandler) buildCategoryInsightsWithFilter(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	filterType string,
) (*webModels.CategoryInsightsCard, error) {
	// Создаем фильтры для базового метода
	filters := &webModels.DashboardFilters{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	// Базовый buildCategoryInsights для получения всех данных
	baseInsights, err := h.buildCategoryInsights(ctx, familyID, filters)
	if err != nil {
		return nil, err
	}

	// Фильтруем данные в зависимости от типа
	switch filterType {
	case "expense":
		return &webModels.CategoryInsightsCard{
			TopExpenseCategories: baseInsights.TopExpenseCategories,
			TopIncomeCategories:  nil,
			PeriodStart:          baseInsights.PeriodStart,
			PeriodEnd:            baseInsights.PeriodEnd,
			TotalExpenses:        baseInsights.TotalExpenses,
			TotalIncome:          0,
		}, nil
	case "income":
		return &webModels.CategoryInsightsCard{
			TopExpenseCategories: nil,
			TopIncomeCategories:  baseInsights.TopIncomeCategories,
			PeriodStart:          baseInsights.PeriodStart,
			PeriodEnd:            baseInsights.PeriodEnd,
			TotalExpenses:        0,
			TotalIncome:          baseInsights.TotalIncome,
		}, nil
	default: // "all"
		return baseInsights, nil
	}
}

// buildDashboardViewModel создает полную модель данных для dashboard
func (h *DashboardHandler) buildDashboardViewModel(
	ctx context.Context,
	familyID uuid.UUID,
	monthlySummary *webModels.MonthlySummaryCard,
	enhancedStats *webModels.EnhancedStatsCard,
	filters *webModels.DashboardFilters,
) *webModels.DashboardViewModel {
	return &webModels.DashboardViewModel{
		MonthlySummary: monthlySummary,
		EnhancedStats:  enhancedStats,
		BudgetOverview: func() *webModels.BudgetOverviewCard {
			budgetOverview, budgetErr := h.buildBudgetOverview(ctx, familyID)
			if budgetErr != nil {
				return &webModels.BudgetOverviewCard{
					TotalBudgets:  0,
					ActiveBudgets: 0,
					OverBudget:    0,
					NearLimit:     0,
					TopBudgets:    []*webModels.BudgetProgressItem{},
					AlertsSummary: &webModels.BudgetAlertsSummary{
						CriticalAlerts: 0,
						WarningAlerts:  0,
						TotalAlerts:    0,
					},
				}
			}
			return budgetOverview
		}(),
		RecentActivity: func() *webModels.RecentActivityCard {
			recentActivity, activityErr := h.buildRecentActivity(ctx, familyID)
			if activityErr != nil {
				return &webModels.RecentActivityCard{
					Transactions: []*webModels.RecentTransactionItem{},
					TotalCount:   0,
					ShowingCount: 0,
					HasMoreData:  false,
					LastUpdated:  time.Now(),
				}
			}
			return recentActivity
		}(),
		CategoryInsights: func() *webModels.CategoryInsightsCard {
			insights, insightsErr := h.buildCategoryInsights(ctx, familyID, filters)
			if insightsErr != nil {
				return &webModels.CategoryInsightsCard{
					TopExpenseCategories: []*webModels.CategoryInsightItem{},
					TopIncomeCategories:  []*webModels.CategoryInsightItem{},
					PeriodStart:          time.Now().AddDate(0, -1, 0),
					PeriodEnd:            time.Now(),
					TotalExpenses:        0.0,
				}
			}
			return insights
		}(),
	}
}

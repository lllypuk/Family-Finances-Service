package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
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
		return c.JSON(500, map[string]string{"error": "Session error: " + err.Error()})
	}

	// Парсим фильтры
	filters := &webModels.DashboardFilters{
		Period: "current_month",
	}
	if bindErr := c.Bind(filters); bindErr != nil {
		// Игнорируем ошибки привязки и используем значения по умолчанию
		filters.Period = "current_month"
	}

	// Создаем минимальные данные для тестирования
	dashboardData := &webModels.DashboardViewModel{
		MonthlySummary: &webModels.MonthlySummaryCard{
			TotalIncome:     1000.0,
			TotalExpenses:   500.0,
			NetIncome:       500.0,
			CurrentMonth:    "Декабрь 2024",
			HasPreviousData: false,
		},
		BudgetOverview: &webModels.BudgetOverviewCard{
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
		},
		RecentActivity: &webModels.RecentActivityCard{
			Transactions: []*webModels.RecentTransactionItem{},
			TotalCount:   0,
			ShowingCount: 0,
			HasMoreData:  false,
			LastUpdated:  time.Now(),
		},
		CategoryInsights: &webModels.CategoryInsightsCard{
			TopExpenseCategories: []*webModels.CategoryInsightItem{},
			TopIncomeCategories:  []*webModels.CategoryInsightItem{},
			PeriodStart:          time.Now().AddDate(0, -1, 0),
			PeriodEnd:            time.Now(),
			TotalExpenses:        0.0,
		},
	}

	if dashboardData == nil {
		return c.JSON(500, map[string]string{"error": "Dashboard data is nil"})
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
			FirstName: "", // Заполним позже если нужно
			LastName:  "", // Заполним позже если нужно
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
		return c.JSON(500, map[string]string{"error": "Render error: " + err.Error()})
	}
	return nil
}

// DashboardStats возвращает обновленную статистику (HTMX endpoint)
func (h *DashboardHandler) DashboardStats(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
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
		return h.handleError(c, err, "Failed to load monthly summary")
	}

	return h.renderPartial(c, "components/dashboard-stats", map[string]interface{}{
		"MonthlySummary": monthlySummary,
	})
}

// RecentTransactions возвращает последние транзакции (HTMX endpoint)
func (h *DashboardHandler) RecentTransactions(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Получаем последние транзакции
	recentActivity, err := h.buildRecentActivity(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return h.handleError(c, err, "Failed to load recent transactions")
	}

	return h.renderPartial(c, "components/recent-transactions", map[string]interface{}{
		"RecentActivity": recentActivity,
	})
}

// BudgetOverview возвращает обзор бюджетов (HTMX endpoint)
func (h *DashboardHandler) BudgetOverview(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Получаем обзор бюджетов
	budgetOverview, err := h.buildBudgetOverview(c.Request().Context(), sessionData.FamilyID)
	if err != nil {
		return h.handleError(c, err, "Failed to load budget overview")
	}

	return h.renderPartial(c, "components/budget-overview", map[string]interface{}{
		"BudgetOverview": budgetOverview,
	})
}

// buildDashboardData собирает все данные для dashboard
func (h *DashboardHandler) buildDashboardData(
	ctx context.Context,
	familyID uuid.UUID,
	filters *webModels.DashboardFilters,
) (*webModels.DashboardViewModel, error) {
	// Собираем данные параллельно
	monthlySummary, err := h.buildMonthlySummary(ctx, familyID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build monthly summary: %w", err)
	}

	budgetOverview, err := h.buildBudgetOverview(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to build budget overview: %w", err)
	}

	recentActivity, err := h.buildRecentActivity(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to build recent activity: %w", err)
	}

	categoryInsights, err := h.buildCategoryInsights(ctx, familyID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build category insights: %w", err)
	}

	return &webModels.DashboardViewModel{
		MonthlySummary:   monthlySummary,
		BudgetOverview:   budgetOverview,
		RecentActivity:   recentActivity,
		CategoryInsights: categoryInsights,
	}, nil
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
		Limit:    1000, // Достаточно для подсчета
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
	var previousIncome, previousExpenses float64
	var incomeChange, expensesChange float64
	hasPreviousData := false

	previousStart, previousEnd := h.getPreviousPeriodDates(startDate, endDate)
	if !previousStart.IsZero() && !previousEnd.IsZero() {
		previousFilter := dto.TransactionFilterDTO{
			FamilyID: familyID,
			DateFrom: &previousStart,
			DateTo:   &previousEnd,
			Limit:    1000,
		}

		previousTransactions, prevErr := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, previousFilter)
		if prevErr == nil && len(previousTransactions) > 0 {
			hasPreviousData = true

			for _, tx := range previousTransactions {
				switch tx.Type {
				case transaction.TypeIncome:
					previousIncome += tx.Amount
				case transaction.TypeExpense:
					previousExpenses += tx.Amount
				}
			}

			// Вычисляем процентные изменения
			if previousIncome > 0 {
				incomeChange = ((totalIncome - previousIncome) / previousIncome) * webModels.PercentageMultiplier
			}
			if previousExpenses > 0 {
				expensesChange = ((totalExpenses - previousExpenses) / previousExpenses) * webModels.PercentageMultiplier
			}
		}
	}

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

	// Подсчитываем статистику
	totalBudgets := len(activeBudgets)
	activeBudgetsCount := 0
	overBudgetCount := 0
	nearLimitCount := 0

	var topBudgets []*webModels.BudgetProgressItem

	for _, b := range activeBudgets {
		if b.IsActive {
			activeBudgetsCount++
		}

		// Рассчитываем прогресс
		percentage := 0.0
		if b.Amount > 0 {
			percentage = (b.Spent / b.Amount) * webModels.PercentageMultiplier
		}

		remaining := b.Amount - b.Spent
		daysRemaining := int(b.EndDate.Sub(now).Hours() / webModels.HoursInDay)
		if daysRemaining < 0 {
			daysRemaining = 0
		}

		isOverBudget := percentage >= webModels.BudgetOverLimitThreshold
		isNearLimit := percentage >= webModels.BudgetNearLimitThreshold && !isOverBudget

		if isOverBudget {
			overBudgetCount++
		} else if isNearLimit {
			nearLimitCount++
		}

		// Определяем уровень алерта
		alertLevel := "success"
		if isOverBudget {
			alertLevel = "danger"
		} else if isNearLimit {
			alertLevel = "warning"
		}

		// Получаем название категории если есть
		categoryName := "Общий бюджет"
		if b.CategoryID != nil {
			if category, catErr := h.services.Category.GetCategoryByID(ctx, *b.CategoryID); catErr == nil {
				categoryName = category.Name
			}
		}

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

		topBudgets = append(topBudgets, budgetItem)
	}

	// Сортируем по проценту использования (убывание) и берем топ 5
	if len(topBudgets) > webModels.MaxTopBudgets {
		// Простая сортировка по percentage
		for i := range len(topBudgets) - 1 {
			for j := i + 1; j < len(topBudgets); j++ {
				if topBudgets[j].Percentage > topBudgets[i].Percentage {
					topBudgets[i], topBudgets[j] = topBudgets[j], topBudgets[i]
				}
			}
		}
		topBudgets = topBudgets[:webModels.MaxTopBudgets]
	}

	return &webModels.BudgetOverviewCard{
		TotalBudgets:  totalBudgets,
		ActiveBudgets: activeBudgetsCount,
		OverBudget:    overBudgetCount,
		NearLimit:     nearLimitCount,
		TopBudgets:    topBudgets,
		AlertsSummary: &webModels.BudgetAlertsSummary{
			CriticalAlerts: overBudgetCount,
			WarningAlerts:  nearLimitCount,
			TotalAlerts:    overBudgetCount + nearLimitCount,
		},
	}, nil
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
		Limit:    1000, // Достаточно для подсчета
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

	// Получаем все транзакции за период
	filter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    1000,
	}

	transactions, err := h.services.Transaction.GetTransactionsByFamily(ctx, familyID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for insights: %w", err)
	}

	// Группируем по категориям
	categoryStats := make(map[uuid.UUID]*struct {
		name     string
		color    string
		icon     string
		income   float64
		expenses float64
		count    int
	})

	var totalIncome, totalExpenses float64

	for _, tx := range transactions {
		// CategoryID всегда uuid.UUID, не указатель
		categoryID := tx.CategoryID
		if _, exists := categoryStats[categoryID]; !exists {
			// Получаем информацию о категории
			category, catErr := h.services.Category.GetCategoryByID(ctx, categoryID)
			if catErr != nil {
				continue
			}

			categoryStats[categoryID] = &struct {
				name     string
				color    string
				icon     string
				income   float64
				expenses float64
				count    int
			}{
				name:  category.Name,
				color: category.Color,
				icon:  category.Icon,
			}
		}

		stats := categoryStats[categoryID]
		stats.count++

		switch tx.Type {
		case transaction.TypeIncome:
			stats.income += tx.Amount
			totalIncome += tx.Amount
		case transaction.TypeExpense:
			stats.expenses += tx.Amount
			totalExpenses += tx.Amount
		}
	}

	// Создаем списки топ категорий
	var topExpenseCategories, topIncomeCategories []*webModels.CategoryInsightItem

	for categoryID, stats := range categoryStats {
		if stats.expenses > 0 {
			percentage := 0.0
			if totalExpenses > 0 {
				percentage = (stats.expenses / totalExpenses) * webModels.PercentageMultiplier
			}

			expenseItem := &webModels.CategoryInsightItem{
				CategoryID:       categoryID,
				CategoryName:     stats.name,
				CategoryColor:    stats.color,
				CategoryIcon:     stats.icon,
				Amount:           stats.expenses,
				TransactionCount: stats.count,
				Percentage:       percentage,
			}
			topExpenseCategories = append(topExpenseCategories, expenseItem)
		}

		if stats.income > 0 {
			percentage := 0.0
			if totalIncome > 0 {
				percentage = (stats.income / totalIncome) * webModels.PercentageMultiplier
			}

			incomeItem := &webModels.CategoryInsightItem{
				CategoryID:       categoryID,
				CategoryName:     stats.name,
				CategoryColor:    stats.color,
				CategoryIcon:     stats.icon,
				Amount:           stats.income,
				TransactionCount: stats.count,
				Percentage:       percentage,
			}
			topIncomeCategories = append(topIncomeCategories, incomeItem)
		}
	}

	// Сортируем по сумме (убывание) и берем топ 5
	h.sortCategoryInsights(topExpenseCategories)
	h.sortCategoryInsights(topIncomeCategories)

	if len(topExpenseCategories) > webModels.MaxTopCategories {
		topExpenseCategories = topExpenseCategories[:webModels.MaxTopCategories]
	}
	if len(topIncomeCategories) > webModels.MaxTopCategories {
		topIncomeCategories = topIncomeCategories[:webModels.MaxTopCategories]
	}

	return &webModels.CategoryInsightsCard{
		TopExpenseCategories: topExpenseCategories,
		TopIncomeCategories:  topIncomeCategories,
		PeriodStart:          startDate,
		PeriodEnd:            endDate,
		TotalExpenses:        totalExpenses,
		TotalIncome:          totalIncome,
	}, nil
}

// Helper methods

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

func (h *DashboardHandler) sortCategoryInsights(insights []*webModels.CategoryInsightItem) {
	// Простая сортировка по убыванию суммы
	for i := range len(insights) - 1 {
		for j := i + 1; j < len(insights); j++ {
			if insights[j].Amount > insights[i].Amount {
				insights[i], insights[j] = insights[j], insights[i]
			}
		}
	}
}

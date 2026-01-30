package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

const (
	// DefaultBudgetLimit is the default pagination limit for budget queries
	DefaultBudgetLimit = 50

	// BudgetExceededThreshold is the percentage threshold when budget is considered exceeded (100%)
	BudgetExceededThreshold = 100
	// BudgetCriticalThreshold is the percentage threshold for critical budget alerts (90%)
	BudgetCriticalThreshold = 90
	// BudgetWarningThreshold is the percentage threshold for budget warning alerts (80%)
	BudgetWarningThreshold = 80

	// DefaultAlertThreshold is the default threshold used for healthy budget status (80%)
	DefaultAlertThreshold = 80
)

// BudgetHandler обрабатывает HTTP запросы для бюджетов
type BudgetHandler struct {
	*BaseHandler

	validator *validator.Validate
}

// NewBudgetHandler создает новый обработчик бюджетов
func NewBudgetHandler(repositories *handlers.Repositories, services *services.Services) *BudgetHandler {
	return &BudgetHandler{
		BaseHandler: NewBaseHandler(repositories, services),
		validator:   validator.New(),
	}
}

// Index отображает список бюджетов с прогрессом
func (h *BudgetHandler) Index(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим параметры фильтрации
	filter := dto.BudgetFilterDTO{
		Limit:  DefaultBudgetLimit, // По умолчанию
		Offset: 0,
	}

	// Парсим фильтры из query parameters
	if isActive := c.QueryParam("is_active"); isActive != "" {
		if active, parseErr := strconv.ParseBool(isActive); parseErr == nil {
			filter.IsActive = &active
		}
	}

	if period := c.QueryParam("period"); period != "" {
		switch period {
		case "weekly", "monthly", "yearly", "custom":
			budgetPeriod := budget.Period(period)
			filter.Period = &budgetPeriod
		}
	}

	// Получаем список бюджетов
	budgets, err := h.services.Budget.GetAllBudgets(c.Request().Context(), filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get budgets")
	}

	// Конвертируем в view модели
	budgetVMs := make([]webModels.BudgetProgressVM, len(budgets))
	for i, b := range budgets {
		budgetVMs[i].FromDomain(b)

		// Добавляем информацию о категории если есть
		if b.CategoryID != nil {
			category, catErr := h.services.Category.GetCategoryByID(c.Request().Context(), *b.CategoryID)
			if catErr == nil {
				budgetVMs[i].CategoryName = category.Name
				budgetVMs[i].CategoryColor = category.Color
			}
		}
	}

	// Подготавливаем данные для фильтрации
	filterForm := webModels.BudgetFilter{
		IsActive: filter.IsActive,
	}
	if filter.Period != nil {
		filterForm.Period = string(*filter.Period)
	}

	pageData := &PageData{
		Title: "Budgets",
	}

	data := map[string]any{
		"PageData": pageData,
		"Budgets":  budgetVMs,
		"Filter":   filterForm,
	}

	return h.renderPage(c, "pages/budgets/index", data)
}

// New отображает форму создания нового бюджета
func (h *BudgetHandler) New(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем список категорий для селектора
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		nil, // Все типы категорий
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Подготавливаем опции категорий
	categoryOptions := make([]map[string]any, len(categories)+1)
	categoryOptions[0] = map[string]any{
		"ID":   "",
		"Name": "All Categories",
	}
	for i, cat := range categories {
		categoryOptions[i+1] = map[string]any{
			"ID":   cat.ID.String(),
			"Name": cat.Name,
		}
	}

	// Предзаполняем форму с умолчательными значениями
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	defaultForm := webModels.BudgetForm{
		Period:    "monthly",
		StartDate: startOfMonth.Format("2006-01-02"),
		EndDate:   endOfMonth.Format("2006-01-02"),
		IsActive:  true,
	}

	// Получаем CSRF токен
	csrfToken, _ := middleware.GetCSRFToken(c)

	pageData := &PageData{
		Title: "New Budget",
	}

	data := map[string]any{
		"PageData":        pageData,
		"CategoryOptions": categoryOptions,
		"DefaultForm":     defaultForm,
		"CSRFToken":       csrfToken,
	}

	return h.renderPage(c, "pages/budgets/new", data)
}

// Create создает новый бюджет
func (h *BudgetHandler) Create(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим данные формы
	var form webModels.BudgetForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			return h.renderPartial(c, "components/form_errors", map[string]any{
				"Errors": validationErrors,
			})
		}

		return h.renderBudgetFormWithErrors(c, form, "New Budget")
	}

	// Парсим сумму
	amount, err := form.GetAmount()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid amount")
	}

	// Парсим даты
	startDate, err := form.GetStartDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date")
	}

	endDate, err := form.GetEndDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date")
	}

	// Создаем DTO для создания бюджета
	createDTO := dto.CreateBudgetDTO{
		Name:       form.Name,
		Amount:     amount,
		Period:     form.ToBudgetPeriod(),
		CategoryID: form.GetCategoryID(),
		StartDate:  startDate,
		EndDate:    endDate,
	}

	// Создаем бюджет через сервис
	createdBudget, err := h.services.Budget.CreateBudget(c.Request().Context(), createDTO)
	if err != nil {
		errorMsg := h.getBudgetServiceErrorMessage(err)
		return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
	}

	// Успешное создание - редирект на просмотр бюджета
	budgetURL := fmt.Sprintf("/budgets/%s", createdBudget.ID)
	if h.IsHTMXRequest(c) {
		// Для HTMX запросов используем Hx-Redirect
		c.Response().Header().Set("Hx-Redirect", budgetURL)
	}

	// Для обычных запросов - стандартный редирект
	return h.redirect(c, budgetURL)
}

// Edit отображает форму редактирования бюджета
func (h *BudgetHandler) Edit(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID бюджета
	id := c.Param("id")
	budgetID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Получаем бюджет
	budgetEntity, err := h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// In single-family model, all budgets belong to the family
	// No additional access check needed

	// Получаем список категорий
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		nil,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Подготавливаем опции категорий
	categoryOptions := make([]map[string]any, len(categories)+1)
	categoryOptions[0] = map[string]any{
		"ID":   "",
		"Name": "All Categories",
	}
	for i, cat := range categories {
		categoryOptions[i+1] = map[string]any{
			"ID":   cat.ID.String(),
			"Name": cat.Name,
		}
	}

	// Подготавливаем форму с данными бюджета
	form := webModels.BudgetForm{
		Name:      budgetEntity.Name,
		Amount:    fmt.Sprintf("%.2f", budgetEntity.Amount),
		Period:    string(budgetEntity.Period),
		StartDate: budgetEntity.StartDate.Format("2006-01-02"),
		EndDate:   budgetEntity.EndDate.Format("2006-01-02"),
		IsActive:  budgetEntity.IsActive,
	}

	if budgetEntity.CategoryID != nil {
		form.CategoryID = budgetEntity.CategoryID.String()
	}

	// Получаем CSRF токен
	csrfToken, _ := middleware.GetCSRFToken(c)

	pageData := &PageData{
		Title: "Edit Budget: " + budgetEntity.Name,
	}

	data := map[string]any{
		"PageData":        pageData,
		"Form":            form,
		"CategoryOptions": categoryOptions,
		"BudgetID":        budgetID.String(),
		"CSRFToken":       csrfToken,
	}

	return h.renderPage(c, "pages/budgets/edit", data)
}

// Update обновляет существующий бюджет
func (h *BudgetHandler) Update(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID бюджета
	id := c.Param("id")
	budgetID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Проверяем, что бюджет существует
	_, err = h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// Single family model - no family check needed

	// Парсим данные формы
	var form webModels.BudgetForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			return h.renderPartial(c, "components/form_errors", map[string]any{
				"Errors": validationErrors,
			})
		}

		return h.renderBudgetFormWithErrors(c, form, "Edit Budget")
	}

	// Парсим новые значения
	amount, err := form.GetAmount()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid amount")
	}

	startDate, err := form.GetStartDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date")
	}

	endDate, err := form.GetEndDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date")
	}

	// Создаем DTO для обновления
	updateDTO := dto.UpdateBudgetDTO{
		Name:      &form.Name,
		Amount:    &amount,
		StartDate: &startDate,
		EndDate:   &endDate,
		IsActive:  &form.IsActive,
	}

	// Обновляем бюджет через сервис
	updatedBudget, err := h.services.Budget.UpdateBudget(c.Request().Context(), budgetID, updateDTO)
	if err != nil {
		errorMsg := h.getBudgetServiceErrorMessage(err)
		return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
	}

	// Успешное обновление - редирект на просмотр
	budgetURL := fmt.Sprintf("/budgets/%s", updatedBudget.ID)
	if h.IsHTMXRequest(c) {
		// Для HTMX запросов используем Hx-Redirect
		c.Response().Header().Set("Hx-Redirect", budgetURL)
		return c.NoContent(http.StatusOK)
	}

	// Для обычных запросов - стандартный редирект
	return h.redirect(c, budgetURL)
}

// Delete удаляет бюджет
func (h *BudgetHandler) Delete(c echo.Context) error {
	return h.handleDelete(c, DeleteEntityParams{
		EntityName: "budget",
		GetEntityFunc: func(ctx echo.Context, entityID uuid.UUID) (any, error) {
			return h.services.Budget.GetBudgetByID(ctx.Request().Context(), entityID)
		},
		DeleteEntityFunc: func(ctx echo.Context, entityID uuid.UUID) error {
			_, err := middleware.GetUserFromContext(ctx)
			if err != nil {
				return err
			}
			return h.services.Budget.DeleteBudget(ctx.Request().Context(), entityID)
		},
		GetErrorMsgFunc: h.getBudgetServiceErrorMessage,
		RedirectURL:     "/budgets",
	})
}

// Progress возвращает обновленный прогресс бюджета (HTMX)
func (h *BudgetHandler) Progress(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID бюджета
	id := c.Param("id")
	budgetID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Получаем бюджет
	budgetEntity, err := h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// Проверяем права доступа
	// Single family model - no family check needed

	// Конвертируем в view модель
	progressVM := webModels.BudgetProgressVM{}
	progressVM.FromDomain(budgetEntity)

	// Добавляем информацию о категории если есть
	if budgetEntity.CategoryID != nil {
		category, catErr := h.services.Category.GetCategoryByID(c.Request().Context(), *budgetEntity.CategoryID)
		if catErr == nil {
			progressVM.CategoryName = category.Name
			progressVM.CategoryColor = category.Color
		}
	}

	data := map[string]any{
		"Progress": progressVM,
	}

	return h.renderPartial(c, "components/budget_progress", data)
}

// Show отображает детальную информацию о бюджете
func (h *BudgetHandler) Show(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID бюджета
	id := c.Param("id")
	budgetID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Получаем бюджет
	budgetEntity, err := h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// Проверяем права доступа
	// Single family model - no family check needed

	// Конвертируем в view модель
	budgetVM := webModels.BudgetProgressVM{}
	budgetVM.FromDomain(budgetEntity)

	// Добавляем информацию о категории если есть
	if budgetEntity.CategoryID != nil {
		category, catErr := h.services.Category.GetCategoryByID(c.Request().Context(), *budgetEntity.CategoryID)
		if catErr == nil {
			budgetVM.CategoryName = category.Name
			budgetVM.CategoryColor = category.Color
		}
	}

	// Получаем данные о тратах для анализа
	var spendingData *webModels.SpendingAnalysis
	if budgetEntity.IsActive && budgetEntity.Spent > 0 {
		dailyAvg := budgetEntity.Spent / float64(budgetVM.DaysElapsed)
		budgetPace := budgetEntity.Amount / float64(budgetVM.DaysTotal)
		spendingData = &webModels.SpendingAnalysis{
			DailyAverage:   dailyAvg,
			BudgetPace:     budgetPace,
			ProjectedTotal: dailyAvg * float64(budgetVM.DaysTotal),
			DaysElapsed:    budgetVM.DaysElapsed,
			Variance:       dailyAvg - budgetPace,
		}
	}

	// Получаем последние транзакции связанные с бюджетом
	recentTransactions, err := h.getRecentTransactionsForBudget(
		c.Request().Context(),
		budgetEntity,
	)
	if err != nil {
		// В случае ошибки получения транзакций, продолжаем без них
		c.Logger().Warnf("Failed to get recent transactions for budget %s: %v", budgetEntity.ID, err)
		recentTransactions = []*webModels.TransactionSummary{}
	}

	pageData := &PageData{
		Title: "Budget: " + budgetEntity.Name,
	}

	data := map[string]any{
		"PageData":           pageData,
		"Budget":             budgetVM,
		"SpendingData":       spendingData,
		"RecentTransactions": recentTransactions,
	}

	return h.renderPage(c, "pages/budgets/show", data)
}

// renderBudgetFormWithErrors отображает форму с ошибками
func (h *BudgetHandler) renderBudgetFormWithErrors(
	c echo.Context,
	form webModels.BudgetForm,
	title string,
) error {
	// Получаем данные пользователя из сессии для категорий
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем список категорий
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		nil,
	)
	if err != nil {
		categories = []*category.Category{} // Пустой список при ошибке
	}

	// Подготавливаем опции категорий
	categoryOptions := make([]map[string]any, len(categories)+1)
	categoryOptions[0] = map[string]any{
		"ID":   "",
		"Name": "All Categories",
	}
	for i, cat := range categories {
		categoryOptions[i+1] = map[string]any{
			"ID":   cat.ID.String(),
			"Name": cat.Name,
		}
	}

	pageData := &PageData{
		Title: title,
		Messages: []Message{
			{Type: "error", Text: "Проверьте правильность заполнения формы"},
		},
	}

	data := map[string]any{
		"PageData":        pageData,
		"Form":            form,
		"CategoryOptions": categoryOptions,
	}

	template := "pages/budgets/new"
	if title == "Edit Budget" {
		template = "pages/budgets/edit"
	}

	return h.renderPage(c, template, data)
}

// handleBudgetActivation общий метод для изменения статуса бюджета
func (h *BudgetHandler) handleBudgetActivation(c echo.Context, isActive bool) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID бюджета
	id := c.Param("id")
	budgetID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Получаем бюджет для проверки прав доступа
	_, err = h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// Проверяем права доступа
	// Single family model - no family check needed

	// Создаем DTO для обновления
	updateDTO := &dto.UpdateBudgetDTO{
		IsActive: &isActive,
	}

	// Обновляем бюджет
	_, err = h.services.Budget.UpdateBudget(c.Request().Context(), budgetID, *updateDTO)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, h.getBudgetServiceErrorMessage(err))
	}

	// Для HTMX запросов возвращаем обновленную страницу
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return h.Show(c)
	}

	// Обычный редирект
	return c.Redirect(http.StatusFound, fmt.Sprintf("/budgets/%s", budgetID))
}

// Activate активирует бюджет
func (h *BudgetHandler) Activate(c echo.Context) error {
	return h.handleBudgetActivation(c, true)
}

// Deactivate деактивирует бюджет
func (h *BudgetHandler) Deactivate(c echo.Context) error {
	return h.handleBudgetActivation(c, false)
}

// Alerts отображает страницу с алертами для бюджетов
func (h *BudgetHandler) Alerts(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем все активные бюджеты семьи
	filter := dto.NewBudgetFilterDTO()
	isActive := true
	filter.IsActive = &isActive

	budgets, err := h.services.Budget.GetAllBudgets(c.Request().Context(), filter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load budgets")
	}

	// Создаем алерты для каждого бюджета
	var alerts []*webModels.BudgetAlertVM
	triggeredCount := 0
	for _, budgetEntity := range budgets {
		// Создаем view модель бюджета
		budgetVM := webModels.BudgetProgressVM{}
		budgetVM.FromDomain(budgetEntity)

		// Создаем алерт на основе текущего состояния бюджета
		alert := &webModels.BudgetAlertVM{
			ID:         uuid.New(), // В реальном приложении это был бы ID из базы
			BudgetID:   budgetEntity.ID,
			BudgetName: budgetEntity.Name,
		}

		// Определяем тип алерта на основе процента использования
		percentage := budgetEntity.GetSpentPercentage()
		switch {
		case percentage >= BudgetExceededThreshold:
			alert.Threshold = BudgetExceededThreshold
			alert.IsTriggered = true
			alert.Message = fmt.Sprintf("Budget exceeded! You've spent %.1f%% of your allocated amount.", percentage)
			alert.AlertClass = "danger"
			triggeredCount++
		case percentage >= BudgetCriticalThreshold:
			alert.Threshold = BudgetCriticalThreshold
			alert.IsTriggered = true
			alert.Message = fmt.Sprintf("Critical alert: You've reached %.1f%% of your budget.", percentage)
			alert.AlertClass = "danger"
			triggeredCount++
		case percentage >= BudgetWarningThreshold:
			alert.Threshold = BudgetWarningThreshold
			alert.IsTriggered = true
			alert.Message = fmt.Sprintf("Warning: You've used %.1f%% of your budget.", percentage)
			alert.AlertClass = "warning"
			triggeredCount++
		default:
			alert.Threshold = DefaultAlertThreshold
			alert.IsTriggered = false
			alert.Message = fmt.Sprintf("Budget is healthy at %.1f%% usage.", percentage)
			alert.AlertClass = "info"
		}

		alerts = append(alerts, alert)
	}

	totalCount := len(alerts)
	healthyCount := totalCount - triggeredCount

	// Получаем CSRF токен
	csrfToken, _ := middleware.GetCSRFToken(c)

	pageData := &PageData{
		Title: "Budget Alerts",
	}

	data := map[string]any{
		"PageData":       pageData,
		"Alerts":         alerts,
		"TotalCount":     totalCount,
		"TriggeredCount": triggeredCount,
		"HealthyCount":   healthyCount,
		"CSRFToken":      csrfToken,
	}

	return h.renderPage(c, "pages/budgets/alerts", data)
}

// CreateAlert создает новый алерт для бюджета
func (h *BudgetHandler) CreateAlert(c echo.Context) error {
	// Получаем данные пользователя из сессии
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим форму
	var form webModels.BudgetAlertForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Form validation failed")
	}

	// Парсим ID бюджета
	budgetID, err := form.GetBudgetID()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid budget ID")
	}

	// Проверяем, что бюджет принадлежит семье пользователя
	_, err = h.services.Budget.GetBudgetByID(c.Request().Context(), budgetID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Budget not found")
	}

	// Single family model - no family check needed

	// В реальном приложении здесь был бы вызов сервиса для создания алерта
	// Сейчас просто возвращаем успех

	// Для HTMX запросов перенаправляем на страницу алертов
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		c.Response().Header().Set("Hx-Redirect", "/budgets/alerts")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/budgets/alerts")
}

// DeleteAlert удаляет алерт
func (h *BudgetHandler) DeleteAlert(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID алерта
	id := c.Param("alert_id")
	alertID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid alert ID")
	}

	// В реальном приложении здесь был бы вызов сервиса для удаления алерта
	// с проверкой прав доступа
	_ = alertID
	_ = sessionData

	// Для HTMX запросов возвращаем пустой ответ для удаления элемента
	if c.Request().Header.Get("Hx-Request") == HTMXRequestHeader {
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusFound, "/budgets/alerts")
}

// getBudgetServiceErrorMessage возвращает пользовательское сообщение об ошибке
func (h *BudgetHandler) getBudgetServiceErrorMessage(err error) string {
	errMsg := err.Error()
	switch errMsg {
	case "budget not found":
		return fmt.Sprintf("Budget not found: %s", errMsg)
	case "invalid budget period":
		return fmt.Sprintf("Invalid budget period - end date must be after start date: %s", errMsg)
	case "budget period overlap":
		return fmt.Sprintf("Budget period overlaps with existing budget for this category: %s", errMsg)
	case "budget already exceeded":
		return fmt.Sprintf("Budget amount is less than already spent amount: %s", errMsg)
	case "invalid budget amount":
		return fmt.Sprintf("Budget amount must be greater than 0: %s", errMsg)
	default:
		return fmt.Sprintf("Failed to process budget: %s", errMsg)
	}
}

// getRecentTransactionsForBudget получает последние транзакции связанные с бюджетом
func (h *BudgetHandler) getRecentTransactionsForBudget(
	ctx context.Context,
	budget *budget.Budget,
) ([]*webModels.TransactionSummary, error) {
	// Создаем фильтр для получения транзакций
	filter := dto.NewTransactionFilterDTO()
	filter.DateFrom = &budget.StartDate
	filter.DateTo = &budget.EndDate
	filter.Limit = 5 // Ограничиваем количество последних транзакций

	// Если у бюджета есть категория, фильтруем по ней
	if budget.CategoryID != nil {
		filter.CategoryID = budget.CategoryID
	}

	// Получаем единственную семью в single-family модели
	family, err := h.services.Family.GetFamily(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family: %w", err)
	}

	// Получаем транзакции через сервис
	transactions, err := h.services.Transaction.GetTransactionsByDateRange(ctx, family.ID, *filter.DateFrom, *filter.DateTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for budget: %w", err)
	}

	// Фильтруем по категории если нужно (сервис не поддерживает фильтрацию по категории в GetTransactionsByDateRange)
	if filter.CategoryID != nil {
		filteredTransactions := make([]*transaction.Transaction, 0)
		for _, tx := range transactions {
			if tx.CategoryID == *filter.CategoryID {
				filteredTransactions = append(filteredTransactions, tx)
			}
		}
		transactions = filteredTransactions
	}

	// Конвертируем в web модели
	result := make([]*webModels.TransactionSummary, 0, len(transactions))
	for _, tx := range transactions {
		summary := h.convertTransactionToSummary(ctx, tx)
		result = append(result, summary)
	}

	return result, nil
}

// convertTransactionToSummary конвертирует доменную модель транзакции в TransactionSummary
func (h *BudgetHandler) convertTransactionToSummary(
	ctx context.Context,
	tx *transaction.Transaction,
) *webModels.TransactionSummary {
	summary := &webModels.TransactionSummary{
		Description: tx.Description,
		Amount:      tx.Amount,
		Type:        string(tx.Type),
		Date:        tx.Date,
	}

	// Форматируем сумму
	summary.FormattedAmount = fmt.Sprintf("%.2f", tx.Amount)

	// Получаем название категории если есть
	if tx.CategoryID != uuid.Nil {
		category, err := h.services.Category.GetCategoryByID(ctx, tx.CategoryID)
		if err != nil {
			// Если не удалось получить категорию, используем ID как название
			summary.CategoryName = tx.CategoryID.String()
		} else {
			summary.CategoryName = category.Name
		}
	} else {
		summary.CategoryName = "Uncategorized"
	}

	return summary
}

package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

const (
	CategoryTypeIncome  = "income"
	CategoryTypeExpense = "expense"

	// TransactionLimitForStats ограничивает количество транзакций для расчета статистики
	TransactionLimitForStats = 1000

	// BudgetPercentageOverspent порог для превышения бюджета (100%)
	BudgetPercentageOverspent = 100

	// BudgetPercentageWarning порог для предупреждения о приближении к лимиту (80%)
	BudgetPercentageWarning = 80

	// BudgetPercentageToMultiplier множитель для преобразования в проценты
	BudgetPercentageToMultiplier = 100
)

// CategoryHandler обрабатывает HTTP запросы для категорий
type CategoryHandler struct {
	*BaseHandler

	validator *validator.Validate
}

// NewCategoryHandler создает новый обработчик категорий
func NewCategoryHandler(repositories *handlers.Repositories, services *services.Services) *CategoryHandler {
	return &CategoryHandler{
		BaseHandler: NewBaseHandler(repositories, services),
		validator:   validator.New(),
	}
}

// Index отображает список всех категорий
func (h *CategoryHandler) Index(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим фильтры из query parameters
	var filters webModels.CategoryFilter
	if bindErr := c.Bind(&filters); bindErr != nil {
		// Игнорируем ошибки привязки для фильтров - просто используем пустые фильтры
		filters = webModels.CategoryFilter{}
	}

	// Получаем категории через сервис
	typeFilter := parseTypeFilter(filters.Type)
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		typeFilter,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Конвертируем и фильтруем категории
	categoryViewModels := convertToViewModels(
		c.Request().Context(),
		categories,
		h.services.Transaction,
		h.services.Budget,
	)
	categoryViewModels = applyNameFilter(categoryViewModels, filters.Name)
	categoryViewModels = applyParentOnlyFilter(categoryViewModels, filters.ParentOnly)

	// Строим дерево категорий
	categoryTree := webModels.BuildCategoryTree(categoryViewModels)

	pageData := &PageData{
		Title: "Categories",
	}

	data := map[string]any{
		"PageData":   pageData,
		"Categories": categoryTree,
		"Filters":    filters,
	}

	return h.renderPage(c, "pages/categories/index", data)
}

// New отображает форму создания новой категории
func (h *CategoryHandler) New(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем родительские категории для select
	parentCategories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get parent categories")
	}

	// Фильтруем только родительские категории (без ParentID)
	var parentOptions []webModels.CategorySelectOption
	for _, cat := range parentCategories {
		if cat.ParentID == nil { // Только родительские категории могут быть родителями
			option := webModels.CategorySelectOption{
				ID:   cat.ID,
				Name: cat.Name,
				Type: string(cat.Type),
			}
			parentOptions = append(parentOptions, option)
		}
	}

	// Получаем CSRF токен
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get CSRF token")
	}

	pageData := &PageData{
		Title: "New Category",
	}

	data := map[string]any{
		"PageData":      pageData,
		"CSRFToken":     csrfToken,
		"ParentOptions": parentOptions,
		"DefaultColors": getDefaultCategoryColors(),
		"DefaultIcons":  getDefaultCategoryIcons(),
	}

	return h.renderPage(c, "pages/categories/new", data)
}

// Create создает новую категорию
func (h *CategoryHandler) Create(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим данные формы
	var form webModels.CategoryForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		// Возвращаем форму с ошибками валидации
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			// Для HTMX возвращаем только errors partial
			return h.renderPartial(c, "components/form_errors", map[string]any{
				"Errors": validationErrors,
			})
		}

		// Для обычных запросов возвращаем форму заново
		pageData := &PageData{
			Title:  "New Category",
			Errors: validationErrors,
			Messages: []Message{
				{Type: "error", Text: "Проверьте правильность заполнения формы"},
			},
		}

		parentCategories, _ := h.services.Category.GetCategories(
			c.Request().Context(),
			nil,
		)
		var parentOptions []webModels.CategorySelectOption
		for _, cat := range parentCategories {
			if cat.ParentID == nil {
				option := webModels.CategorySelectOption{
					ID:   cat.ID,
					Name: cat.Name,
					Type: string(cat.Type),
				}
				parentOptions = append(parentOptions, option)
			}
		}

		csrfToken, _ := middleware.GetCSRFToken(c)

		data := map[string]any{
			"PageData":      pageData,
			"Form":          form,
			"CSRFToken":     csrfToken,
			"ParentOptions": parentOptions,
			"DefaultColors": getDefaultCategoryColors(),
			"DefaultIcons":  getDefaultCategoryIcons(),
		}

		return h.renderPage(c, "pages/categories/new", data)
	}

	// Создаем DTO для сервиса
	createDTO := dto.CreateCategoryDTO{
		Name:     form.Name,
		Type:     form.ToCategoryType(),
		Color:    form.Color,
		Icon:     form.Icon,
		ParentID: form.GetParentID(),
	}

	// Создаем категорию через сервис
	_, err = h.services.Category.CreateCategory(c.Request().Context(), createDTO)
	if err != nil {
		// Обрабатываем специфичные ошибки сервиса
		var errorMsg string
		switch {
		case strings.Contains(err.Error(), "category with this name already exists"):
			errorMsg = "Category with this name already exists"
		case strings.Contains(err.Error(), "parent category not found"):
			errorMsg = "Selected parent category not found"
		case strings.Contains(err.Error(), "parent category must be of the same type"):
			errorMsg = "Parent category must be of the same type (income/expense)"
		case strings.Contains(err.Error(), "cannot create more than 2 levels"):
			errorMsg = "Cannot create more than 2 levels of category hierarchy"
		default:
			errorMsg = "Failed to create category"
		}

		if h.IsHTMXRequest(c) {
			return h.renderPartial(c, "components/form_errors", map[string]any{
				"Errors": map[string]string{"form": errorMsg},
			})
		}

		return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
	}

	// Успешное создание - редирект
	return h.redirect(c, "/categories")
}

// Edit отображает форму редактирования категории
func (h *CategoryHandler) Edit(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid category ID")
	}

	// Получаем категорию
	category, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Category not found")
	}

	// In single-family model, all categories belong to the family
	// No additional access check needed

	// Получаем родительские категории для select (исключая текущую)
	parentCategories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get parent categories")
	}

	// Фильтруем категории: только родительские + не текущая + того же типа
	var parentOptions []webModels.CategorySelectOption
	for _, cat := range parentCategories {
		if cat.ID != categoryID && cat.ParentID == nil && cat.Type == category.Type {
			option := webModels.CategorySelectOption{
				ID:   cat.ID,
				Name: cat.Name,
				Type: string(cat.Type),
			}
			parentOptions = append(parentOptions, option)
		}
	}

	// Создаем форму из данных категории
	form := webModels.CategoryForm{
		Name:     category.Name,
		Type:     string(category.Type),
		Color:    category.Color,
		Icon:     category.Icon,
		ParentID: "",
	}

	if category.ParentID != nil {
		form.ParentID = category.ParentID.String()
	}

	// Получаем CSRF токен
	csrfToken, err := middleware.GetCSRFToken(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get CSRF token")
	}

	pageData := &PageData{
		Title: "Edit Category",
	}

	data := map[string]any{
		"PageData":      pageData,
		"CSRFToken":     csrfToken,
		"Form":          form,
		"Category":      category,
		"ParentOptions": parentOptions,
		"DefaultColors": getDefaultCategoryColors(),
		"DefaultIcons":  getDefaultCategoryIcons(),
	}

	return h.renderPage(c, "pages/categories/edit", data)
}

// Update обновляет существующую категорию
func (h *CategoryHandler) Update(c echo.Context) error {
	_, categoryID, existingCategory, err := h.validateUpdateRequest(c)
	if err != nil {
		return err
	}

	// Парсим данные формы
	var form webModels.CategoryForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		return h.handleUpdateValidationError(c, validationErr, form, categoryID, existingCategory)
	}

	// Создаем DTO для обновления
	updateDTO := dto.UpdateCategoryDTO{
		Name:  &form.Name,
		Color: &form.Color,
		Icon:  &form.Icon,
	}

	// Обновляем категорию через сервис
	_, err = h.services.Category.UpdateCategory(c.Request().Context(), categoryID, updateDTO)
	if err != nil {
		return h.handleUpdateServiceError(c, err)
	}

	// Успешное обновление - редирект
	return h.redirect(c, "/categories")
}

// validateUpdateRequest проверяет права доступа и возвращает необходимые данные
func (h *CategoryHandler) validateUpdateRequest(c echo.Context) (
	*middleware.SessionData,
	uuid.UUID,
	*category.Category,
	error,
) {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return nil, uuid.Nil, nil, echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return nil, uuid.Nil, nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid category ID")
	}

	// Проверяем, что категория существует
	existingCategory, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return nil, uuid.Nil, nil, echo.NewHTTPError(http.StatusNotFound, "Category not found")
	}

	// In single-family model, all categories belong to the family
	// No additional access check needed

	return sessionData, categoryID, existingCategory, nil
}

// handleUpdateValidationError обрабатывает ошибки валидации при обновлении
func (h *CategoryHandler) handleUpdateValidationError(
	c echo.Context,
	validationErr error,
	form webModels.CategoryForm,
	categoryID uuid.UUID,
	existingCategory *category.Category,
) error {
	validationErrors := webModels.GetValidationErrors(validationErr)

	if h.IsHTMXRequest(c) {
		return h.renderPartial(c, "components/form_errors", map[string]any{
			"Errors": validationErrors,
		})
	}

	// Для обычных запросов возвращаем форму заново
	pageData := &PageData{
		Title:  "Edit Category",
		Errors: validationErrors,
		Messages: []Message{
			{Type: "error", Text: "Проверьте правильность заполнения формы"},
		},
	}

	parentOptions := h.getParentOptionsForUpdate(
		c.Request().Context(),
		categoryID,
		existingCategory,
	)
	csrfToken, _ := middleware.GetCSRFToken(c)

	data := map[string]any{
		"PageData":      pageData,
		"Form":          form,
		"CSRFToken":     csrfToken,
		"Category":      existingCategory,
		"ParentOptions": parentOptions,
		"DefaultColors": getDefaultCategoryColors(),
		"DefaultIcons":  getDefaultCategoryIcons(),
	}

	return h.renderPage(c, "pages/categories/edit", data)
}

// getParentOptionsForUpdate получает список родительских категорий для формы редактирования
func (h *CategoryHandler) getParentOptionsForUpdate(
	ctx context.Context,
	categoryID uuid.UUID,
	existingCategory *category.Category,
) []webModels.CategorySelectOption {
	parentCategories, _ := h.services.Category.GetCategories(ctx, nil)
	var parentOptions []webModels.CategorySelectOption
	for _, cat := range parentCategories {
		if cat.ID != categoryID && cat.ParentID == nil && cat.Type == existingCategory.Type {
			option := webModels.CategorySelectOption{
				ID:   cat.ID,
				Name: cat.Name,
				Type: string(cat.Type),
			}
			parentOptions = append(parentOptions, option)
		}
	}
	return parentOptions
}

// handleUpdateServiceError обрабатывает ошибки сервиса при обновлении
func (h *CategoryHandler) handleUpdateServiceError(c echo.Context, err error) error {
	var errorMsg string
	switch {
	case strings.Contains(err.Error(), "category with this name already exists"):
		errorMsg = "Category with this name already exists"
	case strings.Contains(err.Error(), "category not found"):
		errorMsg = "Category not found"
	default:
		errorMsg = "Failed to update category"
	}

	if h.IsHTMXRequest(c) {
		return h.renderPartial(c, "components/form_errors", map[string]any{
			"Errors": map[string]string{"form": errorMsg},
		})
	}

	return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
}

// Show отображает детали конкретной категории
func (h *CategoryHandler) Show(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid category ID")
	}

	// Получаем категорию
	category, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Category not found")
	}

	// In single-family model, all categories belong to the family
	// No additional access check needed

	// Получаем подкатегории
	allCategories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Создаем view model для категории
	var categoryVM webModels.CategoryViewModel
	categoryVM.FromDomain(category)

	// Если это подкатегория, найдем родительскую
	if category.ParentID != nil {
		for _, parent := range allCategories {
			if parent.ID == *category.ParentID {
				categoryVM.ParentName = parent.Name
				break
			}
		}
	}

	// Заполняем статистику транзакций для основной категории
	_ = populateCategoryStats(
		c.Request().Context(),
		&categoryVM,
		h.services.Transaction,
		h.services.Budget,
	)

	// Найдем подкатегории для текущей категории и создадим view models
	var subcategoryVMs []webModels.CategoryViewModel
	allCategoryVMs := convertToViewModels(
		c.Request().Context(),
		allCategories,
		h.services.Transaction,
		h.services.Budget,
	)
	for _, vm := range allCategoryVMs {
		if vm.ParentID != nil && *vm.ParentID == categoryID {
			subcategoryVMs = append(subcategoryVMs, vm)
		}
	}

	// Получаем последние транзакции для этой категории (если есть Transaction сервис)
	var recentTransactions []any // TODO: заменить на Transaction модель когда будет доступна

	data := map[string]any{
		"Category":      categoryVM,
		"Subcategories": subcategoryVMs,
		"Transactions":  recentTransactions,
	}

	return h.renderPage(c, "pages/categories/show", data)
}

// Delete удаляет категорию
func (h *CategoryHandler) Delete(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid category ID")
	}

	// In single-family model, no need to check category ownership
	// All categories belong to the single family

	// Удаляем категорию через сервис
	err = h.services.Category.DeleteCategory(c.Request().Context(), categoryID)
	if err != nil {
		// Обрабатываем специфичные ошибки сервиса
		var errorMsg string
		switch {
		case strings.Contains(err.Error(), "category is used in transactions"):
			errorMsg = "Cannot delete category that is used in transactions"
		case strings.Contains(err.Error(), "category has subcategories"):
			errorMsg = "Cannot delete category that has subcategories"
		case strings.Contains(err.Error(), "category not found"):
			errorMsg = "Category not found"
		default:
			errorMsg = "Failed to delete category"
		}

		if h.IsHTMXRequest(c) {
			return h.renderPartial(c, "components/alert", map[string]any{
				"Type":    "error",
				"Message": errorMsg,
			})
		}

		return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
	}

	if h.IsHTMXRequest(c) {
		// Для HTMX возвращаем пустой ответ для удаления строки
		return c.NoContent(http.StatusOK)
	}

	return h.redirect(c, "/categories")
}

// Search выполняет поиск категорий (HTMX)
func (h *CategoryHandler) Search(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем параметры поиска
	query := strings.TrimSpace(c.QueryParam("q"))
	categoryType := c.QueryParam("type")
	parentOnly := c.QueryParam("parent_only") == "true"

	// Получаем все категории семьи
	typeFilter := parseTypeFilter(categoryType)
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		typeFilter,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Конвертируем и фильтруем категории
	categoryViewModels := convertToViewModels(
		c.Request().Context(),
		categories,
		h.services.Transaction,
		h.services.Budget,
	)
	categoryViewModels = applyNameFilter(categoryViewModels, query)
	categoryViewModels = applyParentOnlyFilter(categoryViewModels, parentOnly)

	// Строим дерево если не только родительские
	if parentOnly {
		data := map[string]any{
			"Categories": categoryViewModels,
		}
		return h.renderPartial(c, "components/category_list", data)
	}

	categoryTree := webModels.BuildCategoryTree(categoryViewModels)
	data := map[string]any{
		"Categories": categoryTree,
	}

	return h.renderPartial(c, "components/category_list", data)
}

// Select возвращает категории для select элементов (HTMX)
func (h *CategoryHandler) Select(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем параметры
	categoryType := c.QueryParam("type") // income или expense
	parentOnly := c.QueryParam("parent_only") == "true"
	excludeID := c.QueryParam("exclude_id") // Исключить категорию по ID (для редактирования)

	// Парсим exclude ID если есть
	var excludeUUID *uuid.UUID
	if excludeID != "" {
		if parsedUUID, parseErr := uuid.Parse(excludeID); parseErr == nil {
			excludeUUID = &parsedUUID
		}
	}

	// Определяем фильтр по типу
	typeFilter := parseTypeFilter(categoryType)

	// Получаем категории
	categories, err := h.services.Category.GetCategories(
		c.Request().Context(),
		typeFilter,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Конвертируем в select опции
	options := buildSelectOptions(categories, parentOnly, excludeUUID)

	data := map[string]any{
		"Options": options,
	}

	return h.renderPartial(c, "components/category_select", data)
}

// Helper functions

// parseTypeFilter парсит строку типа категории в domain фильтр
func parseTypeFilter(typeString string) *category.Type {
	if typeString == "" {
		return nil
	}

	switch typeString {
	case CategoryTypeIncome:
		t := category.TypeIncome
		return &t
	case CategoryTypeExpense:
		t := category.TypeExpense
		return &t
	default:
		return nil
	}
}

// convertToViewModels конвертирует domain модели в view модели с родительскими именами и статистикой
func convertToViewModels(
	ctx context.Context,
	categories []*category.Category,
	transactionService services.TransactionService,
	budgetService services.BudgetService,
) []webModels.CategoryViewModel {
	var categoryViewModels []webModels.CategoryViewModel
	for _, cat := range categories {
		var vm webModels.CategoryViewModel
		vm.FromDomain(cat)

		// Дополнительная информация для отображения
		if cat.ParentID != nil {
			// Найти родительскую категорию для отображения имени
			for _, parent := range categories {
				if parent.ID == *cat.ParentID {
					vm.ParentName = parent.Name
					// Создаем ParentCategory для шаблонов
					var parentVM webModels.CategoryViewModel
					parentVM.FromDomain(parent)
					vm.ParentCategory = &parentVM
					break
				}
			}
		}

		// Получаем статистику транзакций для категории
		_ = populateCategoryStats(ctx, &vm, transactionService, budgetService)

		categoryViewModels = append(categoryViewModels, vm)
	}
	return categoryViewModels
}

// populateCategoryStats заполняет статистику транзакций для категории
func populateCategoryStats(
	ctx context.Context,
	vm *webModels.CategoryViewModel,
	transactionService services.TransactionService,
	budgetService services.BudgetService,
) error {
	// Получаем все транзакции для категории
	filter := dto.TransactionFilterDTO{
		CategoryID: &vm.ID,
		Limit:      TransactionLimitForStats, // Ограничиваем для производительности
		Offset:     0,
	}

	transactions, err := transactionService.GetTransactionsByCategory(ctx, vm.ID, filter)
	if err != nil {
		return err
	}

	// Рассчитываем статистику
	vm.TransactionCount = len(transactions)
	var totalAmount float64
	var currentMonthAmount float64
	var lastUsed *time.Time

	// Получаем текущий месяц и год
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()

	for _, tx := range transactions {
		totalAmount += tx.Amount

		// Рассчитываем сумму за текущий месяц
		txYear, txMonth, _ := tx.Date.Date()
		if txYear == currentYear && txMonth == currentMonth {
			currentMonthAmount += tx.Amount
		}

		// Находим самую позднюю дату
		if lastUsed == nil || tx.Date.After(*lastUsed) {
			lastUsed = &tx.Date
		}
	}

	vm.TotalAmount = totalAmount
	if vm.TransactionCount > 0 {
		vm.AverageAmount = totalAmount / float64(vm.TransactionCount)
	} else {
		vm.AverageAmount = 0
	}
	vm.CurrentMonthAmount = currentMonthAmount
	vm.LastUsed = lastUsed

	// Получаем активные бюджеты для данной категории
	budgets, err := budgetService.GetBudgetsByCategory(ctx, vm.ID)
	if err == nil && len(budgets) > 0 {
		// Берем первый активный бюджет (можно улучшить логику выбора)
		for _, budget := range budgets {
			if budget.IsActive {
				vm.BudgetLimit = &budget.Amount
				break
			}
		}
	}

	// Рассчитываем прогресс бюджета
	calculateBudgetProgress(vm)

	return nil
}

// calculateBudgetProgress рассчитывает прогресс бюджета и устанавливает соответствующие поля
func calculateBudgetProgress(vm *webModels.CategoryViewModel) {
	if vm.BudgetLimit != nil && *vm.BudgetLimit > 0 {
		vm.BudgetPercentage = (vm.CurrentMonthAmount / *vm.BudgetLimit) * BudgetPercentageToMultiplier

		switch {
		case vm.BudgetPercentage >= BudgetPercentageOverspent:
			vm.BudgetProgressClass = "progress-danger"
			vm.BudgetOverspent = vm.CurrentMonthAmount - *vm.BudgetLimit
			vm.BudgetRemaining = 0
		case vm.BudgetPercentage >= BudgetPercentageWarning:
			vm.BudgetProgressClass = "progress-warning"
			vm.BudgetRemaining = *vm.BudgetLimit - vm.CurrentMonthAmount
			vm.BudgetOverspent = 0
		default:
			vm.BudgetProgressClass = "progress-success"
			vm.BudgetRemaining = *vm.BudgetLimit - vm.CurrentMonthAmount
			vm.BudgetOverspent = 0
		}
	} else {
		vm.BudgetPercentage = 0
		vm.BudgetProgressClass = "progress-success"
		vm.BudgetRemaining = 0
		vm.BudgetOverspent = 0
	}
}

// applyNameFilter фильтрует категории по имени
func applyNameFilter(viewModels []webModels.CategoryViewModel, searchName string) []webModels.CategoryViewModel {
	if searchName == "" {
		return viewModels
	}

	var filtered []webModels.CategoryViewModel
	searchTerm := strings.ToLower(searchName)
	for _, vm := range viewModels {
		if strings.Contains(strings.ToLower(vm.Name), searchTerm) {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// applyParentOnlyFilter фильтрует только родительские категории
func applyParentOnlyFilter(viewModels []webModels.CategoryViewModel, parentOnly bool) []webModels.CategoryViewModel {
	if !parentOnly {
		return viewModels
	}

	var filtered []webModels.CategoryViewModel
	for _, vm := range viewModels {
		if vm.ParentID == nil {
			filtered = append(filtered, vm)
		}
	}
	return filtered
}

// buildSelectOptions конвертирует категории в опции для select элементов
func buildSelectOptions(
	categories []*category.Category,
	parentOnly bool,
	excludeUUID *uuid.UUID,
) []webModels.CategorySelectOption {
	var options []webModels.CategorySelectOption
	for _, cat := range categories {
		// Пропускаем исключаемую категорию
		if excludeUUID != nil && cat.ID == *excludeUUID {
			continue
		}

		// Фильтруем только родительские если требуется
		if parentOnly && cat.ParentID != nil {
			continue
		}

		option := webModels.CategorySelectOption{
			ID:   cat.ID,
			Name: cat.Name,
			Type: string(cat.Type),
		}

		// Добавляем индикацию подкатегории
		if cat.ParentID != nil {
			// Находим родительскую категорию
			for _, parent := range categories {
				if parent.ID == *cat.ParentID {
					option.Name = parent.Name + " > " + cat.Name
					break
				}
			}
		}

		options = append(options, option)
	}
	return options
}

// getDefaultCategoryColors возвращает набор предустановленных цветов для категорий
func getDefaultCategoryColors() []string {
	return []string{
		"#007BFF", "#28A745", "#DC3545", "#FFC107", "#17A2B8",
		"#6F42C1", "#E83E8C", "#FD7E14", "#20C997", "#6C757D",
		"#343A40", "#F8F9FA", "#FF6B6B", "#4ECDC4", "#45B7D1",
		"#96CEB4", "#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
	}
}

// getDefaultCategoryIcons возвращает набор предустановленных иконок для категорий
func getDefaultCategoryIcons() []string {
	return []string{
		"home", "food", "transport", "shopping", "health", "education",
		"entertainment", "sports", "travel", "bills", "salary", "gift",
		"investment", "freelance", "business", "other", "restaurant",
		"gas", "clothes", "beauty", "pets", "charity", "insurance",
	}
}

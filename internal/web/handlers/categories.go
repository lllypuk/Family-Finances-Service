package handlers

import (
	"net/http"
	"strings"

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
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим фильтры из query parameters
	var filters webModels.CategoryFilter
	if bindErr := c.Bind(&filters); bindErr != nil {
		// Игнорируем ошибки привязки для фильтров - просто используем пустые фильтры
		filters = webModels.CategoryFilter{}
	}

	// Получаем категории через сервис
	typeFilter := parseTypeFilter(filters.Type)
	categories, err := h.services.Category.GetCategoriesByFamily(
		c.Request().Context(),
		sessionData.FamilyID,
		typeFilter,
	)
	if err != nil {
		return h.handleError(c, err, "Failed to get categories")
	}

	// Конвертируем и фильтруем категории
	categoryViewModels := convertToViewModels(categories)
	categoryViewModels = applyNameFilter(categoryViewModels, filters.Name)
	categoryViewModels = applyParentOnlyFilter(categoryViewModels, filters.ParentOnly)

	// Строим дерево категорий
	categoryTree := webModels.BuildCategoryTree(categoryViewModels)

	pageData := &PageData{
		Title: "Categories",
	}

	data := map[string]interface{}{
		"PageData":   pageData,
		"Categories": categoryTree,
		"Filters":    filters,
	}

	return h.renderPage(c, "pages/categories/index", data)
}

// New отображает форму создания новой категории
func (h *CategoryHandler) New(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Получаем родительские категории для select
	parentCategories, err := h.services.Category.GetCategoriesByFamily(c.Request().Context(), sessionData.FamilyID, nil)
	if err != nil {
		return h.handleError(c, err, "Failed to get parent categories")
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
		return h.handleError(c, err, "Failed to get CSRF token")
	}

	pageData := &PageData{
		Title: "New Category",
	}

	data := map[string]interface{}{
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
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим данные формы
	var form webModels.CategoryForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return h.handleError(c, bindErr, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		// Возвращаем форму с ошибками валидации
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.isHTMXRequest(c) {
			// Для HTMX возвращаем только errors partial
			return h.renderPartial(c, "components/form_errors", map[string]interface{}{
				"Errors": validationErrors,
			})
		}

		// Для обычных запросов возвращаем форму заново
		pageData := &PageData{
			Title: "New Category",
			Messages: []Message{
				{Type: "error", Text: "Проверьте правильность заполнения формы"},
			},
		}

		parentCategories, _ := h.services.Category.GetCategoriesByFamily(
			c.Request().Context(),
			sessionData.FamilyID,
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

		data := map[string]interface{}{
			"PageData":      pageData,
			"Form":          form,
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
		FamilyID: sessionData.FamilyID,
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

		if h.isHTMXRequest(c) {
			return h.renderPartial(c, "components/form_errors", map[string]interface{}{
				"Errors": map[string]string{"form": errorMsg},
			})
		}

		return h.handleError(c, err, errorMsg)
	}

	// Успешное создание - редирект
	return h.redirect(c, "/categories")
}

// Edit отображает форму редактирования категории
func (h *CategoryHandler) Edit(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return h.handleError(c, err, "Invalid category ID")
	}

	// Получаем категорию
	category, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return h.handleError(c, err, "Category not found")
	}

	// Проверяем, что категория принадлежит семье пользователя
	if category.FamilyID != sessionData.FamilyID {
		return h.handleError(c, echo.ErrForbidden, "Access denied")
	}

	// Получаем родительские категории для select (исключая текущую)
	parentCategories, err := h.services.Category.GetCategoriesByFamily(c.Request().Context(), sessionData.FamilyID, nil)
	if err != nil {
		return h.handleError(c, err, "Failed to get parent categories")
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
		return h.handleError(c, err, "Failed to get CSRF token")
	}

	pageData := &PageData{
		Title: "Edit Category",
	}

	data := map[string]interface{}{
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
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return h.handleError(c, err, "Invalid category ID")
	}

	// Проверяем, что категория существует и принадлежит семье
	existingCategory, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return h.handleError(c, err, "Category not found")
	}

	// Проверяем, что категория принадлежит семье пользователя
	if existingCategory.FamilyID != sessionData.FamilyID {
		return h.handleError(c, echo.ErrForbidden, "Access denied")
	}

	// Парсим данные формы
	var form webModels.CategoryForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return h.handleError(c, bindErr, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		// Возвращаем форму с ошибками валидации
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.isHTMXRequest(c) {
			// Для HTMX возвращаем только errors partial
			return h.renderPartial(c, "components/form_errors", map[string]interface{}{
				"Errors": validationErrors,
			})
		}

		// Для обычных запросов возвращаем форму заново
		pageData := &PageData{
			Title: "Edit Category",
			Messages: []Message{
				{Type: "error", Text: "Проверьте правильность заполнения формы"},
			},
		}

		// Получаем родительские категории снова
		parentCategories, _ := h.services.Category.GetCategoriesByFamily(
			c.Request().Context(),
			sessionData.FamilyID,
			nil,
		)
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

		data := map[string]interface{}{
			"PageData":      pageData,
			"Form":          form,
			"Category":      existingCategory,
			"ParentOptions": parentOptions,
			"DefaultColors": getDefaultCategoryColors(),
			"DefaultIcons":  getDefaultCategoryIcons(),
		}

		return h.renderPage(c, "pages/categories/edit", data)
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
		// Обрабатываем специфичные ошибки сервиса
		var errorMsg string
		switch {
		case strings.Contains(err.Error(), "category with this name already exists"):
			errorMsg = "Category with this name already exists"
		case strings.Contains(err.Error(), "category not found"):
			errorMsg = "Category not found"
		default:
			errorMsg = "Failed to update category"
		}

		if h.isHTMXRequest(c) {
			return h.renderPartial(c, "components/form_errors", map[string]interface{}{
				"Errors": map[string]string{"form": errorMsg},
			})
		}

		return h.handleError(c, err, errorMsg)
	}

	// Успешное обновление - редирект
	return h.redirect(c, "/categories")
}

// Show отображает детали конкретной категории
func (h *CategoryHandler) Show(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return h.handleError(c, err, "Invalid category ID")
	}

	// Получаем категорию
	category, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return h.handleError(c, err, "Category not found")
	}

	// Проверяем, что категория принадлежит семье пользователя
	if category.FamilyID != sessionData.FamilyID {
		return h.handleError(c, echo.ErrForbidden, "Access denied")
	}

	// Получаем подкатегории
	allCategories, err := h.services.Category.GetCategoriesByFamily(c.Request().Context(), sessionData.FamilyID, nil)
	if err != nil {
		return h.handleError(c, err, "Failed to get categories")
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

	// Найдем подкатегории для текущей категории и создадим view models
	var subcategoryVMs []webModels.CategoryViewModel
	for _, cat := range allCategories {
		if cat.ParentID != nil && *cat.ParentID == categoryID {
			var subVM webModels.CategoryViewModel
			subVM.FromDomain(cat)
			subcategoryVMs = append(subcategoryVMs, subVM)
		}
	}

	// Получаем последние транзакции для этой категории (если есть Transaction сервис)
	var recentTransactions []interface{} // TODO: заменить на Transaction модель когда будет доступна

	data := map[string]interface{}{
		"Category":      categoryVM,
		"Subcategories": subcategoryVMs,
		"Transactions":  recentTransactions,
	}

	return h.renderPage(c, "pages/categories/show", data)
}

// Delete удаляет категорию
func (h *CategoryHandler) Delete(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Парсим ID категории
	id := c.Param("id")
	categoryID, err := uuid.Parse(id)
	if err != nil {
		return h.handleError(c, err, "Invalid category ID")
	}

	// Проверяем, что категория существует и принадлежит семье
	category, err := h.services.Category.GetCategoryByID(c.Request().Context(), categoryID)
	if err != nil {
		return h.handleError(c, err, "Category not found")
	}

	// Проверяем, что категория принадлежит семье пользователя
	if category.FamilyID != sessionData.FamilyID {
		return h.handleError(c, echo.ErrForbidden, "Access denied")
	}

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

		if h.isHTMXRequest(c) {
			return h.renderPartial(c, "components/alert", map[string]interface{}{
				"Type":    "error",
				"Message": errorMsg,
			})
		}

		return h.handleError(c, err, errorMsg)
	}

	if h.isHTMXRequest(c) {
		// Для HTMX возвращаем пустой ответ для удаления строки
		return c.NoContent(http.StatusOK)
	}

	return h.redirect(c, "/categories")
}

// Search выполняет поиск категорий (HTMX)
func (h *CategoryHandler) Search(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
	}

	// Получаем параметры поиска
	query := strings.TrimSpace(c.QueryParam("q"))
	categoryType := c.QueryParam("type")
	parentOnly := c.QueryParam("parent_only") == "true"

	// Получаем все категории семьи
	typeFilter := parseTypeFilter(categoryType)
	categories, err := h.services.Category.GetCategoriesByFamily(
		c.Request().Context(),
		sessionData.FamilyID,
		typeFilter,
	)
	if err != nil {
		return h.handleError(c, err, "Failed to get categories")
	}

	// Конвертируем и фильтруем категории
	categoryViewModels := convertToViewModels(categories)
	categoryViewModels = applyNameFilter(categoryViewModels, query)
	categoryViewModels = applyParentOnlyFilter(categoryViewModels, parentOnly)

	// Строим дерево если не только родительские
	if parentOnly {
		data := map[string]interface{}{
			"Categories": categoryViewModels,
		}
		return h.renderPartial(c, "components/category_list", data)
	}

	categoryTree := webModels.BuildCategoryTree(categoryViewModels)
	data := map[string]interface{}{
		"Categories": categoryTree,
	}

	return h.renderPartial(c, "components/category_list", data)
}

// Select возвращает категории для select элементов (HTMX)
func (h *CategoryHandler) Select(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return h.handleError(c, err, "Unable to get user session")
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
	categories, err := h.services.Category.GetCategoriesByFamily(
		c.Request().Context(),
		sessionData.FamilyID,
		typeFilter,
	)
	if err != nil {
		return h.handleError(c, err, "Failed to get categories")
	}

	// Конвертируем в select опции
	options := buildSelectOptions(categories, parentOnly, excludeUUID)

	data := map[string]interface{}{
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

// convertToViewModels конвертирует domain модели в view модели с родительскими именами
func convertToViewModels(categories []*category.Category) []webModels.CategoryViewModel {
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
					break
				}
			}
		}

		categoryViewModels = append(categoryViewModels, vm)
	}
	return categoryViewModels
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

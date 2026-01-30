package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
	webModels "family-budget-service/internal/web/models"
)

// Helper functions for TransactionHandler

// buildTransactionFilterDTO конвертирует web фильтры в DTO для сервиса
func (h *TransactionHandler) buildTransactionFilterDTO(
	familyID uuid.UUID,
	filters webModels.TransactionFilters,
) (dto.TransactionFilterDTO, error) {
	filterDTO := dto.TransactionFilterDTO{
		FamilyID: familyID,
		Offset:   (filters.Page - 1) * filters.PageSize,
		Limit:    filters.PageSize,
	}

	// Обрабатываем UUID фильтры
	if err := h.setUUIDFilters(&filterDTO, filters); err != nil {
		return filterDTO, err
	}

	// Обрабатываем тип транзакции
	if err := h.setTransactionTypeFilter(&filterDTO, filters); err != nil {
		return filterDTO, err
	}

	// Обрабатываем даты
	if err := h.setDateFilters(&filterDTO, filters); err != nil {
		return filterDTO, err
	}

	// Обрабатываем суммы
	if err := h.setAmountFilters(&filterDTO, filters); err != nil {
		return filterDTO, err
	}

	// Обрабатываем теги и описание
	h.setTextFilters(&filterDTO, filters)

	return filterDTO, nil
}

// setUUIDFilters обрабатывает UUID фильтры
func (h *TransactionHandler) setUUIDFilters(
	filterDTO *dto.TransactionFilterDTO,
	filters webModels.TransactionFilters,
) error {
	if filters.UserID != "" {
		userUUID, err := uuid.Parse(filters.UserID)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
		filterDTO.UserID = &userUUID
	}

	if filters.CategoryID != "" {
		categoryUUID, err := uuid.Parse(filters.CategoryID)
		if err != nil {
			return fmt.Errorf("invalid category ID: %w", err)
		}
		filterDTO.CategoryID = &categoryUUID
	}

	return nil
}

// setTransactionTypeFilter обрабатывает фильтр типа транзакции
func (h *TransactionHandler) setTransactionTypeFilter(
	filterDTO *dto.TransactionFilterDTO,
	filters webModels.TransactionFilters,
) error {
	if filters.Type != "" {
		switch filters.Type {
		case TransactionTypeIncome:
			transType := transaction.TypeIncome
			filterDTO.Type = &transType
		case TransactionTypeExpense:
			transType := transaction.TypeExpense
			filterDTO.Type = &transType
		default:
			return fmt.Errorf("invalid transaction type: %s", filters.Type)
		}
	}
	return nil
}

// setDateFilters обрабатывает фильтры дат
func (h *TransactionHandler) setDateFilters(
	filterDTO *dto.TransactionFilterDTO,
	filters webModels.TransactionFilters,
) error {
	if filters.DateFrom != "" {
		dateFrom, err := time.Parse("2006-01-02", filters.DateFrom)
		if err != nil {
			return fmt.Errorf("invalid date_from format: %w", err)
		}
		filterDTO.DateFrom = &dateFrom
	}

	if filters.DateTo != "" {
		dateTo, err := time.Parse("2006-01-02", filters.DateTo)
		if err != nil {
			return fmt.Errorf("invalid date_to format: %w", err)
		}
		filterDTO.DateTo = &dateTo
	}

	return nil
}

// setAmountFilters обрабатывает фильтры сумм
func (h *TransactionHandler) setAmountFilters(
	filterDTO *dto.TransactionFilterDTO,
	filters webModels.TransactionFilters,
) error {
	if filters.AmountFrom != "" {
		amountFrom, err := strconv.ParseFloat(filters.AmountFrom, 64)
		if err != nil {
			return fmt.Errorf("invalid amount_from: %w", err)
		}
		filterDTO.AmountFrom = &amountFrom
	}

	if filters.AmountTo != "" {
		amountTo, err := strconv.ParseFloat(filters.AmountTo, 64)
		if err != nil {
			return fmt.Errorf("invalid amount_to: %w", err)
		}
		filterDTO.AmountTo = &amountTo
	}

	return nil
}

// setTextFilters обрабатывает текстовые фильтры
func (h *TransactionHandler) setTextFilters(filterDTO *dto.TransactionFilterDTO, filters webModels.TransactionFilters) {
	if filters.Tags != "" {
		tags := strings.Split(filters.Tags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		filterDTO.Tags = tags
	}

	if filters.Description != "" {
		filterDTO.Description = &filters.Description
	}
}

// convertTransactionsToViewModels конвертирует domain транзакции в view модели
func (h *TransactionHandler) convertTransactionsToViewModels(
	ctx context.Context,
	transactions []*transaction.Transaction,
	familyID uuid.UUID,
) ([]webModels.TransactionViewModel, error) {
	// Получаем категории для заполнения имен
	categories, err := h.services.Category.GetCategories(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	// Создаем мапу для быстрого поиска категорий
	categoryMap := make(map[uuid.UUID]*category.Category)
	for _, cat := range categories {
		categoryMap[cat.ID] = cat
	}

	var viewModels []webModels.TransactionViewModel
	for _, t := range transactions {
		vm := webModels.TransactionViewModel{}
		vm.FromDomain(t)

		// Добавляем имя категории
		if cat, exists := categoryMap[t.CategoryID]; exists {
			vm.CategoryName = cat.Name
		}

		viewModels = append(viewModels, vm)
	}

	return viewModels, nil
}

// buildCategorySelectOptions конвертирует категории в опции для select элементов
func (h *TransactionHandler) buildCategorySelectOptions(
	categories []*category.Category,
) []webModels.CategorySelectOption {
	var options []webModels.CategorySelectOption
	for _, cat := range categories {
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

// calculatePagination рассчитывает данные для пагинации
func (h *TransactionHandler) calculatePagination(totalItems, page, pageSize int) webModels.TransactionListResponse {
	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	return webModels.TransactionListResponse{
		Total:      totalItems,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// buildCreateTransactionDTO создает DTO для создания транзакции
func (h *TransactionHandler) buildCreateTransactionDTO(
	form webModels.TransactionForm,
	userID, familyID uuid.UUID,
) (dto.CreateTransactionDTO, error) {
	// Парсим сумму
	amount, err := strconv.ParseFloat(form.Amount, 64)
	if err != nil {
		return dto.CreateTransactionDTO{}, fmt.Errorf("invalid amount: %w", err)
	}

	// Парсим дату
	date, err := time.Parse("2006-01-02", form.Date)
	if err != nil {
		return dto.CreateTransactionDTO{}, fmt.Errorf("invalid date format: %w", err)
	}

	// Парсим category ID
	categoryID, err := uuid.Parse(form.CategoryID)
	if err != nil {
		return dto.CreateTransactionDTO{}, fmt.Errorf("invalid category ID: %w", err)
	}

	// Определяем тип транзакции
	var transType transaction.Type
	switch form.Type {
	case TransactionTypeIncome:
		transType = transaction.TypeIncome
	case TransactionTypeExpense:
		transType = transaction.TypeExpense
	default:
		return dto.CreateTransactionDTO{}, fmt.Errorf("invalid transaction type: %s", form.Type)
	}

	// Парсим теги
	tags := make([]string, 0) // Всегда инициализируем как пустой массив, а не nil
	if form.Tags != "" {
		splitTags := strings.SplitSeq(form.Tags, ",")
		for tag := range splitTags {
			trimmed := strings.TrimSpace(tag)
			if trimmed != "" { // Добавляем только непустые теги
				tags = append(tags, trimmed)
			}
		}
	}

	return dto.CreateTransactionDTO{
		Amount:      amount,
		Type:        transType,
		Description: form.Description,
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        date,
		Tags:        tags,
	}, nil
}

// buildUpdateTransactionDTO создает DTO для обновления транзакции
func (h *TransactionHandler) buildUpdateTransactionDTO(
	form webModels.TransactionForm,
) (dto.UpdateTransactionDTO, error) {
	updateDTO := dto.UpdateTransactionDTO{}

	// Парсим сумму
	if form.Amount != "" {
		amount, err := strconv.ParseFloat(form.Amount, 64)
		if err != nil {
			return updateDTO, fmt.Errorf("invalid amount: %w", err)
		}
		updateDTO.Amount = &amount
	}

	// Парсим дату
	if form.Date != "" {
		date, err := time.Parse("2006-01-02", form.Date)
		if err != nil {
			return updateDTO, fmt.Errorf("invalid date format: %w", err)
		}
		updateDTO.Date = &date
	}

	// Парсим category ID
	if form.CategoryID != "" {
		categoryID, err := uuid.Parse(form.CategoryID)
		if err != nil {
			return updateDTO, fmt.Errorf("invalid category ID: %w", err)
		}
		updateDTO.CategoryID = &categoryID
	}

	// Определяем тип транзакции
	if form.Type != "" {
		var transType transaction.Type
		switch form.Type {
		case TransactionTypeIncome:
			transType = transaction.TypeIncome
		case TransactionTypeExpense:
			transType = transaction.TypeExpense
		default:
			return updateDTO, fmt.Errorf("invalid transaction type: %s", form.Type)
		}
		updateDTO.Type = &transType
	}

	// Обновляем описание
	if form.Description != "" {
		updateDTO.Description = &form.Description
	}

	// Парсим теги
	if form.Tags != "" {
		tags := strings.Split(form.Tags, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		updateDTO.Tags = tags
	}

	return updateDTO, nil
}

// renderTransactionFormWithErrors отображает форму с ошибками
func (h *TransactionHandler) renderTransactionFormWithErrors(
	c echo.Context,
	form webModels.TransactionForm,
	errors map[string]string,
	familyID uuid.UUID,
	title string,
) error {
	// Получаем категории для селекта
	categories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	categoryOptions := h.buildCategorySelectOptions(categories)

	pageData := &PageData{
		Title:  title,
		Errors: errors,
		Messages: []Message{
			{Type: "error", Text: "Проверьте правильность заполнения формы"},
		},
	}

	data := map[string]any{
		"PageData":        pageData,
		"Form":            form,
		"CategoryOptions": categoryOptions,
	}

	template := "pages/transactions/new"
	if title == "Edit Transaction" {
		template = "pages/transactions/edit"
	}

	return h.renderPage(c, template, data)
}

// getTransactionServiceErrorMessage возвращает пользовательское сообщение об ошибке
func (h *TransactionHandler) getTransactionServiceErrorMessage(err error) string {
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "category not found"):
		return fmt.Sprintf("Selected category not found: %s", errMsg)
	case strings.Contains(errMsg, "insufficient balance"):
		return fmt.Sprintf("Insufficient budget balance for this category: %s", errMsg)
	case strings.Contains(errMsg, "invalid date"):
		return fmt.Sprintf("Invalid transaction date: %s", errMsg)
	case strings.Contains(errMsg, "invalid amount"):
		return fmt.Sprintf("Invalid transaction amount: %s", errMsg)
	default:
		return fmt.Sprintf("Failed to process transaction: %s", errMsg)
	}
}

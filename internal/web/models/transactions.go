package models

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/transaction"
)

const (
	// TypeIncome represents income transaction type
	TypeIncome = "income"
	// TypeExpense represents expense transaction type
	TypeExpense = "expense"

	// DefaultPageSize represents default page size for pagination
	DefaultPageSize = 50
)

// TransactionForm представляет форму создания/редактирования транзакции
type TransactionForm struct {
	Amount      string `form:"amount"      validate:"required,numeric,gt=0"         json:"amount"`
	Type        string `form:"type"        validate:"required,oneof=income expense" json:"type"`
	Description string `form:"description" validate:"required,min=1,max=500"        json:"description"`
	CategoryID  string `form:"category_id" validate:"required,uuid"                 json:"category_id"`
	Date        string `form:"date"        validate:"required"                      json:"date"`
	Tags        string `form:"tags"        validate:"omitempty,max=500"             json:"tags"` // Comma-separated tags
}

// TransactionFilters представляет фильтры для поиска транзакций (на базе domain.transaction.Filter)
type TransactionFilters struct {
	UserID      string `form:"user_id"     validate:"omitempty,uuid"                 json:"user_id,omitempty"`
	CategoryID  string `form:"category_id" validate:"omitempty,uuid"                 json:"category_id,omitempty"`
	Type        string `form:"type"        validate:"omitempty,oneof=income expense" json:"type,omitempty"`
	DateFrom    string `form:"date_from"   validate:"omitempty"                      json:"date_from,omitempty"`
	DateTo      string `form:"date_to"     validate:"omitempty"                      json:"date_to,omitempty"`
	AmountFrom  string `form:"amount_from" validate:"omitempty,numeric,gte=0"        json:"amount_from,omitempty"`
	AmountTo    string `form:"amount_to"   validate:"omitempty,numeric,gte=0"        json:"amount_to,omitempty"`
	Tags        string `form:"tags"        validate:"omitempty,max=500"              json:"tags,omitempty"`
	Description string `form:"description" validate:"omitempty,max=100"              json:"description,omitempty"`
	Page        int    `form:"page"        validate:"omitempty,min=1"                json:"page,omitempty"`
	PageSize    int    `form:"page_size"   validate:"omitempty,min=1,max=100"        json:"page_size,omitempty"`
}

// TransactionViewModel представляет транзакцию для отображения
type TransactionViewModel struct {
	ID              uuid.UUID        `json:"id"`
	Amount          float64          `json:"amount"`
	Type            transaction.Type `json:"type"`
	Description     string           `json:"description"`
	CategoryID      uuid.UUID        `json:"category_id"`
	CategoryName    string           `json:"category_name"`
	CategoryColor   string           `json:"category_color"`
	CategoryIcon    string           `json:"category_icon"`
	UserID          uuid.UUID        `json:"user_id"`
	UserName        string           `json:"user_name"`
	Date            time.Time        `json:"date"`
	Tags            []string         `json:"tags"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CanEdit         bool             `json:"can_edit"`
	CanDelete       bool             `json:"can_delete"`
	FormattedAmount string           `json:"formatted_amount"`
	FormattedDate   string           `json:"formatted_date"`
}

// TransactionListResponse представляет ответ для списка транзакций с пагинацией
type TransactionListResponse struct {
	Transactions []TransactionViewModel `json:"transactions"`
	Total        int                    `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	TotalPages   int                    `json:"total_pages"`
	HasNext      bool                   `json:"has_next"`
	HasPrev      bool                   `json:"has_prev"`
}

// BulkOperationForm представляет форму для bulk операций
type BulkOperationForm struct {
	Action         string   `form:"action"          validate:"required,oneof=delete update_category update_tags" json:"action"`
	TransactionIDs []string `form:"transaction_ids" validate:"required,dive,uuid"                                json:"transaction_ids"`
	CategoryID     string   `form:"category_id"     validate:"omitempty,uuid"                                    json:"category_id,omitempty"`
	Tags           string   `form:"tags"            validate:"omitempty,max=500"                                 json:"tags,omitempty"`
}

// FromDomain создает TransactionViewModel из domain модели
func (vm *TransactionViewModel) FromDomain(t *transaction.Transaction) {
	vm.ID = t.ID
	vm.Amount = t.Amount
	vm.Type = t.Type
	vm.Description = t.Description
	vm.CategoryID = t.CategoryID
	vm.UserID = t.UserID
	vm.Date = t.Date
	vm.Tags = make([]string, len(t.Tags))
	copy(vm.Tags, t.Tags)
	vm.CreatedAt = t.CreatedAt
	vm.UpdatedAt = t.UpdatedAt
	vm.CanEdit = true   // По умолчанию, может быть изменено в зависимости от прав
	vm.CanDelete = true // По умолчанию, может быть изменено в зависимости от прав

	// Форматирование для отображения
	vm.FormattedAmount = formatAmount(t.Amount, t.Type)
	vm.FormattedDate = t.Date.Format("02.01.2006")
}

// GetAmount возвращает сумму как float64
func (f *TransactionForm) GetAmount() (float64, error) {
	return strconv.ParseFloat(f.Amount, 64)
}

// ToTransactionType конвертирует строку в тип транзакции
func (f *TransactionForm) ToTransactionType() transaction.Type {
	switch f.Type {
	case TypeIncome:
		return transaction.TypeIncome
	case TypeExpense:
		return transaction.TypeExpense
	default:
		return transaction.TypeExpense
	}
}

// GetCategoryID возвращает UUID категории
func (f *TransactionForm) GetCategoryID() (uuid.UUID, error) {
	return uuid.Parse(f.CategoryID)
}

// GetDate возвращает дату как time.Time
func (f *TransactionForm) GetDate() (time.Time, error) {
	return time.Parse("2006-01-02", f.Date)
}

// GetTags возвращает теги как слайс строк
func (f *TransactionForm) GetTags() []string {
	if f.Tags == "" {
		return []string{}
	}

	tags := strings.Split(f.Tags, ",")
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// ToDomainFilter конвертирует веб-фильтры в domain фильтр
func (f *TransactionFilters) ToDomainFilter(familyID uuid.UUID) *transaction.Filter {
	filter := &transaction.Filter{
		FamilyID: familyID,
		Limit:    DefaultPageSize, // По умолчанию
		Offset:   0,
	}

	f.setPaginationParams(filter)
	f.setUUIDParams(filter)
	f.setTypeParam(filter)
	f.setDateParams(filter)
	f.setAmountParams(filter)
	f.setTagsParam(filter)
	f.setDescriptionParam(filter)

	return filter
}

// setPaginationParams устанавливает параметры пагинации
func (f *TransactionFilters) setPaginationParams(filter *transaction.Filter) {
	if f.PageSize > 0 {
		filter.Limit = f.PageSize
	}

	if f.Page > 0 {
		filter.Offset = (f.Page - 1) * filter.Limit
	}
}

// setUUIDParams устанавливает UUID параметры
func (f *TransactionFilters) setUUIDParams(filter *transaction.Filter) {
	if f.UserID != "" {
		if userID, err := uuid.Parse(f.UserID); err == nil {
			filter.UserID = &userID
		}
	}

	if f.CategoryID != "" {
		if categoryID, err := uuid.Parse(f.CategoryID); err == nil {
			filter.CategoryID = &categoryID
		}
	}
}

// setTypeParam устанавливает тип транзакции
func (f *TransactionFilters) setTypeParam(filter *transaction.Filter) {
	if f.Type != "" {
		switch f.Type {
		case TypeIncome:
			tType := transaction.TypeIncome
			filter.Type = &tType
		case TypeExpense:
			tType := transaction.TypeExpense
			filter.Type = &tType
		}
	}
}

// setDateParams устанавливает параметры дат
func (f *TransactionFilters) setDateParams(filter *transaction.Filter) {
	if f.DateFrom != "" {
		if date, err := time.Parse("2006-01-02", f.DateFrom); err == nil {
			filter.DateFrom = &date
		}
	}

	if f.DateTo != "" {
		if date, err := time.Parse("2006-01-02", f.DateTo); err == nil {
			// Устанавливаем время на конец дня
			endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
			filter.DateTo = &endOfDay
		}
	}
}

// setAmountParams устанавливает параметры сумм
func (f *TransactionFilters) setAmountParams(filter *transaction.Filter) {
	if f.AmountFrom != "" {
		if amount, err := strconv.ParseFloat(f.AmountFrom, 64); err == nil {
			filter.AmountFrom = &amount
		}
	}

	if f.AmountTo != "" {
		if amount, err := strconv.ParseFloat(f.AmountTo, 64); err == nil {
			filter.AmountTo = &amount
		}
	}
}

// setTagsParam устанавливает теги
func (f *TransactionFilters) setTagsParam(filter *transaction.Filter) {
	if f.Tags != "" {
		tags := strings.Split(f.Tags, ",")
		filter.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				filter.Tags = append(filter.Tags, tag)
			}
		}
	}
}

// setDescriptionParam устанавливает описание
func (f *TransactionFilters) setDescriptionParam(filter *transaction.Filter) {
	if f.Description != "" {
		filter.Description = strings.TrimSpace(f.Description)
	}
}

// GetTransactionIDs возвращает слайс UUID из строк
func (f *BulkOperationForm) GetTransactionIDs() ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(f.TransactionIDs))

	for _, idStr := range f.TransactionIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// formatAmount форматирует сумму для отображения
func formatAmount(amount float64, transactionType transaction.Type) string {
	sign := "+"
	if transactionType == transaction.TypeExpense {
		sign = "-"
	}

	return sign + strconv.FormatFloat(amount, 'f', 2, 64)
}

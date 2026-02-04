package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/transaction"
)

func TestNewTransactionFilterDTO(t *testing.T) {
	filter := NewTransactionFilterDTO()

	assert.Equal(t, DefaultTransactionLimit, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
	assert.NotNil(t, filter.SortBy)
	assert.Equal(t, DefaultSortByDate, *filter.SortBy)
	assert.NotNil(t, filter.SortOrder)
	assert.Equal(t, DefaultSortOrderDesc, *filter.SortOrder)
}

func TestTransactionFilterDTO_ValidateDateRange_Valid(t *testing.T) {
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	filter := TransactionFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	err := filter.ValidateDateRange()
	assert.NoError(t, err)
}

func TestTransactionFilterDTO_ValidateDateRange_Invalid(t *testing.T) {
	dateFrom := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	filter := TransactionFilterDTO{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	err := filter.ValidateDateRange()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidDateRange, err)
}

func TestTransactionFilterDTO_ValidateDateRange_OnlyDateFrom(t *testing.T) {
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	filter := TransactionFilterDTO{
		DateFrom: &dateFrom,
	}

	err := filter.ValidateDateRange()
	assert.NoError(t, err)
}

func TestTransactionFilterDTO_ValidateAmountRange_Valid(t *testing.T) {
	amountFrom := 10.0
	amountTo := 100.0

	filter := TransactionFilterDTO{
		AmountFrom: &amountFrom,
		AmountTo:   &amountTo,
	}

	err := filter.ValidateAmountRange()
	assert.NoError(t, err)
}

func TestTransactionFilterDTO_ValidateAmountRange_Invalid(t *testing.T) {
	amountFrom := 100.0
	amountTo := 10.0

	filter := TransactionFilterDTO{
		AmountFrom: &amountFrom,
		AmountTo:   &amountTo,
	}

	err := filter.ValidateAmountRange()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAmountRange, err)
}

func TestTransactionFilterDTO_ValidateAmountRange_Equal(t *testing.T) {
	amount := 50.0

	filter := TransactionFilterDTO{
		AmountFrom: &amount,
		AmountTo:   &amount,
	}

	err := filter.ValidateAmountRange()
	assert.NoError(t, err)
}

func TestCreateTransactionDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	dto := CreateTransactionDTO{
		Amount:      100.50,
		Type:        transaction.TypeExpense,
		Description: "Groceries",
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        date,
		Tags:        []string{"food", "weekly"},
	}

	assert.Equal(t, 100.50, dto.Amount)
	assert.Equal(t, transaction.TypeExpense, dto.Type)
	assert.Equal(t, "Groceries", dto.Description)
	assert.Equal(t, categoryID, dto.CategoryID)
	assert.Equal(t, userID, dto.UserID)
	assert.Equal(t, date, dto.Date)
	assert.Len(t, dto.Tags, 2)
}

func TestUpdateTransactionDTO_AllFields(t *testing.T) {
	amount := 200.00
	txType := transaction.TypeIncome
	description := "Updated"
	categoryID := uuid.New()
	date := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)

	dto := UpdateTransactionDTO{
		Amount:      &amount,
		Type:        &txType,
		Description: &description,
		CategoryID:  &categoryID,
		Date:        &date,
		Tags:        []string{"updated"},
	}

	assert.NotNil(t, dto.Amount)
	assert.Equal(t, 200.00, *dto.Amount)
	assert.NotNil(t, dto.Type)
	assert.Equal(t, transaction.TypeIncome, *dto.Type)
	assert.NotNil(t, dto.Description)
	assert.Equal(t, "Updated", *dto.Description)
}

func TestUpdateTransactionDTO_PartialUpdate(t *testing.T) {
	amount := 150.00

	dto := UpdateTransactionDTO{
		Amount: &amount,
	}

	assert.NotNil(t, dto.Amount)
	assert.Equal(t, 150.00, *dto.Amount)
	assert.Nil(t, dto.Type)
	assert.Nil(t, dto.Description)
	assert.Nil(t, dto.CategoryID)
	assert.Nil(t, dto.Date)
}

func TestTransactionResponseDTO_AllFields(t *testing.T) {
	now := time.Now()
	transactionID := uuid.New()
	categoryID := uuid.New()
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	response := TransactionResponseDTO{
		ID:          transactionID,
		Amount:      100.50,
		Type:        "expense",
		Description: "Groceries",
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        date,
		Tags:        []string{"food"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, transactionID, response.ID)
	assert.Equal(t, 100.50, response.Amount)
	assert.Equal(t, "expense", response.Type)
	assert.Equal(t, "Groceries", response.Description)
}

func TestBulkCategorizeDTO_AllFields(t *testing.T) {
	tx1 := uuid.New()
	tx2 := uuid.New()
	categoryID := uuid.New()
	userID := uuid.New()

	dto := BulkCategorizeDTO{
		TransactionIDs: []uuid.UUID{tx1, tx2},
		CategoryID:     categoryID,
		UserID:         userID,
	}

	assert.Len(t, dto.TransactionIDs, 2)
	assert.Equal(t, categoryID, dto.CategoryID)
	assert.Equal(t, userID, dto.UserID)
}

func TestTransactionStatsDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	stats := TransactionStatsDTO{
		CategoryID:       &categoryID,
		Period:           "monthly",
		TotalIncome:      5000.00,
		TotalExpense:     3000.00,
		NetFlow:          2000.00,
		TransactionCount: 50,
		StartDate:        startDate,
		EndDate:          endDate,
	}

	assert.NotNil(t, stats.CategoryID)
	assert.Equal(t, "monthly", stats.Period)
	assert.Equal(t, 5000.00, stats.TotalIncome)
	assert.Equal(t, 3000.00, stats.TotalExpense)
	assert.Equal(t, 2000.00, stats.NetFlow)
	assert.Equal(t, 50, stats.TransactionCount)
}

func TestTransactionStatsDTO_WithoutCategory(t *testing.T) {
	stats := TransactionStatsDTO{
		Period:           "yearly",
		TotalIncome:      60000.00,
		TotalExpense:     40000.00,
		NetFlow:          20000.00,
		TransactionCount: 600,
	}

	assert.Nil(t, stats.CategoryID)
	assert.Equal(t, "yearly", stats.Period)
}

func TestTransactionFilterDTO_ComplexFilter(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	txType := transaction.TypeExpense
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	amountFrom := 10.0
	amountTo := 1000.0
	description := "groceries"
	sortBy := "date"
	sortOrder := "desc"

	filter := TransactionFilterDTO{
		UserID:      &userID,
		CategoryID:  &categoryID,
		Type:        &txType,
		DateFrom:    &dateFrom,
		DateTo:      &dateTo,
		AmountFrom:  &amountFrom,
		AmountTo:    &amountTo,
		Description: &description,
		Tags:        []string{"food", "weekly"},
		Limit:       100,
		Offset:      0,
		SortBy:      &sortBy,
		SortOrder:   &sortOrder,
	}

	assert.NotNil(t, filter.UserID)
	assert.NotNil(t, filter.CategoryID)
	assert.NotNil(t, filter.Type)
	assert.NotNil(t, filter.DateFrom)
	assert.NotNil(t, filter.DateTo)
	assert.NotNil(t, filter.Description)
	assert.Len(t, filter.Tags, 2)
	assert.Equal(t, 100, filter.Limit)
}

func TestStringPtr(t *testing.T) {
	str := "test"
	ptr := stringPtr(str)

	assert.NotNil(t, ptr)
	assert.Equal(t, "test", *ptr)
}

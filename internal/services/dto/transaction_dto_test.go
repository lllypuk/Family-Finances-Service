package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
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
	dateFrom := date.New(2024, time.January, 1)
	dateTo := date.New(2024, time.January, 31)

	filter := TransactionFilterDTO{DateFrom: &dateFrom, DateTo: &dateTo}

	assert.NoError(t, filter.ValidateDateRange())
}

func TestTransactionFilterDTO_ValidateDateRange_Invalid(t *testing.T) {
	dateFrom := date.New(2024, time.January, 31)
	dateTo := date.New(2024, time.January, 1)

	filter := TransactionFilterDTO{DateFrom: &dateFrom, DateTo: &dateTo}

	assert.Equal(t, ErrInvalidDateRange, filter.ValidateDateRange())
}

func TestTransactionFilterDTO_ValidateDateRange_OnlyDateFrom(t *testing.T) {
	dateFrom := date.New(2024, time.January, 1)

	filter := TransactionFilterDTO{DateFrom: &dateFrom}

	assert.NoError(t, filter.ValidateDateRange())
}

func TestTransactionFilterDTO_ValidateAmountRange_Valid(t *testing.T) {
	amountFrom := money.Minor(1_000)
	amountTo := money.Minor(10_000)

	filter := TransactionFilterDTO{AmountFromMinor: &amountFrom, AmountToMinor: &amountTo}

	assert.NoError(t, filter.ValidateAmountRange())
}

func TestTransactionFilterDTO_ValidateAmountRange_Invalid(t *testing.T) {
	amountFrom := money.Minor(10_000)
	amountTo := money.Minor(1_000)

	filter := TransactionFilterDTO{AmountFromMinor: &amountFrom, AmountToMinor: &amountTo}

	assert.Equal(t, ErrInvalidAmountRange, filter.ValidateAmountRange())
}

func TestTransactionFilterDTO_ValidateAmountRange_Equal(t *testing.T) {
	amount := money.Minor(5_000)

	filter := TransactionFilterDTO{AmountFromMinor: &amount, AmountToMinor: &amount}

	assert.NoError(t, filter.ValidateAmountRange())
}

func TestCreateTransactionDTO_AllFields(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()
	txDate := date.New(2024, time.January, 15)

	dto := CreateTransactionDTO{
		AmountMinor: 10_050,
		Type:        transaction.TypeExpense,
		Description: "Groceries",
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        txDate,
		Tags:        []string{"food", "weekly"},
	}

	assert.Equal(t, money.Minor(10_050), dto.AmountMinor)
	assert.Equal(t, transaction.TypeExpense, dto.Type)
	assert.Equal(t, "Groceries", dto.Description)
	assert.Equal(t, categoryID, dto.CategoryID)
	assert.Equal(t, userID, dto.UserID)
	assert.Equal(t, txDate, dto.Date)
	assert.Len(t, dto.Tags, 2)
}

func TestUpdateTransactionDTO_AllFields(t *testing.T) {
	amount := money.Minor(20_000)
	txType := transaction.TypeIncome
	description := "Updated"
	categoryID := uuid.New()
	txDate := date.New(2024, time.January, 20)

	dto := UpdateTransactionDTO{
		AmountMinor: &amount,
		Type:        &txType,
		Description: &description,
		CategoryID:  &categoryID,
		Date:        &txDate,
		Tags:        []string{"updated"},
	}

	assert.NotNil(t, dto.AmountMinor)
	assert.Equal(t, money.Minor(20_000), *dto.AmountMinor)
	assert.NotNil(t, dto.Type)
	assert.Equal(t, transaction.TypeIncome, *dto.Type)
	assert.NotNil(t, dto.Description)
	assert.Equal(t, "Updated", *dto.Description)
}

func TestUpdateTransactionDTO_PartialUpdate(t *testing.T) {
	amount := money.Minor(15_000)

	dto := UpdateTransactionDTO{AmountMinor: &amount}

	assert.NotNil(t, dto.AmountMinor)
	assert.Equal(t, money.Minor(15_000), *dto.AmountMinor)
	assert.Nil(t, dto.Type)
	assert.Nil(t, dto.Description)
	assert.Nil(t, dto.CategoryID)
	assert.Nil(t, dto.Date)
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

func TestTransactionFilterDTO_ComplexFilter(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	txType := transaction.TypeExpense
	dateFrom := date.New(2024, time.January, 1)
	dateTo := date.New(2024, time.January, 31)
	amountFrom := money.Minor(1_000)
	amountTo := money.Minor(100_000)
	description := "groceries"
	sortBy := "date"
	sortOrder := "desc"

	filter := TransactionFilterDTO{
		UserID:          &userID,
		CategoryID:      &categoryID,
		Type:            &txType,
		DateFrom:        &dateFrom,
		DateTo:          &dateTo,
		AmountFromMinor: &amountFrom,
		AmountToMinor:   &amountTo,
		Description:     &description,
		Tags:            []string{"food", "weekly"},
		Limit:           100,
		Offset:          0,
		SortBy:          &sortBy,
		SortOrder:       &sortOrder,
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
	ptr := new(string)

	assert.NotNil(t, ptr)
	assert.Equal(t, "", *ptr)
}

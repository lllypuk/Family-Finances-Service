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

func TestTransactionFilterDTO_ComplexFilter(t *testing.T) {
	userID := uuid.New()
	categoryID := uuid.New()
	txType := transaction.TypeExpense
	dateFrom := date.New(2024, time.January, 1)
	dateTo := date.New(2024, time.January, 31)
	amountFrom := money.Minor(1_000)
	amountTo := money.Minor(100_000)
	description := "groceries"

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

package dto

import (
	"errors"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
)

// DTO validation errors
var (
	ErrInvalidDateRange   = errors.New("date_to must be after date_from")
	ErrInvalidAmountRange = errors.New("amount_to must be greater than or equal to amount_from")
)

// CreateTransactionDTO represents the data required to create a new transaction
type CreateTransactionDTO struct {
	// ID — необязательный клиентский идентификатор; пустой означает «сгенерировать».
	ID          *uuid.UUID       `validate:"omitempty"`
	AmountMinor money.Minor      `validate:"required,gt=0"`
	Type        transaction.Type `validate:"required,oneof=income expense"`
	Description string           `validate:"required,min=2,max=200"`
	CategoryID  uuid.UUID        `validate:"required"`
	UserID      uuid.UUID        `validate:"required"`
	Date        date.Date        `validate:"required"`
	Tags        []string         `validate:"omitempty,dive,min=1,max=50"`
}

// UpdateTransactionDTO represents the data that can be updated for an existing transaction
type UpdateTransactionDTO struct {
	AmountMinor *money.Minor      `validate:"omitempty,gt=0"`
	Type        *transaction.Type `validate:"omitempty,oneof=income expense"`
	Description *string           `validate:"omitempty,min=2,max=200"`
	CategoryID  *uuid.UUID        `validate:"omitempty"`
	Date        *date.Date        `validate:"omitempty"`
	Tags        []string          `validate:"omitempty,dive,min=1,max=50"`
}

// TransactionFilterDTO represents filtering and pagination options for transactions
type TransactionFilterDTO struct {
	// Core filters
	UserID     *uuid.UUID        `validate:"omitempty"`
	CategoryID *uuid.UUID        `validate:"omitempty"`
	Type       *transaction.Type `validate:"omitempty,oneof=income expense"`

	// Date range filters
	DateFrom *date.Date `validate:"omitempty"`
	DateTo   *date.Date `validate:"omitempty"`

	// Amount range filters
	AmountFromMinor *money.Minor `validate:"omitempty,gte=0"`
	AmountToMinor   *money.Minor `validate:"omitempty,gte=0"`

	// Text search
	Description *string  `validate:"omitempty,min=1,max=200"`
	Tags        []string `validate:"omitempty,dive,min=1,max=50"`

	// Pagination
	Limit  int `validate:"min=1,max=1000"`
	Offset int `validate:"min=0"`

	// Sorting
	SortBy    *string `validate:"omitempty,oneof=date amount created_at updated_at"`
	SortOrder *string `validate:"omitempty,oneof=asc desc"`
}

// BulkCategorizeDTO represents data for bulk categorization of transactions
type BulkCategorizeDTO struct {
	TransactionIDs []uuid.UUID `validate:"required,min=1,dive,required"`
	CategoryID     uuid.UUID   `validate:"required"`
	UserID         uuid.UUID   `validate:"required"` // For authorization
}

const (
	// DefaultTransactionLimit is the default number of transactions to return
	DefaultTransactionLimit = 50
	// DefaultSortByDate is the default sort field for transactions
	DefaultSortByDate = "date"
	// DefaultSortOrderDesc is the default sort order for transactions
	DefaultSortOrderDesc = "desc"
)

// NewTransactionFilterDTO creates a new TransactionFilterDTO with default values
func NewTransactionFilterDTO() TransactionFilterDTO {
	return TransactionFilterDTO{
		Limit:     DefaultTransactionLimit,
		Offset:    0,
		SortBy:    new(DefaultSortByDate),
		SortOrder: new(DefaultSortOrderDesc),
	}
}

// ValidateDateRange validates that DateTo is after DateFrom if both are provided
func (f *TransactionFilterDTO) ValidateDateRange() error {
	if f.DateFrom != nil && f.DateTo != nil {
		if f.DateTo.Before(*f.DateFrom) {
			return ErrInvalidDateRange
		}
	}
	return nil
}

// ValidateAmountRange validates that AmountTo is greater than AmountFrom if both are provided
func (f *TransactionFilterDTO) ValidateAmountRange() error {
	if f.AmountFromMinor != nil && f.AmountToMinor != nil {
		if *f.AmountToMinor < *f.AmountFromMinor {
			return ErrInvalidAmountRange
		}
	}
	return nil
}

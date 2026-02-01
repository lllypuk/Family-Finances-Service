package dto

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/transaction"
)

// DTO validation errors
var (
	ErrInvalidDateRange   = errors.New("date_to must be after date_from")
	ErrInvalidAmountRange = errors.New("amount_to must be greater than or equal to amount_from")
)

// CreateTransactionDTO represents the data required to create a new transaction
type CreateTransactionDTO struct {
	Amount      float64          `validate:"required,gt=0"`
	Type        transaction.Type `validate:"required,oneof=income expense"`
	Description string           `validate:"required,min=2,max=200"`
	CategoryID  uuid.UUID        `validate:"required"`
	UserID      uuid.UUID        `validate:"required"`
	Date        time.Time        `validate:"required"`
	Tags        []string         `validate:"omitempty,dive,min=1,max=50"`
}

// UpdateTransactionDTO represents the data that can be updated for an existing transaction
type UpdateTransactionDTO struct {
	Amount      *float64          `validate:"omitempty,gt=0"`
	Type        *transaction.Type `validate:"omitempty,oneof=income expense"`
	Description *string           `validate:"omitempty,min=2,max=200"`
	CategoryID  *uuid.UUID        `validate:"omitempty"`
	Date        *time.Time        `validate:"omitempty"`
	Tags        []string          `validate:"omitempty,dive,min=1,max=50"`
}

// TransactionFilterDTO represents filtering and pagination options for transactions
type TransactionFilterDTO struct {
	// Core filters
	UserID     *uuid.UUID        `validate:"omitempty"`
	CategoryID *uuid.UUID        `validate:"omitempty"`
	Type       *transaction.Type `validate:"omitempty,oneof=income expense"`

	// Date range filters
	DateFrom *time.Time `validate:"omitempty"`
	DateTo   *time.Time `validate:"omitempty"`

	// Amount range filters
	AmountFrom *float64 `validate:"omitempty,gte=0"`
	AmountTo   *float64 `validate:"omitempty,gte=0"`

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

// TransactionResponseDTO represents transaction data for API responses
type TransactionResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CategoryID  uuid.UUID `json:"category_id"`
	UserID      uuid.UUID `json:"user_id"`
	Date        time.Time `json:"date"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BulkCategorizeDTO represents data for bulk categorization of transactions
type BulkCategorizeDTO struct {
	TransactionIDs []uuid.UUID `validate:"required,min=1,dive,required"`
	CategoryID     uuid.UUID   `validate:"required"`
	UserID         uuid.UUID   `validate:"required"` // For authorization
}

// TransactionStatsDTO represents transaction statistics for a family or category
type TransactionStatsDTO struct {
	CategoryID       *uuid.UUID `json:"category_id,omitempty"`
	Period           string     `json:"period"` // "daily", "weekly", "monthly", "yearly"
	TotalIncome      float64    `json:"total_income"`
	TotalExpense     float64    `json:"total_expense"`
	NetFlow          float64    `json:"net_flow"`
	TransactionCount int        `json:"transaction_count"`
	StartDate        time.Time  `json:"start_date"`
	EndDate          time.Time  `json:"end_date"`
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
		SortBy:    stringPtr(DefaultSortByDate),
		SortOrder: stringPtr(DefaultSortOrderDesc),
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
	if f.AmountFrom != nil && f.AmountTo != nil {
		if *f.AmountTo < *f.AmountFrom {
			return ErrInvalidAmountRange
		}
	}
	return nil
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

package services

import (
	"context"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/transaction"
)

// TransactionRepositoryForUsage defines minimal interface for checking category usage
type TransactionRepositoryForUsage interface {
	GetTotalByCategory(ctx context.Context, categoryID uuid.UUID, transactionType transaction.Type) (float64, error)
}

// CategoryUsageCheckerImpl implements CategoryUsageChecker interface
type CategoryUsageCheckerImpl struct {
	transactionRepo TransactionRepositoryForUsage
}

// NewCategoryUsageChecker creates a new CategoryUsageChecker instance
func NewCategoryUsageChecker(transactionRepo TransactionRepositoryForUsage) CategoryUsageChecker {
	return &CategoryUsageCheckerImpl{
		transactionRepo: transactionRepo,
	}
}

// IsCategoryUsed checks if a category is used in any transactions
func (c *CategoryUsageCheckerImpl) IsCategoryUsed(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	// Check if category is used in transactions by trying to get total for any transaction type
	// We use empty transaction.Type which should match any type
	total, err := c.transactionRepo.GetTotalByCategory(ctx, categoryID, "")
	if err != nil {
		// If error is "not found" or similar, category is not used
		return false, err
	}

	// If we got a result without error, category is used
	return total >= 0, nil
}

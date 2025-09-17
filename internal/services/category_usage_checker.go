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
	// Check if category is used in income transactions
	incomeTotal, err := c.transactionRepo.GetTotalByCategory(ctx, categoryID, transaction.TypeIncome)
	if err != nil {
		// If error is not "not found", return the error
		return false, err
	}

	// Check if category is used in expense transactions
	expenseTotal, err := c.transactionRepo.GetTotalByCategory(ctx, categoryID, transaction.TypeExpense)
	if err != nil {
		// If error is not "not found", return the error
		return false, err
	}

	// Category is used if there are any transactions of either type
	return incomeTotal > 0 || expenseTotal > 0, nil
}

package validation

import (
	"errors"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
)

// ValidateUUID validates UUID parameter to prevent injection attacks
func ValidateUUID(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("UUID cannot be nil")
	}
	return nil
}

// ValidateCategoryType validates category type parameter
func ValidateCategoryType(categoryType category.Type) error {
	if categoryType != category.TypeIncome && categoryType != category.TypeExpense {
		return errors.New("invalid category type")
	}
	return nil
}

// ValidateTransactionType validates transaction type parameter
func ValidateTransactionType(transactionType transaction.Type) error {
	if transactionType != transaction.TypeIncome && transactionType != transaction.TypeExpense {
		return errors.New("invalid transaction type")
	}
	return nil
}

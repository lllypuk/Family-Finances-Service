package validation

import (
	"errors"
	"regexp"
	"strings"

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

// ValidateEmail validates email format to prevent injection attacks
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}

	// Check for maximum length
	if len(email) > 254 {
		return errors.New("email too long")
	}

	// Email regex pattern - strict validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	// Additional security checks for SQL injection patterns
	suspicious := []string{"'", "--", "/*", "*/", "xp_", "sp_", "DROP", "SELECT", "INSERT", "UPDATE", "DELETE"}
	emailLower := strings.ToLower(email)
	for _, pattern := range suspicious {
		if strings.Contains(emailLower, strings.ToLower(pattern)) {
			return errors.New("email contains suspicious characters")
		}
	}

	return nil
}

// SanitizeEmail sanitizes email by trimming whitespace and converting to lowercase
func SanitizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

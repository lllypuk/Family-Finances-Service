package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
)

// Validation constants
const (
	maxEmailLength        = 254
	maxBudgetAmount       = 999999999.99
	maxBudgetNameLength   = 255
	maxTransactionAmount  = 999999999.99
	maxDescriptionLength  = 1000
	maxCategoryNameLength = 255
	maxReportNameLength   = 255
	currencyCodeLength    = 3
	MaxQueryLimit         = 1000 // Exported for use in repositories
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
	if len(email) > maxEmailLength {
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

// ValidateBudgetPeriod validates budget period
func ValidateBudgetPeriod(period budget.Period) error {
	switch period {
	case budget.PeriodWeekly, budget.PeriodMonthly, budget.PeriodYearly, budget.PeriodCustom:
		return nil
	default:
		return errors.New("invalid budget period")
	}
}

// ValidateBudgetAmount validates budget amount
func ValidateBudgetAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("budget amount must be positive")
	}
	if amount > maxBudgetAmount {
		return errors.New("budget amount too large")
	}
	return nil
}

// ValidateBudgetName validates budget name
func ValidateBudgetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("budget name cannot be empty")
	}
	if len(name) > maxBudgetNameLength {
		return errors.New("budget name too long")
	}
	return nil
}

// ValidateAmount validates transaction amount
func ValidateAmount(amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	if amount > maxTransactionAmount {
		return errors.New("amount too large")
	}
	return nil
}

// ValidateDescription validates transaction description
func ValidateDescription(description string) error {
	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("description cannot be empty")
	}
	if len(description) > maxDescriptionLength {
		return errors.New("description too long")
	}
	return nil
}

// ValidateCategoryName validates category name
func ValidateCategoryName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("category name cannot be empty")
	}
	if len(name) > maxCategoryNameLength {
		return errors.New("category name too long")
	}
	return nil
}

// ValidateReportType validates report type
func ValidateReportType(reportType report.Type) error {
	switch reportType {
	case report.TypeExpenses, report.TypeIncome, report.TypeBudget, report.TypeCashFlow, report.TypeCategoryBreak:
		return nil
	default:
		return errors.New("invalid report type")
	}
}

// ValidateReportPeriod validates report period
func ValidateReportPeriod(period report.Period) error {
	switch period {
	case report.PeriodDaily, report.PeriodWeekly, report.PeriodMonthly, report.PeriodYearly, report.PeriodCustom:
		return nil
	default:
		return errors.New("invalid report period")
	}
}

// ValidateReportName validates report name
func ValidateReportName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("report name cannot be empty")
	}
	if len(name) > maxReportNameLength {
		return errors.New("report name too long")
	}
	return nil
}

// ValidateCurrency validates currency code
func ValidateCurrency(currency string) error {
	if currency == "" {
		return errors.New("currency cannot be empty")
	}

	// Convert to uppercase for consistency
	currency = strings.ToUpper(strings.TrimSpace(currency))

	// Must be exactly 3 characters (ISO 4217)
	if len(currency) != currencyCodeLength {
		return errors.New("currency must be exactly 3 characters")
	}

	// Check for valid characters (A-Z only)
	for _, char := range currency {
		if char < 'A' || char > 'Z' {
			return errors.New("currency must contain only uppercase letters")
		}
	}

	// Common currency codes validation (extend as needed)
	validCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "RUB": true,
		"CNY": true, "CAD": true, "AUD": true, "CHF": true, "SEK": true,
		"NOK": true, "DKK": true, "PLN": true, "CZK": true, "HUF": true,
	}

	if !validCurrencies[currency] {
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	return nil
}

// SanitizeFamilyName sanitizes family name by trimming whitespace
func SanitizeFamilyName(name string) string {
	return strings.TrimSpace(name)
}

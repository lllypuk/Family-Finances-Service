package validation_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/infrastructure/validation"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "valid UUID",
			id:      uuid.New(),
			wantErr: false,
		},
		{
			name:    "nil UUID",
			id:      uuid.Nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateUUID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategoryType(t *testing.T) {
	tests := []struct {
		name    string
		catType category.Type
		wantErr bool
	}{
		{
			name:    "income type",
			catType: category.TypeIncome,
			wantErr: false,
		},
		{
			name:    "expense type",
			catType: category.TypeExpense,
			wantErr: false,
		},
		{
			name:    "invalid type - empty",
			catType: category.Type(""),
			wantErr: true,
		},
		{
			name:    "invalid type - transfer",
			catType: category.Type("transfer"),
			wantErr: true,
		},
		{
			name:    "invalid type - uppercase",
			catType: category.Type("INCOME"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateCategoryType(tt.catType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategoryType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTransactionType(t *testing.T) {
	tests := []struct {
		name    string
		txType  transaction.Type
		wantErr bool
	}{
		{
			name:    "income type",
			txType:  transaction.TypeIncome,
			wantErr: false,
		},
		{
			name:    "expense type",
			txType:  transaction.TypeExpense,
			wantErr: false,
		},
		{
			name:    "invalid type - empty",
			txType:  transaction.Type(""),
			wantErr: true,
		},
		{
			name:    "invalid type - transfer",
			txType:  transaction.Type("transfer"),
			wantErr: true,
		},
		{
			name:    "invalid type - uppercase",
			txType:  transaction.Type("INCOME"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateTransactionType(tt.txType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransactionType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		// Valid emails
		{
			name:    "simple email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "email with plus",
			email:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "email with dots",
			email:   "first.last@example.com",
			wantErr: false,
		},
		{
			name:    "email with numbers",
			email:   "user123@example123.com",
			wantErr: false,
		},
		{
			name:    "email with hyphen in domain",
			email:   "user@my-domain.com",
			wantErr: false,
		},
		{
			name:    "email with underscore",
			email:   "user_name@example.com",
			wantErr: false,
		},

		// Invalid emails
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "no at sign",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "no domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "no local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "double at sign",
			email:   "user@@example.com",
			wantErr: true,
		},
		{
			name:    "spaces in email",
			email:   "user @example.com",
			wantErr: true,
		},
		{
			name:    "special chars",
			email:   "user<>@example.com",
			wantErr: true,
		},
		{
			name:    "no TLD",
			email:   "user@example",
			wantErr: true,
		},
		{
			name:    "too long email",
			email:   strings.Repeat("a", 250) + "@example.com",
			wantErr: true,
		},

		// Security checks - SQL injection attempts
		{
			name:    "SQL injection - single quote",
			email:   "user'@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - double dash",
			email:   "user--@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - DROP command",
			email:   "userDROP@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - SELECT command",
			email:   "userSELECT@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - comment block start",
			email:   "user/*@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - comment block end",
			email:   "user*/@example.com",
			wantErr: true,
		},
		{
			name:    "SQL injection - stored procedure prefix",
			email:   "userxp_@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "already clean",
			email: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "with leading spaces",
			email: "  user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "with trailing spaces",
			email: "user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "with both spaces",
			email: "  user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "uppercase to lowercase",
			email: "User@Example.COM",
			want:  "user@example.com",
		},
		{
			name:  "mixed case with spaces",
			email: "  UsEr@ExAmPlE.com  ",
			want:  "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.SanitizeEmail(tt.email)
			if got != tt.want {
				t.Errorf("SanitizeEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateBudgetPeriod(t *testing.T) {
	tests := []struct {
		name    string
		period  budget.Period
		wantErr bool
	}{
		{
			name:    "weekly period",
			period:  budget.PeriodWeekly,
			wantErr: false,
		},
		{
			name:    "monthly period",
			period:  budget.PeriodMonthly,
			wantErr: false,
		},
		{
			name:    "yearly period",
			period:  budget.PeriodYearly,
			wantErr: false,
		},
		{
			name:    "custom period",
			period:  budget.PeriodCustom,
			wantErr: false,
		},
		{
			name:    "invalid - empty",
			period:  budget.Period(""),
			wantErr: true,
		},
		{
			name:    "invalid - biweekly",
			period:  budget.Period("biweekly"),
			wantErr: true,
		},
		{
			name:    "invalid - uppercase",
			period:  budget.Period("MONTHLY"),
			wantErr: true,
		},
		{
			name:    "invalid - daily",
			period:  budget.Period("daily"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateBudgetPeriod(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBudgetPeriod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBudgetAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "positive integer",
			amount:  100.00,
			wantErr: false,
		},
		{
			name:    "positive decimal",
			amount:  99.99,
			wantErr: false,
		},
		{
			name:    "small amount",
			amount:  0.01,
			wantErr: false,
		},
		{
			name:    "large amount",
			amount:  999999.99,
			wantErr: false,
		},
		{
			name:    "maximum valid amount",
			amount:  999999999.99,
			wantErr: false,
		},

		// Invalid amounts
		{
			name:    "zero",
			amount:  0,
			wantErr: true,
		},
		{
			name:    "negative",
			amount:  -100,
			wantErr: true,
		},
		{
			name:    "too large",
			amount:  999999999.99 + 0.01,
			wantErr: true,
		},
		{
			name:    "extremely large",
			amount:  999999999999.99,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateBudgetAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBudgetAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBudgetName(t *testing.T) {
	tests := []struct {
		name       string
		budgetName string
		wantErr    bool
	}{
		{
			name:       "simple name",
			budgetName: "Monthly Budget",
			wantErr:    false,
		},
		{
			name:       "with numbers",
			budgetName: "Budget 2024",
			wantErr:    false,
		},
		{
			name:       "with special chars",
			budgetName: "Budget (January)",
			wantErr:    false,
		},
		{
			name:       "with unicode",
			budgetName: "Бюджет на месяц",
			wantErr:    false,
		},
		{
			name:       "maximum length",
			budgetName: strings.Repeat("a", 255),
			wantErr:    false,
		},

		// Invalid names
		{
			name:       "empty",
			budgetName: "",
			wantErr:    true,
		},
		{
			name:       "only spaces",
			budgetName: "   ",
			wantErr:    true,
		},
		{
			name:       "too long",
			budgetName: strings.Repeat("a", 255+1),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateBudgetName(tt.budgetName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBudgetName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "positive integer",
			amount:  100.00,
			wantErr: false,
		},
		{
			name:    "positive decimal",
			amount:  99.99,
			wantErr: false,
		},
		{
			name:    "small amount",
			amount:  0.01,
			wantErr: false,
		},
		{
			name:    "large amount",
			amount:  999999.99,
			wantErr: false,
		},
		{
			name:    "maximum valid amount",
			amount:  999999999.99,
			wantErr: false,
		},

		// Invalid amounts
		{
			name:    "zero",
			amount:  0,
			wantErr: true,
		},
		{
			name:    "negative",
			amount:  -100,
			wantErr: true,
		},
		{
			name:    "too large",
			amount:  999999999.99 + 0.01,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     bool
	}{
		{
			name:        "simple description",
			description: "Grocery shopping",
			wantErr:     false,
		},
		{
			name:        "with numbers",
			description: "Invoice #123",
			wantErr:     false,
		},
		{
			name:        "with special chars",
			description: "Coffee & snacks @ cafe",
			wantErr:     false,
		},
		{
			name:        "with unicode",
			description: "Покупка продуктов",
			wantErr:     false,
		},
		{
			name:        "maximum length",
			description: strings.Repeat("a", 1000),
			wantErr:     false,
		},
		{
			name:        "with newlines",
			description: "Line 1\nLine 2",
			wantErr:     false,
		},

		// Invalid descriptions
		{
			name:        "empty",
			description: "",
			wantErr:     true,
		},
		{
			name:        "only spaces",
			description: "   ",
			wantErr:     true,
		},
		{
			name:        "too long",
			description: strings.Repeat("a", 1000+1),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateDescription(tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDescription() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategoryName(t *testing.T) {
	tests := []struct {
		name         string
		categoryName string
		wantErr      bool
	}{
		{
			name:         "simple name",
			categoryName: "Food",
			wantErr:      false,
		},
		{
			name:         "with spaces",
			categoryName: "Public Transport",
			wantErr:      false,
		},
		{
			name:         "with numbers",
			categoryName: "Category 1",
			wantErr:      false,
		},
		{
			name:         "with unicode",
			categoryName: "Продукты",
			wantErr:      false,
		},
		{
			name:         "maximum length",
			categoryName: strings.Repeat("a", 255),
			wantErr:      false,
		},

		// Invalid names
		{
			name:         "empty",
			categoryName: "",
			wantErr:      true,
		},
		{
			name:         "only spaces",
			categoryName: "   ",
			wantErr:      true,
		},
		{
			name:         "too long",
			categoryName: strings.Repeat("a", 255+1),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateCategoryName(tt.categoryName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategoryName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportType(t *testing.T) {
	tests := []struct {
		name       string
		reportType report.Type
		wantErr    bool
	}{
		{
			name:       "expenses type",
			reportType: report.TypeExpenses,
			wantErr:    false,
		},
		{
			name:       "income type",
			reportType: report.TypeIncome,
			wantErr:    false,
		},
		{
			name:       "budget type",
			reportType: report.TypeBudget,
			wantErr:    false,
		},
		{
			name:       "cash flow type",
			reportType: report.TypeCashFlow,
			wantErr:    false,
		},
		{
			name:       "category breakdown type",
			reportType: report.TypeCategoryBreak,
			wantErr:    false,
		},

		// Invalid types
		{
			name:       "invalid - empty",
			reportType: report.Type(""),
			wantErr:    true,
		},
		{
			name:       "invalid - unknown",
			reportType: report.Type("unknown"),
			wantErr:    true,
		},
		{
			name:       "invalid - uppercase",
			reportType: report.Type("EXPENSES"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateReportType(tt.reportType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReportType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportPeriod(t *testing.T) {
	tests := []struct {
		name    string
		period  report.Period
		wantErr bool
	}{
		{
			name:    "daily period",
			period:  report.PeriodDaily,
			wantErr: false,
		},
		{
			name:    "weekly period",
			period:  report.PeriodWeekly,
			wantErr: false,
		},
		{
			name:    "monthly period",
			period:  report.PeriodMonthly,
			wantErr: false,
		},
		{
			name:    "yearly period",
			period:  report.PeriodYearly,
			wantErr: false,
		},
		{
			name:    "custom period",
			period:  report.PeriodCustom,
			wantErr: false,
		},

		// Invalid periods
		{
			name:    "invalid - empty",
			period:  report.Period(""),
			wantErr: true,
		},
		{
			name:    "invalid - unknown",
			period:  report.Period("biweekly"),
			wantErr: true,
		},
		{
			name:    "invalid - uppercase",
			period:  report.Period("MONTHLY"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateReportPeriod(tt.period)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReportPeriod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReportName(t *testing.T) {
	tests := []struct {
		name       string
		reportName string
		wantErr    bool
	}{
		{
			name:       "simple name",
			reportName: "Monthly Report",
			wantErr:    false,
		},
		{
			name:       "with numbers",
			reportName: "Report 2024",
			wantErr:    false,
		},
		{
			name:       "with special chars",
			reportName: "Report (January)",
			wantErr:    false,
		},
		{
			name:       "with unicode",
			reportName: "Отчет за месяц",
			wantErr:    false,
		},
		{
			name:       "maximum length",
			reportName: strings.Repeat("a", 255),
			wantErr:    false,
		},

		// Invalid names
		{
			name:       "empty",
			reportName: "",
			wantErr:    true,
		},
		{
			name:       "only spaces",
			reportName: "   ",
			wantErr:    true,
		},
		{
			name:       "too long",
			reportName: strings.Repeat("a", 255+1),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateReportName(tt.reportName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReportName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		wantErr  bool
	}{
		// Valid currencies
		{
			name:     "USD",
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "EUR",
			currency: "EUR",
			wantErr:  false,
		},
		{
			name:     "GBP",
			currency: "GBP",
			wantErr:  false,
		},
		{
			name:     "JPY",
			currency: "JPY",
			wantErr:  false,
		},
		{
			name:     "RUB",
			currency: "RUB",
			wantErr:  false,
		},
		{
			name:     "lowercase converted to uppercase",
			currency: "usd",
			wantErr:  false,
		},
		{
			name:     "with spaces trimmed",
			currency: "  EUR  ",
			wantErr:  false,
		},

		// Invalid currencies
		{
			name:     "empty",
			currency: "",
			wantErr:  true,
		},
		{
			name:     "too short",
			currency: "US",
			wantErr:  true,
		},
		{
			name:     "too long",
			currency: "USDD",
			wantErr:  true,
		},
		{
			name:     "with numbers",
			currency: "US1",
			wantErr:  true,
		},
		{
			name:     "with special chars",
			currency: "US$",
			wantErr:  true,
		},
		{
			name:     "unsupported currency",
			currency: "XXX",
			wantErr:  true,
		},
		{
			name:     "only spaces",
			currency: "   ",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateCurrency(tt.currency)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCurrency() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFamilyName(t *testing.T) {
	tests := []struct {
		name       string
		familyName string
		want       string
	}{
		{
			name:       "already clean",
			familyName: "Smith Family",
			want:       "Smith Family",
		},
		{
			name:       "with leading spaces",
			familyName: "  Smith Family",
			want:       "Smith Family",
		},
		{
			name:       "with trailing spaces",
			familyName: "Smith Family  ",
			want:       "Smith Family",
		},
		{
			name:       "with both spaces",
			familyName: "  Smith Family  ",
			want:       "Smith Family",
		},
		{
			name:       "with tabs",
			familyName: "\tSmith Family\t",
			want:       "Smith Family",
		},
		{
			name:       "with newlines",
			familyName: "\nSmith Family\n",
			want:       "Smith Family",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.SanitizeFamilyName(tt.familyName)
			if got != tt.want {
				t.Errorf("SanitizeFamilyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Edge cases and security tests

func TestValidateEmail_SecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "XSS attempt - script tag",
			email:   "<script>alert('xss')</script>@example.com",
			wantErr: true,
		},
		{
			name:    "null byte",
			email:   "user\x00@example.com",
			wantErr: true,
		},
		{
			name:    "LDAP injection attempt",
			email:   "user*@example.com",
			wantErr: true, // * is not allowed by our strict email regex
		},
		{
			name:    "command injection attempt",
			email:   "user;ls@example.com",
			wantErr: true,
		},
		{
			name:    "Unicode homograph attack",
			email:   "аdmin@example.com", // 'а' is Cyrillic
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() security edge case error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCategoryName_SecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		categoryName string
		wantErr      bool
	}{
		{
			name:         "with emoji",
			categoryName: "Food 🍕",
			wantErr:      false,
		},
		{
			name:         "with Chinese characters",
			categoryName: "食物",
			wantErr:      false,
		},
		{
			name:         "with Arabic characters",
			categoryName: "طعام",
			wantErr:      false,
		},
		{
			name:         "with mixed scripts",
			categoryName: "Food продукты 食物",
			wantErr:      false,
		},
		{
			name:         "with tab character",
			categoryName: "Food\tCategory",
			wantErr:      false,
		},
		{
			name:         "only tab",
			categoryName: "\t",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateCategoryName(tt.categoryName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCategoryName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAmount_BoundaryValues(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{
			name:    "minimum positive value",
			amount:  0.00001,
			wantErr: false,
		},
		{
			name:    "just below zero",
			amount:  -0.00001,
			wantErr: true,
		},
		{
			name:    "exactly at max boundary",
			amount:  999999999.99,
			wantErr: false,
		},
		{
			name:    "just above max boundary",
			amount:  1000000000.00,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateAmount(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAmount() boundary test error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBudgetName_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name       string
		budgetName string
		wantErr    bool
	}{
		{
			name:       "multiple spaces in middle",
			budgetName: "Monthly    Budget",
			wantErr:    false,
		},
		{
			name:       "tab character",
			budgetName: "Monthly\tBudget",
			wantErr:    false,
		},
		{
			name:       "newline character",
			budgetName: "Monthly\nBudget",
			wantErr:    false,
		},
		{
			name:       "mixed whitespace",
			budgetName: "  Monthly \t Budget \n ",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateBudgetName(tt.budgetName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBudgetName() whitespace test error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests for performance-critical validations

func BenchmarkValidateEmail(b *testing.B) {
	email := "user@example.com"
	for range b.N {
		_ = validation.ValidateEmail(email)
	}
}

func BenchmarkValidateAmount(b *testing.B) {
	amount := 100.50
	for range b.N {
		_ = validation.ValidateAmount(amount)
	}
}

func BenchmarkValidateCurrency(b *testing.B) {
	currency := "USD"
	for range b.N {
		_ = validation.ValidateCurrency(currency)
	}
}

func BenchmarkSanitizeEmail(b *testing.B) {
	email := "  User@Example.COM  "
	for range b.N {
		_ = validation.SanitizeEmail(email)
	}
}

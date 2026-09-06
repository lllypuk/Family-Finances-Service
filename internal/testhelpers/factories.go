package testhelpers

import (
	"fmt"
	"time"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"

	"github.com/google/uuid"
)

const (
	// TestTransactionAmountMinor test transaction amount in minor units (100.50)
	TestTransactionAmountMinor = money.Minor(10_050)
	// TestBudgetAmountMinor test budget amount in minor units (1000.00)
	TestBudgetAmountMinor = money.Minor(100_000)
	// TestReportExpensesMinor represents the test report expenses amount in minor units (500.00)
	TestReportExpensesMinor = money.Minor(50_000)
	// testPeriodDays — длина периода тестовых бюджета и отчёта в днях
	testPeriodDays = 30
)

// CreateTestFamily creates a test family
func CreateTestFamily() *user.Family {
	return &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "RUB",
		Timezone:  "Europe/Moscow",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestUser creates a test user
func CreateTestUser(_ uuid.UUID) *user.User {
	return &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("john.doe+%s@example.com", uuid.New().String()),
		Password:  "hashed_password_for_testing", // Required for database constraint
		Role:      user.RoleAdmin,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestCategory creates a test category
func CreateTestCategory(_ uuid.UUID, categoryType category.Type) *category.Category {
	return &category.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Type:      categoryType,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestTransaction creates a test transaction
func CreateTestTransaction(
	_ uuid.UUID, userID, categoryID uuid.UUID,
	transactionType transaction.Type,
) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		CategoryID:  categoryID,
		AmountMinor: TestTransactionAmountMinor,
		Type:        transactionType,
		Description: "Test transaction",
		Date:        date.Today(time.UTC),
		Tags:        []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestBudget creates a test budget
func CreateTestBudget(_ uuid.UUID, categoryID uuid.UUID) *budget.Budget {
	return &budget.Budget{
		ID:         uuid.New(),
		CategoryID: &categoryID,
		// Имя уникально: UNIQUE (family_id, name, start_date, end_date), а даты теперь
		// календарные — два бюджета одного дня иначе конфликтуют.
		Name:        fmt.Sprintf("Test Budget %s", uuid.New().String()),
		AmountMinor: TestBudgetAmountMinor,
		SpentMinor:  0,
		Period:      budget.PeriodMonthly,
		StartDate:   date.Today(time.UTC),
		EndDate:     date.Today(time.UTC).AddDays(testPeriodDays),
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestReport creates a test report
func CreateTestReport(_ uuid.UUID, userID uuid.UUID) *report.Report {
	return &report.Report{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        "Test Report",
		Type:        report.TypeExpenses,
		Period:      report.PeriodMonthly,
		StartDate:   date.Today(time.UTC).AddDays(-testPeriodDays),
		EndDate:     date.Today(time.UTC),
		Data:        report.Data{TotalExpensesMinor: TestReportExpensesMinor},
		GeneratedAt: time.Now(),
	}
}

package testhelpers

import (
	"fmt"
	"time"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"

	"github.com/google/uuid"
)

const (
	// TestTransactionAmount test transaction amount
	TestTransactionAmount = 100.50
	// TestBudgetAmount test budget amount
	TestBudgetAmount = 1000.0
	// TestReportExpenses represents the test report expenses amount
	TestReportExpenses = 500.0
)

// CreateTestFamily creates a test family
func CreateTestFamily() *user.Family {
	return &user.Family{
		ID:        uuid.New(),
		Name:      "Test Family",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestUser creates a test user
func CreateTestUser(familyID uuid.UUID) *user.User {
	return &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("john.doe+%s@example.com", uuid.New().String()),
		Password:  "hashed_password_for_testing", // Required for database constraint
		Role:      user.RoleAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestCategory creates a test category
func CreateTestCategory(familyID uuid.UUID, categoryType category.Type) *category.Category {
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
	familyID, userID, categoryID uuid.UUID,
	transactionType transaction.Type,
) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		CategoryID:  categoryID,
		Amount:      TestTransactionAmount,
		Type:        transactionType,
		Description: "Test transaction",
		Date:        time.Now(),
		Tags:        []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateTestBudget creates a test budget
func CreateTestBudget(familyID, categoryID uuid.UUID) *budget.Budget {
	return &budget.Budget{
		ID:         uuid.New(),
		CategoryID: &categoryID,
		Name:       "Test Budget",
		Amount:     TestBudgetAmount,
		Spent:      0.0,
		Period:     budget.PeriodMonthly,
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 1, 0),
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// CreateTestReport creates a test report
func CreateTestReport(familyID, userID uuid.UUID) *report.Report {
	return &report.Report{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        "Test Report",
		Type:        report.TypeExpenses,
		Period:      report.PeriodMonthly,
		StartDate:   time.Now().AddDate(0, -1, 0),
		EndDate:     time.Now(),
		Data:        report.Data{TotalExpenses: TestReportExpenses},
		GeneratedAt: time.Now(),
	}
}

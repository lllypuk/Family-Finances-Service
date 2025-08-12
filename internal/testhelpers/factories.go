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
		FamilyID:  familyID,
		FirstName: "John",
		LastName:  "Doe",
		Email:     fmt.Sprintf("john.doe+%s@example.com", uuid.New().String()),
		Role:      user.RoleAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestCategory creates a test category
func CreateTestCategory(familyID uuid.UUID, categoryType category.CategoryType) *category.Category {
	return &category.Category{
		ID:        uuid.New(),
		FamilyID:  familyID,
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
	transactionType transaction.TransactionType,
) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          uuid.New(),
		FamilyID:    familyID,
		UserID:      userID,
		CategoryID:  categoryID,
		Amount:      100.50,
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
		FamilyID:   familyID,
		CategoryID: &categoryID,
		Name:       "Test Budget",
		Amount:     1000.0,
		Spent:      0.0,
		Period:     budget.BudgetPeriodMonthly,
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
		FamilyID:    familyID,
		UserID:      userID,
		Name:        "Test Report",
		Type:        report.ReportTypeExpenses,
		Period:      report.ReportPeriodMonthly,
		StartDate:   time.Now().AddDate(0, -1, 0),
		EndDate:     time.Now(),
		Data:        report.ReportData{TotalExpenses: 500.0},
		GeneratedAt: time.Now(),
	}
}

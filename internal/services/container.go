package services

import (
	"context"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/report"
)

// ReportRepository defines the interface for report data access
type ReportRepository interface {
	Create(ctx context.Context, report *report.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Services contains all business services
type Services struct {
	User        UserService
	Family      FamilyService
	Category    CategoryService
	Transaction TransactionService
	Budget      BudgetService
	Report      ReportService
}

// NewServices creates a new services container with all dependencies
func NewServices(
	userRepo UserRepository,
	familyRepo FamilyRepository,
	categoryRepo CategoryRepository,
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	fullBudgetRepo BudgetRepository,
	reportRepo ReportRepository,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)

	// Create core services first
	userService := NewUserService(userRepo, familyRepo)
	familyService := NewFamilyService(familyRepo)
	categoryService := NewCategoryService(categoryRepo, familyRepo, usageChecker)
	transactionService := NewTransactionService(transactionRepo, budgetRepo, categoryRepo, userRepo)
	budgetService := NewBudgetService(fullBudgetRepo, transactionRepo)

	// Create report service with dependencies on other services
	reportService := NewReportService(
		reportRepo,
		transactionRepo,
		fullBudgetRepo,
		categoryRepo,
		userRepo,
		transactionService,
		budgetService,
		categoryService,
	)

	return &Services{
		User:        userService,
		Family:      familyService,
		Category:    categoryService,
		Transaction: transactionService,
		Budget:      budgetService,
		Report:      reportService,
	}
}

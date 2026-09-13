package services

import (
	"log/slog"

	"family-budget-service/internal/auth"
)

// Services contains all business services
type Services struct {
	User        UserService
	Family      FamilyService
	Category    CategoryService
	Transaction TransactionService
	Budget      BudgetService
	Stats       StatsService
	Backup      BackupService
	// Auth — bearer-сессии; собирается снаружи, как и Backup: ему нужны репозитории, а не сервисы.
	Auth *auth.Service
}

// NewServices creates a new services container with all dependencies
func NewServices(
	userRepo UserRepository,
	familyRepo FamilyRepository,
	categoryRepo CategoryRepository,
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	fullBudgetRepo BudgetRepository,
	backupService BackupService,
	authService *auth.Service,
	logger *slog.Logger,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)

	// Create core services first
	userService := NewUserService(userRepo, familyRepo)
	categoryService := NewCategoryService(categoryRepo, familyRepo, usageChecker)
	familyService := NewFamilyService(familyRepo, transactionRepo)
	transactionService := NewTransactionServiceWithLogger(transactionRepo, budgetRepo, categoryRepo, userRepo, logger)
	budgetService := NewBudgetServiceWithLogger(fullBudgetRepo, transactionRepo, logger)

	statsService := NewStatsService(transactionService, budgetService, categoryService, familyService, transactionRepo)

	return &Services{
		User:        userService,
		Family:      familyService,
		Category:    categoryService,
		Transaction: transactionService,
		Budget:      budgetService,
		Stats:       statsService,
		Backup:      backupService,
		Auth:        authService,
	}
}

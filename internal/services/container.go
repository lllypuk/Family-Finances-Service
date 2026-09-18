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
	Account     AccountService
	Transaction TransactionService
	Budget      BudgetService
	Stats       StatsService
	Backup      BackupService
	Recognize   RecognizeService
	// Auth — bearer-сессии; собирается снаружи, как и Backup: ему нужны репозитории, а не сервисы.
	Auth *auth.Service
}

// NewServices creates a new services container with all dependencies
func NewServices(
	userRepo UserRepository,
	familyRepo FamilyRepository,
	categoryRepo CategoryRepository,
	accountRepo AccountRepository,
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	fullBudgetRepo BudgetRepository,
	backupService BackupService,
	authService *auth.Service,
	recognizer Recognizer,
	recognizeObserver RecognizeObserver,
	logger *slog.Logger,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)

	// Create core services first
	userService := NewUserService(userRepo, familyRepo)
	categoryService := NewCategoryService(categoryRepo, familyRepo, usageChecker)
	familyService := NewFamilyService(familyRepo, transactionRepo)
	transactionService := NewTransactionServiceWithLogger(
		transactionRepo, budgetRepo, categoryRepo, userRepo, accountRepo, logger,
	)
	budgetService := NewBudgetServiceWithLogger(fullBudgetRepo, transactionRepo, logger)

	statsService := NewStatsService(transactionService, budgetService, categoryService, familyService, transactionRepo)

	return &Services{
		User:        userService,
		Family:      familyService,
		Category:    categoryService,
		Account:     NewAccountService(accountRepo),
		Transaction: transactionService,
		Budget:      budgetService,
		Stats:       statsService,
		Backup:      backupService,
		Recognize:   NewRecognizeService(recognizer, familyRepo, categoryRepo, transactionRepo, recognizeObserver),
		Auth:        authService,
	}
}

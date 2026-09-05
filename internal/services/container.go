package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/user"
)

// ReportRepository defines the interface for report data access
type ReportRepository interface {
	Create(ctx context.Context, report *report.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	GetAll(ctx context.Context) ([]*report.Report, error)
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
	Stats       StatsService
	Report      ReportService
	Invite      InviteService
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
	reportRepo ReportRepository,
	inviteRepo user.InviteRepository,
	backupService BackupService,
	authService *auth.Service,
	logger *slog.Logger,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)

	// Create core services first
	userService := NewUserService(userRepo, familyRepo, authService)
	categoryService := NewCategoryService(categoryRepo, familyRepo, usageChecker)
	familyService := NewFamilyService(familyRepo, transactionRepo)
	transactionService := NewTransactionServiceWithLogger(transactionRepo, budgetRepo, categoryRepo, userRepo, logger)
	budgetService := NewBudgetServiceWithLogger(fullBudgetRepo, transactionRepo, logger)
	inviteService := NewInviteService(inviteRepo, userRepo, familyRepo, logger)

	statsService := NewStatsService(transactionService, budgetService, categoryService)

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
		Stats:       statsService,
		Report:      reportService,
		Invite:      inviteService,
		Backup:      backupService,
		Auth:        authService,
	}
}

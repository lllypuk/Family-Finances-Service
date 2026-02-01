package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

// UserService defines business operations for user management
type UserService interface {
	// CRUD Operations
	CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetUsers(ctx context.Context) ([]*user.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// Business Operations
	ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error
	ValidateUserAccess(ctx context.Context, userID, resourceOwnerID uuid.UUID) error
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
}

// FamilyService defines business operations for the single family
type FamilyService interface {
	SetupFamily(ctx context.Context, req dto.SetupFamilyDTO) (*user.Family, error)
	GetFamily(ctx context.Context) (*user.Family, error)
	UpdateFamily(ctx context.Context, req dto.UpdateFamilyDTO) (*user.Family, error)
	IsSetupComplete(ctx context.Context) (bool, error)
}

// CategoryService defines business operations for category management
type CategoryService interface {
	// CRUD Operations
	CreateCategory(ctx context.Context, req dto.CreateCategoryDTO) (*category.Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
	GetCategories(
		ctx context.Context,
		typeFilter *category.Type,
	) ([]*category.Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, req dto.UpdateCategoryDTO) (*category.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetCategoryHierarchy(ctx context.Context) ([]*category.Category, error)
	ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error
	CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error)
	CreateDefaultCategories(ctx context.Context) error
}

// TransactionService defines business operations for transaction management
type TransactionService interface {
	// CRUD Operations
	CreateTransaction(ctx context.Context, req dto.CreateTransactionDTO) (*transaction.Transaction, error)
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetAllTransactions(
		ctx context.Context,
		filter dto.TransactionFilterDTO,
	) ([]*transaction.Transaction, error)
	UpdateTransaction(ctx context.Context, id uuid.UUID, req dto.UpdateTransactionDTO) (*transaction.Transaction, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetTransactionsByCategory(
		ctx context.Context,
		categoryID uuid.UUID,
		filter dto.TransactionFilterDTO,
	) ([]*transaction.Transaction, error)
	GetTransactionsByDateRange(
		ctx context.Context,
		from, to time.Time,
	) ([]*transaction.Transaction, error)
	BulkCategorizeTransactions(
		ctx context.Context,
		transactionIDs []uuid.UUID,
		categoryID uuid.UUID,
	) error
	ValidateTransactionLimits(
		ctx context.Context,
		categoryID uuid.UUID,
		amount float64,
		transactionType transaction.Type,
	) error
}

// BudgetService defines business operations for budget management and calculations
type BudgetService interface {
	// CRUD Operations
	CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error)
	GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
	GetAllBudgets(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, error) // Single family
	UpdateBudget(ctx context.Context, id uuid.UUID, req dto.UpdateBudgetDTO) (*budget.Budget, error)
	DeleteBudget(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetActiveBudgets(ctx context.Context, date time.Time) ([]*budget.Budget, error) // Single family
	UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error
	CheckBudgetLimits(ctx context.Context, categoryID uuid.UUID, amount float64) error // Single family
	GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error)
	CalculateBudgetUtilization(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetUtilizationDTO, error)
	GetBudgetsByCategory(ctx context.Context, categoryID uuid.UUID) ([]*budget.Budget, error) // Single family
	ValidateBudgetPeriod(
		ctx context.Context,
		categoryID *uuid.UUID,
		startDate, endDate time.Time,
	) error
	RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error
}

// ReportService defines business operations for report generation and analytics
type ReportService interface {
	// Report Generation
	GenerateExpenseReport(ctx context.Context, req dto.ReportRequestDTO) (*dto.ExpenseReportDTO, error)
	GenerateIncomeReport(ctx context.Context, req dto.ReportRequestDTO) (*dto.IncomeReportDTO, error)
	GenerateBudgetComparisonReport(
		ctx context.Context,
		period report.Period,
	) (*dto.BudgetComparisonDTO, error)
	GenerateCashFlowReport(ctx context.Context, from, to time.Time) (*dto.CashFlowReportDTO, error)
	GenerateCategoryBreakdownReport(
		ctx context.Context,
		period report.Period,
	) (*dto.CategoryBreakdownDTO, error)

	// Report Management
	SaveReport(
		ctx context.Context,
		reportData any,
		reportType report.Type,
		req dto.ReportRequestDTO,
	) (*report.Report, error)
	GetReportByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	GetReports(ctx context.Context, typeFilter *report.Type) ([]*report.Report, error)
	DeleteReport(ctx context.Context, id uuid.UUID) error

	// Export Operations
	ExportReport(ctx context.Context, reportID uuid.UUID, format string, options dto.ExportOptionsDTO) ([]byte, error)
	ExportReportData(ctx context.Context, reportData any, format string, options dto.ExportOptionsDTO) ([]byte, error)

	// Scheduled Reports
	ScheduleReport(ctx context.Context, req dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error)
	GetScheduledReports(ctx context.Context) ([]*dto.ScheduledReportDTO, error)
	UpdateScheduledReport(ctx context.Context, id uuid.UUID, req dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error)
	DeleteScheduledReport(ctx context.Context, id uuid.UUID) error
	ExecuteScheduledReport(ctx context.Context, scheduledReportID uuid.UUID) error

	// Analytics & Insights
	GenerateTrendAnalysis(
		ctx context.Context,
		categoryID *uuid.UUID,
		period report.Period,
	) (*dto.TrendAnalysisDTO, error)
	GenerateSpendingForecast(ctx context.Context, months int) ([]dto.ForecastDTO, error)
	GenerateFinancialInsights(ctx context.Context) ([]dto.RecommendationDTO, error)
	CalculateBenchmarks(ctx context.Context) (*dto.BenchmarkComparisonDTO, error)
}

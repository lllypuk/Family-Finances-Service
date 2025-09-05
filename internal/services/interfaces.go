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
	GetUsersByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// Business Operations
	ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error
	ValidateUserAccess(ctx context.Context, userID, resourceOwnerID uuid.UUID) error
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
}

// FamilyService defines business operations for family management
type FamilyService interface {
	CreateFamily(ctx context.Context, req dto.CreateFamilyDTO) (*user.Family, error)
	GetFamilyByID(ctx context.Context, id uuid.UUID) (*user.Family, error)
	UpdateFamily(ctx context.Context, id uuid.UUID, req dto.UpdateFamilyDTO) (*user.Family, error)
	DeleteFamily(ctx context.Context, id uuid.UUID) error
}

// CategoryService defines business operations for category management
type CategoryService interface {
	// CRUD Operations
	CreateCategory(ctx context.Context, req dto.CreateCategoryDTO) (*category.Category, error)
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
	GetCategoriesByFamily(
		ctx context.Context,
		familyID uuid.UUID,
		typeFilter *category.Type,
	) ([]*category.Category, error)
	UpdateCategory(ctx context.Context, id uuid.UUID, req dto.UpdateCategoryDTO) (*category.Category, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetCategoryHierarchy(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error)
	ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error
	CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error)
	CreateDefaultCategories(ctx context.Context, familyID uuid.UUID) error
}

// TransactionService defines business operations for transaction management
type TransactionService interface {
	// CRUD Operations
	CreateTransaction(ctx context.Context, req dto.CreateTransactionDTO) (*transaction.Transaction, error)
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetTransactionsByFamily(
		ctx context.Context,
		familyID uuid.UUID,
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
		familyID uuid.UUID,
		from, to time.Time,
	) ([]*transaction.Transaction, error)
	BulkCategorizeTransactions(
		ctx context.Context,
		transactionIDs []uuid.UUID,
		categoryID uuid.UUID,
	) error
	ValidateTransactionLimits(
		ctx context.Context,
		familyID uuid.UUID,
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
	GetBudgetsByFamily(ctx context.Context, familyID uuid.UUID, filter dto.BudgetFilterDTO) ([]*budget.Budget, error)
	UpdateBudget(ctx context.Context, id uuid.UUID, req dto.UpdateBudgetDTO) (*budget.Budget, error)
	DeleteBudget(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetActiveBudgets(ctx context.Context, familyID uuid.UUID, date time.Time) ([]*budget.Budget, error)
	UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error
	CheckBudgetLimits(ctx context.Context, familyID uuid.UUID, categoryID uuid.UUID, amount float64) error
	GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error)
	CalculateBudgetUtilization(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetUtilizationDTO, error)
	GetBudgetsByCategory(ctx context.Context, familyID uuid.UUID, categoryID uuid.UUID) ([]*budget.Budget, error)
	ValidateBudgetPeriod(
		ctx context.Context,
		familyID uuid.UUID,
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
		familyID uuid.UUID,
		period report.Period,
	) (*dto.BudgetComparisonDTO, error)
	GenerateCashFlowReport(ctx context.Context, familyID uuid.UUID, from, to time.Time) (*dto.CashFlowReportDTO, error)
	GenerateCategoryBreakdownReport(
		ctx context.Context,
		familyID uuid.UUID,
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
	GetReportsByFamily(ctx context.Context, familyID uuid.UUID, typeFilter *report.Type) ([]*report.Report, error)
	DeleteReport(ctx context.Context, id uuid.UUID) error

	// Export Operations
	ExportReport(ctx context.Context, reportID uuid.UUID, format string, options dto.ExportOptionsDTO) ([]byte, error)
	ExportReportData(ctx context.Context, reportData any, format string, options dto.ExportOptionsDTO) ([]byte, error)

	// Scheduled Reports
	ScheduleReport(ctx context.Context, req dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error)
	GetScheduledReports(ctx context.Context, familyID uuid.UUID) ([]*dto.ScheduledReportDTO, error)
	UpdateScheduledReport(ctx context.Context, id uuid.UUID, req dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error)
	DeleteScheduledReport(ctx context.Context, id uuid.UUID) error
	ExecuteScheduledReport(ctx context.Context, scheduledReportID uuid.UUID) error

	// Analytics & Insights
	GenerateTrendAnalysis(
		ctx context.Context,
		familyID uuid.UUID,
		categoryID *uuid.UUID,
		period report.Period,
	) (*dto.TrendAnalysisDTO, error)
	GenerateSpendingForecast(ctx context.Context, familyID uuid.UUID, months int) ([]dto.ForecastDTO, error)
	GenerateFinancialInsights(ctx context.Context, familyID uuid.UUID) ([]dto.RecommendationDTO, error)
	CalculateBenchmarks(ctx context.Context, familyID uuid.UUID) (*dto.BenchmarkComparisonDTO, error)
}

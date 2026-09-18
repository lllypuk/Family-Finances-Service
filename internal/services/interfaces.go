package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/recognize"
	"family-budget-service/internal/services/dto"
)

// UserService defines business operations for user management
type UserService interface {
	// CRUD Operations
	CreateUser(ctx context.Context, req dto.CreateUserDTO) (*user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetUsers(ctx context.Context) ([]*user.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserDTO) (*user.User, error)
	// PatchUser пишет роль и активность одной записью; деактивация отзывает сессии,
	// себя и последнего активного админа выключить нельзя.
	PatchUser(ctx context.Context, id uuid.UUID, role *user.Role, active *bool, actorID uuid.UUID) error

	// Business Operations
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
}

// AccountService — справочник счетов; id в Create — клиентский, nil — сгенерировать.
type AccountService interface {
	Create(ctx context.Context, id *uuid.UUID, name string) (*account.Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error)
	List(ctx context.Context, includeArchived bool) ([]*account.Account, error)
	Update(ctx context.Context, id uuid.UUID, name *string, archived *bool) (*account.Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// HoldingService — справочник активов и пассивов; current считается на сегодня в зоне семьи.
type HoldingService interface {
	Create(
		ctx context.Context,
		id *uuid.UUID,
		name string,
		side holding.Side,
		kind holding.Kind,
	) (*holding.Holding, error)
	GetByID(ctx context.Context, id uuid.UUID) (*holding.Holding, error)
	List(ctx context.Context, includeArchived bool) ([]*holding.Holding, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		name *string,
		kind *holding.Kind,
		archived *bool,
	) (*holding.Holding, error)
	Delete(ctx context.Context, id uuid.UUID) error
	PutValue(ctx context.Context, id uuid.UUID, day date.Date, value money.Minor) (*holding.Value, error)
	DeleteValue(ctx context.Context, id uuid.UUID, day date.Date) error
	ListValues(ctx context.Context, id uuid.UUID, limit, offset int) ([]*holding.Value, int, error)
}

// ReconciliationService — сверка счетов по месяцам; month — любой день месяца.
type ReconciliationService interface {
	// Put заменяет сверку целиком; неизвестный счёт — account.ErrNotFound.
	Put(
		ctx context.Context,
		accountID uuid.UUID,
		month date.Date,
		bankExpense money.Minor,
		note string,
	) (*reconciliation.Reconciliation, error)
	Delete(ctx context.Context, accountID uuid.UUID, month date.Date) error
	// Summary — записанное против банка по счетам; nil — текущий месяц в поясе семьи.
	Summary(ctx context.Context, month *date.Date) (*dto.ReconciliationStats, error)
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
	// CountTransactions возвращает общее число транзакций под фильтр, без учёта Limit/Offset.
	CountTransactions(ctx context.Context, filter dto.TransactionFilterDTO) (int, error)
	UpdateTransaction(ctx context.Context, id uuid.UUID, req dto.UpdateTransactionDTO) (*transaction.Transaction, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error
	// BulkDelete удаляет транзакции одним запросом и возвращает число удалённых;
	// отсутствующие id игнорируются.
	BulkDelete(ctx context.Context, ids []uuid.UUID) (int, error)

	// Business Operations
	GetTransactionsByCategory(
		ctx context.Context,
		categoryID uuid.UUID,
		filter dto.TransactionFilterDTO,
	) ([]*transaction.Transaction, error)
	GetTransactionsByDateRange(
		ctx context.Context,
		from, to date.Date,
	) ([]*transaction.Transaction, error)
	BulkCategorizeTransactions(
		ctx context.Context,
		transactionIDs []uuid.UUID,
		categoryID uuid.UUID,
	) error
	// ValidateTransactionLimits — бюджет ищется по календарной дате операции on.
	ValidateTransactionLimits(
		ctx context.Context,
		categoryID uuid.UUID,
		amount money.Minor,
		transactionType transaction.Type,
		on date.Date,
	) error
}

// BudgetService defines business operations for budget management and calculations
type BudgetService interface {
	// CRUD Operations
	CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error)
	GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
	GetAllBudgets(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, error) // Single family
	// GetBudgetsPage — та же выборка плюс общее число подходящих записей, для meta.pagination.
	GetBudgetsPage(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, int, error)
	UpdateBudget(ctx context.Context, id uuid.UUID, req dto.UpdateBudgetDTO) (*budget.Budget, error)
	DeleteBudget(ctx context.Context, id uuid.UUID) error

	// Business Operations
	GetActiveBudgets(ctx context.Context, on date.Date) ([]*budget.Budget, error) // Single family
	UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount money.Minor) error
	CheckBudgetLimits(ctx context.Context, categoryID uuid.UUID, amount money.Minor, on date.Date) error
	GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error)
	CalculateBudgetUtilization(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetUtilizationDTO, error)
	GetBudgetsByCategory(ctx context.Context, categoryID uuid.UUID) ([]*budget.Budget, error) // Single family
	RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error
}

// StatsService defines aggregated statistics over a period (dashboard, GET /stats/summary).
// Пустые границы означают текущий месяц по часовому поясу семьи.
type StatsService interface {
	Summary(ctx context.Context, from, to *date.Date) (*dto.StatsSummary, error)
	// Monthly — ряд по месяцам периода; пустые границы означают двенадцать месяцев по сегодняшний.
	Monthly(ctx context.Context, from, to *date.Date) (*dto.StatsMonthly, error)
	// NetWorth — ряд капитала; границы как у Monthly, to позже сегодня — ErrStatsPeriodInFuture.
	NetWorth(ctx context.Context, from, to *date.Date) (*dto.StatsNetWorth, error)
}

// RecognizeService — кандидаты операций со скриншотов; Budget — худший срок вызова модели, ноль у выключенного плеча.
type RecognizeService interface {
	Recognize(ctx context.Context, images []recognize.Image) (recognize.Result, error)
	Budget() time.Duration
}

// BackupService defines business operations for database backup management
type BackupService interface {
	CreateBackup(ctx context.Context) (*BackupInfo, error)
	ListBackups(ctx context.Context) ([]*BackupInfo, error)
	GetBackup(ctx context.Context, filename string) (*BackupInfo, error)
	DeleteBackup(ctx context.Context, filename string) error
	GetBackupFilePath(filename string) string
}

// BackupInfo contains information about a backup file
type BackupInfo struct {
	Filename  string
	Size      int64
	CreatedAt time.Time
}

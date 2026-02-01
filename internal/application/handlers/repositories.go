package handlers

import (
	"context"
	"time"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"

	"github.com/google/uuid"
)

type Repositories struct {
	User        UserRepository
	Family      FamilyRepository
	Category    CategoryRepository
	Transaction TransactionRepository
	Budget      BudgetRepository
	Report      ReportRepository
}

// UserRepository определяет операции с пользователями
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetAll(ctx context.Context) ([]*user.User, error) // Single family - get all users
	Update(ctx context.Context, user *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// FamilyRepository определяет операции с единственной семьёй
type FamilyRepository interface {
	Create(ctx context.Context, family *user.Family) error
	Get(ctx context.Context) (*user.Family, error)
	Update(ctx context.Context, family *user.Family) error
	Exists(ctx context.Context) (bool, error)
}

// CategoryRepository определяет операции с категориями
type CategoryRepository interface {
	Create(ctx context.Context, category *category.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
	GetAll(ctx context.Context) ([]*category.Category, error) // Single family - get all categories
	GetByType(ctx context.Context, categoryType category.Type) ([]*category.Category, error)
	Update(ctx context.Context, category *category.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TransactionRepository определяет операции с транзакциями
type TransactionRepository interface {
	Create(ctx context.Context, transaction *transaction.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetByFilter(ctx context.Context, filter transaction.Filter) ([]*transaction.Transaction, error)
	GetAll(
		ctx context.Context,
		limit, offset int,
	) ([]*transaction.Transaction, error) // Single family - get all transactions
	Update(ctx context.Context, transaction *transaction.Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetTotalByCategory(
		ctx context.Context,
		categoryID uuid.UUID,
		transactionType transaction.Type,
	) (float64, error)
	GetTotalByDateRange(
		ctx context.Context,
		startDate, endDate time.Time,
		transactionType transaction.Type,
	) (float64, error)
	GetTotalByCategoryAndDateRange(
		ctx context.Context,
		categoryID uuid.UUID,
		startDate, endDate time.Time,
		transactionType transaction.Type,
	) (float64, error)
}

// BudgetRepository определяет операции с бюджетами
type BudgetRepository interface {
	Create(ctx context.Context, budget *budget.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
	GetAll(ctx context.Context) ([]*budget.Budget, error)           // Single family - get all budgets
	GetActiveBudgets(ctx context.Context) ([]*budget.Budget, error) // Single family - get active budgets
	Update(ctx context.Context, budget *budget.Budget) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByCategory(
		ctx context.Context,
		categoryID *uuid.UUID,
	) ([]*budget.Budget, error)
	GetByPeriod(
		ctx context.Context,
		startDate, endDate time.Time,
	) ([]*budget.Budget, error)
}

// ReportRepository определяет операции с отчетами
type ReportRepository interface {
	Create(ctx context.Context, report *report.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	GetAll(ctx context.Context) ([]*report.Report, error) // Single family - get all reports
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

package interfaces

import (
	"context"

	"github.com/google/uuid"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// UserRepository определяет операции с пользователями
type UserRepository interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error)
	Update(ctx context.Context, user *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// FamilyRepository определяет операции с семьями
type FamilyRepository interface {
	Create(ctx context.Context, family *user.Family) error
	GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error)
	Update(ctx context.Context, family *user.Family) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CategoryRepository определяет операции с категориями
type CategoryRepository interface {
	Create(ctx context.Context, category *category.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error)
	GetByType(ctx context.Context, familyID uuid.UUID, categoryType category.CategoryType) ([]*category.Category, error)
	Update(ctx context.Context, category *category.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TransactionRepository определяет операции с транзакциями
type TransactionRepository interface {
	Create(ctx context.Context, transaction *transaction.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetByFilter(ctx context.Context, filter transaction.TransactionFilter) ([]*transaction.Transaction, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID, limit, offset int) ([]*transaction.Transaction, error)
	Update(ctx context.Context, transaction *transaction.Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetTotalByCategory(ctx context.Context, categoryID uuid.UUID, transactionType transaction.TransactionType) (float64, error)
}

// BudgetRepository определяет операции с бюджетами
type BudgetRepository interface {
	Create(ctx context.Context, budget *budget.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error)
	GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error)
	Update(ctx context.Context, budget *budget.Budget) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ReportRepository определяет операции с отчетами
type ReportRepository interface {
	Create(ctx context.Context, report *report.Report) error
	GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error)
	GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
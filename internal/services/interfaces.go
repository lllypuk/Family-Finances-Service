package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
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

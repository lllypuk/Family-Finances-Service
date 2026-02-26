package handlers

import (
	"context"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
)

type Repositories struct {
	User        UserRepository
	Family      FamilyRepository
	Category    CategoryRepository
	Transaction TransactionRepository
	Budget      BudgetRepository
	Report      ReportRepository
	Invite      user.InviteRepository
}

// UserRepository переиспользует сервисный контракт.
type UserRepository = services.UserRepository

// FamilyRepository переиспользует сервисный контракт.
type FamilyRepository = services.FamilyRepository

// CategoryRepository переиспользует сервисный контракт.
type CategoryRepository = services.CategoryRepository

// TransactionRepository расширяет сервисный контракт методом, который нужен только хендлерам.
type TransactionRepository interface {
	services.TransactionRepository
	GetAll(
		ctx context.Context,
		limit, offset int,
	) ([]*transaction.Transaction, error) // Single family - get all transactions
}

// BudgetRepository переиспользует сервисный контракт.
type BudgetRepository = services.BudgetRepository

// ReportRepository переиспользует сервисный контракт.
type ReportRepository = services.ReportRepository

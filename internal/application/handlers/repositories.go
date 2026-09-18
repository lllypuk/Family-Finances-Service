package handlers

import (
	"family-budget-service/internal/auth"
	"family-budget-service/internal/services"
)

type Repositories struct {
	User        UserRepository
	Family      FamilyRepository
	Category    CategoryRepository
	Account     AccountRepository
	Transaction TransactionRepository
	Budget      BudgetRepository
	Session     auth.SessionRepository
}

// UserRepository переиспользует сервисный контракт.
type UserRepository = services.UserRepository

// FamilyRepository переиспользует сервисный контракт.
type FamilyRepository = services.FamilyRepository

// CategoryRepository переиспользует сервисный контракт.
type CategoryRepository = services.CategoryRepository

// AccountRepository переиспользует сервисный контракт.
type AccountRepository = services.AccountRepository

// TransactionRepository переиспользует сервисный контракт.
type TransactionRepository = services.TransactionRepository

// BudgetRepository переиспользует сервисный контракт.
type BudgetRepository = services.BudgetRepository

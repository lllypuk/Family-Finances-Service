package handlers

import (
	"family-budget-service/internal/auth"
	"family-budget-service/internal/services"
)

type Repositories struct {
	User        UserRepository
	Family      FamilyRepository
	Category    CategoryRepository
	Transaction TransactionRepository
	Budget      BudgetRepository
	Report      ReportRepository
	Session     auth.SessionRepository
}

// UserRepository переиспользует сервисный контракт.
type UserRepository = services.UserRepository

// FamilyRepository переиспользует сервисный контракт.
type FamilyRepository = services.FamilyRepository

// CategoryRepository переиспользует сервисный контракт.
type CategoryRepository = services.CategoryRepository

// TransactionRepository переиспользует сервисный контракт.
type TransactionRepository = services.TransactionRepository

// BudgetRepository переиспользует сервисный контракт.
type BudgetRepository = services.BudgetRepository

// ReportRepository переиспользует сервисный контракт.
type ReportRepository = services.ReportRepository

package handlers

import (
	repositories "family-budget-service/internal/application/interfaces"
)

type Repositories struct {
	User        repositories.UserRepository
	Family      repositories.FamilyRepository
	Category    repositories.CategoryRepository
	Transaction repositories.TransactionRepository
	Budget      repositories.BudgetRepository
	Report      repositories.ReportRepository
}

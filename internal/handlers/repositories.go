package handlers

import (
	"family-budget-service/internal/interfaces"
)

type Repositories struct {
	User        interfaces.UserRepository
	Family      interfaces.FamilyRepository
	Category    interfaces.CategoryRepository
	Transaction interfaces.TransactionRepository
	Budget      interfaces.BudgetRepository
	Report      interfaces.ReportRepository
}

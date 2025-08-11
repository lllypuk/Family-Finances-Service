package infrastructure

import (
	"family-budget-service/internal/handlers"
	"family-budget-service/internal/infrastructure/budget"
	"family-budget-service/internal/infrastructure/category"
	"family-budget-service/internal/infrastructure/report"
	"family-budget-service/internal/infrastructure/transaction"
	"family-budget-service/internal/infrastructure/user"
)

// NewRepositories создает и возвращает все репозитории с MongoDB подключениями
func NewRepositories(mongodb *MongoDB) *handlers.Repositories {
	return &handlers.Repositories{
		User:        user.NewRepository(mongodb.Database),
		Family:      user.NewFamilyRepository(mongodb.Database),
		Category:    category.NewRepository(mongodb.Database),
		Transaction: transaction.NewRepository(mongodb.Database),
		Budget:      budget.NewRepository(mongodb.Database),
		Report:      report.NewRepository(mongodb.Database),
	}
}
package infrastructure

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/infrastructure/budget"
	"family-budget-service/internal/infrastructure/category"
	"family-budget-service/internal/infrastructure/report"
	"family-budget-service/internal/infrastructure/transaction"
	"family-budget-service/internal/infrastructure/user"
)

// NewRepositories создает и возвращает все репозитории с PostgreSQL подключениями
func NewRepositories(db *pgxpool.Pool) *handlers.Repositories {
	return &handlers.Repositories{
		User:        user.NewPostgreSQLRepository(db),
		Family:      user.NewPostgreSQLFamilyRepository(db),
		Category:    category.NewPostgreSQLRepository(db),
		Transaction: transaction.NewPostgreSQLRepository(db),
		Budget:      budget.NewPostgreSQLRepository(db),
		Report:      report.NewPostgreSQLRepository(db),
	}
}

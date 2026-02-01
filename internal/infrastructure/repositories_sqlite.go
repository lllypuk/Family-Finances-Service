package infrastructure

import (
	"database/sql"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/infrastructure/budget"
	"family-budget-service/internal/infrastructure/category"
	"family-budget-service/internal/infrastructure/report"
	"family-budget-service/internal/infrastructure/transaction"
	"family-budget-service/internal/infrastructure/user"
)

// NewRepositoriesSQLite создает и возвращает все репозитории с SQLite подключениями
func NewRepositoriesSQLite(db *sql.DB) *handlers.Repositories {
	return &handlers.Repositories{
		User:        user.NewSQLiteRepository(db),
		Family:      user.NewSQLiteFamilyRepository(db),
		Category:    category.NewSQLiteRepository(db),
		Transaction: transaction.NewSQLiteRepository(db),
		Budget:      budget.NewSQLiteRepository(db),
		Report:      report.NewSQLiteRepository(db),
		Invite:      user.NewInviteSQLiteRepository(db),
	}
}

package infrastructure

import (
	"database/sql"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/infrastructure/account"
	"family-budget-service/internal/infrastructure/auth"
	"family-budget-service/internal/infrastructure/budget"
	"family-budget-service/internal/infrastructure/category"
	"family-budget-service/internal/infrastructure/reconciliation"
	"family-budget-service/internal/infrastructure/transaction"
	"family-budget-service/internal/infrastructure/user"
)

// NewRepositoriesSQLite создает и возвращает все репозитории с SQLite подключениями
func NewRepositoriesSQLite(db *sql.DB) *handlers.Repositories {
	userRepo := user.NewSQLiteRepository(db)
	categoryRepo := category.NewSQLiteRepository(db)
	return &handlers.Repositories{
		User:           userRepo,
		Family:         user.NewSQLiteFamilyRepository(db, categoryRepo, userRepo),
		Category:       categoryRepo,
		Account:        account.NewSQLiteRepository(db),
		Reconciliation: reconciliation.NewSQLiteRepository(db),
		Transaction:    transaction.NewSQLiteRepository(db),
		Budget:         budget.NewSQLiteRepository(db),
		Session:        auth.NewSessionSQLiteRepository(db),
	}
}

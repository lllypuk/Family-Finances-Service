package infrastructure_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // Pure Go SQLite driver

	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/testhelpers"
)

// Тестовый путь (testhelpers) исполняет только *.up.sql, поэтому ветка down проверяется здесь.
func TestMigrations_UpAndDownOnEmptyDatabase(t *testing.T) {
	ctx := t.Context()
	root := testhelpers.RepoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "migrations.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath, filepath.Join(root, "migrations"))

	require.NoError(t, manager.Up())

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })

	tables := []string{"families", "users", "categories", "transactions", "budgets", "reports", "sessions"}
	for _, table := range tables {
		var count int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count), table)
	}

	assert.Equal(t, 0, objectCount(ctx, t, db, "name = 'budget_alerts'"), "budget_alerts удалена")

	require.NoError(t, manager.Down())
	left := objectCount(ctx, t, db, "type = 'table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'")
	assert.Equal(t, 0, left, "down не оставляет таблиц")
}

func objectCount(ctx context.Context, t *testing.T, db *sql.DB, where string) int {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE "+where).Scan(&count))
	return count
}

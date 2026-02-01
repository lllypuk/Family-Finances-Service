package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Test configuration constants
const (
	sqliteInMemoryDSN = ":memory:?_foreign_keys=ON&_journal_mode=WAL"
)

// SQLiteTestDB manages SQLite in-memory database for testing
type SQLiteTestDB struct {
	DB *sql.DB
}

// SetupSQLiteTestDB creates an in-memory SQLite database with migrations
func SetupSQLiteTestDB(t *testing.T) *SQLiteTestDB {
	ctx := context.Background()

	// Open in-memory SQLite database
	db, err := sql.Open("sqlite", sqliteInMemoryDSN)
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}

	// Test connection
	if err = db.PingContext(ctx); err != nil {
		_ = db.Close() // Ignore close error since we're failing anyway
		t.Fatalf("Failed to ping SQLite database: %v", err)
	}

	// Run migrations
	if err = runMigrationsSQLite(db); err != nil {
		_ = db.Close() // Ignore close error since we're failing anyway
		t.Fatalf("Failed to run migrations: %v", err)
	}

	testDB := &SQLiteTestDB{
		DB: db,
	}

	// Register cleanup
	t.Cleanup(func() {
		testDB.Cleanup(t)
	})

	return testDB
}

// Cleanup closes the database connection
func (c *SQLiteTestDB) Cleanup(t *testing.T) {
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			t.Logf("Failed to close database: %v", err)
		}
	}
}

// CleanTables deletes all data from tables for test isolation
func (c *SQLiteTestDB) CleanTables(t *testing.T) {
	ctx := context.Background()

	// List of tables to clean in order (respecting FK constraints)
	tables := []string{
		"user_sessions",
		"reports",
		"budget_alerts",
		"budgets",
		"transactions",
		"categories",
		"users",
		"families",
	}

	for _, table := range tables {
		// #nosec G201 - table names are from a fixed list, not user input
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := c.DB.ExecContext(ctx, query); err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}

// GetTestDatabase returns a clean database for tests
func (c *SQLiteTestDB) GetTestDatabase(t *testing.T) *sql.DB {
	c.CleanTables(t)
	return c.DB
}

// runMigrationsSQLite runs database migrations for SQLite by reading and executing SQL files
func runMigrationsSQLite(db *sql.DB) error {
	// Get the project root directory
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsPath := filepath.Join(projectRoot, "migrations")

	// Read all .up.sql files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort migration files
	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Execute each migration
	ctx := context.Background()
	for _, migrationFile := range migrationFiles {
		filePath := filepath.Join(migrationsPath, migrationFile)
		sqlContent, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationFile, readErr)
		}

		// Execute the SQL
		if _, execErr := db.ExecContext(ctx, string(sqlContent)); execErr != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationFile, execErr)
		}
	}

	return nil
}

// TestDataHelper provides utilities for creating test data
type TestDataHelper struct {
	DB *sql.DB
}

// NewTestDataHelper creates a new test data helper
func NewTestDataHelper(db *sql.DB) *TestDataHelper {
	return &TestDataHelper{DB: db}
}

// CreateTestFamily creates a test family and returns its ID
func (h *TestDataHelper) CreateTestFamily(ctx context.Context, name, currency string) (string, error) {
	id := generateUUID()
	query := `
		INSERT INTO families (id, name, currency, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := h.DB.ExecContext(ctx, query, id, name, currency)
	return id, err
}

// CreateTestUser creates a test user and returns its ID
func (h *TestDataHelper) CreateTestUser(
	ctx context.Context,
	email, firstName, lastName, role, familyID string,
) (string, error) {
	id := generateUUID()
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, family_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := h.DB.ExecContext(ctx, query, id, email, "test_hash", firstName, lastName, role, familyID)
	return id, err
}

// CreateTestCategory creates a test category and returns its ID
func (h *TestDataHelper) CreateTestCategory(
	ctx context.Context,
	name, categoryType, familyID string,
	parentID *string,
) (string, error) {
	id := generateUUID()
	query := `
		INSERT INTO categories (id, name, type, parent_id, family_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := h.DB.ExecContext(ctx, query, id, name, categoryType, parentID, familyID)
	return id, err
}

// CreateTestTransaction creates a test transaction and returns its ID
func (h *TestDataHelper) CreateTestTransaction(
	ctx context.Context,
	amount float64,
	description, transactionType, categoryID, userID, familyID string,
) (string, error) {
	id := generateUUID()
	query := `
		INSERT INTO transactions (id, amount, description, type, category_id, user_id, family_id, date, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, DATE('now'), '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := h.DB.ExecContext(ctx, query, id, amount, description, transactionType, categoryID, userID, familyID)
	return id, err
}

// CreateTestBudget creates a test budget and returns its ID
func (h *TestDataHelper) CreateTestBudget(
	ctx context.Context,
	name string,
	amount float64,
	period, familyID string,
	categoryID *string,
) (string, error) {
	id := generateUUID()
	query := `
		INSERT INTO budgets (id, name, amount, spent, period, start_date, end_date, category_id, family_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, DATE('now'), DATE('now', '+1 month'), ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := h.DB.ExecContext(ctx, query, id, name, amount, period, categoryID, familyID)
	return id, err
}

// generateUUID generates a UUID for testing
func generateUUID() string {
	return uuid.New().String()
}

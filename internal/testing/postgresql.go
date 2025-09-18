package testing

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Register PostgreSQL driver for migrations
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"family-budget-service/internal/infrastructure"
)

// Test configuration constants
const (
	testContainerLogOccurrence  = 2
	testContainerStartupTimeout = 30 * time.Second
	testContainerReadyWait      = 2 * time.Second
	testMaxOpenConns            = 5
	testMaxIdleConns            = 1
	testMaxConnIdleTime         = 30 * time.Minute
)

// PostgreSQLTestContainer manages PostgreSQL container for testing
type PostgreSQLTestContainer struct {
	Container testcontainers.Container
	DB        *pgxpool.Pool
	Driver    *infrastructure.PostgreSQLDriver
	URI       string
}

// SetupPostgreSQLContainer creates a PostgreSQL testcontainer with migrations
func SetupPostgreSQLContainer(t *testing.T) *PostgreSQLTestContainer {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17.6-alpine",
		postgres.WithDatabase("family_budget_test"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(testContainerLogOccurrence).
				WithStartupTimeout(testContainerStartupTimeout)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Wait a bit more for container to be fully ready
	time.Sleep(testContainerReadyWait)

	// Run migrations
	if err = runMigrations(connStr); err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create connection pool
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to parse connection config: %v", err)
	}

	// Configure for tests
	poolConfig.MaxConns = testMaxOpenConns
	poolConfig.MinConns = testMaxIdleConns
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = testMaxConnIdleTime

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Test connection
	if err = db.Ping(ctx); err != nil {
		db.Close()
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Create driver
	config := &infrastructure.PostgreSQLConfig{
		URI:             connStr,
		Database:        "family_budget_test",
		MaxOpenConns:    testMaxOpenConns,
		MaxIdleConns:    testMaxIdleConns,
		ConnMaxLifetime: 1 * time.Hour,
		SSLMode:         "disable",
		Schema:          "family_budget",
	}

	driver := infrastructure.NewPostgreSQLDriver(config)
	if err = driver.Connect(ctx); err != nil {
		db.Close()
		_ = postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to connect driver: %v", err)
	}

	return &PostgreSQLTestContainer{
		Container: postgresContainer,
		DB:        db,
		Driver:    driver,
		URI:       connStr,
	}
}

// Cleanup terminates the container and closes connections
func (c *PostgreSQLTestContainer) Cleanup(t *testing.T) {
	ctx := context.Background()

	if c.DB != nil {
		c.DB.Close()
	}

	if c.Driver != nil {
		c.Driver.Close()
	}

	if c.Container != nil {
		if err := c.Container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}
}

// CleanTables truncates all tables for test isolation
func (c *PostgreSQLTestContainer) CleanTables(t *testing.T) {
	ctx := context.Background()

	// List of tables to clean in order (respecting FK constraints)
	tables := []string{
		"family_budget.user_sessions",
		"family_budget.reports",
		"family_budget.budget_alerts",
		"family_budget.budgets",
		"family_budget.transactions",
		"family_budget.categories",
		"family_budget.users",
		"family_budget.families",
	}

	for _, table := range tables {
		if _, err := c.DB.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}

	// Reset sequences
	sequences := []string{
		// Add any sequences here if needed
	}

	for _, seq := range sequences {
		if _, err := c.DB.Exec(ctx, fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seq)); err != nil {
			t.Logf("Failed to reset sequence %s: %v", seq, err)
		}
	}
}

// GetTestDatabase returns a clean database for tests
func (c *PostgreSQLTestContainer) GetTestDatabase(t *testing.T) *pgxpool.Pool {
	c.CleanTables(t)
	return c.DB
}

// runMigrations runs database migrations
func runMigrations(databaseURL string) error {
	// Get the project root directory
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	migrationsPath := filepath.Join(projectRoot, "migrations")

	// Create migrate instance
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// TestDataHelper provides utilities for creating test data
type TestDataHelper struct {
	DB *pgxpool.Pool
}

// NewTestDataHelper creates a new test data helper
func NewTestDataHelper(db *pgxpool.Pool) *TestDataHelper {
	return &TestDataHelper{DB: db}
}

// CreateTestFamily creates a test family and returns its ID
func (h *TestDataHelper) CreateTestFamily(ctx context.Context, name, currency string) (string, error) {
	var id string
	query := `
		INSERT INTO family_budget.families (name, currency, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		RETURNING id`

	err := h.DB.QueryRow(ctx, query, name, currency).Scan(&id)
	return id, err
}

// CreateTestUser creates a test user and returns its ID
func (h *TestDataHelper) CreateTestUser(
	ctx context.Context,
	email, firstName, lastName, role, familyID string,
) (string, error) {
	var id string
	query := `
		INSERT INTO family_budget.users (email, password_hash, first_name, last_name, role, family_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
		RETURNING id`

	err := h.DB.QueryRow(ctx, query, email, "test_hash", firstName, lastName, role, familyID).Scan(&id)
	return id, err
}

// CreateTestCategory creates a test category and returns its ID
func (h *TestDataHelper) CreateTestCategory(
	ctx context.Context,
	name, categoryType, familyID string,
	parentID *string,
) (string, error) {
	var id string
	query := `
		INSERT INTO family_budget.categories (name, type, parent_id, family_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		RETURNING id`

	err := h.DB.QueryRow(ctx, query, name, categoryType, parentID, familyID).Scan(&id)
	return id, err
}

// CreateTestTransaction creates a test transaction and returns its ID
func (h *TestDataHelper) CreateTestTransaction(
	ctx context.Context,
	amount float64,
	description, transactionType, categoryID, userID, familyID string,
) (string, error) {
	var id string
	query := `
		INSERT INTO family_budget.transactions (amount, description, type, category_id, user_id, family_id, date, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_DATE, '[]', NOW(), NOW())
		RETURNING id`

	err := h.DB.QueryRow(ctx, query, amount, description, transactionType, categoryID, userID, familyID).Scan(&id)
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
	var id string
	query := `
		INSERT INTO family_budget.budgets (name, amount, spent, period, start_date, end_date, category_id, family_id, is_active, created_at, updated_at)
		VALUES ($1, $2, 0, $3, CURRENT_DATE, CURRENT_DATE + INTERVAL '1 month', $4, $5, true, NOW(), NOW())
		RETURNING id`

	err := h.DB.QueryRow(ctx, query, name, amount, period, categoryID, familyID).Scan(&id)
	return id, err
}

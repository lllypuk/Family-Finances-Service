package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQLDriver provides PostgreSQL database connection and management
type PostgreSQLDriver struct {
	pool   *pgxpool.Pool
	config *PostgreSQLConfig
}

// PostgreSQLConfig contains PostgreSQL connection configuration
type PostgreSQLConfig struct {
	URI             string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	SSLMode         string
	Schema          string
}

// NewPostgreSQLDriver creates a new PostgreSQL driver instance
func NewPostgreSQLDriver(config *PostgreSQLConfig) *PostgreSQLDriver {
	return &PostgreSQLDriver{
		config: config,
	}
}

// Connect establishes connection to PostgreSQL database
func (d *PostgreSQLDriver) Connect(ctx context.Context) error {
	// Build connection configuration
	poolConfig, err := pgxpool.ParseConfig(d.config.URI)
	if err != nil {
		return fmt.Errorf("failed to parse connection config: %w", err)
	}

	// Set optimized connection pool parameters
	poolConfig.MaxConns = int32(d.config.MaxOpenConns)
	poolConfig.MinConns = int32(d.config.MaxIdleConns)
	poolConfig.MaxConnLifetime = d.config.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = time.Duration(d.config.ConnMaxIdleTime)

	// Set health check interval for better connection management
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Optimize connection acquisition
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	d.pool = pool
	return nil
}

// Close closes the database connection pool
func (d *PostgreSQLDriver) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

// Pool returns the pgxpool.Pool instance
func (d *PostgreSQLDriver) Pool() *pgxpool.Pool {
	return d.pool
}

// HealthCheck performs a health check on the database connection
func (d *PostgreSQLDriver) HealthCheck(ctx context.Context) error {
	if d.pool == nil {
		return errors.New("database connection not initialized")
	}

	// Ping the database
	if err := d.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check connection pool stats
	stats := d.pool.Stat()
	if stats.TotalConns() == 0 {
		return errors.New("no database connections available")
	}

	return nil
}

// Stats returns connection pool statistics
func (d *PostgreSQLDriver) Stats() *pgxpool.Stat {
	if d.pool == nil {
		return nil
	}
	stats := d.pool.Stat()
	return stats
}

// WithTransaction executes a function within a database transaction
func (d *PostgreSQLDriver) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if d.pool == nil {
		return errors.New("database connection not initialized")
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// Log rollback error but don't override the original error
				// In a real application, you would use your logging system here
			}
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecuteQuery executes a query and returns the result
func (d *PostgreSQLDriver) ExecuteQuery(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if d.pool == nil {
		return nil, errors.New("database connection not initialized")
	}

	return d.pool.Query(ctx, query, args...)
}

// ExecuteQueryRow executes a query that returns a single row
func (d *PostgreSQLDriver) ExecuteQueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, query, args...)
}

// DefaultPostgreSQLConfig returns default PostgreSQL configuration
func DefaultPostgreSQLConfig() *PostgreSQLConfig {
	return &PostgreSQLConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 1 * time.Hour,
		SSLMode:         "prefer",
		Schema:          "family_budget",
	}
}

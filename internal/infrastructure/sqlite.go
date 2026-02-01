package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

const (
	// defaultPingTimeout is the default timeout for database ping operations
	defaultPingTimeout = 5 * time.Second
)

// SQLiteConnection represents a SQLite database connection
type SQLiteConnection struct {
	db *sql.DB
}

// NewSQLiteConnection creates a new SQLite database connection
func NewSQLiteConnection(dbPath string) (*SQLiteConnection, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connection string with optimizations for production
	dsn := dbPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite settings for production
	// SQLite uses a single-writer concurrency model: writes acquire a database-wide lock, so
	// having multiple open connections that can write often causes "database is locked" errors,
	// even when using WAL mode. We therefore force a single shared connection and let database/sql
	// serialize all access through it.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Connections never expire

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if pingErr := db.PingContext(ctx); pingErr != nil {
		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	return &SQLiteConnection{db: db}, nil
}

// DB returns the underlying *sql.DB
func (c *SQLiteConnection) DB() *sql.DB {
	return c.db
}

// Close closes the database connection
func (c *SQLiteConnection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// HealthCheck performs a health check on the database connection
func (c *SQLiteConnection) HealthCheck(ctx context.Context) error {
	if c.db == nil {
		return errors.New("database connection is nil")
	}

	return c.db.PingContext(ctx)
}

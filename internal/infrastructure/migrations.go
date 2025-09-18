package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Register PostgreSQL driver for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Register file source driver for migrations
)

// MigrationManager handles database schema migrations
type MigrationManager struct {
	databaseURL    string
	migrationsPath string
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(databaseURL, migrationsPath string) *MigrationManager {
	return &MigrationManager{
		databaseURL:    databaseURL,
		migrationsPath: migrationsPath,
	}
}

// Up runs all up migrations
func (m *MigrationManager) Up() error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

// Down runs all down migrations
func (m *MigrationManager) Down() error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	return nil
}

// Steps runs the specified number of migrations (positive for up, negative for down)
func (m *MigrationManager) Steps(steps int) error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Steps(steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migration steps: %w", err)
	}

	return nil
}

// Migrate runs migrations to the specified version
func (m *MigrationManager) Migrate(version uint) error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	return nil
}

// Force sets the migration version without running migrations
func (m *MigrationManager) Force(version int) error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version to %d: %w", version, err)
	}

	return nil
}

// Version returns the current migration version
func (m *MigrationManager) Version() (uint, bool, error) {
	migration, err := m.createMigration()
	if err != nil {
		return 0, false, err
	}
	defer migration.Close()

	version, dirty, err := migration.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// Drop drops the entire database
func (m *MigrationManager) Drop() error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	if err = migration.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}

// createMigration creates a new migrate.Migrate instance
func (m *MigrationManager) createMigration() (*migrate.Migrate, error) {
	sourceURL := fmt.Sprintf("file://%s", filepath.Clean(m.migrationsPath))

	migration, err := migrate.New(sourceURL, m.databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	return migration, nil
}

// MigrationInfo holds information about a migration
type MigrationInfo struct {
	Version uint
	Name    string
	Dirty   bool
}

// GetMigrationInfo returns current migration information
func (m *MigrationManager) GetMigrationInfo() (*MigrationInfo, error) {
	version, dirty, err := m.Version()
	if err != nil {
		return nil, err
	}

	return &MigrationInfo{
		Version: version,
		Dirty:   dirty,
	}, nil
}

// ValidateMigrations validates that migrations can be applied
func (m *MigrationManager) ValidateMigrations(_ context.Context) error {
	migration, err := m.createMigration()
	if err != nil {
		return err
	}
	defer migration.Close()

	// Get current version
	currentVersion, dirty, err := migration.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in dirty state, version %d needs to be resolved", currentVersion)
	}

	return nil
}

// CreateInitialMigration creates the initial migration files if they don't exist
func CreateInitialMigration(_ string) error {
	// This would create the migration files - for now, we'll use the existing pg-init.sql
	// In a real implementation, you would create numbered migration files
	// like 001_initial_schema.up.sql and 001_initial_schema.down.sql
	return errors.New("manual migration creation required - convert scripts/pg-init.sql to migration format")
}

//nolint:testpackage // Need access to private functions for testing
package services

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // SQLite driver
)

// setupTestDB creates a temporary in-memory database for testing
func setupTestDB(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	// Create temporary directory for database and backups
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	// Create a simple table with some data
	_, err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Insert test data
	_, err = db.Exec("INSERT INTO test_table (name) VALUES (?), (?), (?)", "test1", "test2", "test3")
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, dbPath, cleanup
}

func TestCreateBackup_Success(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create backup
	backupInfo, err := service.CreateBackup(ctx)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, backupInfo)
	assert.NotEmpty(t, backupInfo.Filename)
	assert.Positive(t, backupInfo.Size)
	assert.WithinDuration(t, time.Now(), backupInfo.CreatedAt, 5*time.Second)

	// Verify backup file exists
	backupPath := service.GetBackupFilePath(backupInfo.Filename)
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)
}

func TestListBackups_Empty(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// List backups when none exist
	backups, err := service.ListBackups(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Empty(t, backups)
}

func TestListBackups_Multiple(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create multiple backups
	backup1, err := service.CreateBackup(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // Ensure different timestamps

	backup2, err := service.CreateBackup(ctx)
	require.NoError(t, err)

	// List backups
	backups, err := service.ListBackups(ctx)

	// Assertions
	require.NoError(t, err)
	assert.Len(t, backups, 2)

	// Verify sorting (newest first)
	assert.Equal(t, backup2.Filename, backups[0].Filename)
	assert.Equal(t, backup1.Filename, backups[1].Filename)
}

func TestDeleteBackup_Success(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create backup
	backupInfo, err := service.CreateBackup(ctx)
	require.NoError(t, err)

	// Delete backup
	err = service.DeleteBackup(ctx, backupInfo.Filename)
	require.NoError(t, err)

	// Verify backup file is deleted
	backupPath := service.GetBackupFilePath(backupInfo.Filename)
	_, err = os.Stat(backupPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteBackup_NotFound(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Try to delete non-existent backup
	err := service.DeleteBackup(ctx, "backup_20250101_120000000.db")

	// Assertions
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBackupNotFound)
}

func TestDeleteBackup_InvalidFilename(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	testCases := []struct {
		name     string
		filename string
	}{
		{"path traversal", "../../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"wrong format", "not_a_backup.db"},
		{"wrong extension", "backup_20250101_120000000.txt"},
		{"missing date", "backup_.db"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.DeleteBackup(ctx, tc.filename)
			assert.ErrorIs(t, err, ErrInvalidBackupFilename)
		})
	}
}

func TestGetBackup_Success(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create backup
	created, err := service.CreateBackup(ctx)
	require.NoError(t, err)

	// Get backup info
	backupInfo, err := service.GetBackup(ctx, created.Filename)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, created.Filename, backupInfo.Filename)
	assert.Equal(t, created.Size, backupInfo.Size)
}

func TestGetBackup_NotFound(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Try to get non-existent backup
	_, err := service.GetBackup(ctx, "backup_20250101_120000000.db")

	// Assertions
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBackupNotFound)
}

func TestFilenameValidation(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		valid    bool
	}{
		{"valid filename", "backup_20250202_143022000.db", true},
		{"valid with zeros", "backup_00000000_000000000.db", true},
		{"old format (6 digits)", "backup_20250202_143022.db", false}, // Old format no longer valid
		{"invalid - path traversal", "../backup.db", false},
		{"invalid - absolute path", "/tmp/backup.db", false},
		{"invalid - wrong prefix", "backup2_20250202_143022000.db", false},
		{"invalid - wrong extension", "backup_20250202_143022000.txt", false},
		{"invalid - no date", "backup.db", false},
		{"invalid - incomplete date", "backup_202502_143022000.db", false},
		{"invalid - extra chars", "backup_20250202_143022000x.db", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateFilename(tc.filename)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrInvalidBackupFilename)
			}
		})
	}
}

func TestRestoreBackup(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create backup
	backupInfo, err := service.CreateBackup(ctx)
	require.NoError(t, err)

	// Modify database (insert more data)
	_, err = db.Exec("INSERT INTO test_table (name) VALUES (?)", "test4")
	require.NoError(t, err)

	// Restore from backup
	err = service.RestoreBackup(ctx, backupInfo.Filename)
	require.NoError(t, err)

	// Note: In real scenario, we would need to reconnect to database
	// For this test, we just verify the restore operation succeeded
}

func TestCleanupOldBackups(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)
	ctx := context.Background()

	// Create more than maxBackups (10) backups
	for range 12 {
		_, err := service.CreateBackup(ctx)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List backups
	backups, err := service.ListBackups(ctx)
	require.NoError(t, err)

	// Should have only maxBackups (10) backups
	assert.LessOrEqual(t, len(backups), maxBackups)
}

func TestGetBackupFilePath_InvalidFilename(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	service := NewBackupService(db, dbPath)

	// Try with invalid filename
	path := service.GetBackupFilePath("../../../etc/passwd")

	// Should return empty string for invalid filename
	assert.Empty(t, path)
}

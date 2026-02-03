//nolint:testpackage // Need access to private functions for testing
package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // SQLite driver
)

func TestBackupService_PathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  error
	}{
		{
			name:     "valid filename",
			filename: "backup_20240101_120000000.db",
			wantErr:  ErrBackupNotFound, // file doesn't exist, but format is valid
		},
		{
			name:     "directory traversal with ../",
			filename: "../../../etc/passwd",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "directory traversal with encoded",
			filename: "..%2F..%2Fetc%2Fpasswd",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "absolute path",
			filename: "/etc/passwd",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "null byte injection",
			filename: "backup_20240101_120000000.db\x00.txt",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "hidden file",
			filename: ".backup_20240101_120000000.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "double dots in name",
			filename: "backup..20240101_120000000.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "spaces in filename",
			filename: "backup 20240101_120000000.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "sql injection in filename",
			filename: "backup'; DROP TABLE users;--.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "windows path traversal",
			filename: "..\\..\\..\\windows\\system32\\config\\sam",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "mixed path separators",
			filename: "../backup\\..\\data.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "unicode directory traversal",
			filename: "backup\u2215\u2215etc\u2215passwd.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "command injection attempt",
			filename: "backup_$(rm -rf /)_120000000.db",
			wantErr:  ErrInvalidBackupFilename,
		},
		{
			name:     "xss attempt",
			filename: "backup_<script>alert(1)</script>_120000000.db",
			wantErr:  ErrInvalidBackupFilename,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, dbPath, cleanup := setupTestDB(t)
			defer cleanup()
			svc := NewBackupService(db, dbPath, slog.Default())

			// Test GetBackup
			_, err := svc.GetBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "GetBackup should return expected error")
			}

			// Test DeleteBackup
			err = svc.DeleteBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "DeleteBackup should return expected error")
			}

			// Test RestoreBackup
			err = svc.RestoreBackup(context.Background(), tt.filename)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, "RestoreBackup should return expected error")
			}

			// Test GetBackupFilePath
			path := svc.GetBackupFilePath(tt.filename)
			if errors.Is(tt.wantErr, ErrInvalidBackupFilename) {
				assert.Empty(t, path, "GetBackupFilePath should return empty for invalid filename")
			}
		})
	}
}

func TestBackupService_SafePathDirectoryEscape(t *testing.T) {
	db, dbPath, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewBackupService(db, dbPath, slog.Default()).(*backupService)

	tests := []struct {
		name          string
		filename      string
		shouldEscape  bool
		expectedError error
	}{
		{
			name:          "normal filename stays in directory",
			filename:      "backup_20240101_120000000.db",
			shouldEscape:  false,
			expectedError: nil,
		},
		{
			name:          "path traversal attempt blocked",
			filename:      "../../../etc/passwd",
			shouldEscape:  true,
			expectedError: ErrInvalidBackupFilename,
		},
		{
			name:          "symlink name rejected",
			filename:      "backup_20240101_120000000.db.lnk",
			shouldEscape:  true,
			expectedError: ErrInvalidBackupFilename,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := svc.safePath(tt.filename)

			if tt.shouldEscape || tt.expectedError != nil {
				require.Error(t, err)
				if tt.expectedError != nil {
					require.ErrorIs(t, err, tt.expectedError)
				}
				assert.Empty(t, path)
			} else {
				require.NoError(t, err)
				// Verify path is inside backup directory
				backupDir := filepath.Clean(svc.backupDir)
				resolvedPath := filepath.Clean(path)
				assert.Contains(t, resolvedPath, backupDir,
					"Resolved path should be inside backup directory")
			}
		})
	}
}

func TestBackupService_FilenameValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		valid    bool
	}{
		{
			name:     "exact format match",
			filename: "backup_20250203_050000123.db",
			valid:    true,
		},
		{
			name:     "missing milliseconds",
			filename: "backup_20250203_050000.db",
			valid:    false,
		},
		{
			name:     "too many milliseconds digits",
			filename: "backup_20250203_0500001234.db",
			valid:    false,
		},
		{
			name:     "empty string",
			filename: "",
			valid:    false,
		},
		{
			name:     "only dots",
			filename: "...",
			valid:    false,
		},
		{
			name:     "null bytes in middle",
			filename: "backup_20250203\x00_050000123.db",
			valid:    false,
		},
		{
			name:     "carriage return injection",
			filename: "backup_20250203_050000123.db\r\n../../etc/passwd",
			valid:    false,
		},
		{
			name:     "newline injection",
			filename: "backup_20250203_050000123.db\n",
			valid:    false,
		},
		{
			name:     "tab character",
			filename: "backup_20250203_050000123.db\t",
			valid:    false,
		},
		{
			name:     "uppercase extension",
			filename: "backup_20250203_050000123.DB",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilename(tt.filename)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrInvalidBackupFilename)
			}
		})
	}
}

func TestBackupService_PathValidationDefenseInDepth(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{
			name:  "valid unix path",
			path:  "/tmp/backups/backup_20250203_050000123.db",
			valid: true,
		},
		{
			name:  "valid windows path",
			path:  "C:\\data\\backups\\backup_20250203_050000123.db",
			valid: true,
		},
		{
			name:  "semicolon sql injection",
			path:  "/tmp/backups/backup;DROP TABLE users--.db",
			valid: false,
		},
		{
			name:  "single quote sql injection",
			path:  "/tmp/backups/backup'OR'1'='1.db",
			valid: false,
		},
		{
			name:  "double dash comment",
			path:  "/tmp/backups/backup--.db",
			valid: true, // dashes are allowed
		},
		{
			name:  "pipe command injection",
			path:  "/tmp/backups/backup|cat /etc/passwd.db",
			valid: false,
		},
		{
			name:  "ampersand command chaining",
			path:  "/tmp/backups/backup&whoami.db",
			valid: false,
		},
		{
			name:  "backtick command substitution",
			path:  "/tmp/backups/backup`ls`.db",
			valid: false,
		},
		{
			name:  "dollar command substitution",
			path:  "/tmp/backups/backup$(ls).db",
			valid: false,
		},
		{
			name:  "asterisk wildcard",
			path:  "/tmp/backups/*.db",
			valid: false,
		},
		{
			name:  "question mark wildcard",
			path:  "/tmp/backups/backup?.db",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBackupPath(tt.path)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// setupTestDB creates a temporary database for testing
func setupTestDBSecurity(t *testing.T) (*sql.DB, string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, dbPath, cleanup
}

func TestBackupService_ConcurrentPathTraversalAttempts(t *testing.T) {
	db, dbPath, cleanup := setupTestDBSecurity(t)
	defer cleanup()

	svc := NewBackupService(db, dbPath, slog.Default())
	ctx := context.Background()

	// Concurrent attempts to access invalid paths
	maliciousFilenames := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/etc/shadow",
		"backup'; DROP TABLE users;--.db",
		"backup_$(rm -rf /)_120000000.db",
	}

	done := make(chan bool, len(maliciousFilenames))

	for _, filename := range maliciousFilenames {
		go func(fn string) {
			_, err := svc.GetBackup(ctx, fn)
			assert.ErrorIs(t, err, ErrInvalidBackupFilename)
			done <- true
		}(filename)
	}

	// Wait for all goroutines to complete
	for range maliciousFilenames {
		<-done
	}
}

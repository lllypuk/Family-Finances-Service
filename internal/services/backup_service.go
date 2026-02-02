package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

const (
	maxBackups          = 10
	backupDirPerm       = 0750
	backupFilenameRegex = `^backup_\d{8}_\d{9}\.db$` // Updated for milliseconds
)

var (
	// ErrBackupNotFound is returned when backup file is not found
	ErrBackupNotFound = errors.New("backup not found")
	// ErrInvalidBackupFilename is returned when filename is invalid (path traversal protection)
	ErrInvalidBackupFilename = errors.New("invalid backup filename")
)

// backupService implements BackupService interface
type backupService struct {
	db        *sql.DB
	dbPath    string
	backupDir string
	logger    *slog.Logger
}

// NewBackupService creates a new BackupService instance
func NewBackupService(db *sql.DB, dbPath string, logger *slog.Logger) BackupService {
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	return &backupService{
		db:        db,
		dbPath:    dbPath,
		backupDir: backupDir,
		logger:    logger,
	}
}

// validateFilename validates backup filename to prevent path traversal attacks
func validateFilename(filename string) error {
	matched, err := regexp.MatchString(backupFilenameRegex, filename)
	if err != nil {
		return err
	}
	if !matched {
		return ErrInvalidBackupFilename
	}
	return nil
}

// ensureBackupDir creates backup directory if it doesn't exist
func (s *backupService) ensureBackupDir() error {
	if _, err := os.Stat(s.backupDir); os.IsNotExist(err) {
		return os.MkdirAll(s.backupDir, backupDirPerm)
	}
	return nil
}

// CreateBackup creates a new backup using SQLite VACUUM INTO
func (s *backupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	// Ensure backup directory exists
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with current timestamp (including nanoseconds for uniqueness)
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	const (
		nanosToMillis  = 1e6
		millisInSecond = 1000
	)
	milliseconds := now.UnixNano() / nanosToMillis % millisInSecond // Get milliseconds part
	filename := fmt.Sprintf("backup_%s%03d.db", timestamp, milliseconds)
	backupPath := filepath.Join(s.backupDir, filename)

	// Use VACUUM INTO for atomic backup (safe with WAL mode)
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Get file info
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	backupInfo := &BackupInfo{
		Filename:  filename,
		Size:      info.Size(),
		CreatedAt: now,
	}

	// Clean up old backups if limit exceeded
	if cleanupErr := s.cleanupOldBackups(ctx); cleanupErr != nil {
		s.logger.WarnContext(ctx, "failed to cleanup old backups",
			slog.String("error", cleanupErr.Error()),
		)
	}

	return backupInfo, nil
}

// ListBackups returns list of all backups sorted by date (newest first)
func (s *backupService) ListBackups(_ context.Context) ([]*BackupInfo, error) {
	// Ensure backup directory exists
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("failed to access backup directory: %w", err)
	}

	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Filter by backup pattern
		if validationErr := validateFilename(filename); validationErr != nil {
			continue
		}

		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		backups = append(backups, &BackupInfo{
			Filename:  filename,
			Size:      entryInfo.Size(),
			CreatedAt: entryInfo.ModTime(),
		})
	}

	// Sort by date, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetBackup retrieves information about a specific backup
func (s *backupService) GetBackup(_ context.Context, filename string) (*BackupInfo, error) {
	// Validate filename
	if err := validateFilename(filename); err != nil {
		return nil, err
	}

	backupPath := filepath.Join(s.backupDir, filename)
	info, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return &BackupInfo{
		Filename:  filename,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
	}, nil
}

// DeleteBackup deletes a specific backup file
func (s *backupService) DeleteBackup(_ context.Context, filename string) error {
	// Validate filename
	if err := validateFilename(filename); err != nil {
		return err
	}

	backupPath := filepath.Join(s.backupDir, filename)
	if err := os.Remove(backupPath); err != nil {
		if os.IsNotExist(err) {
			return ErrBackupNotFound
		}
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// RestoreBackup restores database from a backup file
// WARNING: This is a dangerous operation that replaces the current database
func (s *backupService) RestoreBackup(_ context.Context, filename string) error {
	// Validate filename
	if err := validateFilename(filename); err != nil {
		return err
	}

	backupPath := filepath.Join(s.backupDir, filename)

	// Check if backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return ErrBackupNotFound
		}
		return fmt.Errorf("failed to access backup file: %w", err)
	}

	// Copy backup file to main database location
	// Note: In production, this should close all database connections first
	// This implementation assumes the application will be restarted after restore
	data, readErr := os.ReadFile(backupPath)
	if readErr != nil {
		return fmt.Errorf("failed to read backup file: %w", readErr)
	}

	//nolint:gosec // File permissions 0640 are required for database file
	if err := os.WriteFile(s.dbPath, data, 0640); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// GetBackupFilePath returns the full path to a backup file
func (s *backupService) GetBackupFilePath(filename string) string {
	// Validate filename (silent validation for path construction)
	if err := validateFilename(filename); err != nil {
		return ""
	}
	return filepath.Join(s.backupDir, filename)
}

// cleanupOldBackups removes oldest backups if maxBackups limit is exceeded
func (s *backupService) cleanupOldBackups(ctx context.Context) error {
	backups, err := s.ListBackups(ctx)
	if err != nil {
		return err
	}

	if len(backups) <= maxBackups {
		return nil
	}

	// Delete oldest backups
	for i := maxBackups; i < len(backups); i++ {
		if deleteErr := s.DeleteBackup(ctx, backups[i].Filename); deleteErr != nil {
			// Continue deleting others even if one fails
			continue
		}
	}

	return nil
}

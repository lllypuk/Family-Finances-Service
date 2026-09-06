package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultBackupKeep — сколько последних файлов бэкапа хранить, если BACKUP_KEEP не задан.
	DefaultBackupKeep   = 30
	backupDirPerm       = 0750
	backupFilenameRegex = `^backup_\d{8}_\d{9}\.db$` // backup_YYYYMMDD_HHMMSSmmm.db
	// Имя различает бэкапы только до миллисекунды, поэтому одновременные вызовы
	// (API и cron) могут выбрать один путь, а публикация имени отказывает на занятом.
	backupNameAttempts = 5
)

// validPathRegex validates that backup path contains only safe characters.
// This is defense-in-depth for the VACUUM INTO query which cannot use parameterized queries.
var validPathRegex = regexp.MustCompile(`^[a-zA-Z0-9_.\-/\\:]+$`)

var (
	// ErrBackupNotFound is returned when backup file is not found
	ErrBackupNotFound = errors.New("backup not found")
	// ErrInvalidBackupFilename is returned when filename is invalid (path traversal protection)
	ErrInvalidBackupFilename = errors.New("invalid backup filename")
	// errBackupNameTaken — путь занял параллельный вызов, имя берётся заново.
	errBackupNameTaken = errors.New("backup file name is already taken")
)

// backupService implements BackupService interface
type backupService struct {
	db        *sql.DB
	backupDir string
	keep      int
	logger    *slog.Logger
}

// NewBackupService creates a new BackupService instance.
// Пустой backupDir означает <dir(dbPath)>/backups, keep <= 0 — DefaultBackupKeep.
func NewBackupService(db *sql.DB, dbPath, backupDir string, keep int, logger *slog.Logger) BackupService {
	if backupDir == "" {
		backupDir = filepath.Join(filepath.Dir(dbPath), "backups")
	}
	if keep <= 0 {
		keep = DefaultBackupKeep
	}
	return &backupService{
		db:        db,
		backupDir: backupDir,
		keep:      keep,
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

// backupFilename builds backup_YYYYMMDD_HHMMSSmmm.db — millisecond precision.
func backupFilename(t time.Time) string {
	const (
		nanosToMillis  = 1e6
		millisInSecond = 1000
	)
	milliseconds := t.UnixNano() / nanosToMillis % millisInSecond
	return fmt.Sprintf("backup_%s%03d.db", t.Format("20060102_150405"), milliseconds)
}

// tempBackupName builds a per-operation temporary name.
// Имя намеренно не подходит под backupFilenameRegex: недописанный файл не должен
// попасть ни в листинг, ни в счётчик ретеншена.
func tempBackupName() string {
	return fmt.Sprintf(".tmp-%d-%s.db", os.Getpid(), rand.Text())
}

// isValidBackupPath checks that a backup path contains only safe characters.
// This is defense-in-depth for the VACUUM INTO query which cannot use parameterized queries.
func isValidBackupPath(path string) bool {
	return validPathRegex.MatchString(path)
}

// safePath constructs a safe file path within the backup directory.
// It validates the filename, applies filepath.Base to strip directory components,
// and verifies the result stays within backupDir.
func (s *backupService) safePath(filename string) (string, error) {
	if err := validateFilename(filename); err != nil {
		return "", err
	}

	// Strip any directory components — defense in depth
	clean := filepath.Base(filename)

	// Re-validate after Base() in case something changed
	if err := validateFilename(clean); err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.backupDir, clean)

	// Verify the resolved path is still inside backupDir
	cleanBackupDir := filepath.Clean(s.backupDir) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(fullPath)+string(os.PathSeparator), cleanBackupDir) {
		return "", ErrInvalidBackupFilename
	}

	return fullPath, nil
}

// ensureBackupDir creates backup directory if it doesn't exist
func (s *backupService) ensureBackupDir() error {
	if _, err := os.Stat(s.backupDir); os.IsNotExist(err) {
		return os.MkdirAll(s.backupDir, backupDirPerm)
	}
	return nil
}

// vacuumInto пишет снимок базы во временный файл и публикует его под backupPath
// жёсткой ссылкой: link атомарно отказывает на занятом имени (errBackupNameTaken),
// поэтому чужой готовый бэкап нельзя ни перезаписать, ни удалить своей уборкой —
// удаляется только собственный временный файл.
func (s *backupService) vacuumInto(ctx context.Context, backupPath string) error {
	tmpPath := filepath.Join(s.backupDir, tempBackupName())
	if !isValidBackupPath(tmpPath) {
		return fmt.Errorf("unsafe backup path detected: %s", tmpPath)
	}

	// VACUUM INTO does not support parameterized queries in SQLite.
	// The path is constructed entirely from controlled data (backupDir + generated name).
	//nolint:gosec // tmpPath is generated here, not user input
	query := fmt.Sprintf("VACUUM INTO '%s'", tmpPath)

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		s.removeTemp(ctx, tmpPath)
		return err
	}

	linkErr := os.Link(tmpPath, backupPath)
	s.removeTemp(ctx, tmpPath)

	switch {
	case linkErr == nil:
		return nil
	case errors.Is(linkErr, os.ErrExist):
		return errBackupNameTaken
	default:
		return linkErr
	}
}

// removeTemp удаляет временный снимок; провал уборки не отменяет результат операции.
func (s *backupService) removeTemp(ctx context.Context, tmpPath string) {
	if err := os.Remove(tmpPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.logger.WarnContext(ctx, "failed to remove temporary backup file",
			slog.String("path", tmpPath),
			slog.String("error", err.Error()),
		)
	}
}

// CreateBackup creates a new backup using SQLite VACUUM INTO
func (s *backupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	// Ensure backup directory exists
	if err := s.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Занятость имени определяет сама публикация снимка: os.Stat заранее ничего
	// не гарантирует — параллельный вызов успевает занять то же миллисекундное
	// имя между проверкой и записью.
	var (
		now        time.Time
		filename   string
		backupPath string
		lastErr    error
	)

	for range backupNameAttempts {
		now = time.Now()
		filename = backupFilename(now)

		// Validate the generated filename matches expected pattern
		if err := validateFilename(filename); err != nil {
			return nil, fmt.Errorf("generated invalid backup filename: %w", err)
		}

		backupPath = filepath.Join(s.backupDir, filepath.Base(filename))

		// Validate path contains only safe characters (alphanumeric, underscores, dots, slashes)
		if !isValidBackupPath(backupPath) {
			return nil, fmt.Errorf("unsafe backup path detected: %s", backupPath)
		}

		lastErr = s.vacuumInto(ctx, backupPath)
		if lastErr == nil {
			break
		}
		if !errors.Is(lastErr, errBackupNameTaken) {
			return nil, fmt.Errorf("failed to create backup: %w", lastErr)
		}
		time.Sleep(time.Millisecond)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to create backup: %w", lastErr)
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
	backupPath, err := s.safePath(filename)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- Path is validated by safePath() to prevent traversal attacks
	info, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return &BackupInfo{
		Filename:  filepath.Base(backupPath),
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
	}, nil
}

// DeleteBackup deletes a specific backup file
func (s *backupService) DeleteBackup(_ context.Context, filename string) error {
	backupPath, err := s.safePath(filename)
	if err != nil {
		return err
	}

	// #nosec G304 -- Path is validated by safePath() to prevent traversal attacks
	if removeErr := os.Remove(backupPath); removeErr != nil {
		if os.IsNotExist(removeErr) {
			return ErrBackupNotFound
		}
		return fmt.Errorf("failed to delete backup: %w", removeErr)
	}

	return nil
}

// GetBackupFilePath returns the full path to a backup file
func (s *backupService) GetBackupFilePath(filename string) string {
	backupPath, err := s.safePath(filename)
	if err != nil {
		return ""
	}
	return backupPath
}

// cleanupOldBackups removes oldest backups if the keep limit is exceeded
func (s *backupService) cleanupOldBackups(ctx context.Context) error {
	backups, err := s.ListBackups(ctx)
	if err != nil {
		return err
	}

	if len(backups) <= s.keep {
		return nil
	}

	// Delete oldest backups
	for i := s.keep; i < len(backups); i++ {
		if deleteErr := s.DeleteBackup(ctx, backups[i].Filename); deleteErr != nil {
			// Continue deleting others even if one fails
			continue
		}
	}

	return nil
}

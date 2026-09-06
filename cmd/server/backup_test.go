package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// backupEnv — CWD корня репозитория (там ./migrations) и отдельные каталоги БД и бэкапов.
func backupEnv(t *testing.T) string {
	t.Helper()
	t.Chdir(testhelpers.RepoRoot(t))
	dir := t.TempDir()
	t.Setenv("DATABASE_PATH", filepath.Join(dir, "backup-cli.db"))
	backupDir := filepath.Join(dir, "backups")
	t.Setenv("BACKUP_DIR", backupDir)
	return backupDir
}

func countBackups(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	return len(entries)
}

func TestRunBackup_Success(t *testing.T) {
	backupDir := backupEnv(t)

	var out bytes.Buffer
	require.NoError(t, runBackup(context.Background(), nil, nil, &out))

	assert.Contains(t, out.String(), "created")
	assert.Equal(t, 1, countBackups(t, backupDir))
}

func TestRunBackup_KeepRetainsNewest(t *testing.T) {
	backupDir := backupEnv(t)

	const keep = 2
	var out bytes.Buffer
	for range keep + 2 {
		require.NoError(t, runBackup(context.Background(), []string{"--keep", "2"}, nil, &out))
	}

	assert.Equal(t, keep, countBackups(t, backupDir))
}

func TestRunBackup_DatabaseOpenError(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	// Каталог вместо файла: SQLite не откроет такую «БД».
	t.Setenv("DATABASE_PATH", t.TempDir())

	var out bytes.Buffer
	require.Error(t, runBackup(context.Background(), nil, nil, &out))
	assert.Empty(t, out.String())
}

func TestParseBackupArgs_InvalidFlag(t *testing.T) {
	_, err := parseBackupArgs([]string{"--nope"})
	require.Error(t, err)
}

package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/testhelpers"
)

// backupEnv — CWD корня репозитория (там ./migrations) и отдельные каталоги БД и бэкапов.
// БД создаётся здесь: подкоманда backup отказывается работать с отсутствующим файлом.
func backupEnv(t *testing.T) string {
	t.Helper()
	t.Chdir(testhelpers.RepoRoot(t))
	dir := t.TempDir()
	t.Setenv("DATABASE_PATH", filepath.Join(dir, "backup-cli.db"))
	backupDir := filepath.Join(dir, "backups")
	t.Setenv("BACKUP_DIR", backupDir)

	db, err := internal.OpenDatabase(internal.LoadConfig())
	require.NoError(t, err)
	require.NoError(t, db.Close())

	return backupDir
}

// backupNames — имена файлов в каталоге бэкапов, отсортированные по возрастанию
// (os.ReadDir сортирует по имени, а имя — это временная метка).
func backupNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// runBackupOnce возвращает имя созданного файла, вытащенное из строки вида
// `backup <file> created (<n> bytes)`.
func runBackupOnce(t *testing.T, args []string) string {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, runBackup(context.Background(), args, nil, &out))
	fields := strings.Fields(out.String())
	require.Len(t, fields, 5, "unexpected output: %q", out.String())
	return fields[1]
}

func TestRunBackup_Success(t *testing.T) {
	backupDir := backupEnv(t)

	var out bytes.Buffer
	require.NoError(t, runBackup(context.Background(), nil, nil, &out))

	assert.Contains(t, out.String(), "created")
	assert.Len(t, backupNames(t, backupDir), 1)
}

func TestRunBackup_KeepRetainsNewest(t *testing.T) {
	backupDir := backupEnv(t)

	const keep = 2
	created := make([]string, 0, keep+2)
	for range keep + 2 {
		created = append(created, runBackupOnce(t, []string{"--keep", "2"}))
	}

	// Именно последние keep: проверка по количеству прошла бы и при обратной
	// сортировке в cleanupOldBackups, то есть при удалении свежих файлов.
	assert.Equal(t, created[len(created)-keep:], backupNames(t, backupDir))
}

func TestRunBackup_DatabaseOpenError(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	// Каталог вместо файла: SQLite не откроет такую «БД».
	t.Setenv("DATABASE_PATH", t.TempDir())

	var out bytes.Buffer
	require.Error(t, runBackup(context.Background(), nil, nil, &out))
	assert.Empty(t, out.String())
}

// Отсутствующий файл БД — ошибка, а не пустой бэкап: иначе cron рапортует об успехе
// по неверному DATABASE_PATH и вычищает настоящие копии.
func TestRunBackup_DatabaseMissing(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "absent.db"))

	var out bytes.Buffer
	require.Error(t, runBackup(context.Background(), nil, nil, &out))
	assert.Empty(t, out.String())
}

// Без --keep ретеншен берётся из BACKUP_KEEP.
func TestRunBackup_KeepFromEnv(t *testing.T) {
	backupDir := backupEnv(t)
	t.Setenv("BACKUP_KEEP", "1")

	created := make([]string, 0, 2)
	for range 2 {
		created = append(created, runBackupOnce(t, nil))
	}

	assert.Equal(t, created[1:], backupNames(t, backupDir))
}

func TestParseBackupArgs_InvalidFlag(t *testing.T) {
	_, err := parseBackupArgs([]string{"--nope"})
	require.Error(t, err)
}

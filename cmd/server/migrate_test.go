package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/testhelpers"
)

// migrateEnv — CWD корня репозитория (там ./migrations) и мигрированная БД во временном каталоге.
func migrateEnv(t *testing.T) *internal.Config {
	t.Helper()
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "migrate-cli.db"))

	cfg := internal.LoadConfig()
	db, err := internal.OpenDatabase(cfg)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	return cfg
}

func TestParseMigrateArgs_NoFlagsOnlyReads(t *testing.T) {
	target, set, err := parseMigrateArgs(nil)

	require.NoError(t, err)
	assert.False(t, set)
	assert.Equal(t, uint(0), target)
}

func TestParseMigrateArgs_To(t *testing.T) {
	target, set, err := parseMigrateArgs([]string{"--to", "2"})

	require.NoError(t, err)
	assert.True(t, set)
	assert.Equal(t, uint(2), target)
}

// TestParseMigrateArgs_ToZero — golang-migrate не умеет Migrate(0): без этой проверки
// откат «в ноль» отвечал бы "file does not exist".
func TestParseMigrateArgs_ToZero(t *testing.T) {
	_, _, err := parseMigrateArgs([]string{"--to", "0"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--to must be 1 or above")
}

func TestParseMigrateArgs_UnknownFlag(t *testing.T) {
	_, _, err := parseMigrateArgs([]string{"--down"})

	require.Error(t, err)
}

func TestRunMigrate_PrintsVersion(t *testing.T) {
	cfg := migrateEnv(t)
	version, _, err := internal.SchemaVersion(cfg)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, runMigrate(context.Background(), nil, nil, &out))

	assert.Equal(t, fmtVersion(version), out.String())

	after, _, err := internal.SchemaVersion(cfg)
	require.NoError(t, err)
	assert.Equal(t, version, after, "чтение версии не должно двигать схему")
}

func TestRunMigrate_ToLowersSchema(t *testing.T) {
	cfg := migrateEnv(t)
	version, _, err := internal.SchemaVersion(cfg)
	require.NoError(t, err)
	require.Greater(t, version, uint(1))

	var out bytes.Buffer
	require.NoError(t, runMigrate(context.Background(), []string{"--to", "1"}, nil, &out))

	assert.Equal(t, fmtVersion(1), out.String())
	lowered, _, err := internal.SchemaVersion(cfg)
	require.NoError(t, err)
	assert.Equal(t, uint(1), lowered)
}

// TestRunMigrate_EmptyDatabase — база есть, но ни одной миграции не применяли: это версия 0,
// а не ошибка (golang-migrate отдаёт здесь ErrNilVersion).
func TestRunMigrate_EmptyDatabase(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	path := filepath.Join(t.TempDir(), "empty.db")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	t.Setenv("DATABASE_PATH", path)

	var out bytes.Buffer
	require.NoError(t, runMigrate(context.Background(), nil, nil, &out))

	assert.Equal(t, fmtVersion(0), out.String())
}

func TestRunMigrate_MissingDatabase(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "absent.db"))

	var out bytes.Buffer
	err := runMigrate(context.Background(), nil, nil, &out)

	require.Error(t, err)
	assert.Empty(t, out.String())
}

func fmtVersion(version uint) string {
	return fmt.Sprintf("schema version %d (dirty: false)\n", version)
}

package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/services"
	"family-budget-service/internal/testhelpers"
)

// Подкоманды целиком: флаги → БД из DATABASE_PATH → миграции → семья/пароль.
func TestRunSetupAndResetPassword(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	t.Setenv("DATABASE_PATH", filepath.Join(t.TempDir(), "cli.db"))
	ctx := context.Background()

	var out bytes.Buffer
	require.NoError(t, runSetup(ctx, fullSetupArgs(), strings.NewReader("Admin1234!\n"), &out))
	assert.Equal(t, "family \"Test Family\" created, admin admin@example.com\n", out.String())

	err := runSetup(ctx, fullSetupArgs(), strings.NewReader("Admin1234!\n"), &out)
	require.ErrorIs(t, err, services.ErrFamilyAlreadyExists)

	out.Reset()
	require.NoError(t, runResetPassword(ctx,
		[]string{"--email", "admin@example.com", "--password-stdin"}, strings.NewReader("NewPassw0rd!\n"), &out))
	assert.Equal(t, "password for admin@example.com reset, all sessions revoked\n", out.String())

	err = runResetPassword(ctx,
		[]string{"--email", "nobody@example.com", "--password-stdin"}, strings.NewReader("NewPassw0rd!\n"), &out)
	require.ErrorIs(t, err, user.ErrNotFound)

	db, err := internal.OpenDatabase(internal.LoadConfig())
	require.NoError(t, err)
	defer db.Close()
	repos := infrastructure.NewRepositoriesSQLite(db)
	svc := auth.NewService(repos.Session, repos.User, repos.Family)
	_, err = svc.Login(ctx, "admin@example.com", "Admin1234!", "cli-test")
	require.ErrorIs(t, err, auth.ErrInvalidCredentials, "старый пароль отозван")
	_, err = svc.Login(ctx, "admin@example.com", "NewPassw0rd!", "cli-test")
	require.NoError(t, err)
}

func TestRunSetup_BadFlagsDoNotTouchDatabase(t *testing.T) {
	t.Chdir(testhelpers.RepoRoot(t))
	path := filepath.Join(t.TempDir(), "untouched.db")
	t.Setenv("DATABASE_PATH", path)

	err := runSetup(context.Background(), []string{"--family", "X"}, strings.NewReader(""), &bytes.Buffer{})

	require.ErrorContains(t, err, "missing required flags")
	assert.NoFileExists(t, path)
}

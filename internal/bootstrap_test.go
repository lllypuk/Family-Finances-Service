package internal_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

const (
	setupPassword = "Admin1234!"
	setupEmail    = "admin@example.com"
)

func setupParams() dto.SetupFamilyDTO {
	return dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "RUB",
		Timezone:   "Europe/Moscow",
		Email:      setupEmail,
		FirstName:  "Admin",
		LastName:   "User",
		Password:   setupPassword,
	}
}

func newAuthService(t *testing.T, db *sql.DB) *auth.Service {
	t.Helper()
	return auth.NewService(infrastructure.NewRepositoriesSQLite(db).Session,
		infrastructure.NewRepositoriesSQLite(db).User, infrastructure.NewRepositoriesSQLite(db).Family)
}

func TestOpenDatabase_AppliesMigrationsAndIsIdempotent(t *testing.T) {
	// migrations resolve relative to CWD, like the server and CLI
	t.Chdir(testhelpers.RepoRoot(t))
	cfg := internal.LoadConfig()
	cfg.Database.Path = filepath.Join(t.TempDir(), "bootstrap.db")

	db, err := internal.OpenDatabase(cfg)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sessions").Scan(&n))
	assert.Equal(t, 0, n)
	require.NoError(t, db.Close())

	db, err = internal.OpenDatabase(cfg)
	require.NoError(t, err, "second open must treat ErrNoChange as success")
	require.NoError(t, db.Close())
}

func TestSetup_CreatesFamilyOnceAndAdminCanLogin(t *testing.T) {
	db := testhelpers.SetupSQLiteTestDB(t).DB
	ctx := context.Background()

	family, err := internal.Setup(ctx, db, setupParams())
	require.NoError(t, err)
	assert.Equal(t, "Test Family", family.Name)

	var categories int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories").Scan(&categories))
	assert.Len(t, services.DefaultCategories(), categories)

	result, err := newAuthService(t, db).Login(ctx, setupEmail, setupPassword, "test")
	require.NoError(t, err)
	assert.Equal(t, user.RoleAdmin, result.User.Role)

	_, err = internal.Setup(ctx, db, setupParams())
	require.ErrorIs(t, err, services.ErrFamilyAlreadyExists)
}

func TestSetup_ValidationError(t *testing.T) {
	db := testhelpers.SetupSQLiteTestDB(t).DB
	p := setupParams()
	p.Password = "short"

	_, err := internal.Setup(context.Background(), db, p)
	require.ErrorIs(t, err, services.ErrValidationFailed)

	exists, err := infrastructure.NewRepositoriesSQLite(db).Family.Exists(context.Background())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestResetPassword(t *testing.T) {
	db := testhelpers.SetupSQLiteTestDB(t).DB
	ctx := context.Background()
	_, err := internal.Setup(ctx, db, setupParams())
	require.NoError(t, err)
	authService := newAuthService(t, db)

	t.Run("weak password rejected before lookup", func(t *testing.T) {
		require.ErrorIs(t, internal.ResetPassword(ctx, db, setupEmail, "short"), auth.ErrInvalidPassword)
	})

	t.Run("unknown email", func(t *testing.T) {
		require.ErrorIs(t, internal.ResetPassword(ctx, db, "nobody@example.com", "NewPassw0rd!"), user.ErrNotFound)
	})

	t.Run("sets password and revokes sessions", func(t *testing.T) {
		before, loginErr := authService.Login(ctx, setupEmail, setupPassword, "phone")
		require.NoError(t, loginErr)

		require.NoError(t, internal.ResetPassword(ctx, db, setupEmail, "NewPassw0rd!"))

		_, authErr := authService.Authenticate(ctx, before.Token)
		require.ErrorIs(t, authErr, auth.ErrUnauthorized)

		_, loginErr = authService.Login(ctx, setupEmail, setupPassword, "phone")
		require.ErrorIs(t, loginErr, auth.ErrInvalidCredentials)

		_, loginErr = authService.Login(ctx, setupEmail, "NewPassw0rd!", "phone")
		require.NoError(t, loginErr)
	})
}

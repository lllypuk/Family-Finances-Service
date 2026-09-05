package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	authrepo "family-budget-service/internal/infrastructure/auth"
	testutils "family-budget-service/internal/testhelpers"
)

func TestSessionRepositorySQLite_Integration(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	createUser := func(t *testing.T, email string) uuid.UUID {
		t.Helper()
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)
		userID, err := helper.CreateTestUser(ctx, email, "Anna", "Ivanova", "member", familyID)
		require.NoError(t, err)
		return uuid.MustParse(userID)
	}

	t.Run("Create_And_FindByTokenHash", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "anna@example.com")

		s := auth.NewSession(userID, "hash-1", "Pixel 8", now)
		require.NoError(t, repo.Create(ctx, s))

		found, owner, err := repo.FindByTokenHash(ctx, "hash-1")
		require.NoError(t, err)
		assert.Equal(t, s.ID, found.ID)
		assert.Equal(t, userID, found.UserID)
		assert.Equal(t, "Pixel 8", found.DeviceName)
		assert.True(t, found.CreatedAt.Equal(now))
		assert.True(t, found.LastUsedAt.Equal(now))
		assert.True(t, found.ExpiresAt.Equal(now.Add(auth.IdleTTL)))
		assert.Equal(t, userID, owner.ID)
		assert.Equal(t, "anna@example.com", owner.Email)
		assert.Equal(t, "Anna", owner.FirstName)
		assert.Equal(t, user.RoleMember, owner.Role)
	})

	t.Run("FindByTokenHash_UnknownHash", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))

		_, _, err := repo.FindByTokenHash(ctx, "missing")
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
	})

	t.Run("FindByTokenHash_InactiveUser", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := authrepo.NewSessionSQLiteRepository(db)
		userID := createUser(t, "inactive@example.com")
		require.NoError(t, repo.Create(ctx, auth.NewSession(userID, "hash-inactive", "d", now)))

		_, err := db.ExecContext(ctx, `UPDATE users SET is_active = 0 WHERE id = ?`, userID.String())
		require.NoError(t, err)

		_, _, err = repo.FindByTokenHash(ctx, "hash-inactive")
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
	})

	t.Run("Touch_ExtendsIdleAndCapsAtAbsolute", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "touch@example.com")

		fresh := auth.NewSession(userID, "hash-fresh", "d", now)
		require.NoError(t, repo.Create(ctx, fresh))
		old := auth.NewSession(userID, "hash-old", "d", now.Add(-auth.AbsoluteTTL+time.Hour))
		require.NoError(t, repo.Create(ctx, old))

		later := now.Add(2 * time.Hour)
		require.NoError(t, repo.Touch(ctx, fresh.ID, later))
		require.NoError(t, repo.Touch(ctx, old.ID, later))

		got, _, err := repo.FindByTokenHash(ctx, "hash-fresh")
		require.NoError(t, err)
		assert.True(t, got.LastUsedAt.Equal(later))
		assert.True(t, got.ExpiresAt.Equal(later.Add(auth.IdleTTL)))

		got, _, err = repo.FindByTokenHash(ctx, "hash-old")
		require.NoError(t, err)
		assert.True(t, got.LastUsedAt.Equal(later))
		assert.True(t, got.ExpiresAt.Equal(old.CreatedAt.Add(auth.AbsoluteTTL)))

		require.ErrorIs(t, repo.Touch(ctx, uuid.New(), later), auth.ErrSessionNotFound)
	})

	t.Run("Delete", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "delete@example.com")
		s := auth.NewSession(userID, "hash-del", "d", now)
		require.NoError(t, repo.Create(ctx, s))

		require.NoError(t, repo.Delete(ctx, s.ID))
		require.ErrorIs(t, repo.Delete(ctx, s.ID), auth.ErrSessionNotFound)

		_, _, err := repo.FindByTokenHash(ctx, "hash-del")
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
	})

	t.Run("DeleteByUser_KeepsException", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "many@example.com")
		otherID := createUserInSameFamily(ctx, t, container, userID, "other@example.com")

		keep := auth.NewSession(userID, "hash-keep", "phone", now)
		require.NoError(t, repo.Create(ctx, keep))
		require.NoError(t, repo.Create(ctx, auth.NewSession(userID, "hash-a", "tablet", now.Add(-time.Minute))))
		require.NoError(t, repo.Create(ctx, auth.NewSession(userID, "hash-b", "laptop", now.Add(-2*time.Minute))))
		require.NoError(t, repo.Create(ctx, auth.NewSession(otherID, "hash-foreign", "d", now)))

		require.NoError(t, repo.DeleteByUser(ctx, userID, keep.ID))

		left, err := repo.ListByUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, left, 1)
		assert.Equal(t, keep.ID, left[0].ID)

		otherLeft, err := repo.ListByUser(ctx, otherID)
		require.NoError(t, err)
		assert.Len(t, otherLeft, 1)

		require.NoError(t, repo.DeleteByUser(ctx, userID, uuid.Nil))
		left, err = repo.ListByUser(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, left)
	})

	t.Run("ListByUser_NewestFirst", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "list@example.com")
		older := auth.NewSession(userID, "hash-older", "d", now.Add(-time.Hour))
		newer := auth.NewSession(userID, "hash-newer", "d", now)
		require.NoError(t, repo.Create(ctx, older))
		require.NoError(t, repo.Create(ctx, newer))

		list, err := repo.ListByUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, list, 2)
		assert.Equal(t, newer.ID, list[0].ID)
		assert.Equal(t, older.ID, list[1].ID)
	})

	t.Run("DeleteExpired_OnlyExpired", func(t *testing.T) {
		repo := authrepo.NewSessionSQLiteRepository(container.GetTestDatabase(t))
		userID := createUser(t, "expired@example.com")

		alive := auth.NewSession(userID, "hash-alive", "d", now)
		require.NoError(t, repo.Create(ctx, alive))
		expired := auth.NewSession(userID, "hash-expired", "d", now.Add(-auth.IdleTTL-time.Hour))
		require.NoError(t, repo.Create(ctx, expired))

		require.NoError(t, repo.DeleteExpired(ctx, now))

		list, err := repo.ListByUser(ctx, userID)
		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, alive.ID, list[0].ID)
	})
}

// createUserInSameFamily добавляет второго пользователя в семью существующего.
func createUserInSameFamily(
	ctx context.Context,
	t *testing.T,
	container *testutils.SQLiteTestDB,
	existing uuid.UUID,
	email string,
) uuid.UUID {
	t.Helper()
	var familyID string
	err := container.DB.QueryRowContext(ctx, `SELECT family_id FROM users WHERE id = ?`, existing.String()).
		Scan(&familyID)
	require.NoError(t, err)
	id, err := testutils.NewTestDataHelper(container.DB).
		CreateTestUser(ctx, email, "Ivan", "Petrov", "member", familyID)
	require.NoError(t, err)
	return uuid.MustParse(id)
}

func TestSession_ExpiryAfter(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := auth.NewSession(uuid.New(), "h", "d", created)

	assert.Equal(t, created.Add(auth.IdleTTL), s.ExpiresAt)
	assert.Equal(t, created.Add(auth.AbsoluteTTL), s.ExpiryAfter(created.Add(auth.AbsoluteTTL-time.Hour)))
}

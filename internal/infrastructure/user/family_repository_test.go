package user_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/user"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	userrepo "family-budget-service/internal/infrastructure/user"
	testutils "family-budget-service/internal/testhelpers"
)

func bootstrapFixtures() (*user.Family, []*category.Category, *user.User) {
	family := &user.Family{ID: uuid.New(), Name: "Bootstrap Family", Currency: "RUB", Timezone: "Europe/Moscow"}
	categories := []*category.Category{
		category.NewCategory("Продукты", category.TypeExpense),
		category.NewCategory("Зарплата", category.TypeIncome),
	}
	admin := user.NewUser("admin@example.com", "Admin", "User", user.RoleAdmin)
	admin.Password = "hashed"
	return family, categories, admin
}

func newFamilyRepo(db *sql.DB) *userrepo.SQLiteFamilyRepository {
	return userrepo.NewSQLiteFamilyRepository(
		db,
		categoryrepo.NewSQLiteRepository(db),
		userrepo.NewSQLiteRepository(db),
	)
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&n))
	return n
}

func TestFamilyRepository_Bootstrap(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	ctx := context.Background()

	t.Run("creates family, categories and admin", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := newFamilyRepo(db)
		family, categories, admin := bootstrapFixtures()

		require.NoError(t, repo.Bootstrap(ctx, family, categories, admin))

		exists, err := repo.Exists(ctx)
		require.NoError(t, err)
		assert.True(t, exists)
		assert.Len(t, categories, countRows(t, db, "categories"))

		stored, err := userrepo.NewSQLiteRepository(db).GetByEmail(ctx, admin.Email)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, stored.Role)
		assert.True(t, stored.IsActive)
	})

	t.Run("second bootstrap fails with ErrFamilyExists and changes nothing", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := newFamilyRepo(db)
		family, categories, admin := bootstrapFixtures()
		require.NoError(t, repo.Bootstrap(ctx, family, categories, admin))

		second, secondCategories, secondAdmin := bootstrapFixtures()
		secondAdmin.Email = "other@example.com"
		err := repo.Bootstrap(ctx, second, secondCategories, secondAdmin)
		require.ErrorIs(t, err, user.ErrFamilyExists)

		assert.Equal(t, 1, countRows(t, db, "families"))
		assert.Equal(t, 1, countRows(t, db, "users"))
		assert.Len(t, categories, countRows(t, db, "categories"))
	})

	t.Run("admin failure rolls back family and categories", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := newFamilyRepo(db)
		family, categories, admin := bootstrapFixtures()
		admin.Password = "" // CHECK (LENGTH(TRIM(password_hash)) > 0)

		require.Error(t, repo.Bootstrap(ctx, family, categories, admin))

		exists, err := repo.Exists(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
		assert.Equal(t, 0, countRows(t, db, "categories"))
	})

	t.Run("invalid category name rolls back family", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := newFamilyRepo(db)
		family, categories, admin := bootstrapFixtures()
		categories = append(categories, category.NewCategory("", category.TypeExpense))

		require.Error(t, repo.Bootstrap(ctx, family, categories, admin))

		exists, err := repo.Exists(ctx)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFamilyRepository_SchemaConstraints(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	ctx := context.Background()

	t.Run("second family violates singleton UNIQUE", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		family, categories, admin := bootstrapFixtures()
		require.NoError(t, newFamilyRepo(db).Bootstrap(ctx, family, categories, admin))

		_, err := db.ExecContext(ctx,
			`INSERT INTO families (id, name, currency) VALUES (?, 'Second', 'RUB')`, uuid.New().String())
		require.ErrorContains(t, err, "UNIQUE constraint failed")
		assert.Equal(t, 1, countRows(t, db, "families"))
	})

	t.Run("role child violates CHECK", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		family, categories, admin := bootstrapFixtures()
		require.NoError(t, newFamilyRepo(db).Bootstrap(ctx, family, categories, admin))

		_, err := db.ExecContext(ctx,
			`INSERT INTO users (id, email, password_hash, first_name, last_name, role, family_id)
			 VALUES (?, 'child@example.com', 'hashed', 'Child', 'User', 'child', ?)`,
			uuid.New().String(), family.ID.String())
		require.ErrorContains(t, err, "CHECK constraint failed")
		assert.Equal(t, 1, countRows(t, db, "users"))
	})
}

// Часовой пояс переживает запись и чтение: по нему считаются границы периодов,
// а пустая колонка отбивается CHECK-ом.
func TestFamilyRepository_TimezoneRoundTrip(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	ctx := context.Background()
	db := container.GetTestDatabase(t)
	repo := newFamilyRepo(db)

	family := &user.Family{ID: uuid.New(), Name: "TZ Family", Currency: "RUB", Timezone: "Europe/Moscow"}
	require.NoError(t, repo.Create(ctx, family))

	stored, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Europe/Moscow", stored.Timezone)
	assert.Equal(t, "Europe/Moscow", stored.Location().String())

	stored.Timezone = "UTC"
	require.NoError(t, repo.Update(ctx, stored))

	updated, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "UTC", updated.Timezone)
}

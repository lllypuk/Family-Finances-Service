package holding_test

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	holdingrepo "family-budget-service/internal/infrastructure/holding"
	"family-budget-service/internal/testhelpers"
)

// today — фиксированный «сегодня» для всех тестов пакета.
func today() date.Date {
	return date.New(2026, 9, 18)
}

func setupRepo(t *testing.T) (*holdingrepo.SQLiteRepository, *sql.DB) {
	t.Helper()

	container := testhelpers.SetupSQLiteTestDB(t)
	_, err := testhelpers.NewTestDataHelper(container.DB).CreateTestFamily(t.Context(), "Family", "RUB")
	require.NoError(t, err)

	return holdingrepo.NewSQLiteRepository(container.DB), container.DB
}

func create(t *testing.T, repo *holdingrepo.SQLiteRepository, name string, side holding.Side) *holding.Holding {
	t.Helper()

	h := testhelpers.CreateTestHolding(name, side, holding.KindOther)
	require.NoError(t, repo.Create(t.Context(), h))

	return h
}

func putValue(t *testing.T, db *sql.DB, id uuid.UUID, day string, value int64) {
	t.Helper()

	_, err := db.ExecContext(t.Context(),
		`INSERT INTO holding_values (holding_id, date, value_minor) VALUES (?, ?, ?)`, id.String(), day, value)
	require.NoError(t, err)
}

func archive(t *testing.T, repo *holdingrepo.SQLiteRepository, id uuid.UUID) {
	t.Helper()

	archived := true
	require.NoError(t, repo.Update(t.Context(), id, nil, nil, &archived))
}

func TestHoldingRepository_Create_GetByID(t *testing.T) {
	repo, _ := setupRepo(t)
	created := create(t, repo, "Квартира", holding.SideAsset)

	got, err := repo.GetByID(t.Context(), created.ID, today())
	require.NoError(t, err)
	assert.Equal(t, "Квартира", got.Name)
	assert.Equal(t, holding.SideAsset, got.Side)
	assert.Equal(t, holding.KindOther, got.Kind)
	assert.False(t, got.IsArchived)
	assert.Nil(t, got.Current, "без снимков current пуст")

	_, err = repo.GetByID(t.Context(), uuid.New(), today())
	require.ErrorIs(t, err, holding.ErrNotFound)
}

func TestHoldingRepository_Current_LatestByDateNotAfterToday(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Вклад", holding.SideAsset)

	putValue(t, db, h.ID, "2026-09-01", 300)
	putValue(t, db, h.ID, "2026-08-01", 200)
	putValue(t, db, h.ID, "2026-09-19", 999)

	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	assert.Equal(
		t,
		date.New(2026, 9, 1),
		got.Current.Date,
		"позже записанный август не текущий; завтрашний снимок отсечён",
	)
	assert.Equal(t, money.Minor(300), got.Current.ValueMinor)

	list, err := repo.List(t.Context(), false, today())
	require.NoError(t, err)
	require.Len(t, list, 1, "одна строка на позицию")
	assert.Equal(t, money.Minor(300), list[0].Current.ValueMinor)

	got, err = repo.GetByID(t.Context(), h.ID, date.New(2026, 9, 19))
	require.NoError(t, err)
	assert.Equal(t, money.Minor(999), got.Current.ValueMinor)
}

func TestHoldingRepository_Create_NameKeyUnique(t *testing.T) {
	repo, _ := setupRepo(t)
	old := create(t, repo, "Машина", holding.SideAsset)
	archive(t, repo, old.ID)

	err := repo.Create(t.Context(), testhelpers.CreateTestHolding(" машина ", holding.SideLiability, holding.KindLoan))
	require.ErrorIs(t, err, holding.ErrNameExists, "архивная позиция имя занимает, сторона не важна")

	dup := testhelpers.CreateTestHolding("Другое", holding.SideAsset, holding.KindOther)
	dup.ID = old.ID
	err = repo.Create(t.Context(), dup)
	require.Error(t, err)
	assert.NotErrorIs(t, err, holding.ErrNameExists)
}

func TestHoldingRepository_List_Archived(t *testing.T) {
	repo, _ := setupRepo(t)
	create(t, repo, "ипотека", holding.SideLiability)
	create(t, repo, "Вклад", holding.SideAsset)
	old := create(t, repo, "Архив", holding.SideAsset)
	archive(t, repo, old.ID)

	active, err := repo.List(t.Context(), false, today())
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "Вклад", active[0].Name)
	assert.Equal(t, "ипотека", active[1].Name)

	all, err := repo.List(t.Context(), true, today())
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, "Архив", all[0].Name)
	assert.True(t, all[0].IsArchived)
}

func TestHoldingRepository_Update_KeepsFieldsNotGiven(t *testing.T) {
	repo, _ := setupRepo(t)
	h := create(t, repo, "Карта", holding.SideLiability)
	create(t, repo, "Кредит", holding.SideLiability)

	kind := holding.KindCreditCard
	require.NoError(t, repo.Update(t.Context(), h.ID, nil, &kind, nil))
	archive(t, repo, h.ID)
	name := "Карта Сбер"
	require.NoError(t, repo.Update(t.Context(), h.ID, &name, nil, nil))

	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, "Карта Сбер", got.Name)
	assert.Equal(t, holding.KindCreditCard, got.Kind)
	assert.True(t, got.IsArchived)
	assert.Equal(t, holding.SideLiability, got.Side)

	clash := "КРЕДИТ"
	require.ErrorIs(t, repo.Update(t.Context(), h.ID, &clash, nil, nil), holding.ErrNameExists)
	require.ErrorIs(t, repo.Update(t.Context(), uuid.New(), &name, nil, nil), holding.ErrNotFound)
}

func TestHoldingRepository_Delete_CascadesValues(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Вклад", holding.SideAsset)
	putValue(t, db, h.ID, "2026-09-01", 100)

	require.NoError(t, repo.Delete(t.Context(), h.ID))
	require.ErrorIs(t, repo.Delete(t.Context(), h.ID), holding.ErrNotFound)

	var n int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM holding_values`).Scan(&n))
	assert.Zero(t, n)
}

func TestHoldingRepository_UpsertValue_ReplacesOnSameDate(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Вклад", holding.SideAsset)
	day := date.New(2026, 9, 1)

	first := &holding.Value{Date: day, ValueMinor: 100}
	require.NoError(t, repo.UpsertValue(t.Context(), h.ID, first))
	assert.False(t, first.UpdatedAt.IsZero())
	require.NoError(t, repo.UpsertValue(t.Context(), h.ID, &holding.Value{Date: day, ValueMinor: 0}))

	var n int
	require.NoError(t, db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM holding_values`).Scan(&n))
	assert.Equal(t, 1, n)
	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	require.NotNil(t, got.Current)
	assert.Zero(t, got.Current.ValueMinor, "0 заменяет прежнее значение")

	err = repo.UpsertValue(t.Context(), uuid.New(), &holding.Value{Date: day, ValueMinor: 1})
	require.ErrorIs(t, err, holding.ErrNotFound)
}

func TestHoldingRepository_Current_AfterDeleteValue(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Вклад", holding.SideAsset)
	putValue(t, db, h.ID, "2026-07-01", 100)
	putValue(t, db, h.ID, "2026-08-01", 200)
	putValue(t, db, h.ID, "2026-09-01", 300)

	current := func() *holding.Value {
		got, err := repo.GetByID(t.Context(), h.ID, today())
		require.NoError(t, err)
		return got.Current
	}

	require.NoError(t, repo.DeleteValue(t.Context(), h.ID, date.New(2026, 8, 1)))
	assert.Equal(t, money.Minor(300), current().ValueMinor, "удалён непоследний — current прежний")

	require.NoError(t, repo.DeleteValue(t.Context(), h.ID, date.New(2026, 9, 1)))
	assert.Equal(t, date.New(2026, 7, 1), current().Date, "удалён последний — current предыдущий")

	require.NoError(t, repo.DeleteValue(t.Context(), h.ID, date.New(2026, 7, 1)))
	assert.Nil(t, current(), "удалён единственный — current пуст")

	require.ErrorIs(t, repo.DeleteValue(t.Context(), h.ID, date.New(2026, 7, 1)), holding.ErrValueNotFound)
}

func TestHoldingRepository_ListValues_NewestFirstWithTotal(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Вклад", holding.SideAsset)
	other := create(t, repo, "Ипотека", holding.SideLiability)
	putValue(t, db, h.ID, "2026-07-01", 100)
	putValue(t, db, h.ID, "2026-09-01", 300)
	putValue(t, db, h.ID, "2026-08-01", 200)
	putValue(t, db, other.ID, "2026-09-01", 900)

	values, total, err := repo.ListValues(t.Context(), h.ID, 2, 1)
	require.NoError(t, err)
	assert.Equal(t, 3, total, "total — все снимки позиции, не страница")
	require.Len(t, values, 2)
	assert.Equal(t, date.New(2026, 8, 1), values[0].Date)
	assert.Equal(t, date.New(2026, 7, 1), values[1].Date)
	assert.False(t, values[0].UpdatedAt.IsZero())
}

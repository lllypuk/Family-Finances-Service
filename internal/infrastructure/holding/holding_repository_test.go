package holding_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/infrastructure"
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
	require.NoError(t, repo.Update(t.Context(), id, nil, nil, &archived, nil, nil))
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
	require.NoError(t, repo.Update(t.Context(), h.ID, nil, &kind, nil, nil, nil))
	archive(t, repo, h.ID)
	name := "Карта Сбер"
	require.NoError(t, repo.Update(t.Context(), h.ID, &name, nil, nil, nil, nil))

	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, "Карта Сбер", got.Name)
	assert.Equal(t, holding.KindCreditCard, got.Kind)
	assert.True(t, got.IsArchived)
	assert.Equal(t, holding.SideLiability, got.Side)

	restored := false
	require.NoError(t, repo.Update(t.Context(), h.ID, nil, nil, &restored, nil, nil))
	got, err = repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.False(t, got.IsArchived, "false — это значение, а не «не передано»")

	clash := "КРЕДИТ"
	require.ErrorIs(t, repo.Update(t.Context(), h.ID, &clash, nil, nil, nil, nil), holding.ErrNameExists)
	require.ErrorIs(t, repo.Update(t.Context(), uuid.New(), &name, nil, nil, nil, nil), holding.ErrNotFound)
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

func TestHoldingRepository_SeriesValues(t *testing.T) {
	repo, db := setupRepo(t)
	flat := create(t, repo, "Квартира", holding.SideAsset)
	loan := create(t, repo, "Кредит", holding.SideLiability)
	old := create(t, repo, "Машина", holding.SideAsset)
	create(t, repo, "Пустая", holding.SideAsset)

	putValue(t, db, flat.ID, "2025-01-01", 100)
	putValue(t, db, flat.ID, "2025-06-01", 200)
	putValue(t, db, flat.ID, "2025-12-01", 300)
	putValue(t, db, flat.ID, "2026-04-01", 400)
	putValue(t, db, loan.ID, "2026-03-05", 50)
	putValue(t, db, loan.ID, "2026-03-20", 40)
	putValue(t, db, loan.ID, "2026-06-01", 10)
	edge := create(t, repo, "Вклад", holding.SideAsset)
	putValue(t, db, edge.ID, "2026-03-15", 5)
	putValue(t, db, edge.ID, "2026-05-31", 6)
	putValue(t, db, old.ID, "2025-02-01", 70)
	archive(t, repo, old.ID)

	rows, err := repo.SeriesValues(t.Context(), date.New(2026, 3, 15), date.New(2026, 5, 31))
	require.NoError(t, err)

	type got struct {
		id    uuid.UUID
		side  holding.Side
		day   string
		value money.Minor
	}
	actual := make([]got, 0, len(rows))
	for _, r := range rows {
		actual = append(actual, got{r.HoldingID, r.Side, r.Date.String(), r.ValueMinor})
	}
	assert.Equal(t, []got{
		{old.ID, holding.SideAsset, "2025-02-01", 70},
		{flat.ID, holding.SideAsset, "2025-12-01", 300},
		{loan.ID, holding.SideLiability, "2026-03-05", 50},
		{edge.ID, holding.SideAsset, "2026-03-15", 5},
		{loan.ID, holding.SideLiability, "2026-03-20", 40},
		{flat.ID, holding.SideAsset, "2026-04-01", 400},
		{edge.ID, holding.SideAsset, "2026-05-31", 6},
	}, actual, "из снимков до from — только последний, в том числе внутри месяца from; архивная на месте; по дате")
}

func minor(v int64) *money.Minor {
	m := money.Minor(v)
	return &m
}

func planRows(t *testing.T, db *sql.DB, id uuid.UUID) int {
	t.Helper()

	var n int
	require.NoError(t, db.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM holding_plans WHERE holding_id = ?`, id.String()).Scan(&n))
	return n
}

func TestHoldingRepository_Create_Plan(t *testing.T) {
	repo, db := setupRepo(t)

	withPlan := testhelpers.CreateTestHolding("Квартира", holding.SideAsset, holding.KindProperty)
	withPlan.Plan = holding.Plan{MonthlyIncomeMinor: 4_500_000, MonthlyExpenseMinor: 830_000}
	require.NoError(t, repo.Create(t.Context(), withPlan))
	require.NotNil(t, withPlan.Plan.UpdatedAt, "POST отдаёт дату плана без перечитывания")

	got, err := repo.GetByID(t.Context(), withPlan.ID, today())
	require.NoError(t, err)
	assert.Equal(t, money.Minor(4_500_000), got.Plan.MonthlyIncomeMinor)
	assert.Equal(t, money.Minor(830_000), got.Plan.MonthlyExpenseMinor)
	require.NotNil(t, got.Plan.UpdatedAt)
	assert.True(t, withPlan.Plan.UpdatedAt.Equal(*got.Plan.UpdatedAt))

	plain := create(t, repo, "Вклад", holding.SideAsset)
	assert.Nil(t, plain.Plan.UpdatedAt)
	assert.Equal(t, 0, planRows(t, db, plain.ID), "0/0 строку плана не пишет")
	got, err = repo.GetByID(t.Context(), plain.ID, today())
	require.NoError(t, err)
	assert.Equal(t, holding.Plan{}, got.Plan)
}

func TestHoldingRepository_Update_Plan(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Ипотека", holding.SideLiability)

	require.NoError(t, repo.Update(t.Context(), h.ID, nil, nil, nil, minor(10_000), minor(7_430_000)))
	first, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	require.NotNil(t, first.Plan.UpdatedAt)
	assert.Equal(t, money.Minor(10_000), first.Plan.MonthlyIncomeMinor, "доход у пассива законен")

	require.NoError(t, repo.Update(t.Context(), h.ID, nil, nil, nil, nil, minor(7_000_000)))
	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, money.Minor(10_000), got.Plan.MonthlyIncomeMinor, "неприсланное число сохраняется")
	assert.Equal(t, money.Minor(7_000_000), got.Plan.MonthlyExpenseMinor)
	assert.True(t, got.Plan.UpdatedAt.After(*first.Plan.UpdatedAt), "правка плана двигает дату")

	planAt := *got.Plan.UpdatedAt
	name := "Ипотека ВТБ"
	archived := true
	require.NoError(t, repo.Update(t.Context(), h.ID, &name, nil, &archived, nil, nil))
	got, err = repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.True(t, planAt.Equal(*got.Plan.UpdatedAt), "переименование и архив дату плана не двигают")
	assert.Equal(t, money.Minor(7_000_000), got.Plan.MonthlyExpenseMinor)

	require.NoError(t, repo.Update(t.Context(), h.ID, nil, nil, nil, minor(0), minor(0)))
	assert.Equal(t, 0, planRows(t, db, h.ID), "0/0 удаляет строку")
	got, err = repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, holding.Plan{}, got.Plan)

	require.ErrorIs(t, repo.Update(t.Context(), uuid.New(), nil, nil, nil, minor(1), nil), holding.ErrNotFound)
}

func TestHoldingRepository_Update_PlanRefusalRollsBackRename(t *testing.T) {
	repo, db := setupRepo(t)
	h := create(t, repo, "Машина", holding.SideAsset)
	require.NoError(t, repo.Update(t.Context(), h.ID, nil, nil, nil, nil, minor(500_000)))

	name := "Машина Лада"
	err := repo.Update(t.Context(), h.ID, &name, nil, nil, minor(-1), nil)
	var planErr *holding.PlanError
	require.ErrorAs(t, err, &planErr)
	assert.Equal(t, holding.FieldMonthlyIncome, planErr.Field)

	got, err := repo.GetByID(t.Context(), h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, "Машина", got.Name, "отказ плана откатывает переименование")
	assert.Equal(t, money.Minor(500_000), got.Plan.MonthlyExpenseMinor)
	assert.Equal(t, 1, planRows(t, db, h.ID))
}

// Тестовый DSN не несёт MaxOpenConns=1 и _txlock=immediate; чтение через r.db внутри транзакции
// повисло бы только здесь.
func TestHoldingRepository_Update_PlanOnProductionConnection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "prod.db")
	manager := infrastructure.NewMigrationManager("sqlite://"+dbPath,
		filepath.Join(testhelpers.RepoRoot(t), "migrations"))
	require.NoError(t, manager.Up())

	conn, err := infrastructure.NewSQLiteConnection(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	_, err = testhelpers.NewTestDataHelper(conn.DB()).CreateTestFamily(t.Context(), "Family", "RUB")
	require.NoError(t, err)

	repo := holdingrepo.NewSQLiteRepository(conn.DB())
	h := testhelpers.CreateTestHolding("Квартира", holding.SideAsset, holding.KindProperty)
	require.NoError(t, repo.Create(t.Context(), h))

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, repo.Update(ctx, h.ID, nil, nil, nil, minor(4_500_000), nil))
	require.NoError(t, repo.Update(ctx, h.ID, nil, nil, nil, nil, minor(830_000)))

	got, err := repo.GetByID(ctx, h.ID, today())
	require.NoError(t, err)
	assert.Equal(t, money.Minor(4_500_000), got.Plan.MonthlyIncomeMinor)
	assert.Equal(t, money.Minor(830_000), got.Plan.MonthlyExpenseMinor)
}

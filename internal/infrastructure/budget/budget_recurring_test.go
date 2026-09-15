package budget_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	testutils "family-budget-service/internal/testhelpers"
)

const recurringAmount = money.Minor(100_000)

// recurringRepo — репозиторий на свежей базе и id категории, на которую можно ссылаться.
func recurringRepo(t *testing.T) (*budgetrepo.SQLiteRepository, context.Context, uuid.UUID) {
	t.Helper()

	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
	require.NoError(t, err)

	categoryID, err := helper.CreateTestCategory(ctx, "Еда", "expense", familyID, nil)
	require.NoError(t, err)

	return budgetrepo.NewSQLiteRepository(container.DB), ctx, uuid.MustParse(categoryID)
}

// monthBudget — бюджет на календарный месяц, в котором лежит day.
func monthBudget(name string, day date.Date, recurring bool) *budget.Budget {
	start, end := day.MonthBounds()
	id := uuid.New()
	b := &budget.Budget{
		ID:          id,
		Name:        name,
		AmountMinor: recurringAmount,
		Period:      budget.PeriodMonthly,
		StartDate:   start,
		EndDate:     end,
		IsActive:    true,
		Recurring:   recurring,
	}
	if recurring {
		b.SeriesID = &id
	}

	return b
}

func TestBudgetRepositorySQLite_Create_RoundTripsRecurringColumns(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 15), true)
	require.NoError(t, repo.Create(ctx, tail))

	stored, err := repo.GetByID(ctx, tail.ID)
	require.NoError(t, err)
	assert.True(t, stored.Recurring)
	require.NotNil(t, stored.SeriesID)
	assert.Equal(t, tail.ID, *stored.SeriesID)

	plain := monthBudget("Разовый", date.New(2026, time.August, 1), false)
	require.NoError(t, repo.Create(ctx, plain))

	stored, err = repo.GetByID(ctx, plain.ID)
	require.NoError(t, err)
	assert.False(t, stored.Recurring)
	assert.Nil(t, stored.SeriesID)
}

func TestBudgetRepositorySQLite_Create_PeriodTaken(t *testing.T) {
	repo, ctx, categoryID := recurringRepo(t)

	first := monthBudget("Продукты", date.New(2026, time.June, 1), false)
	first.CategoryID = nil
	require.NoError(t, repo.Create(ctx, first))

	t.Run("SameScopeOverlappingDates", func(t *testing.T) {
		other := monthBudget("Другое имя", date.New(2026, time.June, 10), false)
		err := repo.Create(ctx, other)
		require.ErrorIs(t, err, budget.ErrOverlap)
		assert.Contains(t, err.Error(), "Продукты")
	})

	// Имя занято в другой области: подвинуть даты не поможет, поможет переименование,
	// поэтому отказ должен отличаться от пересечения области.
	t.Run("SameNameOtherScope", func(t *testing.T) {
		other := monthBudget("Продукты", date.New(2026, time.June, 10), false)
		other.CategoryID = &categoryID
		err := repo.Create(ctx, other)
		require.ErrorIs(t, err, budget.ErrNameExists)
		assert.NotErrorIs(t, err, budget.ErrOverlap)
	})

	t.Run("FreePeriod", func(t *testing.T) {
		next := monthBudget("Продукты", date.New(2026, time.July, 1), false)
		require.NoError(t, repo.Create(ctx, next))
	})
}

// Update пересчитывает spent_minor в своей транзакции: присланная сумма игнорируется, и
// расход, записанный между сдвигом дат и отдельной записью расхода, не может быть затёрт.
func TestBudgetRepositorySQLite_Update_SyncsSpent(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
	require.NoError(t, err)
	categoryID, err := helper.CreateTestCategory(ctx, "Еда", "expense", familyID, nil)
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "user@test.com", "Test", "User", "admin", familyID)
	require.NoError(t, err)

	repo := budgetrepo.NewSQLiteRepository(container.DB)
	b := monthBudget("Продукты", date.New(2026, time.June, 1), false)
	require.NoError(t, repo.Create(ctx, b))

	_, err = helper.CreateTestTransactionWithDate(
		ctx, 40_000, "Обед", "expense", categoryID, userID, familyID, "2026-06-10")
	require.NoError(t, err)
	_, err = helper.CreateTestTransactionWithDate(
		ctx, 15_000, "Ужин", "expense", categoryID, userID, familyID, "2026-06-20")
	require.NoError(t, err)

	b.Name = "Еда"
	b.SpentMinor = 0
	require.NoError(t, repo.Update(ctx, b, budget.UpdateExpect{}))
	assert.Equal(t, money.Minor(55_000), b.SpentMinor)

	stored, err := repo.GetByID(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "Еда", stored.Name)
	assert.Equal(t, money.Minor(55_000), stored.SpentMinor)

	b.EndDate = date.New(2026, time.June, 15)
	require.NoError(t, repo.Update(ctx, b, budget.UpdateExpect{}))
	assert.Equal(t, money.Minor(40_000), b.SpentMinor)

	stored, err = repo.GetByID(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, money.Minor(40_000), stored.SpentMinor)
}

func TestBudgetRepositorySQLite_Update_StaleRecurringFlag(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	require.NoError(t, repo.Create(ctx, tail))

	tail.Name = "Еда"
	require.ErrorIs(t, repo.Update(ctx, tail, budget.UpdateExpect{}), budget.ErrNotTail)
}

// twoInstanceSeries — прошлый инстанс и последний, флага нет ни у одного: каждый подтест
// строит серию заново, иначе они читают состояние соседа и по одному не запускаются.
func twoInstanceSeries(t *testing.T) (*budgetrepo.SQLiteRepository, context.Context, *budget.Budget, *budget.Budget) {
	t.Helper()

	repo, ctx, _ := recurringRepo(t)

	past := monthBudget("Продукты", date.New(2026, time.June, 1), false)
	past.SeriesID = &past.ID
	require.NoError(t, repo.Create(ctx, past))

	last := monthBudget("Продукты", date.New(2026, time.July, 1), false)
	last.SeriesID = &past.ID
	require.NoError(t, repo.Create(ctx, last))

	return repo, ctx, past, last
}

func TestBudgetRepositorySQLite_Update_TailRule(t *testing.T) {
	t.Run("PastInstanceCannotBecomeTail", func(t *testing.T) {
		repo, ctx, past, _ := twoInstanceSeries(t)

		past.Recurring = true
		require.ErrorIs(t, repo.Update(ctx, past, budget.UpdateExpect{}), budget.ErrNotTail)
	})

	t.Run("LastInstanceBecomesTail", func(t *testing.T) {
		repo, ctx, _, last := twoInstanceSeries(t)

		last.Recurring = true
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{}))

		stored, err := repo.GetByID(ctx, last.ID)
		require.NoError(t, err)
		assert.True(t, stored.Recurring)
	})

	t.Run("SeriesWithTailRefusesSecondOne", func(t *testing.T) {
		repo, ctx, past, last := twoInstanceSeries(t)

		last.Recurring = true
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{}))

		past.Recurring = true
		require.ErrorIs(t, repo.Update(ctx, past, budget.UpdateExpect{}), budget.ErrNotTail)
	})

	// Снятие флага с прошлого инстанса, пока серия жива: проверка живого хвоста идёт внутри
	// транзакции записи, иначе возобновление между чтением и записью дало бы ложный успех.
	t.Run("StopRefusedWhileSeriesHasTail", func(t *testing.T) {
		repo, ctx, past, last := twoInstanceSeries(t)

		last.Recurring = true
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{}))

		require.ErrorIs(t,
			repo.Update(ctx, past, budget.UpdateExpect{StopSeries: true}),
			budget.ErrNotTail)
	})

	// Правка прошлого инстанса без поля recurring хвоста не касается.
	t.Run("PastInstanceEditableWhileSeriesHasTail", func(t *testing.T) {
		repo, ctx, past, last := twoInstanceSeries(t)

		last.Recurring = true
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{}))

		past.Name = "Продукты (июнь)"
		require.NoError(t, repo.Update(ctx, past, budget.UpdateExpect{}))
	})

	t.Run("StopOnStoppedSeriesIsNoop", func(t *testing.T) {
		repo, ctx, past, _ := twoInstanceSeries(t)

		require.NoError(t, repo.Update(ctx, past, budget.UpdateExpect{StopSeries: true}))
	})

	t.Run("TailStops", func(t *testing.T) {
		repo, ctx, _, last := twoInstanceSeries(t)

		last.Recurring = true
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{}))

		last.Recurring = false
		require.NoError(t, repo.Update(ctx, last, budget.UpdateExpect{Recurring: true}))

		stored, err := repo.GetByID(ctx, last.ID)
		require.NoError(t, err)
		assert.False(t, stored.Recurring)
	})
}

func TestBudgetRepositorySQLite_Advance_BuildsWholeSeries(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	require.NoError(t, repo.Create(ctx, tail))

	created, err := repo.Advance(ctx, tail.ID, date.New(2026, time.September, 14))
	require.NoError(t, err)
	assert.Equal(t, 3, created)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 4)

	tails, err := repo.ListRecurring(ctx)
	require.NoError(t, err)
	require.Len(t, tails, 1)
	assert.Equal(t, date.New(2026, time.September, 1), tails[0].StartDate)
	require.NotNil(t, tails[0].SeriesID)
	assert.Equal(t, tail.ID, *tails[0].SeriesID)
	assert.Equal(t, money.Minor(0), tails[0].SpentMinor)

	_, err = repo.Advance(ctx, tail.ID, date.New(2026, time.September, 14))
	require.ErrorIs(t, err, budget.ErrNotTail)
}

// Инстанс достраивается задним числом: расход своего периода у него уже есть, и лимит
// транзакции читает spent_minor из БД без пересчёта.
func TestBudgetRepositorySQLite_Advance_MaterializedInstanceCarriesSpent(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "spent@example.com", "Spent", "Test", "admin", familyID)
	require.NoError(t, err)
	categoryID, err := helper.CreateTestCategory(ctx, "Еда", "expense", familyID, nil)
	require.NoError(t, err)

	repo := budgetrepo.NewSQLiteRepository(container.DB)
	catID := uuid.MustParse(categoryID)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	tail.CategoryID = &catID
	require.NoError(t, repo.Create(ctx, tail))

	_, err = helper.CreateTestTransactionWithDate(ctx, 30_000, "Магазин", "expense",
		categoryID, userID, familyID, "2026-07-10")
	require.NoError(t, err)

	created, err := repo.Advance(ctx, tail.ID, date.New(2026, time.August, 5))
	require.NoError(t, err)
	assert.Equal(t, 2, created)

	july, err := repo.GetActiveBudgets(ctx, date.New(2026, time.July, 10))
	require.NoError(t, err)
	require.Len(t, july, 1)
	assert.Equal(t, money.Minor(30_000), july[0].SpentMinor)

	august, err := repo.GetActiveBudgets(ctx, date.New(2026, time.August, 5))
	require.NoError(t, err)
	require.Len(t, august, 1)
	assert.Equal(t, money.Minor(0), august[0].SpentMinor)
}

func TestBudgetRepositorySQLite_Advance_SkipsTakenPeriod(t *testing.T) {
	repo, ctx, categoryID := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	require.NoError(t, repo.Create(ctx, tail))

	// Июль занят ручным бюджетом той же области, август — чужим бюджетом с тем же именем.
	manual := monthBudget("Ручной июль", date.New(2026, time.July, 1), false)
	require.NoError(t, repo.Create(ctx, manual))

	sameName := monthBudget("Продукты", date.New(2026, time.August, 1), false)
	sameName.CategoryID = &categoryID
	require.NoError(t, repo.Create(ctx, sameName))

	created, err := repo.Advance(ctx, tail.ID, date.New(2026, time.September, 14))
	require.NoError(t, err)
	assert.Equal(t, 1, created)

	tails, err := repo.ListRecurring(ctx)
	require.NoError(t, err)
	require.Len(t, tails, 1)
	assert.Equal(t, date.New(2026, time.September, 1), tails[0].StartDate)
}

func TestBudgetRepositorySQLite_Advance_TodayInsideTail(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	require.NoError(t, repo.Create(ctx, tail))

	created, err := repo.Advance(ctx, tail.ID, date.New(2026, time.June, 30))
	require.NoError(t, err)
	assert.Equal(t, 0, created)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

// TestBudgetRepositorySQLite_Advance_WeeklySeries — недельный шаг идёт через тот же SQL,
// что и месячный: пропуск занятого периода и перенос флага проверяются и на нём.
func TestBudgetRepositorySQLite_Advance_WeeklySeries(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	start := date.New(2026, time.June, 1)
	tail := &budget.Budget{
		ID:          uuid.New(),
		Name:        "Неделя",
		AmountMinor: recurringAmount,
		Period:      budget.PeriodWeekly,
		StartDate:   start,
		EndDate:     start.AddDays(6),
		IsActive:    true,
		Recurring:   true,
	}
	tail.SeriesID = &tail.ID
	require.NoError(t, repo.Create(ctx, tail))

	created, err := repo.Advance(ctx, tail.ID, start.AddDays(21))
	require.NoError(t, err)
	assert.Equal(t, 3, created)

	tails, err := repo.ListRecurring(ctx)
	require.NoError(t, err)
	require.Len(t, tails, 1)
	assert.Equal(t, start.AddDays(21), tails[0].StartDate)
	assert.Equal(t, start.AddDays(27), tails[0].EndDate)
}

// TestBudgetRepositorySQLite_Advance_StepLimit — на границе maxAdvancePeriods серия ещё
// достраивается целиком, на шаг дальше откатывается вся транзакция, а не половина серии.
func TestBudgetRepositorySQLite_Advance_StepLimit(t *testing.T) {
	const limit = 120

	t.Run("AtLimit", func(t *testing.T) {
		repo, ctx, _ := recurringRepo(t)

		start := date.New(2026, time.June, 1)
		tail := monthBudget("Продукты", start, true)
		require.NoError(t, repo.Create(ctx, tail))

		created, err := repo.Advance(ctx, tail.ID, start.AddMonths(limit))
		require.NoError(t, err)
		assert.Equal(t, limit, created)
	})

	t.Run("OneStepBeyond", func(t *testing.T) {
		repo, ctx, _ := recurringRepo(t)

		start := date.New(2026, time.June, 1)
		tail := monthBudget("Продукты", start, true)
		require.NoError(t, repo.Create(ctx, tail))

		_, err := repo.Advance(ctx, tail.ID, start.AddMonths(limit+1))
		require.ErrorIs(t, err, budget.ErrTooFarBehind)

		all, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, all, 1, "откатывается вся серия, а не только последний шаг")
	})
}

// TestBudgetRepositorySQLite_Delete_StopsSeries — удалённый хвост выпадает из ListRecurring,
// и серия молча останавливается: восстановить её можно только флагом на живом инстансе.
func TestBudgetRepositorySQLite_Delete_StopsSeries(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2026, time.June, 1), true)
	require.NoError(t, repo.Create(ctx, tail))
	require.NoError(t, repo.Delete(ctx, tail.ID))

	tails, err := repo.ListRecurring(ctx)
	require.NoError(t, err)
	assert.Empty(t, tails)

	_, err = repo.Advance(ctx, tail.ID, date.New(2026, time.September, 14))
	require.ErrorIs(t, err, budget.ErrNotTail)
}

func TestBudgetRepositorySQLite_Advance_TooFarBehind(t *testing.T) {
	repo, ctx, _ := recurringRepo(t)

	tail := monthBudget("Продукты", date.New(2000, time.January, 1), true)
	require.NoError(t, repo.Create(ctx, tail))

	_, err := repo.Advance(ctx, tail.ID, date.New(2026, time.September, 14))
	require.ErrorIs(t, err, budget.ErrTooFarBehind)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

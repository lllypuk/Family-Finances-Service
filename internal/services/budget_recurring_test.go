package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// monthlyBudget — выровненный по календарю месяц: только такой может быть повторяющимся.
func monthlyBudget(t *testing.T, year int, month time.Month) *budget.Budget {
	t.Helper()

	start, end := date.Date{Year: year, Month: month, Day: 1}.MonthBounds()
	b := createTestBudgetForService()
	b.StartDate, b.EndDate = start, end
	b.SpentMinor = 0

	return b
}

func recurringCreateDTO(t *testing.T) dto.CreateBudgetDTO {
	t.Helper()

	req := createTestBudgetDTO()
	req.Recurring = true
	req.StartDate, req.EndDate = date.Date{Year: 2026, Month: time.September, Day: 1}.MonthBounds()

	return req
}

// expectSpent — пересчёт расхода идёт мимо предмета этих тестов; траты нулевые.
func expectSpent(txRepo *MockTransactionRepositoryForBudgets) {
	txRepo.On(
		"GetTotalByCategoryAndDateRange",
		mock.Anything,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("date.Date"),
		mock.AnythingOfType("date.Date"),
		transaction.TypeExpense,
	).Return(money.Minor(0), nil).Maybe()
}

func TestBudgetService_CreateBudget_RecurringStartsSeries(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	budgetRepo.On("Create", ctx, mock.AnythingOfType("*budget.Budget")).Return(nil)
	budgetRepo.On("GetByID", ctx, mock.AnythingOfType("uuid.UUID")).Return(createTestBudgetForService(), nil)
	expectSpent(txRepo)

	result, err := service.CreateBudget(ctx, recurringCreateDTO(t))

	require.NoError(t, err)
	assert.True(t, result.Recurring)
	require.NotNil(t, result.SeriesID)
	assert.Equal(t, result.ID, *result.SeriesID)

	budgetRepo.AssertExpectations(t)
}

func TestBudgetService_CreateBudget_RecurringCustomRejected(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)

	req := recurringCreateDTO(t)
	req.Period = budget.PeriodCustom

	result, err := service.CreateBudget(context.Background(), req)

	require.ErrorIs(t, err, budget.ErrRecurringCustom)
	assert.Nil(t, result)
	budgetRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestBudgetService_CreateBudget_RecurringNotAligned(t *testing.T) {
	service, budgetRepo, _ := setupBudgetService(t)

	req := recurringCreateDTO(t)
	req.StartDate = req.StartDate.AddDays(1)

	result, err := service.CreateBudget(context.Background(), req)

	require.ErrorIs(t, err, budget.ErrRecurringNotAligned)
	assert.Nil(t, result)
	budgetRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestBudgetService_UpdateBudget_SeriesDatesFixed(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	member := monthlyBudget(t, 2026, time.September)
	seriesID := member.ID
	member.SeriesID = &seriesID

	budgetRepo.On("GetByID", ctx, member.ID).Return(member, nil)
	expectSpent(txRepo)

	newEnd := member.EndDate.AddDays(1)
	result, err := service.UpdateBudget(ctx, member.ID, dto.UpdateBudgetDTO{EndDate: &newEnd})

	require.ErrorIs(t, err, budget.ErrSeriesDatesFixed)
	assert.Nil(t, result)
	budgetRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestBudgetService_UpdateBudget_RecurringTrueStartsSeries(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	plain := monthlyBudget(t, 2026, time.September)

	budgetRepo.On("GetByID", ctx, plain.ID).Return(plain, nil)
	expectSpent(txRepo)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget"), budget.UpdateExpect{}).Return(nil).Once()

	on := true
	result, err := service.UpdateBudget(ctx, plain.ID, dto.UpdateBudgetDTO{Recurring: &on})

	require.NoError(t, err)
	assert.True(t, result.Recurring)
	require.NotNil(t, result.SeriesID)
	assert.Equal(t, result.ID, *result.SeriesID)

	budgetRepo.AssertExpectations(t)
}

// Включить повтор можно только на календарном периоде: иначе серия шагала бы вразнобой.
func TestBudgetService_UpdateBudget_RecurringTrueNotAligned(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	plain := createTestBudgetForService() // произвольные 30 дней от сегодня

	budgetRepo.On("GetByID", ctx, plain.ID).Return(plain, nil)
	expectSpent(txRepo)

	on := true
	result, err := service.UpdateBudget(ctx, plain.ID, dto.UpdateBudgetDTO{Recurring: &on})

	require.ErrorIs(t, err, budget.ErrRecurringNotAligned)
	assert.Nil(t, result)
	budgetRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

// Устаревшая форма: клиент снимает флаг с прошлого инстанса, пока серия уже на новом хвосте.
// Отказ приходит из репозитория — проверка живого хвоста живёт в транзакции записи.
func TestBudgetService_UpdateBudget_RecurringFalseOnPastInstance(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	past := monthlyBudget(t, 2026, time.September)
	seriesID := past.ID
	past.SeriesID = &seriesID

	budgetRepo.On("GetByID", ctx, past.ID).Return(past, nil)
	expectSpent(txRepo)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget"),
		budget.UpdateExpect{StopSeries: true}).Return(budget.ErrNotTail).Once()

	off := false
	result, err := service.UpdateBudget(ctx, past.ID, dto.UpdateBudgetDTO{Recurring: &off})

	require.ErrorIs(t, err, services.ErrBudgetNotTail)
	assert.Nil(t, result)
	budgetRepo.AssertExpectations(t)
}

// Серия уже остановлена — повторное false ничего не меняет и не отказывает.
func TestBudgetService_UpdateBudget_RecurringFalseOnStoppedSeries(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	member := monthlyBudget(t, 2026, time.September)
	seriesID := member.ID
	member.SeriesID = &seriesID

	budgetRepo.On("GetByID", ctx, member.ID).Return(member, nil)
	expectSpent(txRepo)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget"),
		budget.UpdateExpect{StopSeries: true}).Return(nil).Once()

	off := false
	result, err := service.UpdateBudget(ctx, member.ID, dto.UpdateBudgetDTO{Recurring: &off})

	require.NoError(t, err)
	assert.False(t, result.Recurring)

	budgetRepo.AssertExpectations(t)
}

// Хвост продвинулся между чтением и записью — отказ репозитория доходит до клиента.
func TestBudgetService_UpdateBudget_NotTailFromRepository(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	tail := monthlyBudget(t, 2026, time.September)
	tail.Recurring = true
	seriesID := tail.ID
	tail.SeriesID = &seriesID

	newName := "Продукты"
	budgetRepo.On("GetByID", ctx, tail.ID).Return(tail, nil)
	expectSpent(txRepo)
	budgetRepo.On("Update", ctx, mock.AnythingOfType("*budget.Budget"), budget.UpdateExpect{Recurring: true}).
		Return(budget.ErrNotTail).
		Once()

	result, err := service.UpdateBudget(ctx, tail.ID, dto.UpdateBudgetDTO{Name: &newName})

	require.ErrorIs(t, err, services.ErrBudgetNotTail)
	assert.Nil(t, result)

	budgetRepo.AssertExpectations(t)
}

func TestBudgetService_GetBudgetsPage_AdvancesSeriesToToday(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	tail := monthlyBudget(t, 2026, time.June)
	tail.Recurring = true
	today := date.Date{Year: 2026, Month: time.September, Day: 14}

	budgetRepo.On("ListRecurring", ctx).Return([]*budget.Budget{tail}, nil).Once()
	budgetRepo.On("Advance", ctx, tail.ID, today).Return(3, nil).Once()
	budgetRepo.On("GetAll", ctx).Return([]*budget.Budget{tail}, nil)
	expectSpent(txRepo)

	filter := dto.NewBudgetFilterDTO()
	filter.Today = &today

	budgets, total, err := service.GetBudgetsPage(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, budgets, 1)
	assert.Equal(t, 1, total)

	budgetRepo.AssertExpectations(t)
}

// Материализация — побочная задача чтения: её отказ не отменяет список.
func TestBudgetService_GetBudgetsPage_AdvanceFailureKeepsList(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	tail := monthlyBudget(t, 2020, time.January)
	tail.Recurring = true
	today := date.Date{Year: 2026, Month: time.September, Day: 14}

	budgetRepo.On("ListRecurring", ctx).Return([]*budget.Budget{tail}, nil).Once()
	budgetRepo.On("Advance", ctx, tail.ID, today).Return(0, budget.ErrTooFarBehind).Once()
	budgetRepo.On("GetAll", ctx).Return([]*budget.Budget{tail}, nil)
	expectSpent(txRepo)

	filter := dto.NewBudgetFilterDTO()
	filter.Today = &today

	budgets, _, err := service.GetBudgetsPage(ctx, filter)

	require.NoError(t, err)
	assert.Len(t, budgets, 1)

	budgetRepo.AssertExpectations(t)
}

// Параллельное чтение уже достроило серию: ErrNotTail ничего не значит для этого прохода.
func TestBudgetService_GetActiveBudgets_AdvanceNotTailIgnored(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	tail := monthlyBudget(t, 2026, time.September)
	tail.Recurring = true
	on := date.Date{Year: 2026, Month: time.September, Day: 14}

	budgetRepo.On("ListRecurring", ctx).Return([]*budget.Budget{tail}, nil).Once()
	budgetRepo.On("Advance", ctx, tail.ID, on).Return(0, budget.ErrNotTail).Once()
	budgetRepo.On("GetActiveBudgets", ctx, on).Return([]*budget.Budget{tail}, nil)
	expectSpent(txRepo)

	result, err := service.GetActiveBudgets(ctx, on)

	require.NoError(t, err)
	assert.Len(t, result, 1)

	budgetRepo.AssertExpectations(t)
}

// Без Today материализации нет: у чтения может не быть таймзоны семьи.
func TestBudgetService_GetBudgetsPage_WithoutTodaySkipsAdvance(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	budgetRepo.On("GetAll", ctx).Return([]*budget.Budget{}, nil)
	expectSpent(txRepo)

	_, _, err := service.GetBudgetsPage(ctx, dto.NewBudgetFilterDTO())

	require.NoError(t, err)
	budgetRepo.AssertNotCalled(t, "ListRecurring", mock.Anything)
	budgetRepo.AssertNotCalled(t, "Advance", mock.Anything, mock.Anything, mock.Anything)
}

func TestBudgetService_AdvanceRecurring_ListFailureKeepsList(t *testing.T) {
	service, budgetRepo, txRepo := setupBudgetService(t)
	ctx := context.Background()

	today := date.Today(time.UTC)

	budgetRepo.On("ListRecurring", ctx).Return(nil, errors.New("db down")).Once()
	budgetRepo.On("GetActiveBudgets", ctx, today).Return([]*budget.Budget{}, nil)
	expectSpent(txRepo)

	result, err := service.GetActiveBudgets(ctx, today)

	require.NoError(t, err)
	assert.Empty(t, result)

	budgetRepo.AssertExpectations(t)
}

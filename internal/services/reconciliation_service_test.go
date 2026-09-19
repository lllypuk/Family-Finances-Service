package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

type mockBalanceRepo struct {
	mock.Mock
}

func (m *mockBalanceRepo) Upsert(ctx context.Context, b *reconciliation.Balance) error {
	return m.Called(ctx, b).Error(0)
}

func (m *mockBalanceRepo) Delete(ctx context.Context, accountID uuid.UUID, month string) error {
	return m.Called(ctx, accountID, month).Error(0)
}

func (m *mockBalanceRepo) ByMonths(ctx context.Context, prev, month string) ([]*reconciliation.Balance, error) {
	args := m.Called(ctx, prev, month)
	return args.Get(0).([]*reconciliation.Balance), args.Error(1)
}

type reconciliationMocks struct {
	balances *mockBalanceRepo
	accounts *mockAccountRepo
	txs      *MockTransactionRepository
	families *MockFamilyRepository
	svc      services.ReconciliationService
}

func newReconciliationMocks(family *user.Family) reconciliationMocks {
	m := reconciliationMocks{
		balances: &mockBalanceRepo{},
		accounts: &mockAccountRepo{},
		txs:      &MockTransactionRepository{},
		families: &MockFamilyRepository{},
	}
	m.families.On("Get", mock.Anything).Return(family, nil)
	m.svc = services.NewReconciliationService(m.balances, m.accounts, m.txs, m.families)

	return m
}

func augustBounds() (date.Date, date.Date) {
	return date.New(2026, time.August, 1), date.New(2026, time.August, 31)
}

// longAgo — счёт заведён задолго до сверяемого месяца.
func longAgo() time.Time {
	return time.Date(2025, time.January, 10, 12, 0, 0, 0, time.UTC)
}

func balance(a *account.Account, month string, v money.Minor) *reconciliation.Balance {
	return &reconciliation.Balance{AccountID: a.ID, Month: month, BalanceMinor: v}
}

// summarize — сводка за август 2026 с приходом 1000 и расходом 400.
func summarize(
	t *testing.T,
	accounts []*account.Account,
	balances ...*reconciliation.Balance,
) *dto.ReconciliationStats {
	t.Helper()

	m := newReconciliationMocks(&user.Family{Timezone: "UTC"})
	m.accounts.On("List", mock.Anything, true).Return(accounts, nil)
	m.balances.On("ByMonths", mock.Anything, "2026-07", "2026-08").Return(balances, nil)
	august, augustLast := augustBounds()
	m.txs.On("GetTotalsByMonth", mock.Anything, august, augustLast).Return([]transaction.MonthTotal{
		{Month: "2026-08", Type: transaction.TypeIncome, AmountMinor: 1_000},
		{Month: "2026-08", Type: transaction.TypeExpense, AmountMinor: 400},
	}, nil)

	day := date.New(2026, time.August, 17)
	stats, err := m.svc.Summary(t.Context(), &day)
	require.NoError(t, err)
	assert.Equal(t, "2026-08", stats.Month)
	assert.Equal(t, money.Minor(1_000), stats.IncomeMinor)
	assert.Equal(t, money.Minor(400), stats.ExpenseMinor)

	return stats
}

func minor(v money.Minor) *money.Minor { return &v }

func TestReconciliationService_Summary_BothEdges(t *testing.T) {
	card := &account.Account{ID: uuid.New(), Name: "Карта", CreatedAt: longAgo()}
	credit := &account.Account{ID: uuid.New(), Name: "Кредитка", CreatedAt: longAgo()}

	stats := summarize(t, []*account.Account{card, credit},
		balance(card, "2026-07", 10_000), balance(credit, "2026-07", -2_000),
		balance(card, "2026-08", 11_000), balance(credit, "2026-08", -2_500))

	assert.True(t, stats.Complete)
	assert.Equal(t, minor(8_000), stats.OpeningMinor)
	assert.Equal(t, minor(8_500), stats.ClosingMinor)
	// (8500 − 8000) − (1000 − 400)
	assert.Equal(t, minor(-100), stats.GapMinor)
	require.Len(t, stats.Accounts, 2)
	assert.Equal(t, minor(-2_500), stats.Accounts[1].ClosingMinor, "остаток кредитки отрицательный")
	assert.NotNil(t, stats.Accounts[0].UpdatedAt)
}

func TestReconciliationService_Summary_MissingEdge(t *testing.T) {
	card := &account.Account{ID: uuid.New(), Name: "Карта", CreatedAt: longAgo()}
	cash := &account.Account{ID: uuid.New(), Name: "Наличные", CreatedAt: longAgo()}

	stats := summarize(t, []*account.Account{card, cash},
		balance(card, "2026-07", 10_000), balance(cash, "2026-07", 500), balance(card, "2026-08", 11_000))
	assert.Equal(t, minor(10_500), stats.OpeningMinor)
	assert.Nil(t, stats.ClosingMinor)
	assert.Nil(t, stats.GapMinor)
	assert.False(t, stats.Complete)
	assert.Nil(t, stats.Accounts[1].ClosingMinor)
	assert.Nil(t, stats.Accounts[1].UpdatedAt)

	stats = summarize(t, []*account.Account{card, cash},
		balance(card, "2026-07", 10_000), balance(card, "2026-08", 11_000), balance(cash, "2026-08", 0))
	assert.Nil(t, stats.OpeningMinor, "у давнего счёта без строки за прошлый месяц — пропуск")
	assert.Equal(t, minor(11_000), stats.ClosingMinor)
	assert.Nil(t, stats.GapMinor)
	assert.False(t, stats.Complete)
}

func TestReconciliationService_Summary_AccountCreatedInMonth(t *testing.T) {
	fresh := &account.Account{
		ID: uuid.New(), Name: "Новая", CreatedAt: time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC),
	}
	stats := summarize(t, []*account.Account{fresh}, balance(fresh, "2026-08", 700))
	assert.Equal(t, minor(0), stats.Accounts[0].OpeningMinor)
	assert.Equal(t, minor(100), stats.GapMinor)

	stats = summarize(t, []*account.Account{fresh}, balance(fresh, "2026-07", 300), balance(fresh, "2026-08", 700))
	assert.Equal(t, minor(300), stats.OpeningMinor, "строка за прошлый месяц побеждает исключение")
}

func TestReconciliationService_Summary_AccountCreatedAfterMonth(t *testing.T) {
	card := &account.Account{ID: uuid.New(), Name: "Карта", CreatedAt: longAgo()}
	later := &account.Account{
		ID: uuid.New(), Name: "Позже", CreatedAt: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
	}

	stats := summarize(t, []*account.Account{card, later},
		balance(card, "2026-07", 100), balance(card, "2026-08", 700))
	require.Len(t, stats.Accounts, 1)
	assert.Equal(t, card.ID, stats.Accounts[0].Account.ID)
	assert.True(t, stats.Complete)
}

func TestReconciliationService_Summary_Archived(t *testing.T) {
	card := &account.Account{ID: uuid.New(), Name: "Карта", CreatedAt: longAgo()}
	closed := &account.Account{ID: uuid.New(), Name: "Закрытая", IsArchived: true, CreatedAt: longAgo()}
	idle := &account.Account{ID: uuid.New(), Name: "Пустая", IsArchived: true, CreatedAt: longAgo()}
	late := &account.Account{ID: uuid.New(), Name: "Поздняя", IsArchived: true, CreatedAt: longAgo()}
	fresh := &account.Account{
		ID:         uuid.New(),
		Name:       "Новая",
		IsArchived: true,
		CreatedAt:  time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC),
	}

	stats := summarize(t, []*account.Account{card, closed, idle, late, fresh},
		balance(card, "2026-07", 100), balance(closed, "2026-07", 500), balance(idle, "2026-07", 0),
		balance(card, "2026-08", 1_200), balance(late, "2026-07", 0), balance(late, "2026-08", 300))
	require.Len(t, stats.Accounts, 3, "архивный без строки за месяц и без ненулевой за прошлый — не в списке")
	assert.Equal(t, closed.ID, stats.Accounts[1].Account.ID)
	assert.Equal(t, minor(0), stats.Accounts[1].ClosingMinor, "закрытая карта без строки — 0")
	assert.Nil(t, stats.Accounts[1].UpdatedAt)
	assert.Equal(t, late.ID, stats.Accounts[2].Account.ID)
	assert.Equal(t, minor(300), stats.Accounts[2].ClosingMinor, "строка побеждает архивный 0")
	assert.NotNil(t, stats.Accounts[2].UpdatedAt)
	assert.True(t, stats.Complete)

	// Следующий месяц: у закрытой строка за август — 0, за сентябрь нет.
	m := newReconciliationMocks(&user.Family{Timezone: "UTC"})
	m.accounts.On("List", mock.Anything, true).Return([]*account.Account{card, closed}, nil)
	m.balances.On("ByMonths", mock.Anything, "2026-08", "2026-09").Return([]*reconciliation.Balance{
		balance(card, "2026-08", 1_200),
	}, nil)
	m.txs.On("GetTotalsByMonth", mock.Anything, mock.Anything, mock.Anything).Return([]transaction.MonthTotal{}, nil)
	sept := date.New(2026, time.September, 1)
	next, err := m.svc.Summary(t.Context(), &sept)
	require.NoError(t, err)
	require.Len(t, next.Accounts, 1, "закрытая карта выпала через месяц")
}

func TestReconciliationService_Summary_NoAccounts(t *testing.T) {
	stats := summarize(t, []*account.Account{})
	assert.Empty(t, stats.Accounts)
	assert.Nil(t, stats.OpeningMinor)
	assert.Nil(t, stats.ClosingMinor)
	assert.Nil(t, stats.GapMinor)
	assert.False(t, stats.Complete)
}

func TestReconciliationService_Summary_CreatedAtInFamilyZone(t *testing.T) {
	// 31 июля 22:00 UTC — это уже 1 августа во Владивостоке: счёт заведён в сверяемом месяце.
	edge := &account.Account{
		ID: uuid.New(), Name: "Граница", CreatedAt: time.Date(2026, time.July, 31, 22, 0, 0, 0, time.UTC),
	}
	august, augustLast := augustBounds()
	for _, tt := range []struct {
		zone    string
		opening *money.Minor
	}{
		{"Asia/Vladivostok", minor(0)},
		{"UTC", nil},
	} {
		m := newReconciliationMocks(&user.Family{Timezone: tt.zone})
		m.accounts.On("List", mock.Anything, true).Return([]*account.Account{edge}, nil)
		m.balances.On("ByMonths", mock.Anything, "2026-07", "2026-08").Return([]*reconciliation.Balance{}, nil)
		m.txs.On("GetTotalsByMonth", mock.Anything, august, augustLast).Return([]transaction.MonthTotal{}, nil)

		stats, err := m.svc.Summary(t.Context(), &august)
		require.NoError(t, err)
		assert.Equal(t, tt.opening, stats.Accounts[0].OpeningMinor, tt.zone)
	}
}

func TestReconciliationService_Summary_DefaultMonthFromFamilyZone(t *testing.T) {
	family := &user.Family{Timezone: "Pacific/Kiritimati"}
	m := newReconciliationMocks(family)
	first, last := date.Today(family.Location()).MonthBounds()
	m.accounts.On("List", mock.Anything, true).Return([]*account.Account{}, nil)
	m.balances.On("ByMonths", mock.Anything, first.AddMonths(-1).MonthKey(), first.MonthKey()).
		Return([]*reconciliation.Balance{}, nil)
	m.txs.On("GetTotalsByMonth", mock.Anything, first, last).Return([]transaction.MonthTotal(nil), nil)

	stats, err := m.svc.Summary(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, first.MonthKey(), stats.Month)
}

func TestReconciliationService_FutureMonth(t *testing.T) {
	family := &user.Family{Timezone: "UTC"}
	m := newReconciliationMocks(family)
	next := date.Today(family.Location()).AddMonths(1)
	id := uuid.New()

	_, err := m.svc.Summary(t.Context(), &next)
	require.ErrorIs(t, err, services.ErrReconciliationMonthInFuture)
	_, err = m.svc.PutBalance(t.Context(), id, next, 1)
	require.ErrorIs(t, err, services.ErrReconciliationMonthInFuture)
	require.ErrorIs(t, m.svc.DeleteBalance(t.Context(), id, next), services.ErrReconciliationMonthInFuture)
	m.accounts.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestReconciliationService_PutBalance(t *testing.T) {
	m := newReconciliationMocks(&user.Family{Timezone: "UTC"})
	august, _ := augustBounds()
	id := uuid.New()
	m.accounts.On("GetByID", mock.Anything, id).Return(&account.Account{ID: id, IsArchived: true}, nil)
	m.balances.On("Upsert", mock.Anything, mock.AnythingOfType("*reconciliation.Balance")).Return(nil)

	b, err := m.svc.PutBalance(t.Context(), id, august, -money.MaxAmount)
	require.NoError(t, err, "архивный счёт и отрицательный остаток разрешены")
	assert.Equal(t, "2026-08", b.Month)

	today := date.Today(time.UTC)
	b, err = m.svc.PutBalance(t.Context(), id, today, 1)
	require.NoError(t, err, "любой день текущего месяца — не будущее")
	assert.Equal(t, today.MonthKey(), b.Month)

	_, err = m.svc.PutBalance(t.Context(), id, august, money.MaxAmount+1)
	require.ErrorIs(t, err, reconciliation.ErrBalanceOutOfRange)
	_, err = m.svc.PutBalance(t.Context(), id, august, -money.MaxAmount-1)
	require.ErrorIs(t, err, reconciliation.ErrBalanceOutOfRange)

	unknown := uuid.New()
	m.accounts.On("GetByID", mock.Anything, unknown).Return(nil, account.ErrNotFound)
	_, err = m.svc.PutBalance(t.Context(), unknown, august, 1)
	require.ErrorIs(t, err, account.ErrNotFound)
	m.balances.AssertNumberOfCalls(t, "Upsert", 2)
}

func TestReconciliationService_DeleteBalance(t *testing.T) {
	m := newReconciliationMocks(&user.Family{Timezone: "UTC"})
	august, _ := augustBounds()
	id := uuid.New()
	m.accounts.On("GetByID", mock.Anything, id).Return(&account.Account{ID: id}, nil)
	m.balances.On("Delete", mock.Anything, id, "2026-08").Return(reconciliation.ErrBalanceNotFound)

	require.ErrorIs(t, m.svc.DeleteBalance(t.Context(), id, august), reconciliation.ErrBalanceNotFound)
}

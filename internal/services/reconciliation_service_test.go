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
)

type mockReconciliationRepo struct {
	mock.Mock
}

func (m *mockReconciliationRepo) Upsert(ctx context.Context, rec *reconciliation.Reconciliation) error {
	return m.Called(ctx, rec).Error(0)
}

func (m *mockReconciliationRepo) Delete(ctx context.Context, accountID uuid.UUID, month string) error {
	return m.Called(ctx, accountID, month).Error(0)
}

func (m *mockReconciliationRepo) ListByMonth(
	ctx context.Context,
	month string,
) ([]*reconciliation.Reconciliation, error) {
	args := m.Called(ctx, month)
	return args.Get(0).([]*reconciliation.Reconciliation), args.Error(1)
}

type reconciliationMocks struct {
	repo     *mockReconciliationRepo
	accounts *mockAccountRepo
	txs      *MockTransactionRepository
	families *MockFamilyRepository
	svc      services.ReconciliationService
}

func newReconciliationMocks() reconciliationMocks {
	m := reconciliationMocks{
		repo:     &mockReconciliationRepo{},
		accounts: &mockAccountRepo{},
		txs:      &MockTransactionRepository{},
		families: &MockFamilyRepository{},
	}
	m.svc = services.NewReconciliationService(m.repo, m.accounts, m.txs, m.families)

	return m
}

func TestReconciliationService_Summary(t *testing.T) {
	m := newReconciliationMocks()
	active := &account.Account{ID: uuid.New(), Name: "Альфа"}
	unreconciled := &account.Account{ID: uuid.New(), Name: "Наличные"}
	archivedSpent := &account.Account{ID: uuid.New(), Name: "Старая", IsArchived: true}
	archivedZero := &account.Account{ID: uuid.New(), Name: "Закрытая", IsArchived: true}
	archivedIdle := &account.Account{ID: uuid.New(), Name: "Пустая", IsArchived: true}
	first, last := date.New(2026, time.August, 1), date.New(2026, time.August, 31)

	m.accounts.On("List", mock.Anything, true).Return(
		[]*account.Account{active, unreconciled, archivedSpent, archivedZero, archivedIdle}, nil)
	m.txs.On("RecordedByAccount", mock.Anything, first, last).Return([]transaction.AccountTotal{
		{AccountID: &active.ID, AmountMinor: 5_000},
		{AccountID: &unreconciled.ID, AmountMinor: 300},
		{AccountID: &archivedSpent.ID, AmountMinor: 10},
		{AccountID: nil, AmountMinor: 125},
	}, nil)
	m.repo.On("ListByMonth", mock.Anything, "2026-08").Return([]*reconciliation.Reconciliation{
		{AccountID: active.ID, Month: "2026-08", BankExpenseMinor: 4_000, Note: "лишнее"},
		{AccountID: archivedZero.ID, Month: "2026-08", BankExpenseMinor: 0},
	}, nil)

	month := date.New(2026, time.August, 17)
	stats, err := m.svc.Summary(t.Context(), &month)
	require.NoError(t, err)

	assert.Equal(t, "2026-08", stats.Month)
	assert.Equal(t, money.Minor(125), stats.UnassignedMinor)
	require.Len(t, stats.Accounts, 4, "архивный без расхода и сверки не входит")

	row := stats.Accounts[0]
	assert.Equal(t, active.ID, row.Account.ID)
	assert.Equal(t, money.Minor(5_000), row.RecordedMinor)
	require.NotNil(t, row.DiffMinor)
	assert.Equal(t, money.Minor(-1_000), *row.DiffMinor)
	assert.Equal(t, "лишнее", *row.Note)

	row = stats.Accounts[1]
	assert.Equal(t, money.Minor(300), row.RecordedMinor)
	assert.Nil(t, row.BankExpenseMinor)
	assert.Nil(t, row.DiffMinor)
	assert.Nil(t, row.Note)
	assert.Nil(t, row.UpdatedAt)

	assert.Equal(t, archivedSpent.ID, stats.Accounts[2].Account.ID)
	assert.True(t, stats.Accounts[2].Account.IsArchived)
	assert.Equal(t, archivedZero.ID, stats.Accounts[3].Account.ID, "нулевая сверка считается")
	assert.Equal(t, money.Minor(0), *stats.Accounts[3].DiffMinor)
}

func TestReconciliationService_Summary_DefaultMonthFromFamilyZone(t *testing.T) {
	m := newReconciliationMocks()
	family := &user.Family{Timezone: "Pacific/Kiritimati"}
	m.families.On("Get", mock.Anything).Return(family, nil)
	first, last := date.Today(family.Location()).MonthBounds()
	m.accounts.On("List", mock.Anything, true).Return([]*account.Account{}, nil)
	m.txs.On("RecordedByAccount", mock.Anything, first, last).Return([]transaction.AccountTotal(nil), nil)
	m.repo.On("ListByMonth", mock.Anything, first.MonthKey()).Return([]*reconciliation.Reconciliation{}, nil)

	stats, err := m.svc.Summary(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, first.MonthKey(), stats.Month)
	assert.Empty(t, stats.Accounts)
	assert.Equal(t, money.Minor(0), stats.UnassignedMinor)
}

func TestReconciliationService_Put(t *testing.T) {
	m := newReconciliationMocks()
	id := uuid.New()
	m.accounts.On("GetByID", mock.Anything, id).Return(&account.Account{ID: id}, nil)
	m.repo.On("Upsert", mock.Anything, mock.AnythingOfType("*reconciliation.Reconciliation")).Return(nil)

	rec, err := m.svc.Put(t.Context(), id, date.New(2026, time.September, 1), 0, "")
	require.NoError(t, err)
	assert.Equal(t, "2026-09", rec.Month)

	_, err = m.svc.Put(t.Context(), id, date.New(2026, time.September, 1), -1, "")
	require.ErrorIs(t, err, reconciliation.ErrAmountOutOfRange)
	_, err = m.svc.Put(t.Context(), id, date.New(2026, time.September, 1), money.MaxAmount+1, "")
	require.ErrorIs(t, err, reconciliation.ErrAmountOutOfRange)

	unknown := uuid.New()
	m.accounts.On("GetByID", mock.Anything, unknown).Return(nil, account.ErrNotFound)
	_, err = m.svc.Put(t.Context(), unknown, date.New(2026, time.September, 1), 1, "")
	require.ErrorIs(t, err, account.ErrNotFound)
	m.repo.AssertNumberOfCalls(t, "Upsert", 1)
}

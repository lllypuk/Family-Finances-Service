package reconciliation_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/reconciliation"
	accountrepo "family-budget-service/internal/infrastructure/account"
	reconciliationrepo "family-budget-service/internal/infrastructure/reconciliation"
	"family-budget-service/internal/testhelpers"
)

func setupBalances(t *testing.T) (*reconciliationrepo.BalanceSQLiteRepository, *accountrepo.SQLiteRepository) {
	t.Helper()

	container := testhelpers.SetupSQLiteTestDB(t)
	_, err := testhelpers.NewTestDataHelper(container.DB).CreateTestFamily(t.Context(), "Family", "RUB")
	require.NoError(t, err)

	return reconciliationrepo.NewBalanceSQLiteRepository(container.DB), accountrepo.NewSQLiteRepository(container.DB)
}

func byKey(balances []*reconciliation.Balance) map[string]money.Minor {
	out := make(map[string]money.Minor, len(balances))
	for _, b := range balances {
		out[b.AccountID.String()+"/"+b.Month] = b.BalanceMinor
	}
	return out
}

func TestBalanceRepository_Upsert_Overwrites(t *testing.T) {
	repo, accounts := setupBalances(t)
	a := testhelpers.CreateTestAccount("Сбер")
	require.NoError(t, accounts.Create(t.Context(), a))

	first := &reconciliation.Balance{AccountID: a.ID, Month: "2026-09", BalanceMinor: 100}
	require.NoError(t, repo.Upsert(t.Context(), first))
	second := &reconciliation.Balance{AccountID: a.ID, Month: "2026-09", BalanceMinor: -5000}
	require.NoError(t, repo.Upsert(t.Context(), second))

	got, err := repo.ByMonths(t.Context(), "2026-08", "2026-09")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, money.Minor(-5000), got[0].BalanceMinor)
	assert.True(t, got[0].UpdatedAt.After(first.UpdatedAt), "upsert поверх двигает updated_at")
	assert.True(t, got[0].UpdatedAt.Equal(second.UpdatedAt))
}

func TestBalanceRepository_ByMonths_OnlyRequestedMonths(t *testing.T) {
	repo, accounts := setupBalances(t)
	a := testhelpers.CreateTestAccount("Сбер")
	b := testhelpers.CreateTestAccount("Наличные")
	require.NoError(t, accounts.Create(t.Context(), a))
	require.NoError(t, accounts.Create(t.Context(), b))

	for _, bal := range []*reconciliation.Balance{
		{AccountID: a.ID, Month: "2026-07", BalanceMinor: 1},
		{AccountID: a.ID, Month: "2026-08", BalanceMinor: 0},
		{AccountID: a.ID, Month: "2026-09", BalanceMinor: -300},
		{AccountID: b.ID, Month: "2026-09", BalanceMinor: 700},
		{AccountID: b.ID, Month: "2026-10", BalanceMinor: 9},
	} {
		require.NoError(t, repo.Upsert(t.Context(), bal))
	}

	got, err := repo.ByMonths(t.Context(), "2026-08", "2026-09")
	require.NoError(t, err)
	assert.Equal(t, map[string]money.Minor{
		a.ID.String() + "/2026-08": 0,
		a.ID.String() + "/2026-09": -300,
		b.ID.String() + "/2026-09": 700,
	}, byKey(got))
}

func TestBalanceRepository_Upsert_UnknownAccount(t *testing.T) {
	repo, _ := setupBalances(t)

	err := repo.Upsert(t.Context(), &reconciliation.Balance{AccountID: uuid.New(), Month: "2026-09"})
	require.ErrorIs(t, err, account.ErrNotFound)
}

func TestBalanceRepository_Delete(t *testing.T) {
	repo, accounts := setupBalances(t)
	a := testhelpers.CreateTestAccount("Сбер")
	require.NoError(t, accounts.Create(t.Context(), a))
	require.NoError(t, repo.Upsert(t.Context(), &reconciliation.Balance{AccountID: a.ID, Month: "2026-09"}))

	require.ErrorIs(t, accounts.Delete(t.Context(), a.ID), account.ErrInUse, "остаток держит счёт")
	require.NoError(t, repo.Delete(t.Context(), a.ID, "2026-09"))
	require.ErrorIs(t, repo.Delete(t.Context(), a.ID, "2026-09"), reconciliation.ErrBalanceNotFound)
	require.NoError(t, accounts.Delete(t.Context(), a.ID))
}

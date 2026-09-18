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

func setup(t *testing.T) (*reconciliationrepo.SQLiteRepository, *accountrepo.SQLiteRepository) {
	t.Helper()

	container := testhelpers.SetupSQLiteTestDB(t)
	_, err := testhelpers.NewTestDataHelper(container.DB).CreateTestFamily(t.Context(), "Family", "RUB")
	require.NoError(t, err)

	return reconciliationrepo.NewSQLiteRepository(container.DB), accountrepo.NewSQLiteRepository(container.DB)
}

func TestReconciliationRepository_Upsert_ReplacesWholeRow(t *testing.T) {
	repo, accounts := setup(t)
	a := testhelpers.CreateTestAccount("Сбер")
	require.NoError(t, accounts.Create(t.Context(), a))

	first := &reconciliation.Reconciliation{AccountID: a.ID, Month: "2026-09", BankExpenseMinor: 100, Note: "выписка"}
	require.NoError(t, repo.Upsert(t.Context(), first))
	second := &reconciliation.Reconciliation{AccountID: a.ID, Month: "2026-09", BankExpenseMinor: 0}
	require.NoError(t, repo.Upsert(t.Context(), second))

	got, err := repo.ListByMonth(t.Context(), "2026-09")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, money.Minor(0), got[0].BankExpenseMinor)
	assert.Empty(t, got[0].Note)
	assert.False(t, got[0].UpdatedAt.Before(first.UpdatedAt))

	other, err := repo.ListByMonth(t.Context(), "2026-08")
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestReconciliationRepository_Upsert_UnknownAccount(t *testing.T) {
	repo, _ := setup(t)

	err := repo.Upsert(t.Context(), &reconciliation.Reconciliation{AccountID: uuid.New(), Month: "2026-09"})
	require.ErrorIs(t, err, account.ErrNotFound)
}

func TestReconciliationRepository_Delete(t *testing.T) {
	repo, accounts := setup(t)
	a := testhelpers.CreateTestAccount("Сбер")
	require.NoError(t, accounts.Create(t.Context(), a))
	require.NoError(t, repo.Upsert(t.Context(), &reconciliation.Reconciliation{AccountID: a.ID, Month: "2026-09"}))

	require.ErrorIs(t, accounts.Delete(t.Context(), a.ID), account.ErrInUse, "сверка держит счёт")
	require.NoError(t, repo.Delete(t.Context(), a.ID, "2026-09"))
	require.ErrorIs(t, repo.Delete(t.Context(), a.ID, "2026-09"), reconciliation.ErrNotFound)
	require.NoError(t, accounts.Delete(t.Context(), a.ID))
}

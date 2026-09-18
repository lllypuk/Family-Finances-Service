package transaction_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	accountrepo "family-budget-service/internal/infrastructure/account"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	testutils "family-budget-service/internal/testhelpers"
)

func TestTransactionRepositorySQLite_RecordedByAccount(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := t.Context()
	repo := transactionrepo.NewSQLiteRepository(container.DB)

	familyID, err := helper.CreateTestFamily(ctx, "Family", "RUB")
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "rec@example.com", "Rec", "User", "admin", familyID)
	require.NoError(t, err)
	foodID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
	require.NoError(t, err)
	salaryID, err := helper.CreateTestCategory(ctx, "Salary", "income", familyID, nil)
	require.NoError(t, err)
	card := testutils.CreateTestAccount("Карта")
	require.NoError(t, accountrepo.NewSQLiteRepository(container.DB).Create(ctx, card))

	entries := []struct {
		category string
		txType   transaction.Type
		amount   money.Minor
		day      date.Date
		account  *uuid.UUID
	}{
		{foodID, transaction.TypeExpense, 100, date.New(2026, time.August, 31), &card.ID},
		{foodID, transaction.TypeExpense, 200, date.New(2026, time.September, 1), &card.ID},
		{foodID, transaction.TypeExpense, 300, date.New(2026, time.September, 30), &card.ID},
		{foodID, transaction.TypeExpense, 400, date.New(2026, time.October, 1), &card.ID},
		{salaryID, transaction.TypeIncome, 5_000, date.New(2026, time.September, 10), &card.ID},
		{foodID, transaction.TypeExpense, 50, date.New(2026, time.September, 15), nil},
	}
	for i, e := range entries {
		require.NoError(t, repo.Create(ctx, &transaction.Transaction{
			ID:          uuid.New(),
			AmountMinor: e.amount,
			Type:        e.txType,
			Description: "Recorded",
			CategoryID:  uuid.MustParse(e.category),
			UserID:      uuid.MustParse(userID),
			AccountID:   e.account,
			Date:        e.day,
		}), "transaction %d", i)
	}

	totals, err := repo.RecordedByAccount(ctx, date.New(2026, time.September, 1), date.New(2026, time.September, 30))
	require.NoError(t, err)
	require.Len(t, totals, 2)

	byAccount := map[string]money.Minor{}
	for _, total := range totals {
		key := "unassigned"
		if total.AccountID != nil {
			key = total.AccountID.String()
		}
		byAccount[key] = total.AmountMinor
	}
	assert.Equal(t, money.Minor(500), byAccount[card.ID.String()], "только расходы, 1-е и 30-е входят")
	assert.Equal(t, money.Minor(50), byAccount["unassigned"])

	empty, err := repo.RecordedByAccount(ctx, date.New(2026, time.July, 1), date.New(2026, time.July, 31))
	require.NoError(t, err)
	assert.Empty(t, empty)
}

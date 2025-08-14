package transaction_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	"family-budget-service/internal/testhelpers"
)

func TestTransactionRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := transactionrepo.NewRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)

		err := repo.Create(context.Background(), testTransaction)
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)

		err := repo.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		retrievedTransaction, err := repo.GetByID(context.Background(), testTransaction.ID)
		require.NoError(t, err)
		assert.Equal(t, testTransaction.ID, retrievedTransaction.ID)
		assert.InEpsilon(t, testTransaction.Amount, retrievedTransaction.Amount, 0.001)
		assert.Equal(t, testTransaction.Type, retrievedTransaction.Type)
		assert.Equal(t, testTransaction.Description, retrievedTransaction.Description)
		assert.Equal(t, testTransaction.FamilyID, retrievedTransaction.FamilyID)
		assert.Equal(t, testTransaction.UserID, retrievedTransaction.UserID)
		assert.Equal(t, testTransaction.CategoryID, retrievedTransaction.CategoryID)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByFamilyID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		transaction1 := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		transaction1.Amount = 50.0
		transaction2 := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeIncome,
		)
		transaction2.Amount = 100.0

		err := repo.Create(context.Background(), transaction1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), transaction2)
		require.NoError(t, err)

		transactions, err := repo.GetByFamilyID(context.Background(), family.ID, 0, 0)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)

		transactionIDs := make([]uuid.UUID, len(transactions))
		for i, tr := range transactions {
			transactionIDs[i] = tr.ID
		}
		assert.Contains(t, transactionIDs, transaction1.ID)
		assert.Contains(t, transactionIDs, transaction2.ID)
	})

	t.Run("GetByFamilyID_WithPagination", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		// Create 5 transactions
		for i := range 5 {
			tr := testhelpers.CreateTestTransaction(
				family.ID,
				user.ID,
				testCategory.ID,
				transaction.TransactionTypeExpense,
			)
			tr.Amount = float64(10 * (i + 1))
			err := repo.Create(context.Background(), tr)
			require.NoError(t, err)
		}

		// Get first 2 transactions
		transactions, err := repo.GetByFamilyID(context.Background(), family.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)

		// Get next 2 transactions
		transactions, err = repo.GetByFamilyID(context.Background(), family.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)
	})

	t.Run("GetByFilter_ByType", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		expenseTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		incomeTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeIncome,
		)

		err := repo.Create(context.Background(), expenseTransaction)
		require.NoError(t, err)
		err = repo.Create(context.Background(), incomeTransaction)
		require.NoError(t, err)

		expenseType := transaction.TransactionTypeExpense
		filter := transaction.TransactionFilter{
			FamilyID: family.ID,
			Type:     &expenseType,
		}

		transactions, err := repo.GetByFilter(context.Background(), filter)
		require.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, expenseTransaction.ID, transactions[0].ID)
	})

	t.Run("GetByFilter_ByDateRange", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		// Create transaction with older date
		oldTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		oldTransaction.Date = time.Now().AddDate(0, -2, 0) // 2 months ago

		// Create transaction with recent date
		newTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		newTransaction.Date = time.Now().AddDate(0, 0, -1) // 1 day ago

		err := repo.Create(context.Background(), oldTransaction)
		require.NoError(t, err)
		err = repo.Create(context.Background(), newTransaction)
		require.NoError(t, err)

		// Filter for last month
		dateFrom := time.Now().AddDate(0, -1, 0)
		dateTo := time.Now()
		filter := transaction.TransactionFilter{
			FamilyID: family.ID,
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		transactions, err := repo.GetByFilter(context.Background(), filter)
		require.NoError(t, err)
		assert.Len(t, transactions, 1)
		assert.Equal(t, newTransaction.ID, transactions[0].ID)
	})

	t.Run("Update_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)

		err := repo.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		testTransaction.Amount = 200.0
		testTransaction.Description = "Updated transaction"
		err = repo.Update(context.Background(), testTransaction)
		require.NoError(t, err)

		retrievedTransaction, err := repo.GetByID(context.Background(), testTransaction.ID)
		require.NoError(t, err)
		assert.InEpsilon(t, 200.0, retrievedTransaction.Amount, 0.001)
		assert.Equal(t, "Updated transaction", retrievedTransaction.Description)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		nonExistentTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)

		err := repo.Update(context.Background(), nonExistentTransaction)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)
		testTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)

		err := repo.Create(context.Background(), testTransaction)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testTransaction.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), testTransaction.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetTotalByCategory_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		user := testhelpers.CreateTestUser(family.ID)
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		// Create multiple expense transactions for the same category
		transaction1 := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		transaction1.Amount = 100.0
		transaction2 := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeExpense,
		)
		transaction2.Amount = 50.0

		// Create an income transaction (should not be included)
		incomeTransaction := testhelpers.CreateTestTransaction(
			family.ID,
			user.ID,
			testCategory.ID,
			transaction.TransactionTypeIncome,
		)
		incomeTransaction.Amount = 200.0

		err := repo.Create(context.Background(), transaction1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), transaction2)
		require.NoError(t, err)
		err = repo.Create(context.Background(), incomeTransaction)
		require.NoError(t, err)

		total, err := repo.GetTotalByCategory(context.Background(), testCategory.ID, transaction.TransactionTypeExpense)
		require.NoError(t, err)
		assert.InEpsilon(t, 150.0, total, 0.001)

		incomeTotal, err := repo.GetTotalByCategory(
			context.Background(),
			testCategory.ID,
			transaction.TransactionTypeIncome,
		)
		require.NoError(t, err)
		assert.InEpsilon(t, 200.0, incomeTotal, 0.001)
	})

	t.Run("GetTotalByCategory_NoTransactions", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testCategory := testhelpers.CreateTestCategory(family.ID, category.CategoryTypeExpense)

		total, err := repo.GetTotalByCategory(context.Background(), testCategory.ID, transaction.TransactionTypeExpense)
		require.NoError(t, err)
		assert.Equal(t, 0.0, total) //nolint:testifylint // InEpsilon не работает для нуля
	})
}

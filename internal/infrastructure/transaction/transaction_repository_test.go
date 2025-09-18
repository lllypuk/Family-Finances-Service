package transaction_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/transaction"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	testutils "family-budget-service/internal/testing"
)

func TestTransactionRepositoryPostgreSQL_Integration(t *testing.T) {
	// Setup PostgreSQL testcontainer
	container := testutils.SetupPostgreSQLContainer(t)
	defer container.Cleanup(t)

	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "test@example.com", "John", "Doe", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Groceries", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transaction
		testTransaction := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      150.50,
			Type:        transaction.TypeExpense,
			Description: "Weekly groceries",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			FamilyID:    uuid.MustParse(familyID),
			Date:        time.Now().AddDate(0, 0, -1), // Yesterday
			Tags:        []string{"grocery", "weekly"},
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Verify transaction was created
		retrievedTransaction, err := repo.GetByID(ctx, testTransaction.ID)
		require.NoError(t, err)
		assert.Equal(t, testTransaction.ID, retrievedTransaction.ID)
		assert.InEpsilon(t, testTransaction.Amount, retrievedTransaction.Amount, 0.01)
		assert.Equal(t, testTransaction.Type, retrievedTransaction.Type)
		assert.Equal(t, testTransaction.Description, retrievedTransaction.Description)
		assert.Equal(t, testTransaction.CategoryID, retrievedTransaction.CategoryID)
		assert.Equal(t, testTransaction.UserID, retrievedTransaction.UserID)
		assert.Equal(t, testTransaction.FamilyID, retrievedTransaction.FamilyID)
		assert.Equal(t, testTransaction.Tags, retrievedTransaction.Tags)
	})

	t.Run("GetByFilter_DateRange", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Filter Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		userID, err := helper.CreateTestUser(ctx, "filter@example.com", "Filter", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions on different dates
		now := time.Now()
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				Amount:      100.00,
				Type:        transaction.TypeExpense,
				Description: "Old transaction",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now.AddDate(0, 0, -10), // 10 days ago
				Tags:        []string{"old"},
			},
			{
				ID:          uuid.New(),
				Amount:      200.00,
				Type:        transaction.TypeExpense,
				Description: "Recent transaction",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now.AddDate(0, 0, -2), // 2 days ago
				Tags:        []string{"recent"},
			},
			{
				ID:          uuid.New(),
				Amount:      300.00,
				Type:        transaction.TypeIncome,
				Description: "Today income",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now,
				Tags:        []string{"today"},
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Filter by date range (last 5 days)
		dateFrom := now.AddDate(0, 0, -5)
		dateTo := now.AddDate(0, 0, 1) // Include today
		filter := transaction.Filter{
			FamilyID: familyUUID,
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			Limit:    10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Should exclude the 10-day-old transaction

		// Results should be sorted by date DESC
		assert.True(t, results[0].Date.After(results[1].Date) || results[0].Date.Equal(results[1].Date))
	})

	t.Run("GetByFilter_TypeAndAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Amount Filter Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		userID, err := helper.CreateTestUser(ctx, "amount@example.com", "Amount", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions with different amounts
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				Amount:      50.00,
				Type:        transaction.TypeExpense,
				Description: "Small expense",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
			},
			{
				ID:          uuid.New(),
				Amount:      150.00,
				Type:        transaction.TypeExpense,
				Description: "Medium expense",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
			},
			{
				ID:          uuid.New(),
				Amount:      1000.00,
				Type:        transaction.TypeIncome,
				Description: "Large income",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Filter expenses between 100-200
		expenseType := transaction.TypeExpense
		amountFrom := 100.0
		amountTo := 200.0
		filter := transaction.Filter{
			FamilyID:   familyUUID,
			Type:       &expenseType,
			AmountFrom: &amountFrom,
			AmountTo:   &amountTo,
			Limit:      10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1) // Should only get the 150.00 expense
		assert.InEpsilon(t, 150.00, results[0].Amount, 0.01)
		assert.Equal(t, transaction.TypeExpense, results[0].Type)
	})

	t.Run("GetByFilter_Tags", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Tags Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		userID, err := helper.CreateTestUser(ctx, "tags@example.com", "Tags", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions with different tags
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				Amount:      100.00,
				Type:        transaction.TypeExpense,
				Description: "Grocery shopping",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
				Tags:        []string{"grocery", "food", "weekly"},
			},
			{
				ID:          uuid.New(),
				Amount:      200.00,
				Type:        transaction.TypeExpense,
				Description: "Gas station",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
				Tags:        []string{"gas", "car", "transport"},
			},
			{
				ID:          uuid.New(),
				Amount:      50.00,
				Type:        transaction.TypeExpense,
				Description: "Weekly grocery",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        time.Now(),
				Tags:        []string{"grocery", "weekly"},
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Filter by tag "grocery"
		filter := transaction.Filter{
			FamilyID: familyUUID,
			Tags:     []string{"grocery"},
			Limit:    10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Should get both grocery transactions

		// Verify both results have the "grocery" tag
		for _, result := range results {
			assert.Contains(t, result.Tags, "grocery")
		}
	})

	t.Run("GetTransactionSummary", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Summary Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		userID, err := helper.CreateTestUser(ctx, "summary@example.com", "Summary", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create mixed transactions
		now := time.Now()
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				Amount:      1000.00,
				Type:        transaction.TypeIncome,
				Description: "Salary",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now,
			},
			{
				ID:          uuid.New(),
				Amount:      200.00,
				Type:        transaction.TypeExpense,
				Description: "Groceries",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now,
			},
			{
				ID:          uuid.New(),
				Amount:      300.00,
				Type:        transaction.TypeExpense,
				Description: "Utilities",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				FamilyID:    familyUUID,
				Date:        now,
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Get summary
		startDate := now.AddDate(0, 0, -1)
		endDate := now.AddDate(0, 0, 1)
		summary, err := repo.GetSummary(ctx, familyUUID, startDate, endDate)
		require.NoError(t, err)

		assert.Equal(t, familyUUID, summary.FamilyID)
		assert.Equal(t, 3, summary.TotalCount)
		assert.Equal(t, 1, summary.IncomeCount)
		assert.Equal(t, 2, summary.ExpenseCount)
		assert.InEpsilon(t, 1000.00, summary.TotalIncome, 0.01)
		assert.InEpsilon(t, 500.00, summary.TotalExpenses, 0.01) // 200 + 300
		assert.InEpsilon(t, 500.00, summary.Balance, 0.01)       // 1000 - 500
		assert.InEpsilon(t, 1000.00, summary.AvgIncome, 0.01)
		assert.InEpsilon(t, 250.00, summary.AvgExpense, 0.01) // (200 + 300) / 2
	})

	t.Run("Update_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Update Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "update@example.com", "Update", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Original Category", "expense", familyID, nil)
		require.NoError(t, err)

		newCategoryID, err := helper.CreateTestCategory(ctx, "New Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transaction
		testTransaction := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      100.00,
			Type:        transaction.TypeExpense,
			Description: "Original description",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			FamilyID:    uuid.MustParse(familyID),
			Date:        time.Now().AddDate(0, 0, -1),
			Tags:        []string{"original"},
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Update transaction
		testTransaction.Amount = 150.00
		testTransaction.Description = "Updated description"
		testTransaction.CategoryID = uuid.MustParse(newCategoryID)
		testTransaction.Tags = []string{"updated", "modified"}

		err = repo.Update(ctx, testTransaction)
		require.NoError(t, err)

		// Verify update
		retrievedTransaction, err := repo.GetByID(ctx, testTransaction.ID)
		require.NoError(t, err)
		assert.InEpsilon(t, 150.00, retrievedTransaction.Amount, 0.01)
		assert.Equal(t, "Updated description", retrievedTransaction.Description)
		assert.Equal(t, uuid.MustParse(newCategoryID), retrievedTransaction.CategoryID)
		assert.Equal(t, []string{"updated", "modified"}, retrievedTransaction.Tags)
	})

	t.Run("Delete_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewPostgreSQLRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Delete Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "delete@example.com", "Delete", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transaction
		testTransaction := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      100.00,
			Type:        transaction.TypeExpense,
			Description: "To be deleted",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			FamilyID:    uuid.MustParse(familyID),
			Date:        time.Now(),
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Delete transaction
		err = repo.Delete(ctx, testTransaction.ID, testTransaction.FamilyID)
		require.NoError(t, err)

		// Verify transaction is deleted
		_, err = repo.GetByID(ctx, testTransaction.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

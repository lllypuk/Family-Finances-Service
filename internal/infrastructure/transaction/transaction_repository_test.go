package transaction_test

import (
	"context"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
)

func TestTransactionRepositorySQLite_Integration(t *testing.T) {
	// Setup SQLite in-memory database
	container := testutils.SetupSQLiteTestDB(t)

	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

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
			AmountMinor: 15_050,
			Type:        transaction.TypeExpense,
			Description: "Weekly groceries",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        date.Today(time.UTC).AddDays(-1), // Yesterday
			Tags:        []string{"grocery", "weekly"},
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Verify transaction was created
		retrievedTransaction, err := repo.GetByID(ctx, testTransaction.ID)
		require.NoError(t, err)
		assert.Equal(t, testTransaction.ID, retrievedTransaction.ID)
		assert.Equal(t, testTransaction.AmountMinor, retrievedTransaction.AmountMinor)
		assert.Equal(t, testTransaction.Type, retrievedTransaction.Type)
		assert.Equal(t, testTransaction.Description, retrievedTransaction.Description)
		assert.Equal(t, testTransaction.CategoryID, retrievedTransaction.CategoryID)
		assert.Equal(t, testTransaction.UserID, retrievedTransaction.UserID)
		assert.Equal(t, testTransaction.Tags, retrievedTransaction.Tags)
	})

	t.Run("CreateWithBudgetUpdate_AtomicSuccess", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Atomic Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "atomic@example.com", "Atomic", "User", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
		require.NoError(t, err)

		budgetID, err := helper.CreateTestBudget(ctx, "Food Budget", 50_000, "monthly", familyID, &categoryID)
		require.NoError(t, err)

		testTransaction := &transaction.Transaction{
			ID:          uuid.New(),
			AmountMinor: 12_525,
			Type:        transaction.TypeExpense,
			Description: "Atomic create",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        date.Today(time.UTC),
			Tags:        []string{"atomic"},
		}

		err = repo.CreateWithBudgetUpdate(ctx, testTransaction)
		require.NoError(t, err)

		createdTx, err := repo.GetByID(ctx, testTransaction.ID)
		require.NoError(t, err)
		require.NotNil(t, createdTx)
		assert.Equal(t, money.Minor(12_525), createdTx.AmountMinor)

		var spent money.Minor
		err = db.QueryRowContext(ctx, "SELECT spent_minor FROM budgets WHERE id = ?", budgetID).Scan(&spent)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(12_525), spent)
	})

	t.Run("GetByFilter_DateRange", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Filter Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "filter@example.com", "Filter", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions on different dates
		now := date.Today(time.UTC)
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				AmountMinor: 10_000,
				Type:        transaction.TypeExpense,
				Description: "Old transaction",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        now.AddDays(-10), // 10 days ago
				Tags:        []string{"old"},
			},
			{
				ID:          uuid.New(),
				AmountMinor: 20_000,
				Type:        transaction.TypeExpense,
				Description: "Recent transaction",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        now.AddDays(-2), // 2 days ago
				Tags:        []string{"recent"},
			},
			{
				ID:          uuid.New(),
				AmountMinor: 30_000,
				Type:        transaction.TypeIncome,
				Description: "Today income",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
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
		dateFrom := now.AddDays(-5)
		dateTo := now.AddDays(1) // Include today
		filter := transaction.Filter{
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			Limit:    10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Should exclude the 10-day-old transaction

		// Results should be sorted by date DESC
		assert.False(t, results[0].Date.Before(results[1].Date))
	})

	t.Run("GetByFilter_TypeAndAmount", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Amount Filter Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "amount@example.com", "Amount", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions with different amounts
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				AmountMinor: 5_000,
				Type:        transaction.TypeExpense,
				Description: "Small expense",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
			},
			{
				ID:          uuid.New(),
				AmountMinor: 15_000,
				Type:        transaction.TypeExpense,
				Description: "Medium expense",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
			},
			{
				ID:          uuid.New(),
				AmountMinor: 100_000,
				Type:        transaction.TypeIncome,
				Description: "Large income",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Filter expenses between 100-200
		expenseType := transaction.TypeExpense
		amountFrom := money.Minor(10_000)
		amountTo := money.Minor(20_000)
		filter := transaction.Filter{
			Type:            &expenseType,
			AmountFromMinor: &amountFrom,
			AmountToMinor:   &amountTo,
			Limit:           10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1) // Should only get the 150.00 expense
		assert.Equal(t, money.Minor(15_000), results[0].AmountMinor)
		assert.Equal(t, transaction.TypeExpense, results[0].Type)
	})

	t.Run("GetByFilter_Tags", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Tags Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "tags@example.com", "Tags", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create transactions with different tags
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				AmountMinor: 10_000,
				Type:        transaction.TypeExpense,
				Description: "Grocery shopping",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
				Tags:        []string{"grocery", "food", "weekly"},
			},
			{
				ID:          uuid.New(),
				AmountMinor: 20_000,
				Type:        transaction.TypeExpense,
				Description: "Gas station",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
				Tags:        []string{"gas", "car", "transport"},
			},
			{
				ID:          uuid.New(),
				AmountMinor: 5_000,
				Type:        transaction.TypeExpense,
				Description: "Weekly grocery",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC),
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
			Tags:  []string{"grocery"},
			Limit: 10,
		}

		results, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Should get both grocery transactions

		// Verify both results have the "grocery" tag
		for _, result := range results {
			assert.Contains(t, result.Tags, "grocery")
		}
	})

	// CountByFilter обязан считать все подходящие записи, а не страницу: именно на
	// нём держится HasNext в пагинации веб-интерфейса.
	t.Run("CountByFilter_IgnoresPagination", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		familyID, err := helper.CreateTestFamily(ctx, "Count Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "count@example.com", "Count", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		expenseType := transaction.TypeExpense
		for i := range 3 {
			require.NoError(t, repo.Create(ctx, &transaction.Transaction{
				ID:          uuid.New(),
				AmountMinor: 10_000,
				Type:        expenseType,
				Description: "Counted expense",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        date.Today(time.UTC).AddDays(-i),
				Tags:        []string{},
			}))
		}
		require.NoError(t, repo.Create(ctx, &transaction.Transaction{
			ID:          uuid.New(),
			AmountMinor: 50_000,
			Type:        transaction.TypeIncome,
			Description: "Not counted income",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        date.Today(time.UTC),
			Tags:        []string{},
		}))

		filter := transaction.Filter{Type: &expenseType, Limit: 2, Offset: 0}

		page, err := repo.GetByFilter(ctx, filter)
		require.NoError(t, err)
		require.Len(t, page, 2, "страница должна быть урезана LIMIT'ом")

		total, err := repo.CountByFilter(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 3, total, "счётчик обязан игнорировать LIMIT/OFFSET и фильтровать по типу")
	})

	t.Run("GetTransactionSummary", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

		// Create test data
		familyID, err := helper.CreateTestFamily(ctx, "Summary Test Family", "USD")
		require.NoError(t, err)

		userID, err := helper.CreateTestUser(ctx, "summary@example.com", "Summary", "Test", "admin", familyID)
		require.NoError(t, err)

		categoryID, err := helper.CreateTestCategory(ctx, "Test Category", "expense", familyID, nil)
		require.NoError(t, err)

		// Create mixed transactions
		now := date.Today(time.UTC)
		transactions := []*transaction.Transaction{
			{
				ID:          uuid.New(),
				AmountMinor: 100_000,
				Type:        transaction.TypeIncome,
				Description: "Salary",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        now,
			},
			{
				ID:          uuid.New(),
				AmountMinor: 20_000,
				Type:        transaction.TypeExpense,
				Description: "Groceries",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        now,
			},
			{
				ID:          uuid.New(),
				AmountMinor: 30_000,
				Type:        transaction.TypeExpense,
				Description: "Utilities",
				CategoryID:  uuid.MustParse(categoryID),
				UserID:      uuid.MustParse(userID),
				Date:        now,
			},
		}

		// Create all transactions
		for _, tx := range transactions {
			err = repo.Create(ctx, tx)
			require.NoError(t, err)
		}

		// Get summary
		startDate := now.AddDays(-1)
		endDate := now.AddDays(1)
		summary, err := repo.GetSummary(ctx, startDate, endDate)
		require.NoError(t, err)

		assert.Equal(t, 3, summary.TotalCount)
		assert.Equal(t, 1, summary.IncomeCount)
		assert.Equal(t, 2, summary.ExpenseCount)
		assert.Equal(t, money.Minor(100_000), summary.TotalIncomeMinor)
		assert.Equal(t, money.Minor(50_000), summary.TotalExpensesMinor) // 200 + 300
		assert.Equal(t, money.Minor(50_000), summary.BalanceMinor)       // 1000 - 500
		assert.Equal(t, money.Minor(100_000), summary.AvgIncomeMinor)
		assert.Equal(t, money.Minor(25_000), summary.AvgExpenseMinor) // (200 + 300) / 2
	})

	t.Run("Update_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

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
			AmountMinor: 10_000,
			Type:        transaction.TypeExpense,
			Description: "Original description",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        date.Today(time.UTC).AddDays(-1),
			Tags:        []string{"original"},
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Update transaction
		testTransaction.AmountMinor = 15_000
		testTransaction.Description = "Updated description"
		testTransaction.CategoryID = uuid.MustParse(newCategoryID)
		testTransaction.Tags = []string{"updated", "modified"}

		err = repo.Update(ctx, testTransaction)
		require.NoError(t, err)

		// Verify update
		retrievedTransaction, err := repo.GetByID(ctx, testTransaction.ID)
		require.NoError(t, err)
		assert.Equal(t, money.Minor(15_000), retrievedTransaction.AmountMinor)
		assert.Equal(t, "Updated description", retrievedTransaction.Description)
		assert.Equal(t, uuid.MustParse(newCategoryID), retrievedTransaction.CategoryID)
		assert.Equal(t, []string{"updated", "modified"}, retrievedTransaction.Tags)
	})

	t.Run("Delete_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := transactionrepo.NewSQLiteRepository(db)

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
			AmountMinor: 10_000,
			Type:        transaction.TypeExpense,
			Description: "To be deleted",
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        date.Today(time.UTC),
		}

		err = repo.Create(ctx, testTransaction)
		require.NoError(t, err)

		// Delete transaction
		err = repo.Delete(ctx, testTransaction.ID)
		require.NoError(t, err)

		// Verify transaction is deleted
		_, err = repo.GetByID(ctx, testTransaction.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// Копейки не теряются на агрегатах, границы диапазона дат включаются, сортировка идёт
// по календарной дате — три свойства, которые float-колонка и DATETIME ломали молча.
func TestTransactionRepositorySQLite_MinorUnitsAndDateBounds(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	db := container.GetTestDatabase(t)
	repo := transactionrepo.NewSQLiteRepository(db)

	familyID, err := helper.CreateTestFamily(ctx, "Minor Family", "RUB")
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "minor@example.com", "Minor", "User", "admin", familyID)
	require.NoError(t, err)
	categoryID, err := helper.CreateTestCategory(ctx, "Coffee", "expense", familyID, nil)
	require.NoError(t, err)

	days := []date.Date{
		date.New(2026, time.March, 1),
		date.New(2026, time.March, 2),
		date.New(2026, time.March, 3),
	}
	for i, day := range days {
		tx := &transaction.Transaction{
			ID:          uuid.New(),
			AmountMinor: 33,
			Type:        transaction.TypeExpense,
			Description: "Kopeyki " + day.String(),
			CategoryID:  uuid.MustParse(categoryID),
			UserID:      uuid.MustParse(userID),
			Date:        day,
			Tags:        []string{"minor"},
		}
		require.NoError(t, repo.Create(ctx, tx), "transaction %d", i)
	}

	t.Run("SumKeepsMinorUnits", func(t *testing.T) {
		total, totalErr := repo.GetTotalByCategoryAndDateRange(
			ctx, uuid.MustParse(categoryID), days[0], days[2], transaction.TypeExpense,
		)
		require.NoError(t, totalErr)
		assert.Equal(t, money.Minor(99), total)
	})

	t.Run("SummaryAverageRoundsHalfUp", func(t *testing.T) {
		summary, summaryErr := repo.GetSummary(ctx, days[0], days[2])
		require.NoError(t, summaryErr)
		assert.Equal(t, money.Minor(99), summary.TotalExpensesMinor)
		assert.Equal(t, money.Minor(33), summary.AvgExpenseMinor)
	})

	t.Run("DateRangeIncludesBounds", func(t *testing.T) {
		from, to := days[0], days[2]
		results, filterErr := repo.GetByFilter(ctx, transaction.Filter{DateFrom: &from, DateTo: &to, Limit: 10})
		require.NoError(t, filterErr)
		require.Len(t, results, 3)
		// ORDER BY date DESC — самая поздняя операция первой.
		assert.Equal(t, days[2], results[0].Date)
		assert.Equal(t, days[0], results[2].Date)
	})

	t.Run("DateRangeExcludesOutside", func(t *testing.T) {
		from, to := days[1], days[1]
		results, filterErr := repo.GetByFilter(ctx, transaction.Filter{DateFrom: &from, DateTo: &to, Limit: 10})
		require.NoError(t, filterErr)
		require.Len(t, results, 1)
		assert.Equal(t, days[1], results[0].Date)
	})
}

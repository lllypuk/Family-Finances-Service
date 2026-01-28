package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
	testutils "family-budget-service/internal/testhelpers"
)

// Benchmark constants
const (
	benchmarkBudgetStartOffsetDays = 30
	benchmarkBudgetEndOffsetMonths = 1
	benchmarkTransactionSpreadDays = 365
	benchmarkDateRangeDays         = 30
)

var (
	testContainer  *testutils.SQLiteTestDB
	testFamilyID   uuid.UUID
	testUserID     uuid.UUID
	testCategories []*category.Category
)

// setupBenchmarkData creates test data for benchmarks
func setupBenchmarkData(b *testing.B) {
	if testContainer == nil {
		// Setup is done once for all benchmarks
		testContainer = testutils.SetupSQLiteTestDB(&testing.T{})
		initializeBenchmarkData(b)
	}
}

// initializeBenchmarkData initializes the benchmark test data
func initializeBenchmarkData(b *testing.B) {
	helper := testutils.NewTestDataHelper(testContainer.DB)
	ctx := context.Background()

	checkDatabaseSchema(b, ctx)
	disableBudgetTriggers(b, ctx)

	// Create test family and user
	testFamilyID = createTestFamilyAndUser(b, helper, ctx)

	// Create test categories
	testCategories = createTestCategories(b, ctx)

	// Create test budgets
	createTestBudgets(b, ctx)

	// Create test transactions
	createTestTransactions(b, ctx)
}

// checkDatabaseSchema checks and logs database schema information
func checkDatabaseSchema(b *testing.B, ctx context.Context) {
	var tableExists int
	err := testContainer.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='budgets'").
		Scan(&tableExists)
	if err != nil {
		b.Fatalf("Failed to check budgets table existence: %v", err)
	}

	rows, err := testContainer.DB.QueryContext(
		ctx,
		"SELECT name FROM sqlite_master WHERE type='table' ORDER BY name",
	)
	if err != nil {
		b.Fatalf("Failed to list tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			b.Fatalf("Failed to scan table name: %v", err)
		}
		tables = append(tables, tableName)
	}
	b.Logf("SQLite tables: %v", tables)
	b.Logf("Budgets table exists: %v", tableExists > 0)
}

// disableBudgetTriggers disables budget triggers for benchmarks
func disableBudgetTriggers(b *testing.B, ctx context.Context) {
	// SQLite triggers are named differently and don't use ON syntax
	_, err := testContainer.DB.ExecContext(
		ctx,
		"DROP TRIGGER IF EXISTS update_budget_spent_on_transaction",
	)
	if err != nil {
		b.Logf("Note: Failed to drop budget trigger (may not exist): %v", err)
	}
}

// createTestFamilyAndUser creates test family and user
func createTestFamilyAndUser(b *testing.B, helper *testutils.TestDataHelper, ctx context.Context) uuid.UUID {
	familyID, err := helper.CreateTestFamily(ctx, "Benchmark Family", "USD")
	if err != nil {
		b.Fatalf("Failed to create test family: %v", err)
	}
	testFamilyID := uuid.MustParse(familyID)

	userID, err := helper.CreateTestUser(ctx, "benchmark@test.com", "Benchmark", "User", "admin", familyID)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	testUserID = uuid.MustParse(userID)

	return testFamilyID
}

// createTestCategories creates test categories
func createTestCategories(b *testing.B, ctx context.Context) []*category.Category {
	categoryRepo := categoryrepo.NewSQLiteRepository(testContainer.DB)
	categories := make([]*category.Category, 10)

	for i := range 10 {
		cat := &category.Category{
			ID:       uuid.New(),
			Name:     fmt.Sprintf("Category %d", i+1),
			Type:     category.TypeExpense,
			FamilyID: testFamilyID,
			IsActive: true,
		}
		err := categoryRepo.Create(ctx, cat)
		if err != nil {
			b.Fatalf("Failed to create category: %v", err)
		}
		categories[i] = cat
	}

	return categories
}

// createTestBudgets creates test budgets
func createTestBudgets(b *testing.B, ctx context.Context) {
	budgetRepo := budgetrepo.NewSQLiteRepository(testContainer.DB)
	for i := range 5 {
		budgetObj := &budget.Budget{
			ID:         uuid.New(),
			Name:       fmt.Sprintf("Budget %d", i+1),
			Amount:     1000.00,
			Spent:      0.00,
			Period:     budget.PeriodMonthly,
			CategoryID: &testCategories[i%len(testCategories)].ID,
			FamilyID:   testFamilyID,
			StartDate:  time.Now().AddDate(0, 0, -benchmarkBudgetStartOffsetDays),
			EndDate:    time.Now().AddDate(0, benchmarkBudgetEndOffsetMonths, 0),
			IsActive:   true,
		}
		err := budgetRepo.Create(ctx, budgetObj)
		if err != nil {
			b.Fatalf("Failed to create budget: %v", err)
		}
	}
}

// createTestTransactions creates test transactions
func createTestTransactions(b *testing.B, ctx context.Context) {
	transactionRepo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	now := time.Now()

	for i := range 1000 {
		tx := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      float64(10 + (i % 500)),
			Type:        transaction.TypeExpense,
			Description: fmt.Sprintf("Benchmark transaction %d", i+1),
			CategoryID:  testCategories[i%len(testCategories)].ID,
			UserID:      testUserID,
			FamilyID:    testFamilyID,
			Date:        now.AddDate(0, 0, -(i % benchmarkTransactionSpreadDays)),
			Tags:        []string{fmt.Sprintf("tag%d", i%10), "benchmark"},
		}

		err := transactionRepo.Create(ctx, tx)
		if err != nil {
			b.Fatalf("Failed to create transaction %d: %v", i, err)
		}
	}
}

// BenchmarkUserRepository_GetByEmail tests user lookup performance
func BenchmarkUserRepository_GetByEmail(b *testing.B) {
	setupBenchmarkData(b)
	repo := userrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	for b.Loop() {
		_, err := repo.GetByEmail(ctx, "benchmark@test.com")
		if err != nil {
			b.Fatalf("Failed to get user by email: %v", err)
		}
	}
}

// BenchmarkUserRepository_GetByFamilyID tests family user listing performance
func BenchmarkUserRepository_GetByFamilyID(b *testing.B) {
	setupBenchmarkData(b)
	repo := userrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	for b.Loop() {
		_, err := repo.GetByFamilyID(ctx, testFamilyID)
		if err != nil {
			b.Fatalf("Failed to get users by family ID: %v", err)
		}
	}
}

// BenchmarkCategoryRepository_GetCategoryChildren tests hierarchical query performance
func BenchmarkCategoryRepository_GetCategoryChildren(b *testing.B) {
	setupBenchmarkData(b)
	repo := categoryrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	// Use first category as parent
	parentID := testCategories[0].ID

	for b.Loop() {
		_, err := repo.GetCategoryChildren(ctx, parentID)
		if err != nil {
			b.Fatalf("Failed to get category children: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_GetByFilter_Simple tests simple transaction filtering
func BenchmarkTransactionRepository_GetByFilter_Simple(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	filter := transaction.Filter{
		FamilyID: testFamilyID,
		Limit:    50,
	}

	for b.Loop() {
		_, err := repo.GetByFilter(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get transactions by filter: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_GetByFilter_Complex tests complex filtering with multiple conditions
func BenchmarkTransactionRepository_GetByFilter_Complex(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	expenseType := transaction.TypeExpense
	amountFrom := 50.0
	amountTo := 200.0
	dateFrom := time.Now().AddDate(0, 0, -benchmarkDateRangeDays) // Last 30 days
	dateTo := time.Now()

	filter := transaction.Filter{
		FamilyID:   testFamilyID,
		Type:       &expenseType,
		AmountFrom: &amountFrom,
		AmountTo:   &amountTo,
		DateFrom:   &dateFrom,
		DateTo:     &dateTo,
		Tags:       []string{"benchmark"},
		Limit:      50,
	}

	for b.Loop() {
		_, err := repo.GetByFilter(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get transactions by complex filter: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_GetByFilter_Pagination tests pagination performance
func BenchmarkTransactionRepository_GetByFilter_Pagination(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		offset := (i % 10) * 20 // Simulate different pages
		filter := transaction.Filter{
			FamilyID: testFamilyID,
			Limit:    20,
			Offset:   offset,
		}

		_, err := repo.GetByFilter(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get transactions with pagination: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_GetTransactionSummary tests summary calculation performance
func BenchmarkTransactionRepository_GetTransactionSummary(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, 0, -benchmarkDateRangeDays)
	endDate := time.Now()

	for b.Loop() {
		_, err := repo.GetSummary(ctx, testFamilyID, startDate, endDate)
		if err != nil {
			b.Fatalf("Failed to get transaction summary: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_GetMonthlySummary tests monthly aggregation performance
func BenchmarkTransactionRepository_GetMonthlySummary(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	for b.Loop() {
		_, err := repo.GetMonthlySummary(ctx, testFamilyID, year, month)
		if err != nil {
			b.Fatalf("Failed to get monthly summary: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_Create tests transaction creation performance
func BenchmarkTransactionRepository_Create(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		tx := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      float64(10 + (i % 100)),
			Type:        transaction.TypeExpense,
			Description: fmt.Sprintf("Benchmark create transaction %d", i),
			CategoryID:  testCategories[i%len(testCategories)].ID,
			UserID:      testUserID,
			FamilyID:    testFamilyID,
			Date:        time.Now(),
			Tags:        []string{"create-benchmark"},
		}

		err := repo.Create(ctx, tx)
		if err != nil {
			b.Fatalf("Failed to create transaction: %v", err)
		}
	}
}

// BenchmarkTransactionRepository_Update tests transaction update performance
func BenchmarkTransactionRepository_Update(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	// Create a transaction to update
	tx := &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      100.00,
		Type:        transaction.TypeExpense,
		Description: "Transaction to update",
		CategoryID:  testCategories[0].ID,
		UserID:      testUserID,
		FamilyID:    testFamilyID,
		Date:        time.Now(),
		Tags:        []string{"update-benchmark"},
	}

	err := repo.Create(ctx, tx)
	if err != nil {
		b.Fatalf("Failed to create transaction for update benchmark: %v", err)
	}

	for i := 0; b.Loop(); i++ {
		tx.Amount = float64(100 + i)
		tx.Description = fmt.Sprintf("Updated transaction %d", i)

		err := repo.Update(ctx, tx)
		if err != nil {
			b.Fatalf("Failed to update transaction: %v", err)
		}
	}
}

// BenchmarkConcurrentReads tests concurrent read performance
func BenchmarkConcurrentReads(b *testing.B) {
	setupBenchmarkData(b)
	repo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	filter := transaction.Filter{
		FamilyID: testFamilyID,
		Limit:    10,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := repo.GetByFilter(ctx, filter)
			if err != nil {
				b.Fatalf("Failed to get transactions in concurrent test: %v", err)
			}
		}
	})
}

// BenchmarkConnectionPoolUsage tests connection pool efficiency
func BenchmarkConnectionPoolUsage(b *testing.B) {
	setupBenchmarkData(b)
	userRepo := userrepo.NewSQLiteRepository(testContainer.DB)
	transactionRepo := transactionrepo.NewSQLiteRepository(testContainer.DB)
	ctx := context.Background()

	for b.Loop() {
		// Simulate multiple repository operations in sequence
		_, err := userRepo.GetByID(ctx, testUserID)
		if err != nil {
			b.Fatalf("Failed to get user: %v", err)
		}

		filter := transaction.Filter{
			FamilyID: testFamilyID,
			Limit:    5,
		}
		_, err = transactionRepo.GetByFilter(ctx, filter)
		if err != nil {
			b.Fatalf("Failed to get transactions: %v", err)
		}
	}
}

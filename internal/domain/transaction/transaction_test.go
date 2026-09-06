package transaction_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
)

func TestNewTransaction(t *testing.T) {
	// Test data
	amount := money.Minor(10_050)
	transactionType := transaction.TypeExpense
	description := "Grocery shopping"
	categoryID := uuid.New()
	userID := uuid.New()
	on := date.Today(time.UTC)

	// Execute
	txn := transaction.NewTransaction(amount, transactionType, description, categoryID, userID, on)

	// Assert
	assert.NotEqual(t, uuid.Nil, txn.ID)
	assert.Equal(t, amount, txn.AmountMinor)
	assert.Equal(t, transactionType, txn.Type)
	assert.Equal(t, description, txn.Description)
	assert.Equal(t, categoryID, txn.CategoryID)
	assert.Equal(t, userID, txn.UserID)
	assert.Equal(t, on, txn.Date)
	assert.NotNil(t, txn.Tags)
	assert.Empty(t, txn.Tags)
	assert.False(t, txn.CreatedAt.IsZero())
	assert.False(t, txn.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), txn.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), txn.UpdatedAt, time.Second)
}

func TestTransactionType_Constants(t *testing.T) {
	// Test that transaction type constants have expected values
	assert.Equal(t, "income", string(transaction.TypeIncome))
	assert.Equal(t, "expense", string(transaction.TypeExpense))
}

func TestTransaction_AddTag(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)
	originalUpdateTime := txn.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(time.Millisecond)

	// Execute
	txn.AddTag("food")

	// Assert
	assert.Contains(t, txn.Tags, "food")
	assert.Len(t, txn.Tags, 1)
	assert.True(t, txn.UpdatedAt.After(originalUpdateTime))
}

func TestTransaction_AddTag_Duplicate(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)
	txn.AddTag("food")
	originalUpdateTime := txn.UpdatedAt

	// Wait a bit
	time.Sleep(time.Millisecond)

	// Execute - try to add the same tag
	txn.AddTag("food")

	// Assert - tag should not be duplicated and UpdatedAt should not change
	assert.Contains(t, txn.Tags, "food")
	assert.Len(t, txn.Tags, 1)
	assert.Equal(t, originalUpdateTime, txn.UpdatedAt)
}

func TestTransaction_RemoveTag(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)
	txn.AddTag("food")
	txn.AddTag("grocery")
	originalUpdateTime := txn.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(time.Millisecond)

	// Execute
	txn.RemoveTag("food")

	// Assert
	assert.NotContains(t, txn.Tags, "food")
	assert.Contains(t, txn.Tags, "grocery")
	assert.Len(t, txn.Tags, 1)
	assert.True(t, txn.UpdatedAt.After(originalUpdateTime))
}

func TestTransaction_RemoveTag_NonExistent(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)
	txn.AddTag("food")
	originalUpdateTime := txn.UpdatedAt

	// Wait a bit
	time.Sleep(time.Millisecond)

	// Execute - try to remove a non-existent tag
	txn.RemoveTag("nonexistent")

	// Assert - tags should remain unchanged and UpdatedAt should not change
	assert.Contains(t, txn.Tags, "food")
	assert.Len(t, txn.Tags, 1)
	assert.Equal(t, originalUpdateTime, txn.UpdatedAt)
}

func TestTransaction_StructFields(t *testing.T) {
	// Test that Transaction struct has all required fields
	categoryID := uuid.New()
	userID := uuid.New()
	on := date.New(2026, time.September, 4)

	txn := &transaction.Transaction{
		ID:          uuid.New(),
		AmountMinor: money.Minor(15_075),
		Type:        transaction.TypeExpense,
		Description: "Salary",
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        on,
		Tags:        []string{"salary", "work"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, txn.ID)
	assert.Equal(t, money.Minor(15_075), txn.AmountMinor)
	assert.Equal(t, transaction.TypeExpense, txn.Type)
	assert.Equal(t, "Salary", txn.Description)
	assert.Equal(t, categoryID, txn.CategoryID)
	assert.Equal(t, userID, txn.UserID)
	assert.Equal(t, on, txn.Date)
	assert.Equal(t, []string{"salary", "work"}, txn.Tags)
	assert.False(t, txn.CreatedAt.IsZero())
	assert.False(t, txn.UpdatedAt.IsZero())
}

func TestFilter_StructFields(t *testing.T) {
	// Test that Filter struct has all expected fields
	userID := uuid.New()
	categoryID := uuid.New()
	transactionType := transaction.TypeExpense
	dateFrom := date.New(2026, time.August, 1)
	dateTo := date.New(2026, time.September, 1)
	amountFrom := money.Minor(1_000)
	amountTo := money.Minor(100_000)

	filter := &transaction.Filter{
		UserID:          &userID,
		CategoryID:      &categoryID,
		Type:            &transactionType,
		DateFrom:        &dateFrom,
		DateTo:          &dateTo,
		AmountFromMinor: &amountFrom,
		AmountToMinor:   &amountTo,
		Tags:            []string{"food", "grocery"},
		Description:     "test",
		Limit:           10,
		Offset:          0,
	}

	// Assert all fields are accessible
	assert.Equal(t, userID, *filter.UserID)
	assert.Equal(t, categoryID, *filter.CategoryID)
	assert.Equal(t, transactionType, *filter.Type)
	assert.Equal(t, dateFrom, *filter.DateFrom)
	assert.Equal(t, dateTo, *filter.DateTo)
	assert.Equal(t, amountFrom, *filter.AmountFromMinor)
	assert.Equal(t, amountTo, *filter.AmountToMinor)
	assert.Equal(t, []string{"food", "grocery"}, filter.Tags)
	assert.Equal(t, "test", filter.Description)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestNewTransaction_DifferentTypes(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()
	on := date.Today(time.UTC)

	tests := []struct {
		name            string
		transactionType transaction.Type
	}{
		{"Income Transaction", transaction.TypeIncome},
		{"Expense Transaction", transaction.TypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := transaction.NewTransaction(money.Minor(10_000), tt.transactionType, "Test", categoryID, userID, on)
			assert.Equal(t, tt.transactionType, txn.Type)
		})
	}
}

func TestTransaction_TagOperations_Sequence(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)

	// Test adding multiple tags
	txn.AddTag("food")
	txn.AddTag("grocery")
	txn.AddTag("essential")

	assert.Len(t, txn.Tags, 3)
	assert.Contains(t, txn.Tags, "food")
	assert.Contains(t, txn.Tags, "grocery")
	assert.Contains(t, txn.Tags, "essential")

	// Test removing middle tag
	txn.RemoveTag("grocery")

	assert.Len(t, txn.Tags, 2)
	assert.Contains(t, txn.Tags, "food")
	assert.NotContains(t, txn.Tags, "grocery")
	assert.Contains(t, txn.Tags, "essential")

	// Test removing first tag
	txn.RemoveTag("food")

	assert.Len(t, txn.Tags, 1)
	assert.NotContains(t, txn.Tags, "food")
	assert.Contains(t, txn.Tags, "essential")

	// Test removing last tag
	txn.RemoveTag("essential")

	assert.Empty(t, txn.Tags)
}

func TestTransaction_TimestampGeneration(t *testing.T) {
	// Record time before creating transaction
	beforeTime := time.Now()

	// Create transaction
	txn := transaction.NewTransaction(
		money.Minor(10_000),
		transaction.TypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		date.Today(time.UTC),
	)

	// Record time after creating transaction
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, txn.CreatedAt.After(beforeTime) || txn.CreatedAt.Equal(beforeTime))
	assert.True(t, txn.CreatedAt.Before(afterTime) || txn.CreatedAt.Equal(afterTime))
	assert.True(t, txn.UpdatedAt.After(beforeTime) || txn.UpdatedAt.Equal(beforeTime))
	assert.True(t, txn.UpdatedAt.Before(afterTime) || txn.UpdatedAt.Equal(afterTime))
}

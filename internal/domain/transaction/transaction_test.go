package transaction_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/transaction"
)

func TestNewTransaction(t *testing.T) {
	// Test data
	amount := 100.50
	transactionType := transaction.TransactionTypeExpense
	description := "Grocery shopping"
	categoryID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()
	date := time.Now()

	// Execute
	txn := transaction.NewTransaction(amount, transactionType, description, categoryID, userID, familyID, date)

	// Assert
	assert.NotEqual(t, uuid.Nil, txn.ID)
	assert.InEpsilon(t, amount, txn.Amount, 0.001)
	assert.Equal(t, transactionType, txn.Type)
	assert.Equal(t, description, txn.Description)
	assert.Equal(t, categoryID, txn.CategoryID)
	assert.Equal(t, userID, txn.UserID)
	assert.Equal(t, familyID, txn.FamilyID)
	assert.Equal(t, date, txn.Date)
	assert.NotNil(t, txn.Tags)
	assert.Empty(t, txn.Tags)
	assert.False(t, txn.CreatedAt.IsZero())
	assert.False(t, txn.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), txn.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), txn.UpdatedAt, time.Second)
}

func TestTransactionType_Constants(t *testing.T) {
	// Test that transaction type constants have expected values
	assert.Equal(t, "income", string(transaction.TransactionTypeIncome))
	assert.Equal(t, "expense", string(transaction.TransactionTypeExpense))
}

func TestTransaction_AddTag(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
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
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
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
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
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
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
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
	familyID := uuid.New()
	date := time.Now()

	txn := &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      150.75,
		Type:        transaction.TransactionTypeExpense,
		Description: "Salary",
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        date,
		Tags:        []string{"salary", "work"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Assert all fields are accessible
	assert.NotEqual(t, uuid.Nil, txn.ID)
	assert.InEpsilon(t, 150.75, txn.Amount, 0.001)
	assert.Equal(t, transaction.TransactionTypeExpense, txn.Type)
	assert.Equal(t, "Salary", txn.Description)
	assert.Equal(t, categoryID, txn.CategoryID)
	assert.Equal(t, userID, txn.UserID)
	assert.Equal(t, familyID, txn.FamilyID)
	assert.Equal(t, date, txn.Date)
	assert.Equal(t, []string{"salary", "work"}, txn.Tags)
	assert.False(t, txn.CreatedAt.IsZero())
	assert.False(t, txn.UpdatedAt.IsZero())
}

func TestTransactionFilter_StructFields(t *testing.T) {
	// Test that TransactionFilter struct has all expected fields
	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	transactionType := transaction.TransactionTypeExpense
	dateFrom := time.Now().AddDate(0, -1, 0)
	dateTo := time.Now()
	amountFrom := 10.0
	amountTo := 1000.0

	filter := &transaction.TransactionFilter{
		FamilyID:    familyID,
		UserID:      &userID,
		CategoryID:  &categoryID,
		Type:        &transactionType,
		DateFrom:    &dateFrom,
		DateTo:      &dateTo,
		AmountFrom:  &amountFrom,
		AmountTo:    &amountTo,
		Tags:        []string{"food", "grocery"},
		Description: "test",
		Limit:       10,
		Offset:      0,
	}

	// Assert all fields are accessible
	assert.Equal(t, familyID, filter.FamilyID)
	assert.Equal(t, userID, *filter.UserID)
	assert.Equal(t, categoryID, *filter.CategoryID)
	assert.Equal(t, transactionType, *filter.Type)
	assert.Equal(t, dateFrom, *filter.DateFrom)
	assert.Equal(t, dateTo, *filter.DateTo)
	assert.InEpsilon(t, amountFrom, *filter.AmountFrom, 0.001)
	assert.InEpsilon(t, amountTo, *filter.AmountTo, 0.001)
	assert.Equal(t, []string{"food", "grocery"}, filter.Tags)
	assert.Equal(t, "test", filter.Description)
	assert.Equal(t, 10, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestNewTransaction_DifferentTypes(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()
	date := time.Now()

	tests := []struct {
		name            string
		transactionType transaction.TransactionType
	}{
		{"Income Transaction", transaction.TransactionTypeIncome},
		{"Expense Transaction", transaction.TransactionTypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txn := transaction.NewTransaction(100.0, tt.transactionType, "Test", categoryID, userID, familyID, date)
			assert.Equal(t, tt.transactionType, txn.Type)
		})
	}
}

func TestTransaction_TagOperations_Sequence(t *testing.T) {
	// Setup
	txn := transaction.NewTransaction(
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
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
		100.0,
		transaction.TransactionTypeExpense,
		"Test",
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now(),
	)

	// Record time after creating transaction
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, txn.CreatedAt.After(beforeTime) || txn.CreatedAt.Equal(beforeTime))
	assert.True(t, txn.CreatedAt.Before(afterTime) || txn.CreatedAt.Equal(afterTime))
	assert.True(t, txn.UpdatedAt.After(beforeTime) || txn.UpdatedAt.Equal(beforeTime))
	assert.True(t, txn.UpdatedAt.Before(afterTime) || txn.UpdatedAt.Equal(afterTime))
}

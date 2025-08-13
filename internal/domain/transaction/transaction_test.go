package transaction

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewTransaction(t *testing.T) {
	// Test data
	amount := 100.50
	transactionType := TransactionTypeExpense
	description := "Grocery shopping"
	categoryID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()
	date := time.Now()

	// Execute
	transaction := NewTransaction(amount, transactionType, description, categoryID, userID, familyID, date)

	// Assert
	assert.NotEqual(t, uuid.Nil, transaction.ID)
	assert.Equal(t, amount, transaction.Amount)
	assert.Equal(t, transactionType, transaction.Type)
	assert.Equal(t, description, transaction.Description)
	assert.Equal(t, categoryID, transaction.CategoryID)
	assert.Equal(t, userID, transaction.UserID)
	assert.Equal(t, familyID, transaction.FamilyID)
	assert.Equal(t, date, transaction.Date)
	assert.NotNil(t, transaction.Tags)
	assert.Empty(t, transaction.Tags)
	assert.False(t, transaction.CreatedAt.IsZero())
	assert.False(t, transaction.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), transaction.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), transaction.UpdatedAt, time.Second)
}

func TestTransactionType_Constants(t *testing.T) {
	// Test that transaction type constants have expected values
	assert.Equal(t, "income", string(TransactionTypeIncome))
	assert.Equal(t, "expense", string(TransactionTypeExpense))
}

func TestTransaction_AddTag(t *testing.T) {
	// Setup
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())
	originalUpdateTime := transaction.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(time.Millisecond)

	// Execute
	transaction.AddTag("food")

	// Assert
	assert.Contains(t, transaction.Tags, "food")
	assert.Len(t, transaction.Tags, 1)
	assert.True(t, transaction.UpdatedAt.After(originalUpdateTime))
}

func TestTransaction_AddTag_Duplicate(t *testing.T) {
	// Setup
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())
	transaction.AddTag("food")
	originalUpdateTime := transaction.UpdatedAt

	// Wait a bit
	time.Sleep(time.Millisecond)

	// Execute - try to add the same tag
	transaction.AddTag("food")

	// Assert - tag should not be duplicated and UpdatedAt should not change
	assert.Contains(t, transaction.Tags, "food")
	assert.Len(t, transaction.Tags, 1)
	assert.Equal(t, originalUpdateTime, transaction.UpdatedAt)
}

func TestTransaction_RemoveTag(t *testing.T) {
	// Setup
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())
	transaction.AddTag("food")
	transaction.AddTag("grocery")
	originalUpdateTime := transaction.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(time.Millisecond)

	// Execute
	transaction.RemoveTag("food")

	// Assert
	assert.NotContains(t, transaction.Tags, "food")
	assert.Contains(t, transaction.Tags, "grocery")
	assert.Len(t, transaction.Tags, 1)
	assert.True(t, transaction.UpdatedAt.After(originalUpdateTime))
}

func TestTransaction_RemoveTag_NonExistent(t *testing.T) {
	// Setup
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())
	transaction.AddTag("food")
	originalUpdateTime := transaction.UpdatedAt

	// Wait a bit
	time.Sleep(time.Millisecond)

	// Execute - try to remove a non-existent tag
	transaction.RemoveTag("nonexistent")

	// Assert - tags should remain unchanged and UpdatedAt should not change
	assert.Contains(t, transaction.Tags, "food")
	assert.Len(t, transaction.Tags, 1)
	assert.Equal(t, originalUpdateTime, transaction.UpdatedAt)
}

func TestTransaction_StructFields(t *testing.T) {
	// Test that Transaction struct has all required fields
	categoryID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()
	date := time.Now()

	transaction := &Transaction{
		ID:          uuid.New(),
		Amount:      150.75,
		Type:        TransactionTypeIncome,
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
	assert.NotEqual(t, uuid.Nil, transaction.ID)
	assert.Equal(t, 150.75, transaction.Amount)
	assert.Equal(t, TransactionTypeIncome, transaction.Type)
	assert.Equal(t, "Salary", transaction.Description)
	assert.Equal(t, categoryID, transaction.CategoryID)
	assert.Equal(t, userID, transaction.UserID)
	assert.Equal(t, familyID, transaction.FamilyID)
	assert.Equal(t, date, transaction.Date)
	assert.Equal(t, []string{"salary", "work"}, transaction.Tags)
	assert.False(t, transaction.CreatedAt.IsZero())
	assert.False(t, transaction.UpdatedAt.IsZero())
}

func TestTransactionFilter_StructFields(t *testing.T) {
	// Test that TransactionFilter struct has all expected fields
	familyID := uuid.New()
	userID := uuid.New()
	categoryID := uuid.New()
	transactionType := TransactionTypeExpense
	dateFrom := time.Now().AddDate(0, -1, 0)
	dateTo := time.Now()
	amountFrom := 10.0
	amountTo := 1000.0

	filter := &TransactionFilter{
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
	assert.Equal(t, amountFrom, *filter.AmountFrom)
	assert.Equal(t, amountTo, *filter.AmountTo)
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
		transactionType TransactionType
	}{
		{"Income Transaction", TransactionTypeIncome},
		{"Expense Transaction", TransactionTypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transaction := NewTransaction(100.0, tt.transactionType, "Test", categoryID, userID, familyID, date)
			assert.Equal(t, tt.transactionType, transaction.Type)
		})
	}
}

func TestTransaction_TagOperations_Sequence(t *testing.T) {
	// Setup
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())

	// Test adding multiple tags
	transaction.AddTag("food")
	transaction.AddTag("grocery")
	transaction.AddTag("essential")

	assert.Len(t, transaction.Tags, 3)
	assert.Contains(t, transaction.Tags, "food")
	assert.Contains(t, transaction.Tags, "grocery")
	assert.Contains(t, transaction.Tags, "essential")

	// Test removing middle tag
	transaction.RemoveTag("grocery")

	assert.Len(t, transaction.Tags, 2)
	assert.Contains(t, transaction.Tags, "food")
	assert.NotContains(t, transaction.Tags, "grocery")
	assert.Contains(t, transaction.Tags, "essential")

	// Test removing first tag
	transaction.RemoveTag("food")

	assert.Len(t, transaction.Tags, 1)
	assert.NotContains(t, transaction.Tags, "food")
	assert.Contains(t, transaction.Tags, "essential")

	// Test removing last tag
	transaction.RemoveTag("essential")

	assert.Empty(t, transaction.Tags)
}

func TestTransaction_TimestampGeneration(t *testing.T) {
	// Record time before creating transaction
	beforeTime := time.Now()

	// Create transaction
	transaction := NewTransaction(100.0, TransactionTypeExpense, "Test", uuid.New(), uuid.New(), uuid.New(), time.Now())

	// Record time after creating transaction
	afterTime := time.Now()

	// Assert timestamps are within expected range
	assert.True(t, transaction.CreatedAt.After(beforeTime) || transaction.CreatedAt.Equal(beforeTime))
	assert.True(t, transaction.CreatedAt.Before(afterTime) || transaction.CreatedAt.Equal(afterTime))
	assert.True(t, transaction.UpdatedAt.After(beforeTime) || transaction.UpdatedAt.Equal(beforeTime))
	assert.True(t, transaction.UpdatedAt.Before(afterTime) || transaction.UpdatedAt.Equal(afterTime))
}

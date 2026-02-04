package models_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/web/models"
)

func TestTransactionForm_GetAmount(t *testing.T) {
	tests := []struct {
		name      string
		amount    string
		expected  float64
		expectErr bool
	}{
		{
			name:      "valid integer",
			amount:    "100",
			expected:  100.0,
			expectErr: false,
		},
		{
			name:      "valid decimal",
			amount:    "123.45",
			expected:  123.45,
			expectErr: false,
		},
		{
			name:      "invalid - text",
			amount:    "invalid",
			expectErr: true,
		},
		{
			name:      "invalid - empty",
			amount:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.TransactionForm{Amount: tt.amount}
			result, err := form.GetAmount()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTransactionForm_ToTransactionType(t *testing.T) {
	tests := []struct {
		name     string
		formType string
		expected transaction.Type
	}{
		{"income", models.TypeIncome, transaction.TypeIncome},
		{"expense", models.TypeExpense, transaction.TypeExpense},
		{"default expense", "invalid", transaction.TypeExpense},
		{"empty defaults to expense", "", transaction.TypeExpense},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.TransactionForm{Type: tt.formType}
			result := form.ToTransactionType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionForm_GetCategoryID(t *testing.T) {
	validUUID := uuid.New()

	tests := []struct {
		name       string
		categoryID string
		expectErr  bool
	}{
		{
			name:       "valid UUID",
			categoryID: validUUID.String(),
			expectErr:  false,
		},
		{
			name:       "invalid UUID",
			categoryID: "invalid-uuid",
			expectErr:  true,
		},
		{
			name:       "empty",
			categoryID: "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.TransactionForm{CategoryID: tt.categoryID}
			result, err := form.GetCategoryID()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, validUUID, result)
			}
		})
	}
}

func TestTransactionForm_GetDate(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		expectErr bool
	}{
		{
			name:      "valid date",
			date:      "2024-01-15",
			expectErr: false,
		},
		{
			name:      "invalid format",
			date:      "15-01-2024",
			expectErr: true,
		},
		{
			name:      "empty",
			date:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.TransactionForm{Date: tt.date}
			result, err := form.GetDate()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 2024, result.Year())
				assert.Equal(t, time.January, result.Month())
				assert.Equal(t, 15, result.Day())
			}
		})
	}
}

func TestTransactionForm_GetTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     string
		expected []string
	}{
		{
			name:     "single tag",
			tags:     "shopping",
			expected: []string{"shopping"},
		},
		{
			name:     "multiple tags",
			tags:     "food,groceries,weekly",
			expected: []string{"food", "groceries", "weekly"},
		},
		{
			name:     "tags with spaces",
			tags:     "tag1, tag2, tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty string",
			tags:     "",
			expected: []string{},
		},
		{
			name:     "only commas",
			tags:     ",,,",
			expected: []string{},
		},
		{
			name:     "mixed empty and valid",
			tags:     "tag1,,tag2,",
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.TransactionForm{Tags: tt.tags}
			result := form.GetTags()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionViewModel_FromDomain(t *testing.T) {
	now := time.Now()
	txID := uuid.New()
	categoryID := uuid.New()
	userID := uuid.New()

	domainTx := &transaction.Transaction{
		ID:          txID,
		Amount:      150.0,
		Type:        transaction.TypeExpense,
		Description: "Groceries",
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        now,
		Tags:        []string{"food", "weekly"},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	vm := &models.TransactionViewModel{}
	vm.FromDomain(domainTx)

	assert.Equal(t, txID, vm.ID)
	assert.Equal(t, 150.0, vm.Amount)
	assert.Equal(t, transaction.TypeExpense, vm.Type)
	assert.Equal(t, "Groceries", vm.Description)
	assert.Equal(t, categoryID, vm.CategoryID)
	assert.Equal(t, userID, vm.UserID)
	assert.Equal(t, now, vm.Date)
	assert.Equal(t, []string{"food", "weekly"}, vm.Tags)
	assert.True(t, vm.CanEdit)
	assert.True(t, vm.CanDelete)
	assert.NotEmpty(t, vm.FormattedAmount)
	assert.NotEmpty(t, vm.FormattedDate)
}

func TestTransactionViewModel_FromDomain_Income(t *testing.T) {
	now := time.Now()
	domainTx := &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      3000.0,
		Type:        transaction.TypeIncome,
		Description: "Salary",
		CategoryID:  uuid.New(),
		UserID:      uuid.New(),
		Date:        now,
		Tags:        []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	vm := &models.TransactionViewModel{}
	vm.FromDomain(domainTx)

	assert.Equal(t, transaction.TypeIncome, vm.Type)
	assert.Contains(t, vm.FormattedAmount, "+")
	assert.Empty(t, vm.Tags)
}

func TestTransactionFilters_ToDomainFilter(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()

	filters := &models.TransactionFilters{
		UserID:      userID.String(),
		CategoryID:  categoryID.String(),
		Type:        "expense",
		DateFrom:    "2024-01-01",
		DateTo:      "2024-01-31",
		AmountFrom:  "100",
		AmountTo:    "500",
		Tags:        "food,groceries",
		Description: "test",
		Page:        2,
		PageSize:    25,
	}

	domainFilter := filters.ToDomainFilter()

	assert.NotNil(t, domainFilter.UserID)
	assert.Equal(t, userID, *domainFilter.UserID)
	assert.NotNil(t, domainFilter.CategoryID)
	assert.Equal(t, categoryID, *domainFilter.CategoryID)
	assert.NotNil(t, domainFilter.Type)
	assert.Equal(t, transaction.TypeExpense, *domainFilter.Type)
	assert.NotNil(t, domainFilter.DateFrom)
	assert.NotNil(t, domainFilter.DateTo)
	assert.NotNil(t, domainFilter.AmountFrom)
	assert.Equal(t, 100.0, *domainFilter.AmountFrom)
	assert.NotNil(t, domainFilter.AmountTo)
	assert.Equal(t, 500.0, *domainFilter.AmountTo)
	assert.Equal(t, []string{"food", "groceries"}, domainFilter.Tags)
	assert.Equal(t, "test", domainFilter.Description)
	assert.Equal(t, 25, domainFilter.Limit)
	assert.Equal(t, 25, domainFilter.Offset) // (page 2 - 1) * 25
}

func TestTransactionFilters_ToDomainFilter_EmptyFilters(t *testing.T) {
	filters := &models.TransactionFilters{}
	domainFilter := filters.ToDomainFilter()

	assert.Nil(t, domainFilter.UserID)
	assert.Nil(t, domainFilter.CategoryID)
	assert.Nil(t, domainFilter.Type)
	assert.Nil(t, domainFilter.DateFrom)
	assert.Nil(t, domainFilter.DateTo)
	assert.Nil(t, domainFilter.AmountFrom)
	assert.Nil(t, domainFilter.AmountTo)
	assert.Empty(t, domainFilter.Tags)
	assert.Empty(t, domainFilter.Description)
	assert.Equal(t, models.DefaultPageSize, domainFilter.Limit)
	assert.Equal(t, 0, domainFilter.Offset)
}

func TestTransactionFilters_ToDomainFilter_InvalidValues(t *testing.T) {
	filters := &models.TransactionFilters{
		UserID:     "invalid-uuid",
		CategoryID: "invalid-uuid",
		Type:       "invalid",
		DateFrom:   "invalid-date",
		DateTo:     "invalid-date",
		AmountFrom: "invalid",
		AmountTo:   "invalid",
		Tags:       "   ,  ,  ",
	}

	domainFilter := filters.ToDomainFilter()

	// Invalid UUIDs should result in nil
	assert.Nil(t, domainFilter.UserID)
	assert.Nil(t, domainFilter.CategoryID)

	// Invalid type should result in nil
	assert.Nil(t, domainFilter.Type)

	// Invalid dates should result in nil
	assert.Nil(t, domainFilter.DateFrom)
	assert.Nil(t, domainFilter.DateTo)

	// Invalid amounts should result in nil
	assert.Nil(t, domainFilter.AmountFrom)
	assert.Nil(t, domainFilter.AmountTo)

	// Empty tags should be filtered out
	assert.Empty(t, domainFilter.Tags)
}

func TestTransactionFilters_Pagination(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "first page default size",
			page:           1,
			pageSize:       0,
			expectedLimit:  models.DefaultPageSize,
			expectedOffset: 0,
		},
		{
			name:           "second page custom size",
			page:           2,
			pageSize:       10,
			expectedLimit:  10,
			expectedOffset: 10,
		},
		{
			name:           "third page",
			page:           3,
			pageSize:       25,
			expectedLimit:  25,
			expectedOffset: 50,
		},
		{
			name:           "no pagination",
			page:           0,
			pageSize:       0,
			expectedLimit:  models.DefaultPageSize,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters := &models.TransactionFilters{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			domainFilter := filters.ToDomainFilter()

			assert.Equal(t, tt.expectedLimit, domainFilter.Limit)
			assert.Equal(t, tt.expectedOffset, domainFilter.Offset)
		})
	}
}

func TestBulkOperationForm_GetTransactionIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	tests := []struct {
		name           string
		transactionIDs []string
		expectErr      bool
		expectedLen    int
	}{
		{
			name: "valid IDs",
			transactionIDs: []string{
				id1.String(),
				id2.String(),
				id3.String(),
			},
			expectErr:   false,
			expectedLen: 3,
		},
		{
			name:           "empty list",
			transactionIDs: []string{},
			expectErr:      false,
			expectedLen:    0,
		},
		{
			name: "invalid UUID",
			transactionIDs: []string{
				id1.String(),
				"invalid-uuid",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.BulkOperationForm{TransactionIDs: tt.transactionIDs}
			result, err := form.GetTransactionIDs()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}
		})
	}
}

func TestTransactionListResponse(t *testing.T) {
	response := models.TransactionListResponse{
		Transactions: []models.TransactionViewModel{},
		Total:        100,
		Page:         2,
		PageSize:     10,
		TotalPages:   10,
		HasNext:      true,
		HasPrev:      true,
	}

	assert.Equal(t, 100, response.Total)
	assert.Equal(t, 2, response.Page)
	assert.Equal(t, 10, response.PageSize)
	assert.Equal(t, 10, response.TotalPages)
	assert.True(t, response.HasNext)
	assert.True(t, response.HasPrev)
}

func TestTransactionViewModel_EdgeCases(t *testing.T) {
	t.Run("unicode in description", func(t *testing.T) {
		now := time.Now()
		domainTx := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      100.0,
			Type:        transaction.TypeExpense,
			Description: "Покупки в магазине 🛒",
			CategoryID:  uuid.New(),
			UserID:      uuid.New(),
			Date:        now,
			Tags:        []string{"русский", "тег"},
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		vm := &models.TransactionViewModel{}
		vm.FromDomain(domainTx)

		assert.Equal(t, "Покупки в магазине 🛒", vm.Description)
		assert.Contains(t, vm.Tags, "русский")
		assert.Contains(t, vm.Tags, "тег")
	})

	t.Run("empty tags", func(t *testing.T) {
		now := time.Now()
		domainTx := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      50.0,
			Type:        transaction.TypeIncome,
			Description: "Payment",
			CategoryID:  uuid.New(),
			UserID:      uuid.New(),
			Date:        now,
			Tags:        []string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		vm := &models.TransactionViewModel{}
		vm.FromDomain(domainTx)

		assert.NotNil(t, vm.Tags)
		assert.Empty(t, vm.Tags)
	})

	t.Run("large amount", func(t *testing.T) {
		now := time.Now()
		domainTx := &transaction.Transaction{
			ID:          uuid.New(),
			Amount:      999999.99,
			Type:        transaction.TypeIncome,
			Description: "Large payment",
			CategoryID:  uuid.New(),
			UserID:      uuid.New(),
			Date:        now,
			Tags:        []string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		vm := &models.TransactionViewModel{}
		vm.FromDomain(domainTx)

		assert.Equal(t, 999999.99, vm.Amount)
		assert.NotEmpty(t, vm.FormattedAmount)
	})
}

func TestBulkOperationForm(t *testing.T) {
	t.Run("delete action", func(t *testing.T) {
		form := models.BulkOperationForm{
			Action:         "delete",
			TransactionIDs: []string{uuid.New().String()},
		}

		assert.Equal(t, "delete", form.Action)
	})

	t.Run("update category action", func(t *testing.T) {
		categoryID := uuid.New()
		form := models.BulkOperationForm{
			Action:         "update_category",
			TransactionIDs: []string{uuid.New().String()},
			CategoryID:     categoryID.String(),
		}

		assert.Equal(t, "update_category", form.Action)
		assert.Equal(t, categoryID.String(), form.CategoryID)
	})

	t.Run("update tags action", func(t *testing.T) {
		form := models.BulkOperationForm{
			Action:         "update_tags",
			TransactionIDs: []string{uuid.New().String()},
			Tags:           "new,tags",
		}

		assert.Equal(t, "update_tags", form.Action)
		assert.Equal(t, "new,tags", form.Tags)
	})
}

func TestTransactionConstants(t *testing.T) {
	assert.Equal(t, "income", models.TypeIncome)
	assert.Equal(t, "expense", models.TypeExpense)
	assert.Equal(t, 50, models.DefaultPageSize)
}

func TestTransactionFilters_DateTo_EndOfDay(t *testing.T) {
	filters := &models.TransactionFilters{
		DateTo: "2024-01-15",
	}

	domainFilter := filters.ToDomainFilter()

	assert.NotNil(t, domainFilter.DateTo)
	assert.Equal(t, 23, domainFilter.DateTo.Hour())
	assert.Equal(t, 59, domainFilter.DateTo.Minute())
	assert.Equal(t, 59, domainFilter.DateTo.Second())
}

func TestTransactionForm_LongDescription(t *testing.T) {
	longDesc := strings.Repeat("A", 600) // More than 500 characters
	form := models.TransactionForm{
		Description: longDesc,
	}

	assert.Greater(t, len(form.Description), 500)
}

func TestTransactionFilters_WhitespaceHandling(t *testing.T) {
	filters := &models.TransactionFilters{
		Description: "  test description  ",
		Tags:        "  tag1  ,  tag2  ,  tag3  ",
	}

	domainFilter := filters.ToDomainFilter()

	assert.Equal(t, "test description", domainFilter.Description)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, domainFilter.Tags)
}

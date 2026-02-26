package dto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

func TestFromCreateUserAPIRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  CreateUserAPIRequest
		expected CreateUserDTO
	}{
		{
			name: "valid user request",
			request: CreateUserAPIRequest{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      "admin",
			},
			expected: CreateUserDTO{
				Email:     "test@example.com",
				Password:  "password123",
				FirstName: "John",
				LastName:  "Doe",
				Role:      user.RoleAdmin,
			},
		},
		{
			name: "member role",
			request: CreateUserAPIRequest{
				Email:     "member@example.com",
				Password:  "password123",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      "member",
			},
			expected: CreateUserDTO{
				Email:     "member@example.com",
				Password:  "password123",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      user.RoleMember,
			},
		},
		{
			name: "child role",
			request: CreateUserAPIRequest{
				Email:     "child@example.com",
				Password:  "password123",
				FirstName: "Kid",
				LastName:  "Jones",
				Role:      "child",
			},
			expected: CreateUserDTO{
				Email:     "child@example.com",
				Password:  "password123",
				FirstName: "Kid",
				LastName:  "Jones",
				Role:      user.RoleChild,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromCreateUserAPIRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromUpdateUserAPIRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  UpdateUserAPIRequest
		expected UpdateUserDTO
	}{
		{
			name: "update all fields",
			request: UpdateUserAPIRequest{
				FirstName: new("NewFirst"),
				LastName:  new("NewLast"),
				Email:     new("new@example.com"),
			},
			expected: UpdateUserDTO{
				FirstName: new("NewFirst"),
				LastName:  new("NewLast"),
				Email:     new("new@example.com"),
			},
		},
		{
			name: "update only first name",
			request: UpdateUserAPIRequest{
				FirstName: new("UpdatedFirst"),
			},
			expected: UpdateUserDTO{
				FirstName: new("UpdatedFirst"),
			},
		},
		{
			name:     "empty update",
			request:  UpdateUserAPIRequest{},
			expected: UpdateUserDTO{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromUpdateUserAPIRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToUserAPIResponse(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	tests := []struct {
		name     string
		user     *user.User
		expected UserAPIResponse
	}{
		{
			name: "admin user",
			user: &user.User{
				ID:        userID,
				Email:     "admin@example.com",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: UserAPIResponse{
				ID:        userID,
				Email:     "admin@example.com",
				FirstName: "Admin",
				LastName:  "User",
				Role:      "admin",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "member user",
			user: &user.User{
				ID:        userID,
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Role:      user.RoleMember,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: UserAPIResponse{
				ID:        userID,
				Email:     "member@example.com",
				FirstName: "Member",
				LastName:  "User",
				Role:      "member",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUserAPIResponse(tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromCreateCategoryAPIRequest(t *testing.T) {
	parentID := uuid.New()

	tests := []struct {
		name     string
		request  CreateCategoryAPIRequest
		expected CreateCategoryDTO
	}{
		{
			name: "parent category",
			request: CreateCategoryAPIRequest{
				Name:  "Food",
				Type:  "expense",
				Color: "#FF0000",
				Icon:  "food",
			},
			expected: CreateCategoryDTO{
				Name:  "Food",
				Type:  category.TypeExpense,
				Color: "#FF0000",
				Icon:  "food",
			},
		},
		{
			name: "child category with parent",
			request: CreateCategoryAPIRequest{
				Name:     "Groceries",
				Type:     "expense",
				Color:    "#00FF00",
				Icon:     "cart",
				ParentID: &parentID,
			},
			expected: CreateCategoryDTO{
				Name:     "Groceries",
				Type:     category.TypeExpense,
				Color:    "#00FF00",
				Icon:     "cart",
				ParentID: &parentID,
			},
		},
		{
			name: "income category",
			request: CreateCategoryAPIRequest{
				Name:  "Salary",
				Type:  "income",
				Color: "#0000FF",
				Icon:  "money",
			},
			expected: CreateCategoryDTO{
				Name:  "Salary",
				Type:  category.TypeIncome,
				Color: "#0000FF",
				Icon:  "money",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromCreateCategoryAPIRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromUpdateCategoryAPIRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  UpdateCategoryAPIRequest
		expected UpdateCategoryDTO
	}{
		{
			name: "update all fields",
			request: UpdateCategoryAPIRequest{
				Name:  new("NewName"),
				Color: new("#FFFFFF"),
				Icon:  new("new-icon"),
			},
			expected: UpdateCategoryDTO{
				Name:  new("NewName"),
				Color: new("#FFFFFF"),
				Icon:  new("new-icon"),
			},
		},
		{
			name: "update only name",
			request: UpdateCategoryAPIRequest{
				Name: new("UpdatedName"),
			},
			expected: UpdateCategoryDTO{
				Name: new("UpdatedName"),
			},
		},
		{
			name:     "empty update",
			request:  UpdateCategoryAPIRequest{},
			expected: UpdateCategoryDTO{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromUpdateCategoryAPIRequest(tt.request)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToCategoryAPIResponse(t *testing.T) {
	now := time.Now()
	categoryID := uuid.New()
	parentID := uuid.New()

	tests := []struct {
		name     string
		category *category.Category
		expected CategoryAPIResponse
	}{
		{
			name: "parent category",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Food",
				Type:      category.TypeExpense,
				Color:     "#FF0000",
				Icon:      "food",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Food",
				Type:      "expense",
				Color:     "#FF0000",
				Icon:      "food",
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "child category with parent",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Groceries",
				Type:      category.TypeExpense,
				Color:     "#00FF00",
				Icon:      "cart",
				ParentID:  &parentID,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Groceries",
				Type:      "expense",
				Color:     "#00FF00",
				Icon:      "cart",
				ParentID:  &parentID,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "inactive category",
			category: &category.Category{
				ID:        categoryID,
				Name:      "Old Category",
				Type:      category.TypeExpense,
				Color:     "#000000",
				Icon:      "archive",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: CategoryAPIResponse{
				ID:        categoryID,
				Name:      "Old Category",
				Type:      "expense",
				Color:     "#000000",
				Icon:      "archive",
				IsActive:  false,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToCategoryAPIResponse(tt.category)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateTransactionAPIRequest_ToCreateTransactionDTO(t *testing.T) {
	categoryID := uuid.New()
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		request  CreateTransactionAPIRequest
		expected CreateTransactionDTO
	}{
		{
			name: "income transaction",
			request: CreateTransactionAPIRequest{
				Amount:      1500.50,
				Type:        "income",
				Description: "Salary",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"monthly", "salary"},
			},
			expected: CreateTransactionDTO{
				Amount:      1500.50,
				Type:        transaction.TypeIncome,
				Description: "Salary",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"monthly", "salary"},
			},
		},
		{
			name: "expense transaction",
			request: CreateTransactionAPIRequest{
				Amount:      100.00,
				Type:        "expense",
				Description: "Groceries",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
			},
			expected: CreateTransactionDTO{
				Amount:      100.00,
				Type:        transaction.TypeExpense,
				Description: "Groceries",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
			},
		},
		{
			name: "transaction with tags",
			request: CreateTransactionAPIRequest{
				Amount:      50.00,
				Type:        "expense",
				Description: "Coffee",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"cafe", "weekend"},
			},
			expected: CreateTransactionDTO{
				Amount:      50.00,
				Type:        transaction.TypeExpense,
				Description: "Coffee",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"cafe", "weekend"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.ToCreateTransactionDTO()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateTransactionAPIRequest_ToUpdateTransactionDTO(t *testing.T) {
	categoryID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		request  UpdateTransactionAPIRequest
		expected UpdateTransactionDTO
	}{
		{
			name: "update all fields",
			request: UpdateTransactionAPIRequest{
				Amount:      new(200.00),
				Type:        new("expense"),
				Description: new("Updated description"),
				CategoryID:  &categoryID,
				Date:        &date,
				Tags:        []string{"updated"},
			},
			expected: UpdateTransactionDTO{
				Amount:      new(200.00),
				Type:        transactionTypePtr(transaction.TypeExpense),
				Description: new("Updated description"),
				CategoryID:  &categoryID,
				Date:        &date,
				Tags:        []string{"updated"},
			},
		},
		{
			name: "update only amount",
			request: UpdateTransactionAPIRequest{
				Amount: new(150.00),
			},
			expected: UpdateTransactionDTO{
				Amount: new(150.00),
			},
		},
		{
			name: "update type to income",
			request: UpdateTransactionAPIRequest{
				Type: new("income"),
			},
			expected: UpdateTransactionDTO{
				Type: transactionTypePtr(transaction.TypeIncome),
			},
		},
		{
			name:     "empty update",
			request:  UpdateTransactionAPIRequest{},
			expected: UpdateTransactionDTO{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.ToUpdateTransactionDTO()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToTransactionAPIResponse(t *testing.T) {
	now := time.Now()
	transactionID := uuid.New()
	categoryID := uuid.New()
	userID := uuid.New()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		transaction *transaction.Transaction
		expected    TransactionAPIResponse
	}{
		{
			name: "income transaction",
			transaction: &transaction.Transaction{
				ID:          transactionID,
				Amount:      1500.50,
				Type:        transaction.TypeIncome,
				Description: "Salary",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"monthly"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			expected: TransactionAPIResponse{
				ID:          transactionID,
				Amount:      1500.50,
				Type:        "income",
				Description: "Salary",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"monthly"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "expense transaction with tags",
			transaction: &transaction.Transaction{
				ID:          transactionID,
				Amount:      100.00,
				Type:        transaction.TypeExpense,
				Description: "Groceries",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"food", "weekly"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			expected: TransactionAPIResponse{
				ID:          transactionID,
				Amount:      100.00,
				Type:        "expense",
				Description: "Groceries",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        []string{"food", "weekly"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		{
			name: "transaction without tags",
			transaction: &transaction.Transaction{
				ID:          transactionID,
				Amount:      50.00,
				Type:        transaction.TypeExpense,
				Description: "Coffee",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        nil,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			expected: TransactionAPIResponse{
				ID:          transactionID,
				Amount:      50.00,
				Type:        "expense",
				Description: "Coffee",
				CategoryID:  categoryID,
				UserID:      userID,
				Date:        date,
				Tags:        nil,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTransactionAPIResponse(tt.transaction)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func transactionTypePtr(t transaction.Type) *transaction.Type {
	return new(t)
}

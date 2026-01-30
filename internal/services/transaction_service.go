package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
)

var (
	ErrTransactionNotFound       = errors.New("transaction not found")
	ErrInvalidTransactionAmount  = errors.New("transaction amount must be greater than 0")
	ErrInvalidTransactionType    = errors.New("invalid transaction type")
	ErrCategoryNotInFamily       = errors.New("category does not belong to the specified family")
	ErrUserNotInFamily           = errors.New("user does not belong to the specified family")
	ErrInsufficientBudget        = errors.New("transaction would exceed budget limit")
	ErrBudgetNotFound            = errors.New("budget not found")
	ErrTransactionUpdateFailed   = errors.New("failed to update transaction")
	ErrTransactionDeleteFailed   = errors.New("failed to delete transaction")
	ErrBulkCategorizePartialFail = errors.New("some transactions failed to update during bulk categorization")
)

// TransactionRepository defines the data access operations for transactions
type TransactionRepository interface {
	Create(ctx context.Context, transaction *transaction.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetByFilter(ctx context.Context, filter transaction.Filter) ([]*transaction.Transaction, error)
	Update(ctx context.Context, transaction *transaction.Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetTotalByCategory(ctx context.Context, categoryID uuid.UUID, transactionType transaction.Type) (float64, error)
	GetTotalByDateRange(
		ctx context.Context,
		startDate, endDate time.Time,
		transactionType transaction.Type,
	) (float64, error)
	GetTotalByCategoryAndDateRange(
		ctx context.Context,
		categoryID uuid.UUID,
		startDate, endDate time.Time,
		transactionType transaction.Type,
	) (float64, error)
	// Note: UpdateBulkCategory may need to be implemented in the repository
	// For now, we'll use individual updates in a transaction
}

// BudgetRepositoryForTransactions defines the budget operations needed for transaction service
type BudgetRepositoryForTransactions interface {
	GetActiveBudgets(ctx context.Context) ([]*budget.Budget, error)
	Update(ctx context.Context, budget *budget.Budget) error
	// Note: GetByCategoryAndFamily may need to be added to budget repository
	// For now, we'll iterate through active budgets to find the right one
}

// Repository interfaces needed for TransactionService
// Note: These are minimal interfaces that may be satisfied by the full repository implementations

type CategoryRepositoryForTransactions interface {
	GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
}

type UserRepositoryForTransactions interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

// TransactionServiceImpl implements the TransactionService interface
type TransactionServiceImpl struct {
	transactionRepo TransactionRepository
	budgetRepo      BudgetRepositoryForTransactions
	categoryRepo    CategoryRepositoryForTransactions
	userRepo        UserRepositoryForTransactions
	validator       *validator.Validate
}

// NewTransactionService creates a new TransactionService instance
func NewTransactionService(
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	categoryRepo CategoryRepositoryForTransactions,
	userRepo UserRepositoryForTransactions,
) *TransactionServiceImpl {
	return &TransactionServiceImpl{
		transactionRepo: transactionRepo,
		budgetRepo:      budgetRepo,
		categoryRepo:    categoryRepo,
		userRepo:        userRepo,
		validator:       validator.New(),
	}
}

// CreateTransaction creates a new transaction with business logic validation
func (s *TransactionServiceImpl) CreateTransaction(
	ctx context.Context,
	req dto.CreateTransactionDTO,
) (*transaction.Transaction, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate user exists (single-family model - no family validation needed)
	if err := s.validateUserExists(ctx, req.UserID); err != nil {
		return nil, err
	}

	// Validate category exists (single-family model - no family validation needed)
	if err := s.validateCategoryExists(ctx, req.CategoryID); err != nil {
		return nil, err
	}

	// For expense transactions, check budget limits
	if req.Type == transaction.TypeExpense {
		if err := s.ValidateTransactionLimits(ctx, req.CategoryID, req.Amount, req.Type); err != nil {
			return nil, err
		}
	}

	// Create transaction
	newTransaction := &transaction.Transaction{
		ID:          uuid.New(),
		Amount:      req.Amount,
		Type:        req.Type,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		Date:        req.Date,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, newTransaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update budget if it's an expense transaction
	if req.Type == transaction.TypeExpense {
		if budgetErr := s.updateBudgetSpent(ctx, req.CategoryID, req.Amount); budgetErr != nil {
			// Log the error but don't fail the transaction creation
			// In a production system, you might want to use a message queue for this
			// TODO: Replace with proper logging system
			_ = budgetErr // Ignore budget update errors for now
		}
	}

	return newTransaction, nil
}

// GetTransactionByID retrieves a transaction by its ID
func (s *TransactionServiceImpl) GetTransactionByID(
	ctx context.Context,
	id uuid.UUID,
) (*transaction.Transaction, error) {
	tx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}
	return tx, nil
}

// GetTransactionsByFamily retrieves transactions for a family with filtering
func (s *TransactionServiceImpl) GetTransactionsByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	if err := s.validator.Struct(filter); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate date and amount ranges
	if err := filter.ValidateDateRange(); err != nil {
		return nil, err
	}
	if err := filter.ValidateAmountRange(); err != nil {
		return nil, err
	}

	// Set family ID from parameter to ensure consistency
	filter.FamilyID = familyID

	// Convert DTO filter to domain filter
	repoFilter := s.convertDTOFilterToRepoFilter(filter)

	transactions, err := s.transactionRepo.GetByFilter(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

// UpdateTransaction updates an existing transaction
func (s *TransactionServiceImpl) UpdateTransaction(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateTransactionDTO,
) (*transaction.Transaction, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing transaction
	existingTx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTransactionNotFound
	}

	// Store original values for budget adjustment
	originalAmount := existingTx.Amount
	originalType := existingTx.Type
	originalCategoryID := existingTx.CategoryID

	// Update fields if provided
	if req.Amount != nil {
		existingTx.Amount = *req.Amount
	}
	if req.Type != nil {
		existingTx.Type = *req.Type
	}
	if req.Description != nil {
		existingTx.Description = *req.Description
	}
	if req.CategoryID != nil {
		existingTx.CategoryID = *req.CategoryID
	}
	if req.Date != nil {
		existingTx.Date = *req.Date
	}
	if req.Tags != nil {
		existingTx.Tags = req.Tags
	}
	existingTx.UpdatedAt = time.Now()

	// Validate budget limits for new values if it's an expense
	if existingTx.Type == transaction.TypeExpense {
		if limitErr := s.ValidateTransactionLimits(
			ctx,
			existingTx.CategoryID,
			existingTx.Amount,
			existingTx.Type,
		); limitErr != nil {
			return nil, limitErr
		}
	}

	// Update transaction
	if updateErr := s.transactionRepo.Update(ctx, existingTx); updateErr != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", updateErr)
	}

	// Adjust budgets for the changes
	if budgetErr := s.adjustBudgetsForUpdate(
		ctx,
		originalAmount,
		originalType,
		originalCategoryID,
		existingTx,
	); budgetErr != nil {
		// TODO: Replace with proper logging system
		_ = budgetErr // Ignore budget adjustment errors for now
	}

	return existingTx, nil
}

// DeleteTransaction deletes a transaction and adjusts budgets
func (s *TransactionServiceImpl) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	// Get existing transaction for budget adjustment
	existingTx, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return ErrTransactionNotFound
	}

	// Delete transaction - familyID is obtained internally by repository
	if deleteErr := s.transactionRepo.Delete(ctx, id); deleteErr != nil {
		return fmt.Errorf("failed to delete transaction: %w", deleteErr)
	}

	// Reverse budget impact if it was an expense
	if existingTx.Type == transaction.TypeExpense {
		if budgetErr := s.updateBudgetSpent(
			ctx,
			existingTx.CategoryID,
			-existingTx.Amount,
		); budgetErr != nil {
			// TODO: Replace with proper logging system
			_ = budgetErr // Ignore budget reversal errors for now
		}
	}

	return nil
}

// GetTransactionsByCategory retrieves transactions for a specific category
func (s *TransactionServiceImpl) GetTransactionsByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	if err := s.validator.Struct(filter); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set category ID in filter
	filter.CategoryID = &categoryID

	// Convert DTO filter to domain filter
	repoFilter := s.convertDTOFilterToRepoFilter(filter)

	transactions, err := s.transactionRepo.GetByFilter(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by category: %w", err)
	}

	return transactions, nil
}

const (
	// DateRangeQueryLimit is the default limit for date range queries
	DateRangeQueryLimit = 100
)

// GetTransactionsByDateRange retrieves transactions within a date range
func (s *TransactionServiceImpl) GetTransactionsByDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	from, to time.Time,
) ([]*transaction.Transaction, error) {
	if to.Before(from) {
		return nil, dto.ErrInvalidDateRange
	}

	filter := dto.TransactionFilterDTO{
		FamilyID: familyID,
		DateFrom: &from,
		DateTo:   &to,
		Limit:    DateRangeQueryLimit,
		Offset:   0,
	}

	return s.GetTransactionsByFamily(ctx, familyID, filter)
}

// BulkCategorizeTransactions updates categories for multiple transactions
func (s *TransactionServiceImpl) BulkCategorizeTransactions(
	ctx context.Context,
	transactionIDs []uuid.UUID,
	categoryID uuid.UUID,
) error {
	if len(transactionIDs) == 0 {
		return errors.New("no transaction IDs provided")
	}

	// Validate category exists
	_, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("category not found: %w", err)
	}

	// Retrieve all transactions
	transactions, err := s.retrieveTransactions(ctx, transactionIDs)
	if err != nil {
		return err
	}

	// Update transactions individually
	failedUpdates := s.updateTransactionsCategory(ctx, transactions, categoryID)

	if failedUpdates > 0 {
		return fmt.Errorf("%w: %d out of %d transactions failed to update",
			ErrBulkCategorizePartialFail, failedUpdates, len(transactionIDs))
	}

	return nil
}

func (s *TransactionServiceImpl) retrieveTransactions(
	ctx context.Context,
	transactionIDs []uuid.UUID,
) (map[uuid.UUID]*transaction.Transaction, error) {
	transactions := make(map[uuid.UUID]*transaction.Transaction)

	for _, txID := range transactionIDs {
		tx, err := s.transactionRepo.GetByID(ctx, txID)
		if err != nil {
			return nil, fmt.Errorf("transaction %s not found: %w", txID, err)
		}

		transactions[txID] = tx
	}

	return transactions, nil
}

func (s *TransactionServiceImpl) updateTransactionsCategory(
	ctx context.Context,
	transactions map[uuid.UUID]*transaction.Transaction,
	categoryID uuid.UUID,
) int {
	failedUpdates := 0

	for txID, oldTx := range transactions {
		if oldTx.CategoryID == categoryID {
			continue // No change needed
		}

		if err := s.updateSingleTransactionCategory(ctx, oldTx, categoryID, transactions[txID].CategoryID); err != nil {
			// TODO: Replace with proper logging system
			_ = err // Ignore individual update errors for now
			failedUpdates++
		}
	}

	return failedUpdates
}

func (s *TransactionServiceImpl) updateSingleTransactionCategory(
	ctx context.Context,
	tx *transaction.Transaction,
	newCategoryID, originalCategoryID uuid.UUID,
) error {
	// Update transaction
	tx.CategoryID = newCategoryID
	tx.UpdatedAt = time.Now()

	if err := s.transactionRepo.Update(ctx, tx); err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	// Adjust budgets for category changes (only for expense transactions)
	if tx.Type == transaction.TypeExpense {
		s.adjustBudgetsForCategoryChange(ctx, originalCategoryID, newCategoryID, tx.Amount)
	}

	return nil
}

func (s *TransactionServiceImpl) adjustBudgetsForCategoryChange(
	ctx context.Context,
	oldCategoryID, newCategoryID uuid.UUID,
	amount float64,
) {
	// Remove from old category budget
	if budgetErr := s.updateBudgetSpent(ctx, oldCategoryID, -amount); budgetErr != nil {
		// TODO: Replace with proper logging system
		_ = budgetErr // Ignore budget adjustment errors for now
	}

	// Add to new category budget
	if budgetErr := s.updateBudgetSpent(ctx, newCategoryID, amount); budgetErr != nil {
		// TODO: Replace with proper logging system
		_ = budgetErr // Ignore budget adjustment errors for now
	}
}

// ValidateTransactionLimits checks if a transaction would exceed budget limits
func (s *TransactionServiceImpl) ValidateTransactionLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
) error {
	// Only check limits for expense transactions
	if transactionType != transaction.TypeExpense {
		return nil
	}

	// Get active budget for the category
	budget, err := s.findBudgetByCategory(ctx, categoryID)
	if err != nil {
		// No budget means no limit - allow the transaction
		return nil //nolint:nilerr // No budget found is acceptable, not an error condition
	}

	// Check if adding this transaction would exceed the budget limit
	if budget.Spent+amount > budget.Amount {
		return fmt.Errorf("%w: budget amount %.2f, current spent %.2f, transaction amount %.2f",
			ErrInsufficientBudget, budget.Amount, budget.Spent, amount)
	}

	return nil
}

// Helper methods

// validateUserExists checks if a user exists (simplified for single-family model)
func (s *TransactionServiceImpl) validateUserExists(ctx context.Context, userID uuid.UUID) error {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	return nil
}

// validateCategoryExists checks if a category exists (simplified for single-family model)
func (s *TransactionServiceImpl) validateCategoryExists(ctx context.Context, categoryID uuid.UUID) error {
	_, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("category not found: %w", err)
	}
	return nil
}

func (s *TransactionServiceImpl) updateBudgetSpent(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
) error {
	budget, err := s.findBudgetByCategory(ctx, categoryID)
	if err != nil {
		// No budget found - this is acceptable, not all categories need budgets
		return nil //nolint:nilerr // No budget found is acceptable, not an error condition
	}

	budget.Spent += amount
	budget.UpdatedAt = time.Now()

	return s.budgetRepo.Update(ctx, budget)
}

func (s *TransactionServiceImpl) findBudgetByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
) (*budget.Budget, error) {
	// Get active budgets (single family model - repository handles family ID internally)
	budgets, err := s.budgetRepo.GetActiveBudgets(ctx)
	if err != nil {
		return nil, err
	}

	for _, b := range budgets {
		if b.CategoryID != nil && *b.CategoryID == categoryID {
			return b, nil
		}
	}

	return nil, errors.New("budget not found for category")
}

func (s *TransactionServiceImpl) adjustBudgetsForUpdate(
	ctx context.Context,
	originalAmount float64,
	originalType transaction.Type,
	originalCategoryID uuid.UUID,
	newTransaction *transaction.Transaction,
) error {
	// Reverse original budget impact if it was an expense
	if originalType == transaction.TypeExpense {
		if err := s.updateBudgetSpent(ctx, originalCategoryID, -originalAmount); err != nil {
			return err
		}
	}

	// Apply new budget impact if it's an expense
	if newTransaction.Type == transaction.TypeExpense {
		if err := s.updateBudgetSpent(ctx, newTransaction.CategoryID, newTransaction.Amount); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionServiceImpl) convertDTOFilterToRepoFilter(filter dto.TransactionFilterDTO) transaction.Filter {
	repoFilter := transaction.Filter{
		UserID:     filter.UserID,
		CategoryID: filter.CategoryID,
		DateFrom:   filter.DateFrom,
		DateTo:     filter.DateTo,
		AmountFrom: filter.AmountFrom,
		AmountTo:   filter.AmountTo,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	if filter.Type != nil {
		repoFilter.Type = filter.Type
	}

	if filter.Description != nil {
		repoFilter.Description = *filter.Description
	}

	return repoFilter
}

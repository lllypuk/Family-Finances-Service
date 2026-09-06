package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
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
	ErrTransactionAmountTooLarge = errors.New("transaction amount exceeds the maximum")
	ErrTransactionDateOutOfRange = transaction.ErrDateOutOfRange
)

// validateTransactionBounds — границы суммы и даты; здесь они нужны, чтобы клиент
// получил 422, а не 500 из слоя данных.
func validateTransactionBounds(amount money.Minor, on date.Date) error {
	if amount > money.MaxAmount {
		return fmt.Errorf("%w: %d", ErrTransactionAmountTooLarge, amount)
	}

	return transaction.ValidateDate(on)
}

// TransactionRepository defines the data access operations for transactions
type TransactionRepository interface {
	Create(ctx context.Context, transaction *transaction.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
	GetByFilter(ctx context.Context, filter transaction.Filter) ([]*transaction.Transaction, error)
	CountByFilter(ctx context.Context, filter transaction.Filter) (int, error)
	Update(ctx context.Context, transaction *transaction.Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteBulk удаляет переданные id одним запросом и возвращает число удалённых строк;
	// отсутствующие id молча пропускаются.
	DeleteBulk(ctx context.Context, ids []uuid.UUID) (int, error)
	GetTotalByCategory(
		ctx context.Context,
		categoryID uuid.UUID,
		transactionType transaction.Type,
	) (money.Minor, error)
	GetTotalByDateRange(
		ctx context.Context,
		startDate, endDate date.Date,
		transactionType transaction.Type,
	) (money.Minor, error)
	GetTotalByCategoryAndDateRange(
		ctx context.Context,
		categoryID uuid.UUID,
		startDate, endDate date.Date,
		transactionType transaction.Type,
	) (money.Minor, error)
	// Note: UpdateBulkCategory may need to be implemented in the repository
	// For now, we'll use individual updates in a transaction
}

// TransactionRepositoryAtomicCreateWithBudget supports atomic creation of a transaction and budget update.
// Implementations may no-op budget update when no active budget exists for the transaction category.
type TransactionRepositoryAtomicCreateWithBudget interface {
	CreateWithBudgetUpdate(ctx context.Context, tx *transaction.Transaction) error
}

// BudgetRepositoryForTransactions defines the budget operations needed for transaction service
type BudgetRepositoryForTransactions interface {
	GetActiveBudgets(ctx context.Context, on date.Date) ([]*budget.Budget, error)
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
	logger          *slog.Logger
}

// NewTransactionService creates a new TransactionService instance
func NewTransactionService(
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	categoryRepo CategoryRepositoryForTransactions,
	userRepo UserRepositoryForTransactions,
) *TransactionServiceImpl {
	return NewTransactionServiceWithLogger(transactionRepo, budgetRepo, categoryRepo, userRepo, nil)
}

// NewTransactionServiceWithLogger creates a new TransactionService instance with injected logger.
func NewTransactionServiceWithLogger(
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
	categoryRepo CategoryRepositoryForTransactions,
	userRepo UserRepositoryForTransactions,
	logger *slog.Logger,
) *TransactionServiceImpl {
	if logger == nil {
		logger = slog.Default()
	}

	return &TransactionServiceImpl{
		transactionRepo: transactionRepo,
		budgetRepo:      budgetRepo,
		categoryRepo:    categoryRepo,
		userRepo:        userRepo,
		validator:       newValidator(),
		logger:          logger,
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

	if err := validateTransactionBounds(req.AmountMinor, req.Date); err != nil {
		return nil, err
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
		if err := s.ValidateTransactionLimits(ctx, req.CategoryID, req.AmountMinor, req.Type, req.Date); err != nil {
			return nil, err
		}
	}

	// Create transaction
	newTransaction := &transaction.Transaction{
		ID:          dto.EntityID(req.ID),
		AmountMinor: req.AmountMinor,
		Type:        req.Type,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		UserID:      req.UserID,
		Date:        req.Date,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Prefer an atomic create+budget update path when supported by the repository.
	if req.Type == transaction.TypeExpense {
		if atomicRepo, ok := s.transactionRepo.(TransactionRepositoryAtomicCreateWithBudget); ok {
			if err := atomicRepo.CreateWithBudgetUpdate(ctx, newTransaction); err != nil {
				return nil, fmt.Errorf("failed to create transaction atomically: %w", err)
			}
			return newTransaction, nil
		}
	}

	if err := s.transactionRepo.Create(ctx, newTransaction); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Fallback non-atomic budget update path for repositories without atomic support.
	// Return error if budget sync fails to avoid reporting success with inconsistent state.
	if req.Type == transaction.TypeExpense {
		if budgetErr := s.updateBudgetSpent(ctx, req.CategoryID, req.AmountMinor, req.Date); budgetErr != nil {
			return nil, fmt.Errorf("failed to update budget after transaction create: %w", budgetErr)
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

// GetAllTransactions retrieves transactions with filtering (single family model)
func (s *TransactionServiceImpl) GetAllTransactions(
	ctx context.Context,
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

	// Convert DTO filter to domain filter
	repoFilter := s.convertDTOFilterToRepoFilter(filter)

	transactions, err := s.transactionRepo.GetByFilter(ctx, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

// CountTransactions returns the total number of transactions matching the filter,
// ignoring Limit/Offset. Used by the UI to build pagination.
func (s *TransactionServiceImpl) CountTransactions(
	ctx context.Context,
	filter dto.TransactionFilterDTO,
) (int, error) {
	if err := s.validator.Struct(filter); err != nil {
		return 0, fmt.Errorf("validation failed: %w", err)
	}

	if err := filter.ValidateDateRange(); err != nil {
		return 0, err
	}
	if err := filter.ValidateAmountRange(); err != nil {
		return 0, err
	}

	repoFilter := s.convertDTOFilterToRepoFilter(filter)
	repoFilter.Limit = 0
	repoFilter.Offset = 0

	total, err := s.transactionRepo.CountByFilter(ctx, repoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}

	return total, nil
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
	originalAmount := existingTx.AmountMinor
	originalType := existingTx.Type
	originalCategoryID := existingTx.CategoryID
	originalDate := existingTx.Date

	// Update fields if provided
	if req.AmountMinor != nil {
		existingTx.AmountMinor = *req.AmountMinor
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

	if err = validateTransactionBounds(existingTx.AmountMinor, existingTx.Date); err != nil {
		return nil, err
	}

	if limitErr := s.validateUpdatedTransactionLimits(
		ctx, existingTx, originalAmount, originalType, originalCategoryID, originalDate,
	); limitErr != nil {
		return nil, limitErr
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
		originalDate,
		existingTx,
	); budgetErr != nil {
		s.warnBudgetAdjustment(ctx, "failed to adjust budgets after transaction update",
			slog.String("transaction_id", existingTx.ID.String()),
			slog.String("error", budgetErr.Error()),
		)
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
			-existingTx.AmountMinor,
			existingTx.Date,
		); budgetErr != nil {
			s.warnBudgetAdjustment(ctx, "failed to reverse budget spent after transaction delete",
				slog.String("transaction_id", existingTx.ID.String()),
				slog.String("category_id", existingTx.CategoryID.String()),
				slog.String("error", budgetErr.Error()),
			)
		}
	}

	return nil
}

// BulkDelete удаляет транзакции одним запросом и возвращает число фактически удалённых:
// неизвестные id пропускаются молча. Бюджеты корректируются после удаления, по снимку,
// снятому до него, — иначе списанные суммы уже не восстановить.
func (s *TransactionServiceImpl) BulkDelete(ctx context.Context, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// Дубликат в запросе удалил бы строку один раз, но откатил бюджет дважды.
	unique := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	expenses := make([]*transaction.Transaction, 0, len(unique))
	for _, id := range unique {
		existingTx, err := s.transactionRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if existingTx.Type == transaction.TypeExpense {
			expenses = append(expenses, existingTx)
		}
	}

	deleted, err := s.transactionRepo.DeleteBulk(ctx, unique)
	if err != nil {
		return 0, fmt.Errorf("failed to delete transactions: %w", err)
	}

	for _, tx := range expenses {
		if budgetErr := s.updateBudgetSpent(ctx, tx.CategoryID, -tx.AmountMinor, tx.Date); budgetErr != nil {
			s.warnBudgetAdjustment(ctx, "failed to reverse budget spent after bulk delete",
				slog.String("transaction_id", tx.ID.String()),
				slog.String("category_id", tx.CategoryID.String()),
				slog.String("error", budgetErr.Error()),
			)
		}
	}

	return deleted, nil
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
	from, to date.Date,
) ([]*transaction.Transaction, error) {
	if to.Before(from) {
		return nil, dto.ErrInvalidDateRange
	}

	filter := dto.TransactionFilterDTO{
		DateFrom: &from,
		DateTo:   &to,
		Limit:    DateRangeQueryLimit,
		Offset:   0,
	}

	return s.GetAllTransactions(ctx, filter)
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
			s.warnBudgetAdjustment(ctx, "failed to update transaction category in bulk operation",
				slog.String("transaction_id", txID.String()),
				slog.String("category_id", categoryID.String()),
				slog.String("error", err.Error()),
			)
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
		s.adjustBudgetsForCategoryChange(ctx, originalCategoryID, newCategoryID, tx.AmountMinor, tx.Date)
	}

	return nil
}

func (s *TransactionServiceImpl) adjustBudgetsForCategoryChange(
	ctx context.Context,
	oldCategoryID, newCategoryID uuid.UUID,
	amount money.Minor,
	on date.Date,
) {
	// Remove from old category budget
	if budgetErr := s.updateBudgetSpent(ctx, oldCategoryID, -amount, on); budgetErr != nil {
		s.warnBudgetAdjustment(ctx, "failed to decrease old category budget after recategorization",
			slog.String("old_category_id", oldCategoryID.String()),
			slog.Int64("amount_minor", int64(amount)),
			slog.String("error", budgetErr.Error()),
		)
	}

	// Add to new category budget
	if budgetErr := s.updateBudgetSpent(ctx, newCategoryID, amount, on); budgetErr != nil {
		s.warnBudgetAdjustment(ctx, "failed to increase new category budget after recategorization",
			slog.String("new_category_id", newCategoryID.String()),
			slog.Int64("amount_minor", int64(amount)),
			slog.String("error", budgetErr.Error()),
		)
	}
}

func (s *TransactionServiceImpl) warnBudgetAdjustment(ctx context.Context, msg string, attrs ...slog.Attr) {
	if s.logger == nil {
		return
	}

	s.logger.WarnContext(ctx, msg, attrsToAny(attrs)...)
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		out = append(out, attr)
	}
	return out
}

// ValidateTransactionLimits checks if a transaction would exceed budget limits
func (s *TransactionServiceImpl) ValidateTransactionLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount money.Minor,
	transactionType transaction.Type,
	on date.Date,
) error {
	// Only check limits for expense transactions
	if transactionType != transaction.TypeExpense {
		return nil
	}

	// Get active budget for the category
	budget, err := s.findBudgetByCategory(ctx, categoryID, on)
	if err != nil {
		// No budget means no limit - allow the transaction
		return nil //nolint:nilerr // No budget found is acceptable, not an error condition
	}

	// Check if adding this transaction would exceed the budget limit
	if budget.SpentMinor+amount > budget.AmountMinor {
		return fmt.Errorf("%w: budget amount %d, current spent %d, transaction amount %d",
			ErrInsufficientBudget, budget.AmountMinor, budget.SpentMinor, amount)
	}

	return nil
}

// validateUpdatedTransactionLimits — проверка лимита при правке операции. Сохранённая
// версия уже сидит в budget.SpentMinor, поэтому её вклад вычитается: иначе операция
// переставала бы редактироваться, как только съедала половину бюджета.
func (s *TransactionServiceImpl) validateUpdatedTransactionLimits(
	ctx context.Context,
	updated *transaction.Transaction,
	originalAmount money.Minor,
	originalType transaction.Type,
	originalCategoryID uuid.UUID,
	originalDate date.Date,
) error {
	if updated.Type != transaction.TypeExpense {
		return nil
	}

	target, err := s.findBudgetByCategory(ctx, updated.CategoryID, updated.Date)
	if err != nil {
		// Нет бюджета — нет лимита.
		return nil //nolint:nilerr // No budget found is acceptable, not an error condition
	}

	spent := target.SpentMinor
	if originalType == transaction.TypeExpense {
		original, origErr := s.findBudgetByCategory(ctx, originalCategoryID, originalDate)
		if origErr == nil && original.ID == target.ID {
			spent -= originalAmount
		}
	}

	if spent+updated.AmountMinor > target.AmountMinor {
		return fmt.Errorf("%w: budget amount %d, current spent %d, transaction amount %d",
			ErrInsufficientBudget, target.AmountMinor, spent, updated.AmountMinor)
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
	amount money.Minor,
	on date.Date,
) error {
	budget, err := s.findBudgetByCategory(ctx, categoryID, on)
	if err != nil {
		// No budget found - this is acceptable, not all categories need budgets
		return nil //nolint:nilerr // No budget found is acceptable, not an error condition
	}

	budget.SpentMinor += amount
	budget.UpdatedAt = time.Now()

	return s.budgetRepo.Update(ctx, budget)
}

func (s *TransactionServiceImpl) findBudgetByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	on date.Date,
) (*budget.Budget, error) {
	// Бюджет выбирается по дате самой операции, а не по «сегодня»: иначе правка
	// прошлого месяца попала бы в текущий бюджет.
	budgets, err := s.budgetRepo.GetActiveBudgets(ctx, on)
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
	originalAmount money.Minor,
	originalType transaction.Type,
	originalCategoryID uuid.UUID,
	originalDate date.Date,
	newTransaction *transaction.Transaction,
) error {
	// Reverse original budget impact if it was an expense
	if originalType == transaction.TypeExpense {
		if err := s.updateBudgetSpent(ctx, originalCategoryID, -originalAmount, originalDate); err != nil {
			return err
		}
	}

	// Apply new budget impact if it's an expense
	if newTransaction.Type == transaction.TypeExpense {
		if err := s.updateBudgetSpent(
			ctx,
			newTransaction.CategoryID,
			newTransaction.AmountMinor,
			newTransaction.Date,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransactionServiceImpl) convertDTOFilterToRepoFilter(filter dto.TransactionFilterDTO) transaction.Filter {
	repoFilter := transaction.Filter{
		UserID:     filter.UserID,
		CategoryID: filter.CategoryID,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	}

	repoFilter.DateFrom = filter.DateFrom
	repoFilter.DateTo = filter.DateTo
	repoFilter.AmountFromMinor = filter.AmountFromMinor
	repoFilter.AmountToMinor = filter.AmountToMinor

	if filter.Type != nil {
		repoFilter.Type = filter.Type
	}

	if filter.Description != nil {
		repoFilter.Description = *filter.Description
	}

	return repoFilter
}

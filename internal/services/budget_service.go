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
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services/dto"
)

var (
	ErrBudgetNotFoundService   = errors.New("budget not found")
	ErrBudgetAmountInvalid     = errors.New("budget amount must be greater than 0")
	ErrBudgetPeriodInvalid     = errors.New("budget end date must be after start date")
	ErrBudgetOverlapExists     = errors.New("budget period overlaps with existing budget")
	ErrBudgetAlreadyExceeded   = errors.New("cannot update budget: amount is less than already spent")
	ErrBudgetCalculationFailed = errors.New("failed to calculate budget metrics")
	ErrInsufficientBudgetFunds = errors.New("insufficient budget funds")
)

// Repository interfaces needed for BudgetService

type BudgetRepository interface {
	Create(ctx context.Context, budget *budget.Budget) error
	GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
	GetAll(ctx context.Context) ([]*budget.Budget, error)
	GetActiveBudgets(ctx context.Context) ([]*budget.Budget, error)
	Update(ctx context.Context, budget *budget.Budget) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByCategory(ctx context.Context, categoryID *uuid.UUID) ([]*budget.Budget, error)
	GetByPeriod(ctx context.Context, startDate, endDate time.Time) ([]*budget.Budget, error)
}

type TransactionRepositoryForBudgets interface {
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
}

// BudgetServiceImpl implements the BudgetService interface
type BudgetServiceImpl struct {
	budgetRepo      BudgetRepository
	transactionRepo TransactionRepositoryForBudgets
	validator       *validator.Validate
	logger          *slog.Logger
}

// NewBudgetService creates a new BudgetService instance
func NewBudgetService(
	budgetRepo BudgetRepository,
	transactionRepo TransactionRepositoryForBudgets,
) *BudgetServiceImpl {
	return NewBudgetServiceWithLogger(budgetRepo, transactionRepo, nil)
}

// NewBudgetServiceWithLogger creates a new BudgetService instance with injected logger.
func NewBudgetServiceWithLogger(
	budgetRepo BudgetRepository,
	transactionRepo TransactionRepositoryForBudgets,
	logger *slog.Logger,
) *BudgetServiceImpl {
	if logger == nil {
		logger = slog.Default()
	}

	return &BudgetServiceImpl{
		budgetRepo:      budgetRepo,
		transactionRepo: transactionRepo,
		validator:       validator.New(),
		logger:          logger,
	}
}

// CreateBudget creates a new budget with validation and business logic
func (s *BudgetServiceImpl) CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate budget period
	if err := req.ValidatePeriod(); err != nil {
		return nil, err
	}

	// Check for overlapping budgets in the same category
	if err := s.ValidateBudgetPeriod(ctx, req.CategoryID, req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	// Create new budget
	newBudget := &budget.Budget{
		ID:         uuid.New(),
		Name:       req.Name,
		Amount:     req.Amount,
		Spent:      0.0, // Always starts with 0
		Period:     req.Period,
		CategoryID: req.CategoryID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.budgetRepo.Create(ctx, newBudget); err != nil {
		return nil, fmt.Errorf("failed to create budget: %w", err)
	}

	// Recalculate spent amount based on existing transactions
	if recalcErr := s.RecalculateBudgetSpent(ctx, newBudget.ID); recalcErr != nil {
		s.logRecalculationWarning(ctx, "create_budget", newBudget.ID, recalcErr)
	}

	return newBudget, nil
}

// GetBudgetByID retrieves a budget by its ID with calculated spent amount
func (s *BudgetServiceImpl) GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	budget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrBudgetNotFoundService
	}

	// Recalculate spent amount from actual transactions
	if recalcErr := s.recalculateAndUpdateSpent(ctx, budget); recalcErr != nil {
		s.logRecalculationWarning(ctx, "get_budget_by_id", budget.ID, recalcErr)
	}

	return budget, nil
}

// GetAllBudgets retrieves budgets with optional filtering
func (s *BudgetServiceImpl) GetAllBudgets(
	ctx context.Context,
	filter dto.BudgetFilterDTO,
) ([]*budget.Budget, error) {
	if err := s.validator.Struct(filter); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate filter ranges
	if err := filter.ValidateDateRange(); err != nil {
		return nil, err
	}
	if err := filter.ValidateAmountRange(); err != nil {
		return nil, err
	}

	// Get budgets based on filter criteria
	budgets, err := s.getBudgetsWithFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets: %w", err)
	}

	// Recalculate spent amounts for all budgets
	for _, b := range budgets {
		if recalcErr := s.recalculateAndUpdateSpent(ctx, b); recalcErr != nil {
			s.logRecalculationWarning(ctx, "get_all_budgets", b.ID, recalcErr)
		}
	}

	return s.applyBudgetFilters(budgets, filter), nil
}

// UpdateBudget updates an existing budget
func (s *BudgetServiceImpl) UpdateBudget(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateBudgetDTO,
) (*budget.Budget, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate period if both dates are being updated
	if err := req.ValidatePeriod(); err != nil {
		return nil, err
	}

	// Get existing budget
	existingBudget, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrBudgetNotFoundService
	}

	// Recalculate current spent amount
	if recalcErr := s.recalculateAndUpdateSpent(ctx, existingBudget); recalcErr != nil {
		s.logRecalculationWarning(ctx, "update_budget", existingBudget.ID, recalcErr)
	}

	// Store original values for validation
	originalStartDate := existingBudget.StartDate
	originalEndDate := existingBudget.EndDate

	// Update fields if provided
	if req.Name != nil {
		existingBudget.Name = *req.Name
	}
	if req.Amount != nil {
		// Validate that new amount is not less than already spent
		if *req.Amount < existingBudget.Spent {
			return nil, fmt.Errorf("%w: new amount %.2f is less than spent %.2f",
				ErrBudgetAlreadyExceeded, *req.Amount, existingBudget.Spent)
		}
		existingBudget.Amount = *req.Amount
	}
	if req.StartDate != nil {
		existingBudget.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		existingBudget.EndDate = *req.EndDate
	}
	if req.IsActive != nil {
		existingBudget.IsActive = *req.IsActive
	}
	existingBudget.UpdatedAt = time.Now()

	// If dates changed, validate period overlap
	if req.StartDate != nil || req.EndDate != nil {
		if existingBudget.StartDate != originalStartDate || existingBudget.EndDate != originalEndDate {
			if validateErr := s.validateBudgetPeriodForUpdate(ctx, existingBudget); validateErr != nil {
				return nil, validateErr
			}
		}
	}

	// Update budget
	if updateErr := s.budgetRepo.Update(ctx, existingBudget); updateErr != nil {
		return nil, fmt.Errorf("failed to update budget: %w", updateErr)
	}

	return existingBudget, nil
}

// DeleteBudget deletes a budget
func (s *BudgetServiceImpl) DeleteBudget(ctx context.Context, id uuid.UUID) error {
	// Verify budget exists
	_, err := s.budgetRepo.GetByID(ctx, id)
	if err != nil {
		return ErrBudgetNotFoundService
	}

	// Delete budget
	if deleteErr := s.budgetRepo.Delete(ctx, id); deleteErr != nil {
		return fmt.Errorf("failed to delete budget: %w", deleteErr)
	}

	return nil
}

// GetActiveBudgets retrieves active budgets on a specific date
func (s *BudgetServiceImpl) GetActiveBudgets(
	ctx context.Context,
	date time.Time,
) ([]*budget.Budget, error) {
	allBudgets, err := s.budgetRepo.GetActiveBudgets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}

	// Filter budgets active on the specified date
	var activeBudgets []*budget.Budget
	for _, b := range allBudgets {
		if s.isBudgetActiveOnDate(b, date) {
			// Recalculate spent amount
			if recalcErr := s.recalculateAndUpdateSpent(ctx, b); recalcErr != nil {
				s.logRecalculationWarning(ctx, "get_active_budgets", b.ID, recalcErr)
			}
			activeBudgets = append(activeBudgets, b)
		}
	}

	return activeBudgets, nil
}

// UpdateBudgetSpent updates the spent amount for a budget
func (s *BudgetServiceImpl) UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error {
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return ErrBudgetNotFoundService
	}

	budget.Spent += amount
	budget.UpdatedAt = time.Now()

	return s.budgetRepo.Update(ctx, budget)
}

// CheckBudgetLimits checks if a transaction would exceed budget limits
func (s *BudgetServiceImpl) CheckBudgetLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
) error {
	budgets, err := s.GetBudgetsByCategory(ctx, categoryID)
	if err != nil {
		// No budgets found is acceptable
		return nil //nolint:nilerr // No budgets found is acceptable
	}

	// Check each active budget for the category
	for _, b := range budgets {
		if !s.isBudgetActiveOnDate(b, time.Now()) {
			continue
		}

		// Recalculate spent amount to ensure accuracy
		if recalcErr := s.recalculateAndUpdateSpent(ctx, b); recalcErr != nil {
			s.logRecalculationWarning(ctx, "check_budget_limits", b.ID, recalcErr)
		}

		// Check if adding this amount would exceed the budget
		if b.Spent+amount > b.Amount {
			return fmt.Errorf("%w: budget '%s' limit %.2f, current spent %.2f, transaction amount %.2f",
				ErrInsufficientBudgetFunds, b.Name, b.Amount, b.Spent, amount)
		}
	}

	return nil
}

// GetBudgetStatus returns detailed status information for a budget
func (s *BudgetServiceImpl) GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error) {
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return nil, ErrBudgetNotFoundService
	}

	// Recalculate spent amount
	if recalcErr := s.recalculateAndUpdateSpent(ctx, budget); recalcErr != nil {
		s.logRecalculationWarning(ctx, "get_budget_status", budget.ID, recalcErr)
	}

	return s.calculateBudgetStatus(budget), nil
}

// CalculateBudgetUtilization calculates budget utilization analytics
func (s *BudgetServiceImpl) CalculateBudgetUtilization(
	ctx context.Context,
	budgetID uuid.UUID,
) (*dto.BudgetUtilizationDTO, error) {
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return nil, ErrBudgetNotFoundService
	}

	// Recalculate spent amount
	if recalcErr := s.recalculateAndUpdateSpent(ctx, budget); recalcErr != nil {
		s.logRecalculationWarning(ctx, "calculate_budget_utilization", budget.ID, recalcErr)
	}

	return s.calculateBudgetUtilization(budget), nil
}

// GetBudgetsByCategory retrieves budgets for a specific category
func (s *BudgetServiceImpl) GetBudgetsByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]*budget.Budget, error) {
	budgets, err := s.budgetRepo.GetByCategory(ctx, &categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by category: %w", err)
	}

	// Recalculate spent amounts
	for _, b := range budgets {
		if recalcErr := s.recalculateAndUpdateSpent(ctx, b); recalcErr != nil {
			s.logRecalculationWarning(ctx, "get_budgets_by_category", b.ID, recalcErr)
		}
	}

	return budgets, nil
}

// ValidateBudgetPeriod validates that budget period doesn't overlap with existing budgets
func (s *BudgetServiceImpl) ValidateBudgetPeriod(
	ctx context.Context,
	categoryID *uuid.UUID,
	startDate, endDate time.Time,
) error {
	existingBudgets, err := s.budgetRepo.GetByPeriod(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to validate budget period: %w", err)
	}

	for _, existing := range existingBudgets {
		if s.budgetPeriodsOverlap(existing, categoryID, startDate, endDate) {
			return fmt.Errorf("%w: overlaps with budget '%s'", ErrBudgetOverlapExists, existing.Name)
		}
	}

	return nil
}

// RecalculateBudgetSpent recalculates and updates the spent amount for a budget
func (s *BudgetServiceImpl) RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error {
	budget, err := s.budgetRepo.GetByID(ctx, budgetID)
	if err != nil {
		return ErrBudgetNotFoundService
	}

	return s.recalculateAndUpdateSpent(ctx, budget)
}

// Helper methods

func (s *BudgetServiceImpl) getBudgetsWithFilter(
	ctx context.Context,
	filter dto.BudgetFilterDTO,
) ([]*budget.Budget, error) {
	if filter.ActiveOn != nil {
		return s.GetActiveBudgets(ctx, *filter.ActiveOn)
	}

	if filter.CategoryID != nil {
		return s.GetBudgetsByCategory(ctx, *filter.CategoryID)
	}

	// Default: get all budgets for single family
	return s.budgetRepo.GetAll(ctx)
}

func (s *BudgetServiceImpl) applyBudgetFilters(budgets []*budget.Budget, filter dto.BudgetFilterDTO) []*budget.Budget {
	var filtered []*budget.Budget

	for _, b := range budgets {
		if s.budgetMatchesFilter(b, filter) {
			filtered = append(filtered, b)
		}
	}

	// Apply pagination
	start := filter.Offset
	end := start + filter.Limit

	if start >= len(filtered) {
		return []*budget.Budget{}
	}

	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end]
}

func (s *BudgetServiceImpl) budgetMatchesFilter(b *budget.Budget, filter dto.BudgetFilterDTO) bool {
	// Period filter
	if filter.Period != nil && b.Period != *filter.Period {
		return false
	}

	// Active filter
	if filter.IsActive != nil && b.IsActive != *filter.IsActive {
		return false
	}

	// Date range filters
	if filter.DateFrom != nil && b.EndDate.Before(*filter.DateFrom) {
		return false
	}
	if filter.DateTo != nil && b.StartDate.After(*filter.DateTo) {
		return false
	}

	// Amount filters
	if filter.AmountFrom != nil && b.Amount < *filter.AmountFrom {
		return false
	}
	if filter.AmountTo != nil && b.Amount > *filter.AmountTo {
		return false
	}

	// Status filters
	utilizationPercent := dto.CalculateUtilizationPercent(b.Spent, b.Amount)

	if filter.IsOverBudget != nil && *filter.IsOverBudget != (utilizationPercent >= dto.BudgetAlertOverBudget) {
		return false
	}
	if filter.IsNearLimit != nil && *filter.IsNearLimit != (utilizationPercent >= dto.BudgetAlertNearLimit) {
		return false
	}
	if filter.HasUnspentFunds != nil && *filter.HasUnspentFunds != (b.Amount > b.Spent) {
		return false
	}

	return true
}

func (s *BudgetServiceImpl) recalculateAndUpdateSpent(ctx context.Context, b *budget.Budget) error {
	var spent float64
	var err error

	if b.CategoryID != nil {
		// Calculate spent for specific category within budget period
		spent, err = s.transactionRepo.GetTotalByCategoryAndDateRange(
			ctx, *b.CategoryID, b.StartDate, b.EndDate, transaction.TypeExpense)
	} else {
		// Calculate spent for entire family within budget period
		spent, err = s.transactionRepo.GetTotalByDateRange(
			ctx, b.StartDate, b.EndDate, transaction.TypeExpense)
	}

	if err != nil {
		return fmt.Errorf("failed to recalculate spent amount: %w", err)
	}

	if b.Spent != spent {
		b.Spent = spent
		b.UpdatedAt = time.Now()
		return s.budgetRepo.Update(ctx, b)
	}

	return nil
}

func (s *BudgetServiceImpl) isBudgetActiveOnDate(b *budget.Budget, date time.Time) bool {
	return b.IsActive &&
		!date.Before(b.StartDate) &&
		!date.After(b.EndDate)
}

func (s *BudgetServiceImpl) budgetPeriodsOverlap(
	existing *budget.Budget,
	categoryID *uuid.UUID,
	startDate, endDate time.Time,
) bool {
	// Only check overlap if it's the same category (or both are family-wide)
	if !s.sameBudgetScope(existing.CategoryID, categoryID) {
		return false
	}

	// Check if periods overlap
	return endDate.After(existing.StartDate) && startDate.Before(existing.EndDate)
}

func (s *BudgetServiceImpl) sameBudgetScope(existingCategoryID, newCategoryID *uuid.UUID) bool {
	// Both are family-wide budgets
	if existingCategoryID == nil && newCategoryID == nil {
		return true
	}

	// One is family-wide, one is category-specific
	if existingCategoryID == nil || newCategoryID == nil {
		return false
	}

	// Both are category-specific - check if same category
	return *existingCategoryID == *newCategoryID
}

func (s *BudgetServiceImpl) validateBudgetPeriodForUpdate(ctx context.Context, budget *budget.Budget) error {
	existingBudgets, err := s.budgetRepo.GetByPeriod(ctx, budget.StartDate, budget.EndDate)
	if err != nil {
		return fmt.Errorf("failed to validate budget period: %w", err)
	}

	for _, existing := range existingBudgets {
		// Skip checking against itself
		if existing.ID == budget.ID {
			continue
		}

		if s.budgetPeriodsOverlap(existing, budget.CategoryID, budget.StartDate, budget.EndDate) {
			return fmt.Errorf("%w: overlaps with budget '%s'", ErrBudgetOverlapExists, existing.Name)
		}
	}

	return nil
}

func (s *BudgetServiceImpl) calculateBudgetStatus(b *budget.Budget) *dto.BudgetStatusDTO {
	utilizationPercent := dto.CalculateUtilizationPercent(b.Spent, b.Amount)
	daysTotal := int(b.EndDate.Sub(b.StartDate).Hours() / dto.HoursPerDay)
	daysElapsed := int(time.Since(b.StartDate).Hours() / dto.HoursPerDay)
	daysRemaining := dto.CalculateDaysRemaining(b.EndDate)

	status := &dto.BudgetStatusDTO{
		BudgetID:           b.ID,
		Name:               b.Name,
		TotalAmount:        b.Amount,
		SpentAmount:        b.Spent,
		RemainingAmount:    b.Amount - b.Spent,
		UtilizationPercent: utilizationPercent,
		DaysTotal:          daysTotal,
		DaysElapsed:        daysElapsed,
		DaysRemaining:      daysRemaining,
		IsOverBudget:       utilizationPercent >= dto.BudgetAlertOverBudget,
		IsNearLimit:        utilizationPercent >= dto.BudgetAlertNearLimit,
		IsCriticalLimit:    utilizationPercent >= dto.BudgetAlertCritical,
		Status:             dto.DetermineBudgetStatus(utilizationPercent),
	}

	// Calculate daily metrics
	if daysTotal > 0 {
		status.DailyBudget = b.Amount / float64(daysTotal)
	}
	if daysElapsed > 0 {
		status.DailySpent = b.Spent / float64(daysElapsed)
	}

	// Calculate projected overrun
	if status.DailySpent > 0 && daysRemaining > 0 {
		projectedTotal := b.Spent + (status.DailySpent * float64(daysRemaining))
		if projectedTotal > b.Amount {
			status.ProjectedOverrun = projectedTotal - b.Amount
		}
	}

	return status
}

func (s *BudgetServiceImpl) calculateBudgetUtilization(b *budget.Budget) *dto.BudgetUtilizationDTO {
	utilizationPercent := dto.CalculateUtilizationPercent(b.Spent, b.Amount)
	daysElapsed := int(time.Since(b.StartDate).Hours() / dto.HoursPerDay)

	utilization := &dto.BudgetUtilizationDTO{
		BudgetID:           b.ID,
		Period:             string(b.Period),
		UtilizationPercent: utilizationPercent,
		Recommendations:    []string{},
	}

	// Calculate spending velocity
	if daysElapsed > 0 {
		utilization.SpendingVelocity = b.Spent / float64(daysElapsed)
	}

	// Calculate projected completion
	if utilization.SpendingVelocity > 0 {
		daysToCompletion := (b.Amount - b.Spent) / utilization.SpendingVelocity
		if daysToCompletion > 0 {
			completionDate := time.Now().Add(time.Duration(daysToCompletion) * 24 * time.Hour)
			utilization.ProjectedCompletion = &completionDate
		}
	}

	// Generate recommendations
	utilization.Recommendations = s.generateBudgetRecommendations(b, utilizationPercent)

	return utilization
}

func (s *BudgetServiceImpl) generateBudgetRecommendations(b *budget.Budget, utilizationPercent float64) []string {
	var recommendations []string

	switch {
	case utilizationPercent >= dto.BudgetAlertOverBudget:
		recommendations = append(recommendations, "Budget exceeded! Review and reduce spending immediately.")
		recommendations = append(recommendations, "Consider increasing budget amount if necessary.")
	case utilizationPercent >= dto.BudgetAlertCritical:
		recommendations = append(recommendations, "Critical budget level reached. Monitor spending closely.")
		recommendations = append(recommendations, "Consider adjusting spending plans for remainder of period.")
	case utilizationPercent >= dto.BudgetAlertNearLimit:
		recommendations = append(recommendations, "Approaching budget limit. Review upcoming expenses.")
		recommendations = append(recommendations, "Consider prioritizing essential expenses only.")
	default:
		recommendations = append(recommendations, "Budget is healthy. Continue current spending patterns.")
	}

	// Time-based recommendations
	daysRemaining := dto.CalculateDaysRemaining(b.EndDate)
	if daysRemaining <= 7 && utilizationPercent < 50 {
		recommendations = append(
			recommendations,
			"Significant budget remaining with little time left. Consider planned expenses.",
		)
	}

	return recommendations
}

func (s *BudgetServiceImpl) logRecalculationWarning(
	ctx context.Context,
	operation string,
	budgetID uuid.UUID,
	err error,
) {
	if s.logger == nil {
		return
	}

	s.logger.WarnContext(ctx, "budget spent recalculation failed",
		slog.String("operation", operation),
		slog.String("budget_id", budgetID.String()),
		slog.String("error", err.Error()),
	)
}

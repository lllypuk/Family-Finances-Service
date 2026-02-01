package services_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

// Common Mock Repositories

// MockFamilyRepository is a shared mock implementation of FamilyRepository
type MockFamilyRepository struct {
	mock.Mock
}

func (m *MockFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) Get(ctx context.Context) (*user.Family, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	family, ok := args.Get(0).(*user.Family)
	if !ok {
		return nil, args.Error(1)
	}
	return family, args.Error(1)
}

func (m *MockFamilyRepository) Update(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) Exists(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetAll(ctx context.Context) ([]*user.User, error) {
	args := m.Called(ctx)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockBudgetRepository is a mock implementation of BudgetRepository
type MockBudgetRepository struct {
	mock.Mock
}

func (m *MockBudgetRepository) Create(ctx context.Context, budget *budget.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockBudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetAll(ctx context.Context) ([]*budget.Budget, error) {
	args := m.Called(ctx)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetActiveBudgets(ctx context.Context) ([]*budget.Budget, error) {
	args := m.Called(ctx)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) Update(ctx context.Context, budget *budget.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockBudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBudgetRepository) GetByCategory(
	ctx context.Context,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, categoryID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetByPeriod(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, startDate, endDate)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

// MockCategoryRepository is a mock implementation of CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *category.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetAll(ctx context.Context) ([]*category.Category, error) {
	args := m.Called(ctx)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByType(
	ctx context.Context,
	categoryType category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, categoryType)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *category.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockTransactionRepository is a mock implementation of TransactionRepository
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

// Updated: GetAll replaces GetByFamilyID (no familyID param)
func (m *MockTransactionRepository) GetAll(
	ctx context.Context,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// Updated: Delete no longer requires familyID
func (m *MockTransactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, txType)
	return args.Get(0).(float64), args.Error(1)
}

// Added: GetTotalByCategoryAndDateRange to match TransactionRepository interface
func (m *MockTransactionRepository) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
}

// Updated: GetTotalByDateRange replaces GetTotalByFamilyAndDateRange
func (m *MockTransactionRepository) GetTotalByDateRange(
	ctx context.Context,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
}

// MockReportRepository is a mock implementation of ReportRepository
type MockReportRepository struct {
	mock.Mock
}

func (m *MockReportRepository) Create(ctx context.Context, report *report.Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

func (m *MockReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

// Updated: GetAll replaces GetByFamilyID
func (m *MockReportRepository) GetAll(ctx context.Context) ([]*report.Report, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportRepository) Update(ctx context.Context, report *report.Report) error {
	args := m.Called(ctx, report)
	return args.Error(0)
}

// Updated: Delete no longer requires familyID
func (m *MockReportRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Common Mock Services

// MockTransactionService is a mock implementation of TransactionService
type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) CreateTransaction(
	ctx context.Context,
	req dto.CreateTransactionDTO,
) (*transaction.Transaction, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) GetTransactionByID(
	ctx context.Context,
	id uuid.UUID,
) (*transaction.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) GetAllTransactions(
	ctx context.Context,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) UpdateTransaction(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateTransactionDTO,
) (*transaction.Transaction, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionService) GetTransactionsByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, categoryID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) GetTransactionsByDateRange(
	ctx context.Context,
	from, to time.Time,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionService) BulkCategorizeTransactions(
	ctx context.Context,
	transactionIDs []uuid.UUID,
	categoryID uuid.UUID,
) error {
	args := m.Called(ctx, transactionIDs, categoryID)
	return args.Error(0)
}

func (m *MockTransactionService) ValidateTransactionLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
) error {
	args := m.Called(ctx, categoryID, amount, transactionType)
	return args.Error(0)
}

// MockBudgetService is a mock implementation of BudgetService
type MockBudgetService struct {
	mock.Mock
}

func (m *MockBudgetService) CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

func (m *MockBudgetService) GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

// Updated: GetAllBudgets replaces GetBudgetsByFamily (single-family model)
func (m *MockBudgetService) GetAllBudgets(ctx context.Context, filter dto.BudgetFilterDTO) ([]*budget.Budget, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetService) UpdateBudget(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateBudgetDTO,
) (*budget.Budget, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

// Updated: DeleteBudget no longer requires familyID
func (m *MockBudgetService) DeleteBudget(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Updated: GetActiveBudgets signature (no familyID)
func (m *MockBudgetService) GetActiveBudgets(
	ctx context.Context,
	date time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetService) UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error {
	args := m.Called(ctx, budgetID, amount)
	return args.Error(0)
}

// Updated: CheckBudgetLimits no longer takes familyID
func (m *MockBudgetService) CheckBudgetLimits(
	ctx context.Context,
	categoryID uuid.UUID,
	amount float64,
) error {
	args := m.Called(ctx, categoryID, amount)
	return args.Error(0)
}

// Updated: GetBudgetsByCategory no longer takes familyID
func (m *MockBudgetService) GetBudgetsByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

// Updated: ValidateBudgetPeriod signature (no familyID)
func (m *MockBudgetService) ValidateBudgetPeriod(
	ctx context.Context,
	categoryID *uuid.UUID,
	startDate, endDate time.Time,
) error {
	args := m.Called(ctx, categoryID, startDate, endDate)
	return args.Error(0)
}

func (m *MockBudgetService) RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error {
	args := m.Called(ctx, budgetID)
	return args.Error(0)
}

// Added: GetBudgetStatus to satisfy BudgetService interface
func (m *MockBudgetService) GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error) {
	args := m.Called(ctx, budgetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetStatusDTO), args.Error(1)
}

// Added: CalculateBudgetUtilization to satisfy BudgetService interface
func (m *MockBudgetService) CalculateBudgetUtilization(
	ctx context.Context,
	budgetID uuid.UUID,
) (*dto.BudgetUtilizationDTO, error) {
	args := m.Called(ctx, budgetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetUtilizationDTO), args.Error(1)
}

// MockCategoryService is a mock implementation of CategoryService
type MockCategoryService struct {
	mock.Mock
}

func (m *MockCategoryService) CreateCategory(
	ctx context.Context,
	req dto.CreateCategoryDTO,
) (*category.Category, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) GetCategories(
	ctx context.Context,
	typeFilter *category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryService) UpdateCategory(
	ctx context.Context,
	id uuid.UUID,
	req dto.UpdateCategoryDTO,
) (*category.Category, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryService) GetCategoryHierarchy(
	ctx context.Context,
) ([]*category.Category, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryService) ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error {
	args := m.Called(ctx, categoryID, parentID)
	return args.Error(0)
}

func (m *MockCategoryService) CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockCategoryService) CreateDefaultCategories(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Common Test Helper Functions

// createTestTransaction creates a test transaction with all required parameters
func createTestTransaction(
	id uuid.UUID,
	amount float64,
	transactionType transaction.Type,
	date time.Time,
) *transaction.Transaction {
	categoryID := uuid.New()
	return createTestTransactionWithCategory(id, categoryID, amount, transactionType, date)
}

// createTestTransactionWithCategory creates a test transaction with specific category
func createTestTransactionWithCategory(
	id uuid.UUID,
	categoryID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
	date time.Time,
) *transaction.Transaction {
	return &transaction.Transaction{
		ID:          id,
		Amount:      amount,
		Description: "Test transaction",
		Type:        transactionType,
		CategoryID:  categoryID,
		UserID:      uuid.New(),
		Date:        date,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// createTestCategory creates a test category with all required parameters
func createTestCategory(id uuid.UUID, name string, categoryType category.Type) *category.Category {
	return &category.Category{
		ID:        id,
		Name:      name,
		Type:      categoryType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createTestBudget creates a test budget with all required parameters
func createTestBudget(id uuid.UUID, amount float64, categoryID uuid.UUID) *budget.Budget {
	return &budget.Budget{
		ID:         id,
		Name:       "Test Budget",
		Amount:     amount,
		CategoryID: &categoryID,
		StartDate:  time.Now().AddDate(0, 0, -30),
		EndDate:    time.Now().AddDate(0, 0, 30),
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// createTestUser creates a test user
func createTestUser(_ uuid.UUID) *user.User {
	return &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Role:      user.RoleMember,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// setupReportService creates a properly configured report service for testing
func setupReportService() (
	services.ReportService,
	*MockReportRepository,
	*MockUserRepository,
	*MockTransactionService,
	*MockBudgetService,
	*MockCategoryService,
) {
	mockReportRepo := &MockReportRepository{}
	mockTransactionRepo := &MockTransactionRepository{}
	mockBudgetRepo := &MockBudgetRepository{}
	mockCategoryRepo := &MockCategoryRepository{}
	mockUserRepo := &MockUserRepository{}
	mockTransactionService := &MockTransactionService{}
	mockBudgetService := &MockBudgetService{}
	mockCategoryService := &MockCategoryService{}

	service := services.NewReportService(
		mockReportRepo,
		mockTransactionRepo,
		mockBudgetRepo,
		mockCategoryRepo,
		mockUserRepo,
		mockTransactionService,
		mockBudgetService,
		mockCategoryService,
	)

	return service, mockReportRepo, mockUserRepo, mockTransactionService, mockBudgetService, mockCategoryService
}

// setupTransactionService creates a properly configured transaction service for testing
func setupTransactionService() (
	services.TransactionService,
	*MockTransactionRepository,
	*MockBudgetRepository,
	*MockCategoryRepository,
	*MockUserRepository,
) {
	txRepo := &MockTransactionRepository{}
	budgetRepo := &MockBudgetRepository{}
	categoryRepo := &MockCategoryRepository{}
	userRepo := &MockUserRepository{}

	service := services.NewTransactionService(txRepo, budgetRepo, categoryRepo, userRepo)

	return service, txRepo, budgetRepo, categoryRepo, userRepo
}

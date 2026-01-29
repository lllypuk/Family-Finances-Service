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

func (m *MockUserRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *user.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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

func (m *MockBudgetRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) Update(ctx context.Context, budget *budget.Budget) error {
	args := m.Called(ctx, budget)
	return args.Error(0)
}

func (m *MockBudgetRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
	return args.Error(0)
}

func (m *MockBudgetRepository) GetByFamilyAndCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, categoryID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepository) GetByPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
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

func (m *MockCategoryRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	args := m.Called(ctx, familyID)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetByType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID, categoryType)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *category.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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

func (m *MockTransactionRepository) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *transaction.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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

func (m *MockTransactionRepository) GetTotalByFamilyAndDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, familyID, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepository) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	txType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, txType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepository) GetByDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) BulkUpdate(ctx context.Context, transactions []*transaction.Transaction) error {
	args := m.Called(ctx, transactions)
	return args.Error(0)
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

func (m *MockReportRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, familyID)
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

func (m *MockReportRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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

func (m *MockTransactionService) GetTransactionsByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, filter)
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

func (m *MockTransactionService) DeleteTransaction(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
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
	familyID uuid.UUID,
	from, to time.Time,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, from, to)
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
	familyID uuid.UUID,
	categoryID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
) error {
	args := m.Called(ctx, familyID, categoryID, amount, transactionType)
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

func (m *MockBudgetService) GetBudgetsByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	filter dto.BudgetFilterDTO,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, filter)
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

func (m *MockBudgetService) DeleteBudget(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
	return args.Error(0)
}

func (m *MockBudgetService) GetActiveBudgets(
	ctx context.Context,
	familyID uuid.UUID,
	date time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetService) UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error {
	args := m.Called(ctx, budgetID, amount)
	return args.Error(0)
}

func (m *MockBudgetService) CheckBudgetLimits(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID uuid.UUID,
	amount float64,
) error {
	args := m.Called(ctx, familyID, categoryID, amount)
	return args.Error(0)
}

func (m *MockBudgetService) GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error) {
	args := m.Called(ctx, budgetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetStatusDTO), args.Error(1)
}

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

func (m *MockBudgetService) GetBudgetsByCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetService) ValidateBudgetPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
	startDate, endDate time.Time,
) error {
	args := m.Called(ctx, familyID, categoryID, startDate, endDate)
	return args.Error(0)
}

func (m *MockBudgetService) RecalculateBudgetSpent(ctx context.Context, budgetID uuid.UUID) error {
	args := m.Called(ctx, budgetID)
	return args.Error(0)
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

func (m *MockCategoryService) GetCategoriesByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	typeFilter *category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID, typeFilter)
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

func (m *MockCategoryService) DeleteCategory(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	args := m.Called(ctx, id, familyID)
	return args.Error(0)
}

func (m *MockCategoryService) GetCategoryHierarchy(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID)
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

func (m *MockCategoryService) CreateDefaultCategories(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

// Common Test Helper Functions

// createTestTransaction creates a test transaction with all required parameters
func createTestTransaction(
	id uuid.UUID,
	familyID uuid.UUID,
	amount float64,
	transactionType transaction.Type,
	date time.Time,
) *transaction.Transaction {
	categoryID := uuid.New()
	return createTestTransactionWithCategory(id, familyID, categoryID, amount, transactionType, date)
}

// createTestTransactionWithCategory creates a test transaction with specific category
func createTestTransactionWithCategory(
	id uuid.UUID,
	familyID uuid.UUID,
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
		FamilyID:    familyID,
		Date:        date,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// createTestCategory creates a test category with all required parameters
func createTestCategory(id uuid.UUID, familyID uuid.UUID, name string, categoryType category.Type) *category.Category {
	return &category.Category{
		ID:        id,
		Name:      name,
		Type:      categoryType,
		FamilyID:  familyID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createTestBudget creates a test budget with all required parameters
func createTestBudget(id uuid.UUID, familyID uuid.UUID, amount float64, categoryID uuid.UUID) *budget.Budget {
	return &budget.Budget{
		ID:         id,
		Name:       "Test Budget",
		Amount:     amount,
		FamilyID:   familyID,
		CategoryID: &categoryID,
		StartDate:  time.Now().AddDate(0, 0, -30),
		EndDate:    time.Now().AddDate(0, 0, 30),
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// createTestUser creates a test user
func createTestUser(familyID uuid.UUID) *user.User {
	return &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com",
		FamilyID:  familyID,
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

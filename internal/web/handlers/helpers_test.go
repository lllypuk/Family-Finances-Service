package handlers_test

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
)

// MockUserRepositoryWeb is a mock implementation of user repository for web tests
type MockUserRepositoryWeb struct {
	mock.Mock
}

func (m *MockUserRepositoryWeb) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryWeb) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepositoryWeb) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepositoryWeb) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

// MockFamilyRepositoryWeb is a mock implementation of family repository for web tests
type MockFamilyRepositoryWeb struct {
	mock.Mock
}

func (m *MockFamilyRepositoryWeb) Create(ctx context.Context, f *user.Family) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *MockFamilyRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.Family), args.Error(1)
}

func (m *MockFamilyRepositoryWeb) Update(ctx context.Context, f *user.Family) error {
	args := m.Called(ctx, f)
	return args.Error(0)
}

func (m *MockFamilyRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockCategoryRepositoryWeb is a mock implementation of category repository for web tests
type MockCategoryRepositoryWeb struct {
	mock.Mock
}

func (m *MockCategoryRepositoryWeb) Create(ctx context.Context, c *category.Category) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCategoryRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func (m *MockCategoryRepositoryWeb) Update(ctx context.Context, c *category.Category) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCategoryRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepositoryWeb) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

func (m *MockCategoryRepositoryWeb) GetByType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.Type,
) ([]*category.Category, error) {
	args := m.Called(ctx, familyID, categoryType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*category.Category), args.Error(1)
}

// MockTransactionRepositoryWeb is a mock implementation of transaction repository for web tests
type MockTransactionRepositoryWeb struct {
	mock.Mock
}

func (m *MockTransactionRepositoryWeb) Create(ctx context.Context, t *transaction.Transaction) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTransactionRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) Update(ctx context.Context, t *transaction.Transaction) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTransactionRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTransactionRepositoryWeb) GetByFamilyID(
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

func (m *MockTransactionRepositoryWeb) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetByDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetByCategoryID(
	ctx context.Context,
	categoryID uuid.UUID,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, categoryID, startDate, endDate, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetTotalByFamilyAndDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	args := m.Called(ctx, familyID, startDate, endDate, transactionType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockTransactionRepositoryWeb) GetMonthlyTotals(
	ctx context.Context,
	familyID uuid.UUID,
	year int,
) (map[string]float64, error) {
	args := m.Called(ctx, familyID, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]float64), args.Error(1)
}

// MockBudgetRepositoryWeb is a mock implementation of budget repository for web tests
type MockBudgetRepositoryWeb struct {
	mock.Mock
}

func (m *MockBudgetRepositoryWeb) Create(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepositoryWeb) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryWeb) Update(ctx context.Context, b *budget.Budget) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBudgetRepositoryWeb) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBudgetRepositoryWeb) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryWeb) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryWeb) GetByFamilyAndCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryWeb) GetByCategoryID(ctx context.Context, categoryID uuid.UUID) ([]*budget.Budget, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func (m *MockBudgetRepositoryWeb) GetByPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

// Mock services for web tests
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(
	ctx context.Context,
	familyName, firstName, lastName, email, password string,
) (*user.User, *user.Family, error) {
	args := m.Called(ctx, familyName, firstName, lastName, email, password)
	var u *user.User
	var f *user.Family
	if args.Get(0) != nil {
		u = args.Get(0).(*user.User)
	}
	if args.Get(1) != nil {
		f = args.Get(1).(*user.Family)
	}
	return u, f, args.Error(2)
}

func (m *MockUserService) LoginUser(ctx context.Context, email, password string) (*user.User, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) GetFamilyMembers(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*user.User), args.Error(1)
}

func (m *MockUserService) AddFamilyMember(
	ctx context.Context,
	familyID uuid.UUID,
	firstName, lastName, email, password string,
	role user.Role,
) (*user.User, error) {
	args := m.Called(ctx, familyID, firstName, lastName, email, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

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

func (m *MockTransactionService) GetTransactionsByDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*transaction.Transaction), args.Error(1)
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

func (m *MockTransactionService) GetTransactionsByFilter(
	ctx context.Context,
	filter dto.TransactionFilterDTO,
) ([]*transaction.Transaction, error) {
	args := m.Called(ctx, filter)
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

func (m *MockBudgetService) DeleteBudget(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
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

type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) GetMonthlyReport(
	ctx context.Context,
	familyID uuid.UUID,
	month time.Time,
) (map[string]interface{}, error) {
	args := m.Called(ctx, familyID, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockReportService) GetYearlyReport(
	ctx context.Context,
	familyID uuid.UUID,
	year int,
) (map[string]interface{}, error) {
	args := m.Called(ctx, familyID, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockReportService) GetCategoryReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (map[string]interface{}, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockReportService) GetBudgetReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (map[string]interface{}, error) {
	args := m.Called(ctx, familyID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockReportService) ExportTransactions(
	ctx context.Context,
	familyID uuid.UUID,
	format string,
	startDate, endDate time.Time,
) ([]byte, error) {
	args := m.Called(ctx, familyID, format, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) GetReportsByFamily(
	ctx context.Context,
	familyID uuid.UUID,
	typeFilter *report.Type,
) ([]*report.Report, error) {
	args := m.Called(ctx, familyID, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportService) GetReportByID(ctx context.Context, reportID uuid.UUID) (*report.Report, error) {
	args := m.Called(ctx, reportID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) SaveReport(
	ctx context.Context,
	reportData any,
	reportType report.Type,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	args := m.Called(ctx, reportData, reportType, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) DeleteReport(ctx context.Context, reportID uuid.UUID) error {
	args := m.Called(ctx, reportID)
	return args.Error(0)
}

func (m *MockReportService) GenerateExpenseReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.ExpenseReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ExpenseReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateIncomeReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.IncomeReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.IncomeReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateBudgetComparisonReport(
	ctx context.Context,
	familyID uuid.UUID,
	period report.Period,
) (*dto.BudgetComparisonDTO, error) {
	args := m.Called(ctx, familyID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetComparisonDTO), args.Error(1)
}

func (m *MockReportService) GenerateCashFlowReport(
	ctx context.Context,
	familyID uuid.UUID,
	from, to time.Time,
) (*dto.CashFlowReportDTO, error) {
	args := m.Called(ctx, familyID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CashFlowReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	familyID uuid.UUID,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	args := m.Called(ctx, familyID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CategoryBreakdownDTO), args.Error(1)
}

func (m *MockReportService) CalculateBenchmarks(
	ctx context.Context,
	familyID uuid.UUID,
) (*dto.BenchmarkComparisonDTO, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BenchmarkComparisonDTO), args.Error(1)
}

// Additional ReportService interface methods
func (m *MockReportService) ExportReport(
	ctx context.Context,
	reportID uuid.UUID,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportID, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) ExportReportData(
	ctx context.Context,
	reportData any,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportData, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) ScheduleReport(
	ctx context.Context,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) GetScheduledReports(
	ctx context.Context,
	familyID uuid.UUID,
) ([]*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) UpdateScheduledReport(
	ctx context.Context,
	id uuid.UUID,
	req dto.ScheduleReportDTO,
) (*dto.ScheduledReportDTO, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ScheduledReportDTO), args.Error(1)
}

func (m *MockReportService) DeleteScheduledReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportService) ExecuteScheduledReport(ctx context.Context, scheduledReportID uuid.UUID) error {
	args := m.Called(ctx, scheduledReportID)
	return args.Error(0)
}

func (m *MockReportService) GenerateTrendAnalysis(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
	period report.Period,
) (*dto.TrendAnalysisDTO, error) {
	args := m.Called(ctx, familyID, categoryID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TrendAnalysisDTO), args.Error(1)
}

func (m *MockReportService) GenerateSpendingForecast(
	ctx context.Context,
	familyID uuid.UUID,
	months int,
) ([]dto.ForecastDTO, error) {
	args := m.Called(ctx, familyID, months)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.ForecastDTO), args.Error(1)
}

func (m *MockReportService) GenerateFinancialInsights(
	ctx context.Context,
	familyID uuid.UUID,
) ([]dto.RecommendationDTO, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.RecommendationDTO), args.Error(1)
}

// MockTemplateRenderer is a mock implementation of template renderer
type MockTemplateRenderer struct {
	mock.Mock
}

func (m *MockTemplateRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	args := m.Called(w, name, data, c)
	return args.Error(0)
}

// MockValidator is a mock implementation of validator
type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(i interface{}) error {
	args := m.Called(i)
	return args.Error(0)
}

// setupRepositories creates mock repositories for testing
func setupRepositories() *handlers.Repositories {
	return &handlers.Repositories{
		User:        &MockUserRepositoryWeb{},
		Family:      &MockFamilyRepositoryWeb{},
		Category:    &MockCategoryRepositoryWeb{},
		Transaction: &MockTransactionRepositoryWeb{},
		Budget:      &MockBudgetRepositoryWeb{},
	}
}

// createTestUser creates a test user for testing
func createTestUser() *user.User {
	userID := uuid.New()
	familyID := uuid.New()

	return &user.User{
		ID:        userID,
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		FamilyID:  familyID,
		Role:      user.RoleAdmin,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createTestFamily creates a test family for testing

// createTestCategory creates a test category for testing
func createTestCategory(familyID uuid.UUID) *category.Category {
	return &category.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Type:      category.TypeExpense,
		FamilyID:  familyID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// createTestTransaction creates a test transaction for testing

// createTestBudget creates a test budget for testing
func createTestBudget(familyID uuid.UUID, categoryID uuid.UUID) *budget.Budget {
	now := time.Now()
	return &budget.Budget{
		ID:         uuid.New(),
		Name:       "Test Budget",
		Amount:     1000.0,
		CategoryID: &categoryID,
		FamilyID:   familyID,
		Period:     budget.PeriodMonthly,
		StartDate:  now,
		EndDate:    now.AddDate(0, 1, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// setupEchoWithSession creates an Echo instance with session middleware for testing
func setupEchoWithSession() *echo.Echo {
	e := echo.New()

	// Set up mock template renderer to avoid "renderer not registered" errors
	mockRenderer := &MockTemplateRenderer{}
	mockRenderer.On("Render", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	e.Renderer = mockRenderer

	// Initialize session middleware first
	sessionMiddleware := middleware.SessionStore("test-secret-key-for-testing-that-is-long-enough", false)
	e.Use(sessionMiddleware)

	// Add simple CSRF token setup for tests (without validation)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Set up CSRF token in session for GET requests (like the New handler needs)
			if c.Request().Method == http.MethodGet {
				sess, err := session.Get("family-budget-session", c)
				if err == nil {
					sess.Values["csrf_token"] = "test-csrf-token"
					sess.Save(c.Request(), c.Response())
				}
			}
			return next(c)
		}
	})

	return e
}

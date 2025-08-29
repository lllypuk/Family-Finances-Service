package services

// Services contains all business services
type Services struct {
	User        UserService
	Family      FamilyService
	Category    CategoryService
	Transaction TransactionService
}

// NewServices creates a new services container with all dependencies
func NewServices(
	userRepo UserRepository,
	familyRepo FamilyRepository,
	categoryRepo CategoryRepository,
	transactionRepo TransactionRepository,
	budgetRepo BudgetRepositoryForTransactions,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)
	return &Services{
		User:        NewUserService(userRepo, familyRepo),
		Family:      NewFamilyService(familyRepo),
		Category:    NewCategoryService(categoryRepo, familyRepo, usageChecker),
		Transaction: NewTransactionService(transactionRepo, budgetRepo, categoryRepo, userRepo),
	}
}

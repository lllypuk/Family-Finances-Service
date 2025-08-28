package services

// Services contains all business services
type Services struct {
	User     UserService
	Family   FamilyService
	Category CategoryService
}

// NewServices creates a new services container with all dependencies
func NewServices(
	userRepo UserRepository,
	familyRepo FamilyRepository,
	categoryRepo CategoryRepository,
	transactionRepo TransactionRepositoryForUsage,
) *Services {
	usageChecker := NewCategoryUsageChecker(transactionRepo)
	return &Services{
		User:     NewUserService(userRepo, familyRepo),
		Family:   NewFamilyService(familyRepo),
		Category: NewCategoryService(categoryRepo, familyRepo, usageChecker),
	}
}

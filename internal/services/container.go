package services

// Services contains all business services
type Services struct {
	User   UserService
	Family FamilyService
}

// NewServices creates a new services container with all dependencies
func NewServices(userRepo UserRepository, familyRepo FamilyRepository) *Services {
	return &Services{
		User:   NewUserService(userRepo, familyRepo),
		Family: NewFamilyService(familyRepo),
	}
}

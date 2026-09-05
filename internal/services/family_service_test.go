package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
)

func validSetupDTO() dto.SetupFamilyDTO {
	return dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "RUB",
		Timezone:   "Europe/Moscow",
		Email:      "admin@example.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "Admin1234!",
	}
}

func TestFamilyService_SetupFamily_Success(t *testing.T) {
	familyRepo := new(MockFamilyRepository)
	svc := services.NewFamilyService(familyRepo, new(MockTransactionRepository))
	req := validSetupDTO()

	var gotAdmin *user.User
	var gotCategories []*category.Category
	familyRepo.On("Bootstrap", mock.Anything, mock.AnythingOfType("*user.Family"), mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			gotCategories, _ = args.Get(2).([]*category.Category)
			gotAdmin, _ = args.Get(3).(*user.User)
		}).
		Return(nil)

	family, err := svc.SetupFamily(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, req.FamilyName, family.Name)
	assert.Equal(t, req.Currency, family.Currency)

	require.NotNil(t, gotAdmin)
	assert.Equal(t, user.RoleAdmin, gotAdmin.Role)
	assert.Equal(t, req.Email, gotAdmin.Email)
	assert.True(t, auth.ComparePassword(gotAdmin.Password, req.Password))
	assert.Len(t, gotCategories, len(services.DefaultCategories()))
	familyRepo.AssertExpectations(t)
}

func TestFamilyService_SetupFamily_AlreadyExists(t *testing.T) {
	familyRepo := new(MockFamilyRepository)
	svc := services.NewFamilyService(familyRepo, new(MockTransactionRepository))
	familyRepo.On("Bootstrap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(user.ErrFamilyExists)

	_, err := svc.SetupFamily(context.Background(), validSetupDTO())
	require.ErrorIs(t, err, services.ErrFamilyAlreadyExists)
}

func TestFamilyService_SetupFamily_BootstrapFailure(t *testing.T) {
	familyRepo := new(MockFamilyRepository)
	svc := services.NewFamilyService(familyRepo, new(MockTransactionRepository))
	boom := errors.New("disk full")
	familyRepo.On("Bootstrap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(boom)

	_, err := svc.SetupFamily(context.Background(), validSetupDTO())
	require.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, services.ErrFamilyAlreadyExists)
}

func TestFamilyService_SetupFamily_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*dto.SetupFamilyDTO)
	}{
		{"short family name", func(d *dto.SetupFamilyDTO) { d.FamilyName = "A" }},
		{"bad currency", func(d *dto.SetupFamilyDTO) { d.Currency = "RUBL" }},
		{"bad timezone", func(d *dto.SetupFamilyDTO) { d.Timezone = "Mars/Olympus" }},
		{"bad email", func(d *dto.SetupFamilyDTO) { d.Email = "nope" }},
		{"weak password", func(d *dto.SetupFamilyDTO) { d.Password = "short" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			familyRepo := new(MockFamilyRepository)
			svc := services.NewFamilyService(familyRepo, new(MockTransactionRepository))
			req := validSetupDTO()
			tt.mutate(&req)

			_, err := svc.SetupFamily(context.Background(), req)
			require.ErrorIs(t, err, services.ErrValidationFailed)
			familyRepo.AssertNotCalled(t, "Bootstrap", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestDefaultCategories(t *testing.T) {
	defaults := services.DefaultCategories()
	require.NotEmpty(t, defaults)

	seen := make(map[string]bool, len(defaults))
	for _, c := range defaults {
		assert.NotEmpty(t, c.Name)
		assert.True(t, c.IsActive)
		assert.False(t, seen[c.Name+string(c.Type)], "duplicate default category %s", c.Name)
		seen[c.Name+string(c.Type)] = true
	}
	// каждый вызов — новые ID, иначе второй bootstrap упал бы на PK, а не на singleton
	assert.NotEqual(t, defaults[0].ID, services.DefaultCategories()[0].ID)
}

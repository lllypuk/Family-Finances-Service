package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/user"
)

// MockFamilyRepository is a shared mock implementation of FamilyRepository
type MockFamilyRepository struct {
	mock.Mock
}

func (m *MockFamilyRepository) Create(ctx context.Context, family *user.Family) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *MockFamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	args := m.Called(ctx, id)
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

func (m *MockFamilyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

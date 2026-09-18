package services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
	"family-budget-service/internal/services"
)

type mockAccountRepo struct {
	mock.Mock
}

func (m *mockAccountRepo) Create(ctx context.Context, a *account.Account) error {
	return m.Called(ctx, a).Error(0)
}

func (m *mockAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *mockAccountRepo) List(ctx context.Context, includeArchived bool) ([]*account.Account, error) {
	args := m.Called(ctx, includeArchived)
	return args.Get(0).([]*account.Account), args.Error(1)
}

func (m *mockAccountRepo) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	archived *bool,
) (*account.Account, error) {
	args := m.Called(ctx, id, name, archived)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *mockAccountRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func TestAccountService_Create_TrimsNameAndKeepsClientID(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()
	repo.On("Create", mock.Anything, mock.MatchedBy(func(a *account.Account) bool {
		return a.ID == id && a.Name == "Сбер" && !a.IsArchived
	})).Return(nil)

	created, err := services.NewAccountService(repo).Create(t.Context(), &id, "  Сбер ")
	require.NoError(t, err)
	assert.Equal(t, id, created.ID)
	repo.AssertExpectations(t)
}

func TestAccountService_Create_EmptyName(t *testing.T) {
	_, err := services.NewAccountService(&mockAccountRepo{}).Create(t.Context(), nil, "   ")
	require.ErrorIs(t, err, account.ErrNameEmpty)
}

func TestAccountService_Create_NameExists(t *testing.T) {
	repo := &mockAccountRepo{}
	repo.On("Create", mock.Anything, mock.Anything).Return(account.ErrNameExists)

	created, err := services.NewAccountService(repo).Create(t.Context(), nil, "Сбер")
	require.ErrorIs(t, err, account.ErrNameExists)
	assert.Nil(t, created)
}

func TestAccountService_Update_PassesOnlyGivenFields(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()
	archived := true
	repo.On("Update", mock.Anything, id, (*string)(nil), &archived).
		Return(&account.Account{ID: id, Name: "Сбер", IsArchived: true}, nil)

	updated, err := services.NewAccountService(repo).Update(t.Context(), id, nil, &archived)
	require.NoError(t, err)
	assert.True(t, updated.IsArchived)
	repo.AssertExpectations(t)
}

func TestAccountService_Update_RenameIsNormalized(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()
	repo.On("Update", mock.Anything, id, mock.MatchedBy(func(n *string) bool {
		return n != nil && *n == "Альфа"
	}), (*bool)(nil)).Return(&account.Account{ID: id, Name: "Альфа"}, nil)

	name := " Альфа"
	_, err := services.NewAccountService(repo).Update(t.Context(), id, &name, nil)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAccountService_Update_NotFound(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()
	repo.On("Update", mock.Anything, id, mock.Anything, mock.Anything).Return(nil, account.ErrNotFound)

	name := "Альфа"
	_, err := services.NewAccountService(repo).Update(t.Context(), id, &name, nil)
	require.ErrorIs(t, err, account.ErrNotFound)
}

func TestAccountService_Update_BlankNameNotWritten(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()

	name := " "
	_, err := services.NewAccountService(repo).Update(t.Context(), id, &name, nil)
	require.ErrorIs(t, err, account.ErrNameEmpty)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestAccountService_Delete_InUse(t *testing.T) {
	repo := &mockAccountRepo{}
	id := uuid.New()
	repo.On("Delete", mock.Anything, id).Return(account.ErrInUse)

	require.ErrorIs(t, services.NewAccountService(repo).Delete(t.Context(), id), account.ErrInUse)
}

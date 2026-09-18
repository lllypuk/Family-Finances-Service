package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/holding"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
)

type mockHoldingRepo struct {
	mock.Mock
}

func (m *mockHoldingRepo) Create(ctx context.Context, h *holding.Holding) error {
	return m.Called(ctx, h).Error(0)
}

func (m *mockHoldingRepo) GetByID(ctx context.Context, id uuid.UUID, today date.Date) (*holding.Holding, error) {
	args := m.Called(ctx, id, today)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*holding.Holding), args.Error(1)
}

func (m *mockHoldingRepo) List(ctx context.Context, includeArchived bool, today date.Date) ([]*holding.Holding, error) {
	args := m.Called(ctx, includeArchived, today)
	return args.Get(0).([]*holding.Holding), args.Error(1)
}

func (m *mockHoldingRepo) Update(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	kind *holding.Kind,
	archived *bool,
) error {
	return m.Called(ctx, id, name, kind, archived).Error(0)
}

func (m *mockHoldingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// setupHoldingService — «сегодня» семьи в зоне, далёкой от UTC, чтобы сервер в UTC дал другую дату.
func setupHoldingService(t *testing.T) (services.HoldingService, *mockHoldingRepo, date.Date) {
	t.Helper()

	families := &MockFamilyRepository{}
	families.On("Get", mock.Anything).Return(&user.Family{Timezone: "Pacific/Kiritimati"}, nil)
	loc, err := time.LoadLocation("Pacific/Kiritimati")
	require.NoError(t, err)

	repo := &mockHoldingRepo{}

	return services.NewHoldingService(repo, families), repo, date.Today(loc)
}

func TestHoldingService_Create_TrimsNameAndKeepsClientID(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	id := uuid.New()
	repo.On("Create", mock.Anything, mock.MatchedBy(func(h *holding.Holding) bool {
		return h.ID == id && h.Name == "Квартира" && h.Side == holding.SideAsset && h.Kind == holding.KindProperty
	})).Return(nil)

	created, err := svc.Create(t.Context(), &id, "  Квартира ", holding.SideAsset, holding.KindProperty)
	require.NoError(t, err)
	assert.Equal(t, id, created.ID)
	assert.Nil(t, created.Current)
	repo.AssertExpectations(t)
}

func TestHoldingService_Create_Refusals(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)

	_, err := svc.Create(t.Context(), nil, "Квартира", holding.SideAsset, holding.KindMortgage)
	require.ErrorIs(t, err, holding.ErrInvalidKind)

	_, err = svc.Create(t.Context(), nil, "Квартира", "equity", holding.KindOther)
	require.ErrorIs(t, err, holding.ErrInvalidSide)

	_, err = svc.Create(t.Context(), nil, "  ", holding.SideAsset, holding.KindOther)
	require.ErrorIs(t, err, holding.ErrNameEmpty)

	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestHoldingService_Create_NameExists(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(holding.ErrNameExists)

	_, err := svc.Create(t.Context(), nil, "Вклад", holding.SideAsset, holding.KindDeposit)
	require.ErrorIs(t, err, holding.ErrNameExists)
}

func TestHoldingService_List_UsesFamilyToday(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	repo.On("List", mock.Anything, true, today).Return([]*holding.Holding{{Name: "Вклад"}}, nil)

	list, err := svc.List(t.Context(), true)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	repo.AssertExpectations(t)
}

func TestHoldingService_Update_KindCheckedAgainstStoredSide(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	id := uuid.New()
	stored := &holding.Holding{ID: id, Side: holding.SideAsset, Kind: holding.KindOther}
	repo.On("GetByID", mock.Anything, id, today).Return(stored, nil)

	mortgage := holding.KindMortgage
	_, err := svc.Update(t.Context(), id, nil, &mortgage, nil)
	require.ErrorIs(t, err, holding.ErrInvalidKind)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	deposit := holding.KindDeposit
	repo.On("Update", mock.Anything, id, (*string)(nil), &deposit, (*bool)(nil)).Return(nil)
	_, err = svc.Update(t.Context(), id, nil, &deposit, nil)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHoldingService_Update_NotFound(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	id := uuid.New()
	archived := true
	repo.On("Update", mock.Anything, id, (*string)(nil), (*holding.Kind)(nil), &archived).
		Return(holding.ErrNotFound)

	_, err := svc.Update(t.Context(), id, nil, nil, &archived)
	require.ErrorIs(t, err, holding.ErrNotFound)
}

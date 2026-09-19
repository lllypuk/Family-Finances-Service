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
	"family-budget-service/internal/domain/money"
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
	income, expense *money.Minor,
) error {
	return m.Called(ctx, id, name, kind, archived, income, expense).Error(0)
}

func (m *mockHoldingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockHoldingRepo) UpsertValue(ctx context.Context, holdingID uuid.UUID, v *holding.Value) error {
	return m.Called(ctx, holdingID, v).Error(0)
}

func (m *mockHoldingRepo) DeleteValue(ctx context.Context, holdingID uuid.UUID, day date.Date) error {
	return m.Called(ctx, holdingID, day).Error(0)
}

func (m *mockHoldingRepo) ListValues(
	ctx context.Context,
	holdingID uuid.UUID,
	limit, offset int,
) ([]*holding.Value, int, error) {
	args := m.Called(ctx, holdingID, limit, offset)
	return args.Get(0).([]*holding.Value), args.Int(1), args.Error(2)
}

func (m *mockHoldingRepo) SeriesValues(ctx context.Context, from, to date.Date) ([]holding.SeriesRow, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]holding.SeriesRow), args.Error(1)
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

	created, err := svc.Create(t.Context(), &id, "  Квартира ", holding.SideAsset, holding.KindProperty, holding.Plan{})
	require.NoError(t, err)
	assert.Equal(t, id, created.ID)
	assert.Nil(t, created.Current)
	repo.AssertExpectations(t)
}

func TestHoldingService_Create_Refusals(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)

	_, err := svc.Create(t.Context(), nil, "Квартира", holding.SideAsset, holding.KindMortgage, holding.Plan{})
	require.ErrorIs(t, err, holding.ErrInvalidKind)

	_, err = svc.Create(t.Context(), nil, "Квартира", "equity", holding.KindOther, holding.Plan{})
	require.ErrorIs(t, err, holding.ErrInvalidSide)

	_, err = svc.Create(t.Context(), nil, "  ", holding.SideAsset, holding.KindOther, holding.Plan{})
	require.ErrorIs(t, err, holding.ErrNameEmpty)

	_, err = svc.Create(t.Context(), nil, "Вклад", holding.SideAsset, holding.KindDeposit,
		holding.Plan{MonthlyIncomeMinor: 100, MonthlyExpenseMinor: -1})
	var planErr *holding.PlanError
	require.ErrorAs(t, err, &planErr)
	assert.Equal(t, holding.FieldMonthlyExpense, planErr.Field)

	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestHoldingService_Create_PassesPlan(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	plan := holding.Plan{MonthlyIncomeMinor: 4_500_000, MonthlyExpenseMinor: 830_000}
	repo.On("Create", mock.Anything, mock.MatchedBy(func(h *holding.Holding) bool {
		return h.Plan == plan
	})).Return(nil)

	_, err := svc.Create(t.Context(), nil, "Квартира", holding.SideAsset, holding.KindProperty, plan)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHoldingService_Update_PassesPlanPointersAsIs(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	id := uuid.New()
	expense := money.Minor(-1)
	repo.On("Update", mock.Anything, id, (*string)(nil), (*holding.Kind)(nil), (*bool)(nil),
		(*money.Minor)(nil), &expense).Return(&holding.PlanError{Field: holding.FieldMonthlyExpense})

	_, err := svc.Update(t.Context(), id, nil, nil, nil, nil, &expense)
	var planErr *holding.PlanError
	require.ErrorAs(t, err, &planErr, "проверку итога делает репозиторий")
	repo.AssertNotCalled(t, "GetByID", mock.Anything, id, today)
}

func TestHoldingService_Create_NameExists(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	repo.On("Create", mock.Anything, mock.Anything).Return(holding.ErrNameExists)

	_, err := svc.Create(t.Context(), nil, "Вклад", holding.SideAsset, holding.KindDeposit, holding.Plan{})
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
	_, err := svc.Update(t.Context(), id, nil, &mortgage, nil, nil, nil)
	require.ErrorIs(t, err, holding.ErrInvalidKind)
	repo.AssertNotCalled(t, "Update",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	deposit := holding.KindDeposit
	repo.On("Update", mock.Anything, id, (*string)(nil), &deposit, (*bool)(nil), (*money.Minor)(nil), (*money.Minor)(nil)).
		Return(nil)
	_, err = svc.Update(t.Context(), id, nil, &deposit, nil, nil, nil)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHoldingService_Update_Name(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	id := uuid.New()
	trimmed := "Дача"
	repo.On("Update", mock.Anything, id, &trimmed, (*holding.Kind)(nil), (*bool)(nil), (*money.Minor)(nil), (*money.Minor)(nil)).
		Return(nil)
	repo.On("GetByID", mock.Anything, id, today).Return(&holding.Holding{ID: id, Name: trimmed}, nil)

	name := "  Дача "
	_, err := svc.Update(t.Context(), id, &name, nil, nil, nil, nil)
	require.NoError(t, err)

	blank := "   "
	_, err = svc.Update(t.Context(), id, &blank, nil, nil, nil, nil)
	require.ErrorIs(t, err, holding.ErrNameEmpty)
	repo.AssertNumberOfCalls(t, "Update", 1)
}

func TestHoldingService_Update_NotFound(t *testing.T) {
	svc, repo, _ := setupHoldingService(t)
	id := uuid.New()
	archived := true
	repo.On("Update", mock.Anything, id, (*string)(nil), (*holding.Kind)(nil), &archived, (*money.Minor)(nil), (*money.Minor)(nil)).
		Return(holding.ErrNotFound)

	_, err := svc.Update(t.Context(), id, nil, nil, &archived, nil, nil)
	require.ErrorIs(t, err, holding.ErrNotFound)
}

func TestHoldingService_PutValue_Checks(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	id := uuid.New()
	repo.On("UpsertValue", mock.Anything, id, mock.MatchedBy(func(v *holding.Value) bool {
		return v.Date == today && v.ValueMinor == 0
	})).Return(nil)

	v, err := svc.PutValue(t.Context(), id, today, 0)
	require.NoError(t, err, "0 законен, архивной позиции тоже можно")
	assert.Zero(t, v.ValueMinor)

	_, err = svc.PutValue(t.Context(), id, today.AddDays(1), 1)
	require.ErrorIs(t, err, holding.ErrValueDateFuture, "завтра по зоне семьи")

	_, err = svc.PutValue(t.Context(), id, today, -1)
	require.ErrorIs(t, err, holding.ErrValueOutOfRange)
	_, err = svc.PutValue(t.Context(), id, today, money.MaxAmount+1)
	require.ErrorIs(t, err, holding.ErrValueOutOfRange)

	repo.AssertNumberOfCalls(t, "UpsertValue", 1)
}

func TestHoldingService_Values_HoldingMissing(t *testing.T) {
	svc, repo, today := setupHoldingService(t)
	id := uuid.New()
	repo.On("GetByID", mock.Anything, id, today).Return(nil, holding.ErrNotFound)
	repo.On("UpsertValue", mock.Anything, id, mock.Anything).Return(holding.ErrNotFound)

	_, err := svc.PutValue(t.Context(), id, today, 1)
	require.ErrorIs(t, err, holding.ErrNotFound)
	require.ErrorIs(t, svc.DeleteValue(t.Context(), id, today), holding.ErrNotFound)
	_, _, err = svc.ListValues(t.Context(), id, 10, 0)
	require.ErrorIs(t, err, holding.ErrNotFound)
	repo.AssertNotCalled(t, "DeleteValue", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "ListValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

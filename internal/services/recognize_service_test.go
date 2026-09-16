package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/recognize"
	"family-budget-service/internal/services"
)

type fakeRecognizer struct {
	input  recognize.Input
	result recognize.Result
	err    error
}

func (f *fakeRecognizer) Recognize(_ context.Context, input recognize.Input) (recognize.Result, error) {
	f.input = input
	return f.result, f.err
}

func (f *fakeRecognizer) Budget() time.Duration { return 130 * time.Second }

type recognitionRecord struct {
	outcome string
	items   int
}

type recordingRecognizeObserver struct {
	records []recognitionRecord
}

func (o *recordingRecognizeObserver) ObserveRecognition(outcome string, _ time.Duration, items int) {
	o.records = append(o.records, recognitionRecord{outcome: outcome, items: items})
}

type recognizeFixture struct {
	families     *MockFamilyRepository
	categories   *MockCategoryRepository
	transactions *MockTransactionRepository
	observer     *recordingRecognizeObserver
	family       *user.Family
}

func newRecognizeFixture(cats ...*category.Category) *recognizeFixture {
	f := &recognizeFixture{
		families:     &MockFamilyRepository{},
		categories:   &MockCategoryRepository{},
		transactions: &MockTransactionRepository{},
		observer:     &recordingRecognizeObserver{},
		family:       user.NewFamily("Семья", "RUB", "Asia/Yekaterinburg"),
	}
	f.families.On("Get", mock.Anything).Return(f.family, nil)
	f.categories.On("GetAll", mock.Anything).Return(cats, nil)

	return f
}

func (f *recognizeFixture) service(engine services.Recognizer) services.RecognizeService {
	return services.NewRecognizeService(engine, f.families, f.categories, f.transactions, f.observer)
}

func testImages() []recognize.Image {
	return []recognize.Image{{MIME: "image/png", Data: []byte("png")}}
}

func TestRecognizeService_Recognize_Disabled(t *testing.T) {
	f := newRecognizeFixture()
	svc := f.service(nil)

	_, err := svc.Recognize(t.Context(), testImages())

	require.ErrorIs(t, err, recognize.ErrDisabled)
	assert.Zero(t, svc.Budget())
	assert.Equal(t, []recognitionRecord{{outcome: "unavailable"}}, f.observer.records)
	f.families.AssertNotCalled(t, "Get", mock.Anything)
}

func TestRecognizeService_Recognize_Input(t *testing.T) {
	food := &category.Category{ID: uuid.New(), Name: "Еда", Type: category.TypeExpense, IsActive: true}
	cafe := &category.Category{
		ID: uuid.New(), Name: "Кафе", Type: category.TypeExpense, ParentID: &food.ID, IsActive: true,
	}
	salary := &category.Category{ID: uuid.New(), Name: "Зарплата", Type: category.TypeIncome, IsActive: true}
	archived := &category.Category{ID: uuid.New(), Name: "Архив", Type: category.TypeExpense}
	orphan := &category.Category{
		ID: uuid.New(), Name: "Старое", Type: category.TypeExpense, ParentID: &archived.ID, IsActive: true,
	}

	f := newRecognizeFixture(food, cafe, salary, archived, orphan)
	engine := &fakeRecognizer{}
	svc := f.service(engine)

	result, err := svc.Recognize(t.Context(), testImages())
	require.NoError(t, err)

	assert.Equal(t, 130*time.Second, svc.Budget())
	assert.Equal(t, testImages(), engine.input.Images)
	assert.Equal(t, []recognize.Category{
		{ID: food.ID, Type: transaction.TypeExpense, Path: "Еда"},
		{ID: cafe.ID, Type: transaction.TypeExpense, Path: "Еда / Кафе"},
		{ID: salary.ID, Type: transaction.TypeIncome, Path: "Зарплата"},
	}, engine.input.Categories)
	assert.Equal(t, date.Today(f.family.Location()), engine.input.Today)
	assert.Equal(t, "RUB", engine.input.Currency)
	assert.Empty(t, result.Items)
	assert.Equal(t, []recognitionRecord{{outcome: "empty"}}, f.observer.records)
}

func TestRecognizeService_Recognize_Similar(t *testing.T) {
	f := newRecognizeFixture()
	day := date.New(2026, time.September, 14)
	amount := money.Minor(30000)
	existing := &transaction.Transaction{ID: uuid.New(), Date: day.AddDays(1), Description: "шавуха"}

	f.transactions.On("GetByFilter", mock.Anything, mock.MatchedBy(func(filter transaction.Filter) bool {
		return filter.Type != nil && *filter.Type == transaction.TypeExpense &&
			filter.AmountFromMinor != nil && *filter.AmountFromMinor == amount &&
			filter.AmountToMinor != nil && *filter.AmountToMinor == amount &&
			filter.DateFrom != nil && *filter.DateFrom == day.AddDays(-1) &&
			filter.DateTo != nil && *filter.DateTo == day.AddDays(1) &&
			filter.Limit == 3
	})).Return([]*transaction.Transaction{existing}, nil).Once()

	engine := &fakeRecognizer{result: recognize.Result{Model: "gemma4:31b", Items: []recognize.Item{
		{
			AmountMinor: amount,
			Type:        transaction.TypeExpense,
			Date:        &day,
			Description: "Шаурма",
			Similar:     []recognize.Similar{},
		},
		{AmountMinor: amount, Type: transaction.TypeExpense, Description: "Вчера", Similar: []recognize.Similar{}},
	}}}

	result, err := f.service(engine).Recognize(t.Context(), testImages())
	require.NoError(t, err)

	require.Len(t, result.Items, 2)
	assert.Equal(t, []recognize.Similar{
		{ID: existing.ID, Date: existing.Date, Description: "шавуха"},
	}, result.Items[0].Similar)
	assert.NotNil(t, result.Items[1].Similar)
	assert.Empty(t, result.Items[1].Similar)
	f.transactions.AssertExpectations(t)
	assert.Equal(t, []recognitionRecord{{outcome: "ok", items: 2}}, f.observer.records)
}

func TestRecognizeService_Recognize_EngineErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		outcome string
	}{
		{"unavailable", &recognize.UnavailableError{RetryAfter: time.Second}, "unavailable"},
		{"bad answer", recognize.ErrBadAnswer, "failed"},
		{"other", errors.New("boom"), "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecognizeFixture()

			_, err := f.service(&fakeRecognizer{err: tt.err}).Recognize(t.Context(), testImages())

			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, []recognitionRecord{{outcome: tt.outcome}}, f.observer.records)
		})
	}
}

func TestRecognizeService_Recognize_RepositoryErrors(t *testing.T) {
	boom := errors.New("boom")
	day := date.New(2026, time.September, 14)
	dated := recognize.Result{Items: []recognize.Item{{AmountMinor: 100, Type: transaction.TypeExpense, Date: &day}}}

	tests := []struct {
		name         string
		breakRepo    func(f *recognizeFixture)
		engineCalled bool
	}{
		{
			name: "family",
			breakRepo: func(f *recognizeFixture) {
				f.families.ExpectedCalls = nil
				f.families.On("Get", mock.Anything).Return(nil, boom)
			},
		},
		{
			name: "categories",
			breakRepo: func(f *recognizeFixture) {
				f.categories.ExpectedCalls = nil
				f.categories.On("GetAll", mock.Anything).Return(nil, boom)
			},
		},
		{
			name: "similar",
			breakRepo: func(f *recognizeFixture) {
				f.transactions.On("GetByFilter", mock.Anything, mock.Anything).Return(nil, boom)
			},
			engineCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRecognizeFixture()
			tt.breakRepo(f)
			engine := &fakeRecognizer{result: dated}

			result, err := f.service(engine).Recognize(t.Context(), testImages())

			require.ErrorIs(t, err, boom)
			assert.Empty(t, result.Items)
			assert.Equal(t, tt.engineCalled, engine.input.Images != nil)
			assert.Equal(t, []recognitionRecord{{outcome: "failed"}}, f.observer.records)
		})
	}
}

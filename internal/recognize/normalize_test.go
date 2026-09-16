package recognize_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/recognize"
)

func ptr[T any](v T) *T {
	return &v
}

func expenseItem() recognize.AnswerItem {
	return recognize.AnswerItem{
		Source:      json.Number("1"),
		Amount:      "100",
		Type:        "expense",
		Date:        ptr("2026-09-01"),
		YearPresent: ptr(true),
		Description: "Покупка",
	}
}

func normalizeOne(t *testing.T, raw recognize.AnswerItem, input recognize.Input) recognize.Item {
	t.Helper()

	result := recognize.Normalize(recognize.Answer{Items: []recognize.AnswerItem{raw}}, input)
	require.Len(t, result.Items, 1)

	return result.Items[0]
}

func defaultInput() recognize.Input {
	return recognize.Input{
		Images:   make([]recognize.Image, 2),
		Today:    date.New(2026, time.September, 16),
		Currency: "RUB",
	}
}

func TestNormalize_ProbeFixture(t *testing.T) {
	answer, err := recognize.Parse(readFixture(t, "probe_list.json"))
	require.NoError(t, err)

	groceries, cafe, salary, transport := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	input := recognize.Input{
		Images: make([]recognize.Image, 1),
		Categories: []recognize.Category{
			{ID: groceries, Type: transaction.TypeExpense, Path: "Еда / Продукты"},
			{ID: cafe, Type: transaction.TypeExpense, Path: "Еда / Кафе"},
			{ID: salary, Type: transaction.TypeIncome, Path: "Зарплата"},
			{ID: transport, Type: transaction.TypeExpense, Path: "Транспорт"},
		},
		Today:    date.New(2026, time.September, 16),
		Currency: "RUB",
	}

	result := recognize.Normalize(answer, input)
	require.Len(t, result.Items, 4)
	assert.False(t, result.Incomplete)

	sep14 := date.New(2026, time.September, 14)
	expected := []recognize.Item{
		{
			Source: ptr(0), AmountMinor: 34990, Currency: ptr("RUB"), Type: transaction.TypeExpense,
			Date: &sep14, DateAssumed: true, Description: "Пятёрочка", CategoryID: &groceries,
			Similar: []recognize.Similar{},
		},
		{
			Source: ptr(0), AmountMinor: 125000, Currency: ptr("RUB"), Type: transaction.TypeExpense,
			Date: &sep14, DateAssumed: true, Description: "Кафе Ромашка", CategoryID: &cafe,
			Similar: []recognize.Similar{},
		},
		{
			Source: ptr(0), AmountMinor: 8500000, Currency: ptr("RUB"), Type: transaction.TypeIncome,
			Date: ptr(date.New(2026, time.September, 13)), DateAssumed: true, Description: "Зарплата",
			CategoryID: &salary, Similar: []recognize.Similar{},
		},
		{
			Source: ptr(0), AmountMinor: 30000, Type: transaction.TypeExpense,
			Description: "Метро", CategoryID: &transport, Similar: []recognize.Similar{},
		},
	}
	assert.Equal(t, expected, result.Items)
}

func TestNormalize_YearAbsent(t *testing.T) {
	tests := []struct {
		name     string
		today    date.Date
		raw      string
		expected *date.Date
	}{
		{
			name:  "December operation seen in January goes to last year",
			today: date.New(2026, time.January, 10), raw: "2000-12-28",
			expected: ptr(date.New(2025, time.December, 28)),
		},
		{
			name:  "January operation seen in December stays this year",
			today: date.New(2026, time.December, 28), raw: "2000-01-05",
			expected: ptr(date.New(2026, time.January, 5)),
		},
		{
			name:  "reference day itself stays this year",
			today: date.New(2026, time.January, 10), raw: "2000-01-10",
			expected: ptr(date.New(2026, time.January, 10)),
		},
		{
			name:  "day after reference goes to last year",
			today: date.New(2026, time.January, 10), raw: "2000-01-11",
			expected: ptr(date.New(2025, time.January, 11)),
		},
		{
			name:  "leap day without a leap year nearby is dropped",
			today: date.New(2026, time.March, 1), raw: "2000-02-29",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := expenseItem()
			raw.Date, raw.YearPresent = ptr(tt.raw), ptr(false)
			input := defaultInput()
			input.Today = tt.today

			item := normalizeOne(t, raw, input)
			assert.Equal(t, tt.expected, item.Date)
			assert.Equal(t, tt.expected != nil, item.DateAssumed)
		})
	}
}

func TestNormalize_YearPresent(t *testing.T) {
	input := defaultInput()

	raw := expenseItem()
	raw.Date = ptr("2025-12-31")
	item := normalizeOne(t, raw, input)
	assert.Equal(t, ptr(date.New(2025, time.December, 31)), item.Date)
	assert.False(t, item.DateAssumed)

	raw.Date = ptr("2026-09-17")
	item = normalizeOne(t, raw, input)
	assert.Nil(t, item.Date, "будущая дата с годом — null, допуска «завтра» нет")
	assert.False(t, item.DateAssumed)

	raw.Date, raw.YearPresent = ptr("2026-09-10"), nil
	item = normalizeOne(t, raw, input)
	assert.Equal(t, ptr(date.New(2026, time.September, 10)), item.Date)
	assert.True(t, item.DateAssumed, "без year_present год не доверяем")

	raw.Date = ptr("14 сентября")
	item = normalizeOne(t, raw, input)
	assert.Nil(t, item.Date)
}

func TestNormalize_Amount(t *testing.T) {
	tests := []struct {
		name     string
		amount   any
		expected money.Minor
	}{
		{name: "comma and space", amount: "1 234,50", expected: 123450},
		{name: "non-breaking space", amount: "12 000", expected: 1200000},
		{name: "one fraction digit", amount: "1234.5", expected: 123450},
		{name: "JSON number", amount: json.Number("300"), expected: 30000},
		{name: "JSON number with fraction", amount: json.Number("99.99"), expected: 9999},
		{name: "sign ignored", amount: "-300", expected: 30000},
		{name: "max amount", amount: "999999999.99", expected: money.MaxAmount},
		{name: "zero dropped", amount: "0"},
		{name: "three fraction digits dropped", amount: "12.345"},
		{name: "exponent dropped", amount: json.Number("1e3")},
		{name: "letters dropped", amount: "300 ₽"},
		{name: "above max dropped", amount: "1000000000"},
		{name: "two separators dropped", amount: "1.234,50"},
		{name: "null dropped", amount: nil},
		{name: "empty dropped", amount: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := expenseItem()
			raw.Amount = tt.amount

			result := recognize.Normalize(recognize.Answer{Items: []recognize.AnswerItem{raw}}, defaultInput())
			if tt.expected == 0 {
				assert.Empty(t, result.Items)
				assert.True(t, result.Incomplete, "отброшенная строка помечает список неполным")

				return
			}

			require.Len(t, result.Items, 1)
			assert.Equal(t, tt.expected, result.Items[0].AmountMinor)
		})
	}
}

func TestNormalize_Currency(t *testing.T) {
	tests := []struct {
		currency *string
		expected *string
	}{
		{currency: ptr("₽"), expected: ptr("RUB")},
		{currency: ptr("руб."), expected: ptr("RUB")},
		{currency: ptr(" Руб "), expected: ptr("RUB")},
		{currency: ptr("RUB"), expected: ptr("RUB")},
		{currency: ptr("$"), expected: ptr("USD")},
		{currency: ptr("€"), expected: ptr("EUR")},
		{currency: ptr("тугрик"), expected: nil},
		{currency: ptr(""), expected: nil},
		{currency: nil, expected: nil},
	}

	for _, tt := range tests {
		raw := expenseItem()
		raw.Currency = tt.currency

		assert.Equal(t, tt.expected, normalizeOne(t, raw, defaultInput()).Currency)
	}
}

func TestNormalize_Category(t *testing.T) {
	cafe, otherA, otherB, salary := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	input := defaultInput()
	input.Categories = []recognize.Category{
		{ID: cafe, Type: transaction.TypeExpense, Path: "Еда / Кафе"},
		{ID: otherA, Type: transaction.TypeExpense, Path: "Прочее"},
		{ID: otherB, Type: transaction.TypeExpense, Path: "Прочее"},
		{ID: salary, Type: transaction.TypeIncome, Path: "Зарплата"},
	}

	tests := []struct {
		name     string
		category *string
		expected *uuid.UUID
	}{
		{name: "exact path", category: ptr("Еда / Кафе"), expected: &cafe},
		{name: "spacing around separator", category: ptr("Еда/Кафе"), expected: &cafe},
		{name: "type prefix from the prompt list", category: ptr("expense: Еда / Кафе"), expected: &cafe},
		{name: "leaf name only", category: ptr("Кафе")},
		{name: "ambiguous path", category: ptr("Прочее")},
		{name: "other type", category: ptr("Зарплата")},
		{name: "unknown", category: ptr("Путешествия")},
		{name: "null", category: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := expenseItem()
			raw.Category = tt.category

			assert.Equal(t, tt.expected, normalizeOne(t, raw, input).CategoryID)
		})
	}
}

func TestNormalize_Source(t *testing.T) {
	tests := []struct {
		name     string
		source   any
		expected *int
	}{
		{name: "first image", source: json.Number("1"), expected: ptr(0)},
		{name: "last image", source: json.Number("2"), expected: ptr(1)},
		{name: "string number", source: "2", expected: ptr(1)},
		{name: "zero is out of range", source: json.Number("0")},
		{name: "past the last image", source: json.Number("3")},
		{name: "fraction", source: json.Number("1.5")},
		{name: "null", source: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := expenseItem()
			raw.Source = tt.source

			assert.Equal(t, tt.expected, normalizeOne(t, raw, defaultInput()).Source)
		})
	}
}

func TestNormalize_TypeAndDescription(t *testing.T) {
	raw := expenseItem()
	raw.Type = " Income "
	raw.Description = "  " + strings.Repeat("я", 250) + "  "

	item := normalizeOne(t, raw, defaultInput())
	assert.Equal(t, transaction.TypeIncome, item.Type)
	assert.Equal(t, strings.Repeat("я", 200), item.Description)

	raw.Type = "transfer"
	result := recognize.Normalize(recognize.Answer{Items: []recognize.AnswerItem{raw, expenseItem()}}, defaultInput())
	assert.Len(t, result.Items, 1)
	assert.True(t, result.Incomplete)
}

func TestNormalize_TruncatesToMaxItems(t *testing.T) {
	items := make([]recognize.AnswerItem, 60)
	for i := range items {
		items[i] = expenseItem()
	}

	result := recognize.Normalize(recognize.Answer{Items: items}, defaultInput())
	assert.Len(t, result.Items, recognize.MaxItems)
	assert.True(t, result.Incomplete)

	result = recognize.Normalize(recognize.Answer{Items: items[:recognize.MaxItems]}, defaultInput())
	assert.Len(t, result.Items, recognize.MaxItems)
	assert.False(t, result.Incomplete)
}

func TestNormalize_EmptyItemsIsSuccess(t *testing.T) {
	result := recognize.Normalize(recognize.Answer{}, defaultInput())

	assert.NotNil(t, result.Items)
	assert.Empty(t, result.Items)
	assert.False(t, result.Incomplete)
}

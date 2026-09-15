package budget_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

func TestBudget_GetRemainingAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   money.Minor
		spent    money.Minor
		expected money.Minor
	}{
		{name: "No spending", amount: 100_000, spent: 0, expected: 100_000},
		{name: "Partial spending", amount: 100_000, spent: 30_000, expected: 70_000},
		{name: "Full spending", amount: 100_000, spent: 100_000, expected: 0},
		{name: "Over spending", amount: 100_000, spent: 120_000, expected: -20_000},
		{name: "Zero budget with spending", amount: 0, spent: 10_000, expected: -10_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				AmountMinor: tt.amount,
				SpentMinor:  tt.spent,
			}

			assert.Equal(t, tt.expected, budgetItem.GetRemainingAmount())
		})
	}
}

func TestBudget_GetSpentPercentage(t *testing.T) {
	tests := []struct {
		name     string
		amount   money.Minor
		spent    money.Minor
		expected float64
	}{
		{name: "No spending", amount: 100_000, spent: 0, expected: 0.0},
		{name: "25% spent", amount: 100_000, spent: 25_000, expected: 25.0},
		{name: "50% spent", amount: 100_000, spent: 50_000, expected: 50.0},
		{name: "100% spent", amount: 100_000, spent: 100_000, expected: 100.0},
		{name: "Over 100% spent", amount: 100_000, spent: 120_000, expected: 120.0},
		{name: "Zero budget protection", amount: 0, spent: 10_000, expected: 0.0},
		{name: "Fractional percentage", amount: 33_300, spent: 11_100, expected: 33.33333333333333},
		// Копейки не теряются: целочисленное деление дало бы 0.
		{name: "One of three minor units", amount: 3, spent: 1, expected: 33.33333333333333},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				AmountMinor: tt.amount,
				SpentMinor:  tt.spent,
			}

			assert.InDelta(t, tt.expected, budgetItem.GetSpentPercentage(), 0.0001)
		})
	}
}

func TestBudget_IsOverBudget(t *testing.T) {
	tests := []struct {
		name         string
		amount       money.Minor
		spent        money.Minor
		isOverBudget bool
	}{
		{name: "Under budget", amount: 100_000, spent: 80_000, isOverBudget: false},
		{name: "Exactly on budget", amount: 100_000, spent: 100_000, isOverBudget: false},
		{name: "Over budget", amount: 100_000, spent: 110_000, isOverBudget: true},
		{name: "Zero budget with any spending", amount: 0, spent: 1, isOverBudget: true},
		{name: "Zero budget with no spending", amount: 0, spent: 0, isOverBudget: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				AmountMinor: tt.amount,
				SpentMinor:  tt.spent,
			}

			assert.Equal(t, tt.isOverBudget, budgetItem.IsOverBudget())
		})
	}
}

func TestBudget_UpdateSpent(t *testing.T) {
	// Arrange
	budgetItem := &budget.Budget{
		AmountMinor: 100_000,
		SpentMinor:  20_000,
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	initialUpdatedAt := budgetItem.UpdatedAt

	// Act
	budgetItem.UpdateSpent(15_000)

	// Assert
	assert.Equal(t, money.Minor(35_000), budgetItem.SpentMinor)
	assert.True(t, budgetItem.UpdatedAt.After(initialUpdatedAt))
}

func TestUpdateSpent_MultipleOperations(t *testing.T) {
	// Arrange
	budgetItem := newBudget(
		"Monthly Budget",
		100_000,
		budget.PeriodMonthly,
		date.New(2025, time.January, 1),
		date.New(2025, time.January, 31),
	)

	// Несколько операций обновления
	budgetItem.UpdateSpent(10_000)
	assert.Equal(t, money.Minor(10_000), budgetItem.SpentMinor)

	budgetItem.UpdateSpent(5_000)
	assert.Equal(t, money.Minor(15_000), budgetItem.SpentMinor)

	budgetItem.UpdateSpent(-2_500) // Возврат/корректировка
	assert.Equal(t, money.Minor(12_500), budgetItem.SpentMinor)

	// Проверяем что бюджет не превышен
	assert.False(t, budgetItem.IsOverBudget())
	assert.Equal(t, money.Minor(87_500), budgetItem.GetRemainingAmount())
	assert.InDelta(t, 12.5, budgetItem.GetSpentPercentage(), 0.01)
}

func TestBudget_GetSpentShare(t *testing.T) {
	budgetItem := &budget.Budget{AmountMinor: 80_000, SpentMinor: 20_000}
	assert.InDelta(t, 0.25, budgetItem.GetSpentShare(), 1e-9)

	empty := &budget.Budget{}
	assert.InDelta(t, 0.0, empty.GetSpentShare(), 1e-9)
}

func TestPeriod_Constants(t *testing.T) {
	// Проверяем что все константы периодов определены корректно
	assert.Equal(t, budget.PeriodWeekly, budget.Period("weekly"))
	assert.Equal(t, budget.PeriodMonthly, budget.Period("monthly"))
	assert.Equal(t, budget.PeriodYearly, budget.Period("yearly"))
	assert.Equal(t, budget.PeriodCustom, budget.Period("custom"))
}

func TestBudget_RealWorldScenarios(t *testing.T) {
	t.Run("Monthly grocery budget workflow", func(t *testing.T) {
		startDate := date.New(2025, time.August, 1)
		endDate := date.New(2025, time.August, 31)

		// Создаем месячный бюджет на продукты
		groceryBudget := newBudget("Grocery Budget", 80_000, budget.PeriodMonthly, startDate, endDate)

		// Первая покупка
		groceryBudget.UpdateSpent(12_050)
		assert.Equal(t, money.Minor(12_050), groceryBudget.SpentMinor)
		assert.Equal(t, money.Minor(67_950), groceryBudget.GetRemainingAmount())
		assert.InDelta(t, 15.06, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Несколько покупок в течение месяца
		groceryBudget.UpdateSpent(9_525)  // Вторая покупка
		groceryBudget.UpdateSpent(15_000) // Третья покупка
		groceryBudget.UpdateSpent(20_075) // Четвертая покупка

		assert.Equal(t, money.Minor(12_050+9_525+15_000+20_075), groceryBudget.SpentMinor)
		assert.InDelta(t, 70.8125, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Превышение бюджета
		groceryBudget.UpdateSpent(25_000) // Большая покупка
		assert.True(t, groceryBudget.IsOverBudget())
		assert.Greater(t, groceryBudget.GetSpentPercentage(), 100.0)
		assert.Negative(t, groceryBudget.GetRemainingAmount())
	})

	t.Run("Weekly entertainment budget", func(t *testing.T) {
		startDate := date.New(2025, time.August, 11) // Понедельник
		endDate := date.New(2025, time.August, 17)   // Воскресенье

		entertainmentBudget := newBudget(
			"Entertainment",
			20_000,
			budget.PeriodWeekly,
			startDate,
			endDate,
		)

		// Развлечения в течение недели
		entertainmentBudget.UpdateSpent(4_500) // Кино
		entertainmentBudget.UpdateSpent(3_000) // Ресторан
		entertainmentBudget.UpdateSpent(2_500) // Кафе

		assert.Equal(t, money.Minor(10_000), entertainmentBudget.SpentMinor)
		assert.InDelta(t, 50.0, entertainmentBudget.GetSpentPercentage(), 0.01)
		assert.Equal(t, money.Minor(10_000), entertainmentBudget.GetRemainingAmount())
		assert.False(t, entertainmentBudget.IsOverBudget())
	})
}

func TestBudget_EdgeCases(t *testing.T) {
	t.Run("Very small amounts", func(t *testing.T) {
		budgetItem := newBudget(
			"Micro Budget",
			2, // две минимальные единицы
			budget.PeriodCustom,
			date.New(2025, time.January, 1),
			date.New(2025, time.January, 2),
		)
		budgetItem.UpdateSpent(1)

		assert.InDelta(t, 50.0, budgetItem.GetSpentPercentage(), 0.01)
		assert.Equal(t, money.Minor(1), budgetItem.GetRemainingAmount())
		assert.False(t, budgetItem.IsOverBudget())

		budgetItem.UpdateSpent(2)
		assert.True(t, budgetItem.IsOverBudget())
	})
}

// newBudget — то, что раньше давал конструктор домена: он ушёл, продакшен собирает
// структуру литералом.
func newBudget(
	name string,
	amountMinor money.Minor,
	period budget.Period,
	startDate, endDate date.Date,
) *budget.Budget {
	return &budget.Budget{
		ID:          uuid.New(),
		Name:        name,
		AmountMinor: amountMinor,
		Period:      period,
		StartDate:   startDate,
		EndDate:     endDate,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestValidateRecurring(t *testing.T) {
	tests := []struct {
		name    string
		period  budget.Period
		start   date.Date
		end     date.Date
		wantErr error
	}{
		{
			name:   "Monthly aligned",
			period: budget.PeriodMonthly,
			start:  date.New(2026, time.September, 1),
			end:    date.New(2026, time.September, 30),
		},
		{
			name:   "Monthly february leap year",
			period: budget.PeriodMonthly,
			start:  date.New(2024, time.February, 1),
			end:    date.New(2024, time.February, 29),
		},
		{
			name:    "Monthly not first day",
			period:  budget.PeriodMonthly,
			start:   date.New(2026, time.September, 2),
			end:     date.New(2026, time.September, 30),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:    "Monthly not last day",
			period:  budget.PeriodMonthly,
			start:   date.New(2026, time.September, 1),
			end:     date.New(2026, time.September, 29),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:    "Monthly spans two months",
			period:  budget.PeriodMonthly,
			start:   date.New(2026, time.September, 1),
			end:     date.New(2026, time.October, 31),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:   "Yearly aligned",
			period: budget.PeriodYearly,
			start:  date.New(2026, time.January, 1),
			end:    date.New(2026, time.December, 31),
		},
		{
			name:    "Yearly ends next year",
			period:  budget.PeriodYearly,
			start:   date.New(2026, time.January, 1),
			end:     date.New(2027, time.December, 31),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:    "Yearly starts in february",
			period:  budget.PeriodYearly,
			start:   date.New(2026, time.February, 1),
			end:     date.New(2026, time.December, 31),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:   "Weekly seven days",
			period: budget.PeriodWeekly,
			start:  date.New(2026, time.September, 14),
			end:    date.New(2026, time.September, 20),
		},
		{
			name:   "Weekly across month boundary",
			period: budget.PeriodWeekly,
			start:  date.New(2026, time.September, 28),
			end:    date.New(2026, time.October, 4),
		},
		{
			name:    "Weekly six days",
			period:  budget.PeriodWeekly,
			start:   date.New(2026, time.September, 14),
			end:     date.New(2026, time.September, 19),
			wantErr: budget.ErrRecurringNotAligned,
		},
		{
			name:    "Custom rejected",
			period:  budget.PeriodCustom,
			start:   date.New(2026, time.September, 1),
			end:     date.New(2026, time.September, 30),
			wantErr: budget.ErrRecurringCustom,
		},
		{
			name:    "Unknown period rejected",
			period:  budget.Period("quarterly"),
			start:   date.New(2026, time.September, 1),
			end:     date.New(2026, time.September, 30),
			wantErr: budget.ErrRecurringNotAligned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := budget.ValidateRecurring(tt.period, tt.start, tt.end)
			if tt.wantErr == nil {
				assert.NoError(t, err)

				return
			}

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestBudget_Next(t *testing.T) {
	tests := []struct {
		name      string
		period    budget.Period
		start     date.Date
		end       date.Date
		wantStart date.Date
		wantEnd   date.Date
	}{
		{
			name:      "Monthly december rolls over the year",
			period:    budget.PeriodMonthly,
			start:     date.New(2026, time.December, 1),
			end:       date.New(2026, time.December, 31),
			wantStart: date.New(2027, time.January, 1),
			wantEnd:   date.New(2027, time.January, 31),
		},
		{
			name:      "Monthly january to leap february",
			period:    budget.PeriodMonthly,
			start:     date.New(2024, time.January, 1),
			end:       date.New(2024, time.January, 31),
			wantStart: date.New(2024, time.February, 1),
			wantEnd:   date.New(2024, time.February, 29),
		},
		{
			name:      "Monthly january to short february",
			period:    budget.PeriodMonthly,
			start:     date.New(2026, time.January, 1),
			end:       date.New(2026, time.January, 31),
			wantStart: date.New(2026, time.February, 1),
			wantEnd:   date.New(2026, time.February, 28),
		},
		{
			name:      "Weekly shifts by seven days",
			period:    budget.PeriodWeekly,
			start:     date.New(2026, time.September, 28),
			end:       date.New(2026, time.October, 4),
			wantStart: date.New(2026, time.October, 5),
			wantEnd:   date.New(2026, time.October, 11),
		},
		{
			name:      "Yearly",
			period:    budget.PeriodYearly,
			start:     date.New(2026, time.January, 1),
			end:       date.New(2026, time.December, 31),
			wantStart: date.New(2027, time.January, 1),
			wantEnd:   date.New(2027, time.December, 31),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seriesID := uuid.New()
			current := &budget.Budget{
				ID:          uuid.New(),
				Name:        "Продукты",
				AmountMinor: 100_000,
				SpentMinor:  40_000,
				Period:      tt.period,
				StartDate:   tt.start,
				EndDate:     tt.end,
				IsActive:    true,
				Recurring:   true,
				SeriesID:    &seriesID,
			}

			next := current.Next()
			require.NotNil(t, next)
			assert.Equal(t, tt.wantStart, next.StartDate)
			assert.Equal(t, tt.wantEnd, next.EndDate)
			require.NoError(t, budget.ValidateRecurring(tt.period, next.StartDate, next.EndDate))
			assert.NotEqual(t, current.ID, next.ID)
			assert.Equal(t, current.Name, next.Name)
			assert.Equal(t, current.AmountMinor, next.AmountMinor)
			assert.Equal(t, money.Minor(0), next.SpentMinor)
			assert.True(t, next.Recurring)
			assert.True(t, next.IsActive)
			assert.Equal(t, &seriesID, next.SeriesID)
			assert.Equal(t, tt.start, current.StartDate)
		})
	}
}

func TestBudget_Next_CustomPeriod(t *testing.T) {
	current := &budget.Budget{
		Period:    budget.PeriodCustom,
		StartDate: date.New(2026, time.September, 1),
		EndDate:   date.New(2026, time.September, 10),
	}

	assert.Nil(t, current.Next())
}

func TestOverlapError_Is(t *testing.T) {
	err := error(&budget.OverlapError{Name: "Продукты"})

	require.ErrorIs(t, err, budget.ErrOverlap)
	assert.Contains(t, err.Error(), "Продукты")
}

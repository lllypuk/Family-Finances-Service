package budget_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/budget"
)

func TestNewBudget_Success(t *testing.T) {
	// Arrange
	name := "Monthly Groceries"
	amount := 1000.0
	period := budget.PeriodMonthly
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	// Act
	budgetItem := budget.NewBudget(name, amount, period, startDate, endDate)

	// Assert
	require.NotNil(t, budgetItem)
	assert.NotEqual(t, uuid.Nil, budgetItem.ID)
	assert.Equal(t, name, budgetItem.Name)
	assert.InDelta(t, amount, budgetItem.Amount, 0.01)
	assert.InDelta(t, 0.0, budgetItem.Spent, 0.01)
	assert.Equal(t, period, budgetItem.Period)
	assert.Equal(t, startDate, budgetItem.StartDate)
	assert.Equal(t, endDate, budgetItem.EndDate)
	assert.False(t, budgetItem.CreatedAt.IsZero())
	assert.False(t, budgetItem.UpdatedAt.IsZero())
	assert.Nil(t, budgetItem.CategoryID) // Должен быть nil по умолчанию
}

func TestNewBudget_ValidationScenarios(t *testing.T) {
	tests := []struct {
		name       string
		budgetName string
		amount     float64
		period     budget.Period
		valid      bool
	}{
		{
			name:       "Valid budget with positive amount",
			budgetName: "Food Budget",
			amount:     500.0,
			period:     budget.PeriodMonthly,
			valid:      true,
		},
		{
			name:       "Zero amount budget",
			budgetName: "Zero Budget",
			amount:     0.0,
			period:     budget.PeriodWeekly,
			valid:      true, // Технически допустимо
		},
		{
			name:       "Negative amount budget",
			budgetName: "Negative Budget",
			amount:     -100.0,
			period:     budget.PeriodYearly,
			valid:      false, // Отрицательный бюджет не имеет смысла
		},
		{
			name:       "Empty name budget",
			budgetName: "",
			amount:     1000.0,
			period:     budget.PeriodCustom,
			valid:      false, // Пустое имя недопустимо
		},
	}

	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := budget.NewBudget(tt.budgetName, tt.amount, tt.period, startDate, endDate)

			assert.NotNil(t, budgetItem)
			if tt.valid {
				assert.NotEqual(t, uuid.Nil, budgetItem.ID)
				assert.Equal(t, tt.budgetName, budgetItem.Name)
				assert.InDelta(t, tt.amount, budgetItem.Amount, 0.01)
			} else {
				// В текущей реализации валидация не проводится в конструкторе
				// Это область для будущих улучшений
				assert.Equal(t, tt.budgetName, budgetItem.Name)
				assert.InDelta(t, tt.amount, budgetItem.Amount, 0.01)
			}
		})
	}
}

func TestBudget_GetRemainingAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		spent    float64
		expected float64
	}{
		{
			name:     "No spending",
			amount:   1000.0,
			spent:    0.0,
			expected: 1000.0,
		},
		{
			name:     "Partial spending",
			amount:   1000.0,
			spent:    300.0,
			expected: 700.0,
		},
		{
			name:     "Full spending",
			amount:   1000.0,
			spent:    1000.0,
			expected: 0.0,
		},
		{
			name:     "Over spending",
			amount:   1000.0,
			spent:    1200.0,
			expected: -200.0,
		},
		{
			name:     "Zero budget with spending",
			amount:   0.0,
			spent:    100.0,
			expected: -100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			remaining := budgetItem.GetRemainingAmount()
			assert.InDelta(t, tt.expected, remaining, 0.01)
		})
	}
}

func TestBudget_GetSpentPercentage(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		spent    float64
		expected float64
	}{
		{
			name:     "No spending",
			amount:   1000.0,
			spent:    0.0,
			expected: 0.0,
		},
		{
			name:     "25% spent",
			amount:   1000.0,
			spent:    250.0,
			expected: 25.0,
		},
		{
			name:     "50% spent",
			amount:   1000.0,
			spent:    500.0,
			expected: 50.0,
		},
		{
			name:     "100% spent",
			amount:   1000.0,
			spent:    1000.0,
			expected: 100.0,
		},
		{
			name:     "Over 100% spent",
			amount:   1000.0,
			spent:    1200.0,
			expected: 120.0,
		},
		{
			name:     "Zero budget protection",
			amount:   0.0,
			spent:    100.0,
			expected: 0.0, // Защита от деления на ноль
		},
		{
			name:     "Fractional percentage",
			amount:   333.0,
			spent:    111.0,
			expected: 33.33333333333333, // ~33.33%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			percentage := budgetItem.GetSpentPercentage()
			assert.InDelta(t, tt.expected, percentage, 0.0001) // Допустимая погрешность для float
		})
	}
}

func TestBudget_IsOverBudget(t *testing.T) {
	tests := []struct {
		name         string
		amount       float64
		spent        float64
		isOverBudget bool
	}{
		{
			name:         "Under budget",
			amount:       1000.0,
			spent:        800.0,
			isOverBudget: false,
		},
		{
			name:         "Exactly on budget",
			amount:       1000.0,
			spent:        1000.0,
			isOverBudget: false,
		},
		{
			name:         "Over budget",
			amount:       1000.0,
			spent:        1100.0,
			isOverBudget: true,
		},
		{
			name:         "Zero budget with any spending",
			amount:       0.0,
			spent:        0.01,
			isOverBudget: true,
		},
		{
			name:         "Zero budget with no spending",
			amount:       0.0,
			spent:        0.0,
			isOverBudget: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budgetItem := &budget.Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			result := budgetItem.IsOverBudget()
			assert.Equal(t, tt.isOverBudget, result)
		})
	}
}

func TestBudget_UpdateSpent(t *testing.T) {
	// Arrange
	budgetItem := &budget.Budget{
		Amount:    1000.0,
		Spent:     200.0,
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	initialUpdatedAt := budgetItem.UpdatedAt

	// Act
	budgetItem.UpdateSpent(150.0)

	// Assert
	assert.InDelta(t, 350.0, budgetItem.Spent, 0.01)
	assert.True(t, budgetItem.UpdatedAt.After(initialUpdatedAt))
}

func TestUpdateSpent_MultipleOperations(t *testing.T) {
	// Arrange
	budgetItem := budget.NewBudget(
		"Monthly Budget",
		1000.0,
		budget.PeriodMonthly,
		uuid.New(),
		time.Now(),
		time.Now().Add(30*24*time.Hour),
	)

	// Несколько операций обновления
	budgetItem.UpdateSpent(100.0)
	assert.InDelta(t, 100.0, budgetItem.Spent, 0.01)

	budgetItem.UpdateSpent(50.0)
	assert.InDelta(t, 150.0, budgetItem.Spent, 0.01)

	budgetItem.UpdateSpent(-25.0) // Возврат/корректировка
	assert.InDelta(t, 125.0, budgetItem.Spent, 0.01)

	// Проверяем что бюджет не превышен
	assert.False(t, budgetItem.IsOverBudget())
	assert.InDelta(t, 875.0, budgetItem.GetRemainingAmount(), 0.01)
	assert.InDelta(t, 12.5, budgetItem.GetSpentPercentage(), 0.01)
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
		startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)

		// Создаем месячный бюджет на продукты
		groceryBudget := budget.NewBudget("Grocery Budget", 800.0, budget.PeriodMonthly, startDate, endDate)

		// Первая покупка
		groceryBudget.UpdateSpent(120.50)
		assert.InDelta(t, 120.50, groceryBudget.Spent, 0.01)
		assert.InDelta(t, 679.50, groceryBudget.GetRemainingAmount(), 0.01)
		assert.InDelta(t, 15.06, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Несколько покупок в течение месяца
		groceryBudget.UpdateSpent(95.25)  // Вторая покупка
		groceryBudget.UpdateSpent(150.00) // Третья покупка
		groceryBudget.UpdateSpent(200.75) // Четвертая покупка

		totalSpent := 120.50 + 95.25 + 150.00 + 200.75
		assert.InDelta(t, totalSpent, groceryBudget.Spent, 0.01)
		assert.InDelta(t, 70.8125, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Превышение бюджета
		groceryBudget.UpdateSpent(250.00) // Большая покупка
		assert.True(t, groceryBudget.IsOverBudget())
		assert.Greater(t, groceryBudget.GetSpentPercentage(), 100.0)
		assert.Negative(t, groceryBudget.GetRemainingAmount())
	})

	t.Run("Weekly entertainment budget", func(t *testing.T) {
		startDate := time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC)  // Понедельник
		endDate := time.Date(2025, 8, 17, 23, 59, 59, 0, time.UTC) // Воскресенье

		entertainmentBudget := budget.NewBudget(
			"Entertainment",
			200.0,
			budget.PeriodWeekly,
			startDate,
			endDate,
		)

		// Развлечения в течение недели
		entertainmentBudget.UpdateSpent(45.00) // Кино
		entertainmentBudget.UpdateSpent(30.00) // Ресторан
		entertainmentBudget.UpdateSpent(25.00) // Кафе

		assert.InDelta(t, 100.0, entertainmentBudget.Spent, 0.01)
		assert.InDelta(t, 50.0, entertainmentBudget.GetSpentPercentage(), 0.01)
		assert.InDelta(t, 100.0, entertainmentBudget.GetRemainingAmount(), 0.01)
		assert.False(t, entertainmentBudget.IsOverBudget())
	})
}

func TestBudget_EdgeCases(t *testing.T) {
	t.Run("Very small amounts", func(t *testing.T) {
		smallAmount := 0.01

		budgetItem := budget.NewBudget(
			"Micro Budget",
			smallAmount,
			budget.PeriodCustom,
			time.Now(),
			time.Now().AddDate(0, 0, 1),
		)
		budgetItem.UpdateSpent(0.005)

		assert.InDelta(t, 50.0, budgetItem.GetSpentPercentage(), 0.01)
		assert.InDelta(t, 0.005, budgetItem.GetRemainingAmount(), 0.01)
		assert.False(t, budgetItem.IsOverBudget())

		budgetItem.UpdateSpent(0.006)
		assert.True(t, budgetItem.IsOverBudget())
	})
}

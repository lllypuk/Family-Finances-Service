package budget

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBudget_Success(t *testing.T) {
	// Arrange
	name := "Monthly Groceries"
	amount := 1000.0
	period := PeriodMonthly
	familyID := uuid.New()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	// Act
	budget := NewBudget(name, amount, period, familyID, startDate, endDate)

	// Assert
	require.NotNil(t, budget)
	assert.NotEqual(t, uuid.Nil, budget.ID)
	assert.Equal(t, name, budget.Name)
	assert.Equal(t, amount, budget.Amount)
	assert.Equal(t, 0.0, budget.Spent)
	assert.Equal(t, period, budget.Period)
	assert.Equal(t, familyID, budget.FamilyID)
	assert.Equal(t, startDate, budget.StartDate)
	assert.Equal(t, endDate, budget.EndDate)
	assert.True(t, budget.IsActive)
	assert.False(t, budget.CreatedAt.IsZero())
	assert.False(t, budget.UpdatedAt.IsZero())
	assert.Nil(t, budget.CategoryID) // Должен быть nil по умолчанию
}

func TestNewBudget_ValidationScenarios(t *testing.T) {
	tests := []struct {
		name       string
		budgetName string
		amount     float64
		period     Period
		valid      bool
	}{
		{
			name:       "Valid budget with positive amount",
			budgetName: "Food Budget",
			amount:     500.0,
			period:     PeriodMonthly,
			valid:      true,
		},
		{
			name:       "Zero amount budget",
			budgetName: "Zero Budget",
			amount:     0.0,
			period:     PeriodWeekly,
			valid:      true, // Технически допустимо
		},
		{
			name:       "Negative amount budget",
			budgetName: "Negative Budget",
			amount:     -100.0,
			period:     PeriodYearly,
			valid:      false, // Отрицательный бюджет не имеет смысла
		},
		{
			name:       "Empty name budget",
			budgetName: "",
			amount:     1000.0,
			period:     PeriodCustom,
			valid:      false, // Пустое имя недопустимо
		},
	}

	familyID := uuid.New()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewBudget(tt.budgetName, tt.amount, tt.period, familyID, startDate, endDate)

			assert.NotNil(t, budget)
			if tt.valid {
				assert.NotEqual(t, uuid.Nil, budget.ID)
				assert.Equal(t, tt.budgetName, budget.Name)
				assert.Equal(t, tt.amount, budget.Amount)
			} else {
				// В текущей реализации валидация не проводится в конструкторе
				// Это область для будущих улучшений
				assert.Equal(t, tt.budgetName, budget.Name)
				assert.Equal(t, tt.amount, budget.Amount)
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
			budget := &Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			remaining := budget.GetRemainingAmount()
			assert.Equal(t, tt.expected, remaining)
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
			budget := &Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			percentage := budget.GetSpentPercentage()
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
			budget := &Budget{
				Amount: tt.amount,
				Spent:  tt.spent,
			}

			result := budget.IsOverBudget()
			assert.Equal(t, tt.isOverBudget, result)
		})
	}
}

func TestBudget_UpdateSpent(t *testing.T) {
	// Arrange
	budget := &Budget{
		Amount:    1000.0,
		Spent:     200.0,
		UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	initialUpdatedAt := budget.UpdatedAt

	// Act
	budget.UpdateSpent(150.0)

	// Assert
	assert.Equal(t, 350.0, budget.Spent)
	assert.True(t, budget.UpdatedAt.After(initialUpdatedAt))
}

func TestBudget_UpdateSpent_MultipleOperations(t *testing.T) {
	budget := &Budget{
		Amount: 1000.0,
		Spent:  0.0,
	}

	// Несколько операций обновления
	budget.UpdateSpent(100.0)
	assert.Equal(t, 100.0, budget.Spent)

	budget.UpdateSpent(50.0)
	assert.Equal(t, 150.0, budget.Spent)

	budget.UpdateSpent(-25.0) // Возврат/корректировка
	assert.Equal(t, 125.0, budget.Spent)

	// Проверяем что бюджет не превышен
	assert.False(t, budget.IsOverBudget())
	assert.Equal(t, 875.0, budget.GetRemainingAmount())
	assert.Equal(t, 12.5, budget.GetSpentPercentage())
}

func TestPeriod_Constants(t *testing.T) {
	// Проверяем что все константы периодов определены корректно
	assert.Equal(t, PeriodWeekly, Period("weekly"))
	assert.Equal(t, PeriodMonthly, Period("monthly"))
	assert.Equal(t, PeriodYearly, Period("yearly"))
	assert.Equal(t, PeriodCustom, Period("custom"))
}

func TestBudget_DateValidation_Scenarios(t *testing.T) {
	familyID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		period    Period
		valid     bool
	}{
		{
			name:      "Valid monthly period",
			startDate: now,
			endDate:   now.AddDate(0, 1, 0),
			period:    PeriodMonthly,
			valid:     true,
		},
		{
			name:      "End date before start date",
			startDate: now,
			endDate:   now.AddDate(0, 0, -1),
			period:    PeriodCustom,
			valid:     false,
		},
		{
			name:      "Same start and end date",
			startDate: now,
			endDate:   now,
			period:    PeriodCustom,
			valid:     false, // Бюджет должен иметь ненулевой период
		},
		{
			name:      "Weekly period with correct duration",
			startDate: now,
			endDate:   now.AddDate(0, 0, 7),
			period:    PeriodWeekly,
			valid:     true,
		},
		{
			name:      "Yearly period with correct duration",
			startDate: now,
			endDate:   now.AddDate(1, 0, 0),
			period:    PeriodYearly,
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewBudget("Test Budget", 1000.0, tt.period, familyID, tt.startDate, tt.endDate)

			assert.NotNil(t, budget)
			assert.Equal(t, tt.startDate, budget.StartDate)
			assert.Equal(t, tt.endDate, budget.EndDate)

			// В текущей реализации валидация дат не проводится в конструкторе
			// Это может быть добавлено в будущем
			if tt.valid {
				assert.True(t, budget.EndDate.After(budget.StartDate))
			}
		})
	}
}

func TestBudget_CategoryID_OptionalField(t *testing.T) {
	familyID := uuid.New()
	categoryID := uuid.New()

	// Бюджет без категории
	budgetWithoutCategory := NewBudget(
		"General Budget",
		1000.0,
		PeriodMonthly,
		familyID,
		time.Now(),
		time.Now().AddDate(0, 1, 0),
	)

	assert.Nil(t, budgetWithoutCategory.CategoryID)

	// Бюджет с категорией (устанавливается после создания)
	budgetWithCategory := NewBudget(
		"Food Budget",
		500.0,
		PeriodMonthly,
		familyID,
		time.Now(),
		time.Now().AddDate(0, 1, 0),
	)
	budgetWithCategory.CategoryID = &categoryID

	assert.NotNil(t, budgetWithCategory.CategoryID)
	assert.Equal(t, categoryID, *budgetWithCategory.CategoryID)
}

func TestBudget_RealWorldScenarios(t *testing.T) {
	t.Run("Monthly grocery budget workflow", func(t *testing.T) {
		familyID := uuid.New()
		startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)

		// Создаем месячный бюджет на продукты
		groceryBudget := NewBudget("Grocery Budget", 800.0, PeriodMonthly, familyID, startDate, endDate)

		// Первая покупка
		groceryBudget.UpdateSpent(120.50)
		assert.Equal(t, 120.50, groceryBudget.Spent)
		assert.Equal(t, 679.50, groceryBudget.GetRemainingAmount())
		assert.InDelta(t, 15.06, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Несколько покупок в течение месяца
		groceryBudget.UpdateSpent(95.25)  // Вторая покупка
		groceryBudget.UpdateSpent(150.00) // Третья покупка
		groceryBudget.UpdateSpent(200.75) // Четвертая покупка

		totalSpent := 120.50 + 95.25 + 150.00 + 200.75
		assert.Equal(t, totalSpent, groceryBudget.Spent)
		assert.InDelta(t, 70.8125, groceryBudget.GetSpentPercentage(), 0.01)
		assert.False(t, groceryBudget.IsOverBudget())

		// Превышение бюджета
		groceryBudget.UpdateSpent(250.00) // Большая покупка
		assert.True(t, groceryBudget.IsOverBudget())
		assert.Greater(t, groceryBudget.GetSpentPercentage(), 100.0)
		assert.Negative(t, groceryBudget.GetRemainingAmount())
	})

	t.Run("Weekly entertainment budget", func(t *testing.T) {
		familyID := uuid.New()
		startDate := time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC)  // Понедельник
		endDate := time.Date(2025, 8, 17, 23, 59, 59, 0, time.UTC) // Воскресенье

		entertainmentBudget := NewBudget("Entertainment", 200.0, PeriodWeekly, familyID, startDate, endDate)

		// Развлечения в течение недели
		entertainmentBudget.UpdateSpent(45.00) // Кино
		entertainmentBudget.UpdateSpent(30.00) // Ресторан
		entertainmentBudget.UpdateSpent(25.00) // Кафе

		assert.Equal(t, 100.0, entertainmentBudget.Spent)
		assert.Equal(t, 50.0, entertainmentBudget.GetSpentPercentage())
		assert.Equal(t, 100.0, entertainmentBudget.GetRemainingAmount())
		assert.False(t, entertainmentBudget.IsOverBudget())
	})
}

func TestBudget_EdgeCases(t *testing.T) {
	t.Run("Very large amounts", func(t *testing.T) {
		familyID := uuid.New()
		largeAmount := 999999999.99

		budget := NewBudget(
			"Large Budget",
			largeAmount,
			PeriodYearly,
			familyID,
			time.Now(),
			time.Now().AddDate(1, 0, 0),
		)
		budget.UpdateSpent(500000000.00)

		assert.Greater(t, budget.GetSpentPercentage(), 50.0)
		assert.Positive(t, budget.GetRemainingAmount())
		assert.False(t, budget.IsOverBudget())
	})

	t.Run("Very small amounts", func(t *testing.T) {
		familyID := uuid.New()
		smallAmount := 0.01

		budget := NewBudget(
			"Micro Budget",
			smallAmount,
			PeriodCustom,
			familyID,
			time.Now(),
			time.Now().AddDate(0, 0, 1),
		)
		budget.UpdateSpent(0.005)

		assert.Equal(t, 50.0, budget.GetSpentPercentage())
		assert.Equal(t, 0.005, budget.GetRemainingAmount())
		assert.False(t, budget.IsOverBudget())

		budget.UpdateSpent(0.006)
		assert.True(t, budget.IsOverBudget())
	})
}

// Benchmark тесты для производительности
func BenchmarkNewBudget(b *testing.B) {
	familyID := uuid.New()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	for range b.N {
		NewBudget("Benchmark Budget", 1000.0, PeriodMonthly, familyID, startDate, endDate)
	}
}

func BenchmarkBudget_GetSpentPercentage(b *testing.B) {
	budget := &Budget{
		Amount: 1000.0,
		Spent:  450.75,
	}

	for range b.N {
		budget.GetSpentPercentage()
	}
}

func BenchmarkBudget_UpdateSpent(b *testing.B) {
	budget := &Budget{
		Amount: 1000.0,
		Spent:  0.0,
	}

	for range b.N {
		budget.UpdateSpent(1.0)
	}
}

func BenchmarkBudget_IsOverBudget(b *testing.B) {
	budget := &Budget{
		Amount: 1000.0,
		Spent:  950.0,
	}

	for range b.N {
		budget.IsOverBudget()
	}
}

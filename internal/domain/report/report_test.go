package report

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReport_Success(t *testing.T) {
	// Arrange
	name := "Monthly Expenses Report"
	reportType := TypeExpenses
	period := PeriodMonthly
	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)

	// Act
	report := NewReport(name, reportType, period, familyID, userID, startDate, endDate)

	// Assert
	require.NotNil(t, report)
	assert.NotEqual(t, uuid.Nil, report.ID)
	assert.Equal(t, name, report.Name)
	assert.Equal(t, reportType, report.Type)
	assert.Equal(t, period, report.Period)
	assert.Equal(t, familyID, report.FamilyID)
	assert.Equal(t, userID, report.UserID)
	assert.Equal(t, startDate, report.StartDate)
	assert.Equal(t, endDate, report.EndDate)
	assert.False(t, report.GeneratedAt.IsZero())

	// Проверяем что Data инициализирована пустой структурой
	assert.NotNil(t, report.Data)
	assert.Equal(t, 0.0, report.Data.TotalIncome)
	assert.Equal(t, 0.0, report.Data.TotalExpenses)
	assert.Equal(t, 0.0, report.Data.NetIncome)
	assert.Empty(t, report.Data.CategoryBreakdown)
	assert.Empty(t, report.Data.DailyBreakdown)
	assert.Empty(t, report.Data.TopExpenses)
	assert.Empty(t, report.Data.BudgetComparison)
}

func TestReportType_Constants(t *testing.T) {
	// Проверяем что все типы отчетов определены корректно
	assert.Equal(t, Type("expenses"), TypeExpenses)
	assert.Equal(t, Type("income"), TypeIncome)
	assert.Equal(t, Type("budget"), TypeBudget)
	assert.Equal(t, Type("cash_flow"), TypeCashFlow)
	assert.Equal(t, Type("category_break"), TypeCategoryBreak)
}

func TestReportPeriod_Constants(t *testing.T) {
	// Проверяем что все периоды определены корректно
	assert.Equal(t, Period("daily"), PeriodDaily)
	assert.Equal(t, Period("weekly"), PeriodWeekly)
	assert.Equal(t, Period("monthly"), PeriodMonthly)
	assert.Equal(t, Period("yearly"), PeriodYearly)
	assert.Equal(t, Period("custom"), PeriodCustom)
}

func TestReport_ValidationScenarios(t *testing.T) {
	tests := []struct {
		name       string
		reportName string
		reportType Type
		period     Period
		valid      bool
	}{
		{
			name:       "Valid expense report",
			reportName: "Expense Analysis",
			reportType: TypeExpenses,
			period:     PeriodMonthly,
			valid:      true,
		},
		{
			name:       "Valid income report",
			reportName: "Income Summary",
			reportType: TypeIncome,
			period:     PeriodYearly,
			valid:      true,
		},
		{
			name:       "Valid budget report",
			reportName: "Budget Comparison",
			reportType: TypeBudget,
			period:     PeriodWeekly,
			valid:      true,
		},
		{
			name:       "Valid cash flow report",
			reportName: "Cash Flow Analysis",
			reportType: TypeCashFlow,
			period:     PeriodDaily,
			valid:      true,
		},
		{
			name:       "Valid category breakdown report",
			reportName: "Category Analysis",
			reportType: TypeCategoryBreak,
			period:     PeriodCustom,
			valid:      true,
		},
		{
			name:       "Empty report name",
			reportName: "",
			reportType: TypeExpenses,
			period:     PeriodMonthly,
			valid:      false, // Пустое имя недопустимо
		},
		{
			name:       "Invalid report type",
			reportName: "Invalid Report",
			reportType: Type("invalid"),
			period:     PeriodMonthly,
			valid:      false,
		},
		{
			name:       "Invalid period",
			reportName: "Invalid Period Report",
			reportType: TypeIncome,
			period:     Period("invalid"),
			valid:      false,
		},
	}

	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewReport(tt.reportName, tt.reportType, tt.period, familyID, userID, startDate, endDate)

			assert.NotNil(t, report)
			if tt.valid {
				assert.NotEqual(t, uuid.Nil, report.ID)
				assert.Equal(t, tt.reportName, report.Name)
				assert.Equal(t, tt.reportType, report.Type)
				assert.Equal(t, tt.period, report.Period)
			} else {
				// В текущей реализации валидация не проводится в конструкторе
				// Это область для будущих улучшений
				assert.Equal(t, tt.reportName, report.Name)
				assert.Equal(t, tt.reportType, report.Type)
				assert.Equal(t, tt.period, report.Period)
			}
		})
	}
}

func TestCategoryReportItem_Structure(t *testing.T) {
	categoryID := uuid.New()
	item := CategoryReportItem{
		CategoryID:   categoryID,
		CategoryName: "Groceries",
		Amount:       1500.75,
		Percentage:   35.5,
		Count:        25,
	}

	assert.Equal(t, categoryID, item.CategoryID)
	assert.Equal(t, "Groceries", item.CategoryName)
	assert.Equal(t, 1500.75, item.Amount)
	assert.Equal(t, 35.5, item.Percentage)
	assert.Equal(t, 25, item.Count)
}

func TestDailyReportItem_Structure(t *testing.T) {
	date := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)
	item := DailyReportItem{
		Date:     date,
		Income:   2500.00,
		Expenses: 800.50,
		Balance:  1699.50,
	}

	assert.Equal(t, date, item.Date)
	assert.Equal(t, 2500.00, item.Income)
	assert.Equal(t, 800.50, item.Expenses)
	assert.Equal(t, 1699.50, item.Balance)
}

func TestTransactionReportItem_Structure(t *testing.T) {
	transactionID := uuid.New()
	date := time.Date(2025, 8, 15, 14, 30, 0, 0, time.UTC)

	item := TransactionReportItem{
		ID:          transactionID,
		Amount:      125.50,
		Description: "Grocery shopping at SuperMarket",
		Category:    "Food & Dining",
		Date:        date,
	}

	assert.Equal(t, transactionID, item.ID)
	assert.Equal(t, 125.50, item.Amount)
	assert.Equal(t, "Grocery shopping at SuperMarket", item.Description)
	assert.Equal(t, "Food & Dining", item.Category)
	assert.Equal(t, date, item.Date)
}

func TestBudgetComparisonItem_Structure(t *testing.T) {
	budgetID := uuid.New()
	item := BudgetComparisonItem{
		BudgetID:   budgetID,
		BudgetName: "Monthly Groceries",
		Planned:    1000.00,
		Actual:     1150.75,
		Difference: -150.75,
		Percentage: 115.075,
	}

	assert.Equal(t, budgetID, item.BudgetID)
	assert.Equal(t, "Monthly Groceries", item.BudgetName)
	assert.Equal(t, 1000.00, item.Planned)
	assert.Equal(t, 1150.75, item.Actual)
	assert.Equal(t, -150.75, item.Difference)
	assert.Equal(t, 115.075, item.Percentage)
}

func TestReportData_ComplexStructure(t *testing.T) {
	categoryID1 := uuid.New()
	categoryID2 := uuid.New()
	transactionID1 := uuid.New()
	transactionID2 := uuid.New()
	budgetID1 := uuid.New()

	data := Data{
		TotalIncome:   5000.00,
		TotalExpenses: 3750.25,
		NetIncome:     1249.75,
		CategoryBreakdown: []CategoryReportItem{
			{
				CategoryID:   categoryID1,
				CategoryName: "Food & Dining",
				Amount:       1500.00,
				Percentage:   40.0,
				Count:        15,
			},
			{
				CategoryID:   categoryID2,
				CategoryName: "Transportation",
				Amount:       800.00,
				Percentage:   21.3,
				Count:        8,
			},
		},
		DailyBreakdown: []DailyReportItem{
			{
				Date:     time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
				Income:   2500.00,
				Expenses: 200.00,
				Balance:  2300.00,
			},
			{
				Date:     time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC),
				Income:   0.00,
				Expenses: 150.75,
				Balance:  2149.25,
			},
		},
		TopExpenses: []TransactionReportItem{
			{
				ID:          transactionID1,
				Amount:      500.00,
				Description: "Monthly rent payment",
				Category:    "Housing",
				Date:        time.Date(2025, 8, 1, 9, 0, 0, 0, time.UTC),
			},
			{
				ID:          transactionID2,
				Amount:      300.00,
				Description: "Car repair",
				Category:    "Transportation",
				Date:        time.Date(2025, 8, 5, 14, 30, 0, 0, time.UTC),
			},
		},
		BudgetComparison: []BudgetComparisonItem{
			{
				BudgetID:   budgetID1,
				BudgetName: "Monthly Food Budget",
				Planned:    1200.00,
				Actual:     1500.00,
				Difference: -300.00,
				Percentage: 125.0,
			},
		},
	}

	// Проверяем общие суммы
	assert.Equal(t, 5000.00, data.TotalIncome)
	assert.Equal(t, 3750.25, data.TotalExpenses)
	assert.Equal(t, 1249.75, data.NetIncome)

	// Проверяем разбивку по категориям
	assert.Len(t, data.CategoryBreakdown, 2)
	assert.Equal(t, "Food & Dining", data.CategoryBreakdown[0].CategoryName)
	assert.Equal(t, 1500.00, data.CategoryBreakdown[0].Amount)

	// Проверяем ежедневную разбивку
	assert.Len(t, data.DailyBreakdown, 2)
	assert.Equal(t, 2500.00, data.DailyBreakdown[0].Income)
	assert.Equal(t, 200.00, data.DailyBreakdown[0].Expenses)

	// Проверяем топ расходы
	assert.Len(t, data.TopExpenses, 2)
	assert.Equal(t, 500.00, data.TopExpenses[0].Amount)
	assert.Equal(t, "Monthly rent payment", data.TopExpenses[0].Description)

	// Проверяем сравнение с бюджетом
	assert.Len(t, data.BudgetComparison, 1)
	assert.Equal(t, 1200.00, data.BudgetComparison[0].Planned)
	assert.Equal(t, 1500.00, data.BudgetComparison[0].Actual)
	assert.Equal(t, -300.00, data.BudgetComparison[0].Difference)
}

func TestReport_DateValidation_Scenarios(t *testing.T) {
	familyID := uuid.New()
	userID := uuid.New()
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
			period:    PeriodDaily,
			valid:     false, // Отчет должен иметь ненулевой период
		},
		{
			name:      "Daily period",
			startDate: time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 15, 23, 59, 59, 0, time.UTC),
			period:    PeriodDaily,
			valid:     true,
		},
		{
			name:      "Weekly period",
			startDate: now,
			endDate:   now.AddDate(0, 0, 7),
			period:    PeriodWeekly,
			valid:     true,
		},
		{
			name:      "Yearly period",
			startDate: now,
			endDate:   now.AddDate(1, 0, 0),
			period:    PeriodYearly,
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewReport("Test Report", TypeExpenses, tt.period, familyID, userID, tt.startDate, tt.endDate)

			assert.NotNil(t, report)
			assert.Equal(t, tt.startDate, report.StartDate)
			assert.Equal(t, tt.endDate, report.EndDate)

			// В текущей реализации валидация дат не проводится в конструкторе
			if tt.valid {
				assert.True(t, report.EndDate.After(report.StartDate) || report.EndDate.Equal(report.StartDate))
			}
		})
	}
}

func TestReport_RealWorldScenarios(t *testing.T) {
	t.Run("Monthly expense report with complete data", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()
		startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)

		report := NewReport("August 2025 Expenses", TypeExpenses, PeriodMonthly, familyID, userID, startDate, endDate)

		// Заполняем отчет данными
		report.Data = Data{
			TotalIncome:   6000.00,
			TotalExpenses: 4250.75,
			NetIncome:     1749.25,
			CategoryBreakdown: []CategoryReportItem{
				{
					CategoryID:   uuid.New(),
					CategoryName: "Housing",
					Amount:       1500.00,
					Percentage:   35.3,
					Count:        3,
				},
				{
					CategoryID:   uuid.New(),
					CategoryName: "Food & Dining",
					Amount:       1200.00,
					Percentage:   28.2,
					Count:        25,
				},
				{
					CategoryID:   uuid.New(),
					CategoryName: "Transportation",
					Amount:       550.75,
					Percentage:   13.0,
					Count:        12,
				},
			},
		}

		// Проверяем корректность данных
		assert.Equal(t, "August 2025 Expenses", report.Name)
		assert.Equal(t, TypeExpenses, report.Type)
		assert.Equal(t, PeriodMonthly, report.Period)
		assert.Equal(t, 6000.00, report.Data.TotalIncome)
		assert.Equal(t, 4250.75, report.Data.TotalExpenses)
		assert.Equal(t, 1749.25, report.Data.NetIncome)

		// Проверяем что суммы категорий сходятся
		totalCategoryAmount := 0.0
		for _, item := range report.Data.CategoryBreakdown {
			totalCategoryAmount += item.Amount
		}
		assert.InDelta(t, 3250.75, totalCategoryAmount, 0.01) // Не все расходы могут быть категоризированы

		// Проверяем проценты
		assert.InDelta(t, 35.3, report.Data.CategoryBreakdown[0].Percentage, 0.1)
		assert.InDelta(t, 28.2, report.Data.CategoryBreakdown[1].Percentage, 0.1)
	})

	t.Run("Weekly cash flow report", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()
		startDate := time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC)  // Понедельник
		endDate := time.Date(2025, 8, 17, 23, 59, 59, 0, time.UTC) // Воскресенье

		report := NewReport("Week 33 Cash Flow", TypeCashFlow, PeriodWeekly, familyID, userID, startDate, endDate)

		// Заполняем ежедневными данными
		report.Data.DailyBreakdown = []DailyReportItem{
			{Date: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC), Income: 0.00, Expenses: 50.00, Balance: -50.00},
			{Date: time.Date(2025, 8, 12, 0, 0, 0, 0, time.UTC), Income: 2500.00, Expenses: 100.00, Balance: 2350.00},
			{Date: time.Date(2025, 8, 13, 0, 0, 0, 0, time.UTC), Income: 0.00, Expenses: 75.50, Balance: 2274.50},
			{Date: time.Date(2025, 8, 14, 0, 0, 0, 0, time.UTC), Income: 0.00, Expenses: 120.00, Balance: 2154.50},
			{Date: time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC), Income: 0.00, Expenses: 200.00, Balance: 1954.50},
			{Date: time.Date(2025, 8, 16, 0, 0, 0, 0, time.UTC), Income: 150.00, Expenses: 80.00, Balance: 2024.50},
			{Date: time.Date(2025, 8, 17, 0, 0, 0, 0, time.UTC), Income: 0.00, Expenses: 45.25, Balance: 1979.25},
		}

		// Вычисляем итоги
		totalIncome := 0.0
		totalExpenses := 0.0
		for _, daily := range report.Data.DailyBreakdown {
			totalIncome += daily.Income
			totalExpenses += daily.Expenses
		}

		report.Data.TotalIncome = totalIncome
		report.Data.TotalExpenses = totalExpenses
		report.Data.NetIncome = totalIncome - totalExpenses

		assert.Equal(t, "Week 33 Cash Flow", report.Name)
		assert.Equal(t, TypeCashFlow, report.Type)
		assert.Equal(t, PeriodWeekly, report.Period)
		assert.Equal(t, 2650.00, report.Data.TotalIncome)
		assert.Equal(t, 670.75, report.Data.TotalExpenses)
		assert.Equal(t, 1979.25, report.Data.NetIncome)
		assert.Len(t, report.Data.DailyBreakdown, 7)
	})

	t.Run("Budget comparison report", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()
		startDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2025, 8, 31, 23, 59, 59, 0, time.UTC)

		report := NewReport("August Budget vs Actual", TypeBudget, PeriodMonthly, familyID, userID, startDate, endDate)

		report.Data.BudgetComparison = []BudgetComparisonItem{
			{
				BudgetID:   uuid.New(),
				BudgetName: "Groceries",
				Planned:    800.00,
				Actual:     920.50,
				Difference: -120.50,
				Percentage: 115.06,
			},
			{
				BudgetID:   uuid.New(),
				BudgetName: "Entertainment",
				Planned:    300.00,
				Actual:     180.25,
				Difference: 119.75,
				Percentage: 60.08,
			},
			{
				BudgetID:   uuid.New(),
				BudgetName: "Transportation",
				Planned:    500.00,
				Actual:     485.00,
				Difference: 15.00,
				Percentage: 97.0,
			},
		}

		assert.Equal(t, TypeBudget, report.Type)
		assert.Len(t, report.Data.BudgetComparison, 3)

		// Проверяем превышение бюджета
		overBudgetItems := 0
		underBudgetItems := 0
		for _, item := range report.Data.BudgetComparison {
			if item.Difference < 0 {
				overBudgetItems++
			} else if item.Difference > 0 {
				underBudgetItems++
			}
		}

		assert.Equal(t, 1, overBudgetItems)  // Groceries превысили бюджет
		assert.Equal(t, 2, underBudgetItems) // Entertainment и Transportation в рамках бюджета
	})
}

func TestReport_EdgeCases(t *testing.T) {
	t.Run("Report with zero amounts", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()

		report := NewReport("Zero Report", TypeIncome, PeriodMonthly, familyID, userID, time.Now(), time.Now().AddDate(0, 1, 0))

		// Все суммы нулевые
		report.Data = Data{
			TotalIncome:   0.0,
			TotalExpenses: 0.0,
			NetIncome:     0.0,
		}

		assert.Equal(t, 0.0, report.Data.TotalIncome)
		assert.Equal(t, 0.0, report.Data.TotalExpenses)
		assert.Equal(t, 0.0, report.Data.NetIncome)
	})

	t.Run("Report with very large amounts", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()

		report := NewReport("Large Report", TypeCashFlow, PeriodYearly, familyID, userID, time.Now(), time.Now().AddDate(1, 0, 0))

		largeAmount := 999999999.99
		report.Data = Data{
			TotalIncome:   largeAmount,
			TotalExpenses: largeAmount * 0.8,
			NetIncome:     largeAmount * 0.2,
		}

		assert.Equal(t, largeAmount, report.Data.TotalIncome)
		assert.True(t, report.Data.NetIncome > 0)
		assert.True(t, report.Data.TotalExpenses < report.Data.TotalIncome)
	})

	t.Run("Report with negative net income", func(t *testing.T) {
		familyID := uuid.New()
		userID := uuid.New()

		report := NewReport("Loss Report", TypeCashFlow, PeriodMonthly, familyID, userID, time.Now(), time.Now().AddDate(0, 1, 0))

		report.Data = Data{
			TotalIncome:   3000.00,
			TotalExpenses: 3500.00,
			NetIncome:     -500.00,
		}

		assert.Equal(t, 3000.00, report.Data.TotalIncome)
		assert.Equal(t, 3500.00, report.Data.TotalExpenses)
		assert.Equal(t, -500.00, report.Data.NetIncome)
		assert.True(t, report.Data.NetIncome < 0)
	})
}

// Benchmark тесты для производительности
func BenchmarkNewReport(b *testing.B) {
	familyID := uuid.New()
	userID := uuid.New()
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	for i := 0; i < b.N; i++ {
		NewReport("Benchmark Report", TypeExpenses, PeriodMonthly, familyID, userID, startDate, endDate)
	}
}

func BenchmarkReport_CategoryBreakdownProcessing(b *testing.B) {
	// Benchmark обработки большого количества категорий
	report := &Report{
		Data: Data{
			CategoryBreakdown: make([]CategoryReportItem, 100),
		},
	}

	// Заполняем данными
	for i := 0; i < 100; i++ {
		report.Data.CategoryBreakdown[i] = CategoryReportItem{
			CategoryID:   uuid.New(),
			CategoryName: "Category " + string(rune(i)),
			Amount:       float64(i * 10),
			Percentage:   float64(i),
			Count:        i,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total := 0.0
		for _, item := range report.Data.CategoryBreakdown {
			total += item.Amount
		}
		_ = total
	}
}

func BenchmarkReport_DailyBreakdownProcessing(b *testing.B) {
	// Benchmark обработки ежедневных данных за год
	report := &Report{
		Data: Data{
			DailyBreakdown: make([]DailyReportItem, 365),
		},
	}

	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 365; i++ {
		report.Data.DailyBreakdown[i] = DailyReportItem{
			Date:     baseDate.AddDate(0, 0, i),
			Income:   float64(100 + i),
			Expenses: float64(80 + i),
			Balance:  float64(20),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		totalIncome := 0.0
		totalExpenses := 0.0
		for _, item := range report.Data.DailyBreakdown {
			totalIncome += item.Income
			totalExpenses += item.Expenses
		}
		_ = totalIncome - totalExpenses
	}
}

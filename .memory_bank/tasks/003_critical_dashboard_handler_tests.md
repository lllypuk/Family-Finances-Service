# Task 003: Dashboard Handler Tests (CRITICAL)

## Priority: CRITICAL

## Status: Pending

## Estimated LOC: ~1,077

## Overview

Dashboard Handler (`internal/web/handlers/dashboard.go`) имеет **0% покрытия** и содержит самую сложную бизнес-логику
веб-слоя: финансовое прогнозирование, сравнение периодов, аналитику.

## Файлы

- **Source**: `internal/web/handlers/dashboard.go`
- **Test**: `internal/web/handlers/dashboard_test.go` (создать)

## Методы для тестирования

### HTTP Handlers

| Метод                  | Описание                 | LOC | Сложность |
|------------------------|--------------------------|-----|-----------|
| `Dashboard()`          | Главная страница         | ~80 | HIGH      |
| `DashboardFilter()`    | HTMX фильтры             | ~40 | MEDIUM    |
| `DashboardStats()`     | HTMX статистика          | ~30 | MEDIUM    |
| `RecentTransactions()` | HTMX последние операции  | ~25 | LOW       |
| `BudgetOverview()`     | HTMX обзор бюджетов      | ~30 | MEDIUM    |
| `CategoryInsights()`   | HTMX аналитика категорий | ~35 | MEDIUM    |

### Helper Methods (ВЫСОКИЙ ПРИОРИТЕТ)

| Метод                           | Описание               | LOC | Сложность |
|---------------------------------|------------------------|-----|-----------|
| `buildMonthlySummary()`         | Агрегация по месяцам   | ~58 | HIGH      |
| `calculatePreviousData()`       | Сравнение периодов     | ~32 | HIGH      |
| `buildBudgetOverview()`         | Аналитика бюджетов     | ~27 | MEDIUM    |
| `processBudgets()`              | Статистика бюджетов    | ~16 | MEDIUM    |
| `createBudgetItem()`            | Форматирование бюджета | ~40 | LOW       |
| `buildRecentActivity()`         | Сборка активности      | ~54 | MEDIUM    |
| `buildCategoryInsights()`       | Аналитика категорий    | ~31 | MEDIUM    |
| `groupTransactionsByCategory()` | Группировка            | ~23 | MEDIUM    |
| `buildEnhancedStats()`          | Расширенная аналитика  | ~75 | HIGH      |
| `buildForecast()`               | Прогнозирование        | ~25 | HIGH      |

## Ключевые сценарии

### 1. Главный Dashboard Handler

```go
func TestDashboardHandler_Dashboard(t *testing.T) {
tests := []struct {
name           string
setupMock      func (*MockServices)
expectedStatus int
checkResponse  func (t *testing.T, body string)
}{
{
name: "full data",
setupMock: func (m *MockServices) {
m.TransactionService.GetAllFunc = func (...) ([]domain.Transaction, error) {
return testTransactions, nil
}
},
expectedStatus: http.StatusOK,
},
{"empty transactions", ...},
{"service error", ...},
{"unauthorized", ...},
}
}
```

### 2. Месячная сводка

```go
func TestDashboardHandler_buildMonthlySummary(t *testing.T) {
tests := []struct {
name         string
transactions []domain.Transaction
expected     []MonthlySummary
}{
{
name: "single month",
transactions: []domain.Transaction{
{Date: parseDate("2024-01-15"), Amount: 1000, Type: "income"},
{Date: parseDate("2024-01-20"), Amount: 500, Type: "expense"},
},
expected: []MonthlySummary{
{Month: "January 2024", Income: 1000, Expenses: 500, Balance: 500},
},
},
{"multiple months", ...},
{"empty transactions", ...},
}
}
```

### 3. Сравнение с предыдущим периодом

```go
func TestDashboardHandler_calculatePreviousData(t *testing.T) {
tests := []struct {
name           string
current        PeriodData
previous       PeriodData
expectedDeltas DeltaData
}{
{
name:     "positive growth",
current:  PeriodData{Income: 5000, Expenses: 3000},
previous: PeriodData{Income: 4000, Expenses: 3500},
expectedDeltas: DeltaData{
IncomeChange:  25.0, // +25%
ExpenseChange: -14.29, // -14.29%
},
},
{"negative growth", ...},
{"zero previous (div by zero)", ...},
}
}
```

### 4. Обзор бюджетов

```go
func TestDashboardHandler_buildBudgetOverview(t *testing.T) {
tests := []struct {
name     string
budgets  []domain.Budget
expected BudgetOverview
}{
{"all under limit", ...},
{"some over limit", ...},
{"warning threshold 80%", ...},
}
}
```

### 5. Финансовое прогнозирование

```go
func TestDashboardHandler_buildForecast(t *testing.T) {
tests := []struct {
name             string
historicalData   []MonthlyData
expectedForecast Forecast
}{
{"steady income pattern", ...},
{"growing expenses trend", ...},
{"insufficient data", ...},
}
}
```

### 6. Расширенная статистика

```go
func TestDashboardHandler_buildEnhancedStats(t *testing.T) {
tests := []struct {
name     string
data     DashboardData
expected EnhancedStats
}{
{
name: "savings rate calculation",
data: DashboardData{
TotalIncome:   10000,
TotalExpenses: 7000,
},
expected: EnhancedStats{
SavingsRate: 30.0, // (10000-7000)/10000 * 100
},
},
{"average daily spending", ...},
{"expense by category %", ...},
}
}
```

## Mock Services

```go
type MockServices struct {
TransactionService *MockTransactionService
BudgetService      *MockBudgetService
CategoryService    *MockCategoryService
}

type MockTransactionService struct {
GetAllFunc        func (ctx context.Context, familyID int64) ([]domain.Transaction, error)
GetByDateRangeFunc func (ctx context.Context, familyID int64, start, end time.Time) ([]domain.Transaction, error)
}

type MockBudgetService struct {
GetActiveBudgetsFunc func (ctx context.Context, familyID int64) ([]domain.Budget, error)
GetUsageStatsFunc    func (ctx context.Context, budgetID int64) (*domain.BudgetStats, error)
}
```

## HTMX тестирование

```go
func TestDashboardHandler_HTMXEndpoints(t *testing.T) {
// Проверить что HTMX endpoints возвращают partial HTML
// Верифицировать HX-Trigger headers
// Тестировать error responses с HTMX error partials

t.Run("filter returns partial", func (t *testing.T) {
// Request с HX-Request header
// Response НЕ содержит <!DOCTYPE html>
// Response содержит только нужный partial
})
}
```

## Edge Cases

1. **Новый пользователь** - нет транзакций, бюджетов, категорий
2. **Первый день месяца** - расчёты периодов
3. **Високосный год** - 29 февраля
4. **Большой датасет** - 10,000+ транзакций
5. **Конкурентные запросы** - несколько загрузок
6. **Ошибки сервисов** - graceful degradation
7. **Невалидная сессия** - редирект на login
8. **Нулевые значения** - защита от деления на ноль

## Утилиты для тестов

```go
func setupDashboardTestServer(t *testing.T) (*echo.Echo, *MockServices) {
e := echo.New()
mocks := &MockServices{...}

handler := NewDashboardHandler(mocks)
e.GET("/dashboard", handler.Dashboard)
// ... register routes

return e, mocks
}

func createTestTransaction(date string, amount float64, txType string) domain.Transaction {
return domain.Transaction{
Date:   parseDate(date),
Amount: amount,
Type:   txType,
}
}
```

## Критерии приёмки

- [ ] Все 6 публичных handlers покрыты
- [ ] Все 10 helper методов имеют unit тесты
- [ ] Финансовые расчёты верифицированы
- [ ] HTMX responses возвращают корректный partial HTML
- [ ] Edge cases покрыты
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 001: Budget Repository Tests (зависимость)
- Task 002: Report Repository Tests (зависимость)
- Task 004: Transactions Handler Tests

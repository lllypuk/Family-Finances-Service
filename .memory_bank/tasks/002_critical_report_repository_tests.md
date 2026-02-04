# Task 002: Report Repository Tests (CRITICAL)

## Priority: CRITICAL

## Status: Pending

## Estimated LOC: ~700+

## Overview

Report Repository (`internal/infrastructure/report/report_repository_sqlite.go`) имеет **0% покрытия** и содержит
сложные алгоритмы генерации отчётов с SQL-агрегациями.

## Файлы

- **Source**: `internal/infrastructure/report/report_repository_sqlite.go`
- **Test**: `internal/infrastructure/report/report_repository_test.go` (создать)

## Методы для тестирования

### CRUD операции

| Метод                           | Описание        | Тест-кейсы                              |
|---------------------------------|-----------------|-----------------------------------------|
| `Create()`                      | Создание отчёта | Валидные данные, невалидный family ID   |
| `GetByID()`                     | Получение по ID | Существующий, несуществующий            |
| `GetAll()`                      | Все отчёты      | Пустой список, несколько                |
| `GetByFamilyIDWithPagination()` | С пагинацией    | Страница 1, середина, последняя, пустая |
| `GetByUserID()`                 | По пользователю | Валидный user, нет отчётов              |
| `Delete()`                      | Удаление        | Существующий, несуществующий            |
| `GetSummary()`                  | Сводка          | С данными, пустая                       |

### Генерация отчётов (ВЫСОКИЙ ПРИОРИТЕТ)

| Метод                              | Описание             | LOC  | Сложность |
|------------------------------------|----------------------|------|-----------|
| `GenerateExpenseReport()`          | Анализ расходов      | ~116 | HIGH      |
| `GenerateIncomeReport()`           | Анализ доходов       | ~82  | HIGH      |
| `GenerateCashFlowReport()`         | Денежный поток       | ~66  | HIGH      |
| `GenerateBudgetComparisonReport()` | Сравнение с бюджетом | ~72  | HIGH      |

## Ключевые сценарии

### 1. Генерация отчёта по расходам

```go
func TestReportRepository_GenerateExpenseReport(t *testing.T) {
tests := []struct {
name         string
transactions []domain.Transaction
dateRange    DateRange
expected     ExpenseReport
}{
{
name: "multiple categories",
transactions: []domain.Transaction{
{CategoryID: 1, Amount: 500, Type: "expense"},
{CategoryID: 1, Amount: 300, Type: "expense"},
{CategoryID: 2, Amount: 200, Type: "expense"},
},
expected: ExpenseReport{
Total: 1000,
ByCategory: map[int64]float64{
1: 800, // 80%
2: 200, // 20%
},
},
},
{"single category", ...},
{"no expenses", ...},
{"date range filter", ...},
}
}
```

### 2. Генерация отчёта по доходам

```go
func TestReportRepository_GenerateIncomeReport(t *testing.T) {
// - Несколько источников дохода
// - Один источник
// - Нет доходов
// - Фильтрация по датам
// - Группировка по источникам
}
```

### 3. Отчёт по денежному потоку

```go
func TestReportRepository_GenerateCashFlowReport(t *testing.T) {
tests := []struct {
name     string
income   float64
expenses float64
expected CashFlowReport
}{
{"positive flow", 5000, 3000, CashFlowReport{Net: 2000}},
{"negative flow", 3000, 5000, CashFlowReport{Net: -2000}},
{"zero flow", 3000, 3000, CashFlowReport{Net: 0}},
}
// Также:
// - Помесячная разбивка
// - Сравнение периодов
// - Входящий/исходящий баланс
}
```

### 4. Сравнение с бюджетом

```go
func TestReportRepository_GenerateBudgetComparisonReport(t *testing.T) {
tests := []struct {
name     string
budget   float64
actual   float64
expected BudgetComparison
}{
{
name:   "under budget",
budget: 1000,
actual: 800,
expected: BudgetComparison{
Variance:    200,
VariancePct: 20.0,
Status:      "under",
},
},
{"over budget", 1000, 1200, ...},
{"exactly on budget", 1000, 1000, ...},
}
}
```

## Тестовые данные

### Структура seed данных

```go
type reportTestData struct {
family       *domain.Family
users        []*domain.User
categories   []*domain.Category
transactions []*domain.Transaction
budgets      []*domain.Budget
}

// Примерные транзакции:
// | Дата       | Категория     | Сумма   | Тип     |
// |------------|---------------|---------|---------|
// | 2024-01-05 | Зарплата      | 5000.00 | income  |
// | 2024-01-10 | Еда           | 500.00  | expense |
// | 2024-01-15 | Транспорт     | 150.00  | expense |
// | 2024-02-05 | Зарплата      | 5000.00 | income  |
// | 2024-02-12 | Еда           | 600.00  | expense |
```

## Верификация расчётов

### Expense Report

- Total = сумма всех категорий
- Category % = (category_total / grand_total) * 100
- Иерархия категорий агрегирована

### Cash Flow

- Net = Total Income - Total Expenses
- Месячные итоги агрегируются корректно
- Дельта между периодами верна

### Budget Comparison

- Variance = Budget Amount - Actual Spent
- Variance % = (Variance / Budget Amount) * 100
- Флаги over/under корректны

## Edge Cases

1. **Пустые данные** - нет транзакций в периоде
2. **Одна транзакция** - минимум данных
3. **Большой датасет** - 1000+ транзакций
4. **Точность decimal** - валютные расчёты
5. **Границы дат** - первый/последний день
6. **Null категории** - транзакции без категории

## Критерии приёмки

- [ ] Все 11 методов покрыты тестами
- [ ] Алгоритмы верифицированы известными данными
- [ ] SQL агрегации совпадают с ручными расчётами
- [ ] Пагинация работает корректно
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 003: Dashboard Handler Tests
- Task 010: Reports Handler Tests

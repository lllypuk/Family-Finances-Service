# Task 001: Budget Repository Tests (CRITICAL)

## Priority: CRITICAL

## Status: ✅ COMPLETED

## Estimated LOC: ~770

## Overview

Budget Repository (`internal/infrastructure/budget/budget_repository_sqlite.go`) имеет **0% покрытия** и содержит
критическую логику финансовых расчётов.

## Файлы

- **Source**: `internal/infrastructure/budget/budget_repository_sqlite.go`
- **Test**: `internal/infrastructure/budget/budget_repository_test.go` (создать)

## Методы для тестирования

### CRUD операции

| Метод       | Описание         | Тест-кейсы                                      |
|-------------|------------------|-------------------------------------------------|
| `Create()`  | Создание бюджета | Валидные данные, дубликат, невалидный family ID |
| `GetByID()` | Получение по ID  | Существующий, несуществующий, удалённый         |
| `GetAll()`  | Все бюджеты      | Пустой список, несколько, пагинация             |
| `Update()`  | Обновление       | Валидное, несуществующий ID, конкурентное       |
| `Delete()`  | Удаление         | Существующий, несуществующий, каскад            |

### Бизнес-логика

| Метод                                | Описание                 | Тест-кейсы                         |
|--------------------------------------|--------------------------|------------------------------------|
| `GetActiveBudgets()`                 | Активные бюджеты         | Только активные, смешанные статусы |
| `GetUsageStats()`                    | Статистика использования | 0%, 50%, 100%, >100%               |
| `UpdateSpentAmount()`                | Обновление расходов      | Положительные, отрицательные       |
| `RecalculateSpent()`                 | Пересчёт расходов        | Агрегация, границы дат             |
| `FindBudgetsAffectedByTransaction()` | Влияние транзакции       | Один/несколько бюджетов            |
| `GetByCategory()`                    | По категории             | Валидная/невалидная категория      |
| `GetByPeriod()`                      | По периоду               | Месяц, год, кастомный              |

### Алерты

| Метод           | Описание          | Тест-кейсы                           |
|-----------------|-------------------|--------------------------------------|
| `GetAlerts()`   | Получение алертов | Нет алертов, несколько, фильтрация   |
| `CreateAlert()` | Создание алерта   | Валидный, дубликат, валидация порога |

## Ключевые сценарии

### 1. Расчёт статистики бюджета

```go
func TestBudgetRepository_GetUsageStats(t *testing.T) {
tests := []struct {
name           string
budgetAmount   float64
transactions   []float64
expectedUsage  float64
}{
{"0% usage", 1000, []float64{}, 0},
{"50% usage", 1000, []float64{500}, 50},
{"100% usage", 1000, []float64{600, 400}, 100},
{"over budget", 1000, []float64{800, 500}, 130},
}
}
```

### 2. Пересчёт потраченной суммы

```go
func TestBudgetRepository_RecalculateSpent(t *testing.T) {
// - Одна транзакция в периоде
// - Несколько транзакций (агрегация)
// - Транзакции вне периода (исключены)
// - Граничные условия дат
}
```

### 3. Поиск затронутых бюджетов

```go
func TestBudgetRepository_FindBudgetsAffectedByTransaction(t *testing.T) {
// - Транзакция влияет на один бюджет
// - Транзакция влияет на несколько
// - Транзакция не влияет ни на один
// - Иерархия категорий
}
```

## Настройка тестов

```go
func setupBudgetTestDB(t *testing.T) (*sql.DB, func ()) {
db, err := sql.Open("sqlite", ":memory:")
require.NoError(t, err)

// Миграции и сид данные
seedTestFamily(t, db)
seedTestCategories(t, db)

return db, func() { db.Close() }
}
```

## Edge Cases

1. **Нулевая сумма бюджета** - защита от деления на ноль
2. **Точность decimal** - валютные расчёты (2 знака)
3. **Границы периода** - первый/последний день месяца
4. **Конкурентные обновления** - race conditions
5. **Soft delete** - исключение из запросов

## Критерии приёмки

- [x] Все 14 методов покрыты тестами
- [x] Edge cases покрыты
- [x] Финансовые расчёты верифицированы
- [x] Coverage > 80% (достигнуто 81.8%)
- [x] `make test` проходит
- [x] `make lint` проходит

## Итоги выполнения

- **Файл**: `internal/infrastructure/budget/budget_repository_test.go`
- **Количество строк**: 1413
- **Покрытие**: 81.8%
- **Количество тестов**: 58 test cases
- **Методы покрыты**: 14/14 (100%)

### Реализованные тестовые сценарии:

#### CRUD операции (7 тестов)
- Create: Success_ValidData, Success_WithCategory, Error_DuplicateName, Error_InvalidBudgetID, Error_InvalidPeriod, Error_InvalidAmount, Error_InvalidDateRange
- GetByID: Success_ExistingBudget, Error_NonExistentBudget, Error_InvalidID
- GetAll: Success_EmptyList, Success_MultipleBudgets, Success_OnlyActiveBudgets
- Update: Success_ValidUpdate, Error_NonExistentBudget, Error_InvalidID
- Delete: Success_SoftDelete, Error_NonExistentBudget, Error_InvalidID

#### Бизнес-логика (24 теста)
- GetActiveBudgets: Success_OnlyActiveInDateRange, Success_ExcludesInactive
- GetUsageStats: Success_ZeroPercent, Success_FiftyPercent, Success_OneHundredPercent, Success_OverBudget, EdgeCase_ZeroAmount
- UpdateSpentAmount: Success_PositiveAmount, Error_NegativeAmount, Error_InvalidBudgetID, Error_NonExistentBudget
- RecalculateSpent: Success_OneTransaction, Success_MultipleTransactions, Success_NoTransactions, Error_InvalidBudgetID, Error_NonExistentBudget
- FindBudgetsAffectedByTransaction: Success_OneBudgetAffected, Success_MultipleBudgetsAffected, Success_NoBudgetsAffected, Error_InvalidFamilyID, Error_InvalidCategoryID
- GetByCategory: Success_ValidCategory, Success_NilCategory, Success_NonExistentCategory, Error_InvalidCategoryID
- GetByPeriod: Success_MonthPeriod, Success_CustomPeriod, Success_NoBudgetsInPeriod

#### Алерты (9 тестов)
- GetAlerts: Success_NoAlerts, Success_MultipleAlerts, Error_InvalidBudgetID
- CreateAlert: Success_ValidAlert, Error_DuplicateThreshold, Error_InvalidAlertID, Error_InvalidBudgetID, Error_InvalidThresholdZero, Error_InvalidThresholdOver100

### Edge Cases протестированы:
- ✅ Нулевая сумма бюджета (защита от деления на ноль)
- ✅ Точность decimal (валютные расчёты)
- ✅ Границы периода (тестирование дат)
- ✅ Конкурентные обновления (проверка RowsAffected)
- ✅ Soft delete (проверка is_active флага)

## Связанные задачи

- Task 003: Dashboard Handler Tests
- Task 006: Budgets Handler Tests

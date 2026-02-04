# Task 004: Transactions Handler Tests (CRITICAL)

## Priority: CRITICAL

## Status: Pending

## Estimated LOC: ~538

## Overview

Transactions Handler (`internal/web/handlers/transactions.go`) имеет **0% покрытия** и обрабатывает все операции с
финансовыми транзакциями, включая bulk delete.

## Файлы

- **Source**: `internal/web/handlers/transactions.go`
- **Test**: `internal/web/handlers/transactions_test.go` (создать)

## Методы для тестирования

### CRUD операции

| Метод      | Описание             | Тест-кейсы                        |
|------------|----------------------|-----------------------------------|
| `Index()`  | Список с фильтрами   | Пагинация, фильтры, сортировка    |
| `New()`    | Форма создания       | Рендеринг, опции категорий        |
| `Create()` | Создание             | Валидные данные, ошибки валидации |
| `Edit()`   | Форма редактирования | Существующая, не найдена          |
| `Update()` | Обновление           | Валидное, ошибки, не найдена      |
| `Delete()` | Удаление             | Существующая, не найдена          |

### Bulk операции

| Метод          | Описание          | Тест-кейсы                                            |
|----------------|-------------------|-------------------------------------------------------|
| `BulkDelete()` | Массовое удаление | Все успешно, частичный сбой, все ошибки, пустой выбор |

### HTMX Endpoints

| Метод      | Описание        | Тест-кейсы                    |
|------------|-----------------|-------------------------------|
| `Filter()` | HTMX фильтрация | Различные комбинации фильтров |
| `List()`   | HTMX пагинация  | Навигация, пустые результаты  |

## Ключевые сценарии

### 1. Список транзакций

```go
func TestTransactionsHandler_Index(t *testing.T) {
tests := []struct {
name           string
queryParams    map[string]string
setupMock      func (*MockTransactionService)
expectedStatus int
}{
{
name:           "list all",
queryParams:    nil,
expectedStatus: http.StatusOK,
},
{
name: "filter by date range",
queryParams: map[string]string{
"start_date": "2024-01-01",
"end_date":   "2024-01-31",
},
},
{
name:        "filter by category",
queryParams: map[string]string{"category_id": "1"},
},
{
name:        "filter by type income",
queryParams: map[string]string{"type": "income"},
},
{
name:        "filter by type expense",
queryParams: map[string]string{"type": "expense"},
},
{
name:        "pagination page 2",
queryParams: map[string]string{"page": "2", "per_page": "10"},
},
{
name:        "empty results",
queryParams: map[string]string{"category_id": "999"},
},
}
}
```

### 2. Создание транзакции

```go
func TestTransactionsHandler_Create(t *testing.T) {
tests := []struct {
name             string
formData         url.Values
expectedStatus   int
expectedRedirect string
checkErrors      func (t *testing.T, body string)
}{
{
name: "valid income",
formData: url.Values{
"amount":      {"1000.50"},
"type":        {"income"},
"category_id": {"1"},
"date":        {"2024-01-15"},
"description": {"Salary"},
},
expectedStatus:   http.StatusSeeOther,
expectedRedirect: "/transactions",
},
{
name: "valid expense",
formData: url.Values{
"amount":      {"50.00"},
"type":        {"expense"},
"category_id": {"2"},
"date":        {"2024-01-16"},
"description": {"Groceries"},
},
expectedStatus: http.StatusSeeOther,
},
{
name: "missing amount",
formData: url.Values{
"type":        {"income"},
"category_id": {"1"},
"date":        {"2024-01-15"},
},
expectedStatus: http.StatusOK,
checkErrors: func(t *testing.T, body string) {
assert.Contains(t, body, "amount")
},
},
{
name: "negative amount",
formData: url.Values{
"amount": {"-100"},
"type":   {"income"},
},
expectedStatus: http.StatusOK,
},
{
name: "invalid date format",
formData: url.Values{
"amount": {"100"},
"date":   {"15-01-2024"}, // неверный формат
},
expectedStatus: http.StatusOK,
},
}
}
```

### 3. Обновление транзакции

```go
func TestTransactionsHandler_Update(t *testing.T) {
tests := []struct {
name           string
transactionID  string
formData       url.Values
expectedStatus int
}{
{
name:           "update amount",
transactionID:  "1",
formData:       url.Values{"amount": {"2000.00"}},
expectedStatus: http.StatusSeeOther,
},
{
name:           "not found",
transactionID:  "999",
formData:       url.Values{"amount": {"100"}},
expectedStatus: http.StatusNotFound,
},
{
name:           "invalid data",
transactionID:  "1",
formData:       url.Values{"amount": {"invalid"}},
expectedStatus: http.StatusOK,
},
}
}
```

### 4. Массовое удаление (ВЫСОКИЙ ПРИОРИТЕТ)

```go
func TestTransactionsHandler_BulkDelete(t *testing.T) {
tests := []struct {
name           string
selectedIDs    []string
setupMock      func(*MockTransactionService)
expectedStatus int
expectedBody   string
}{
{
name:        "all success",
selectedIDs: []string{"1", "2", "3"},
setupMock: func (m *MockTransactionService) {
m.DeleteFunc = func (ctx context.Context, id int64) error {
return nil
}
},
expectedStatus: http.StatusOK,
expectedBody:   "3 transactions deleted",
},
{
name:        "partial failure",
selectedIDs: []string{"1", "2", "3"},
setupMock: func (m *MockTransactionService) {
m.DeleteFunc = func (ctx context.Context, id int64) error {
if id == 2 {
return errors.New("not found")
}
return nil
}
},
expectedStatus: http.StatusOK,
expectedBody:   "2 deleted, 1 failed",
},
{
name:        "all fail",
selectedIDs: []string{"1", "2"},
setupMock: func (m *MockTransactionService) {
m.DeleteFunc = func (ctx context.Context, id int64) error {
return errors.New("db error")
}
},
expectedStatus: http.StatusInternalServerError,
},
{
name:           "empty selection",
selectedIDs:    []string{},
expectedStatus: http.StatusBadRequest,
expectedBody:   "no transactions selected",
},
{
name:           "invalid ID format",
selectedIDs:    []string{"1", "invalid", "3"},
expectedStatus: http.StatusBadRequest,
},
}
}
```

### 5. HTMX Filter Endpoint

```go
func TestTransactionsHandler_Filter(t *testing.T) {
tests := []struct {
name           string
filterParams   map[string]string
checkHTMX      func (t *testing.T, resp *http.Response, body string)
}{
{
name:         "returns partial HTML",
filterParams: map[string]string{"type": "expense"},
checkHTMX: func (t *testing.T, resp *http.Response, body string) {
assert.NotContains(t, body, "<!DOCTYPE html>")
assert.Contains(t, body, "<tr")
},
},
}
}
```

## Mock Service

```go
type MockTransactionService struct {
GetAllFunc  func (ctx context.Context, familyID int64, opts QueryOptions) ([]domain.Transaction, int64, error)
GetByIDFunc func (ctx context.Context, id int64) (*domain.Transaction, error)
CreateFunc  func (ctx context.Context, tx *domain.Transaction) error
UpdateFunc  func (ctx context.Context, tx *domain.Transaction) error
DeleteFunc  func(ctx context.Context, id int64) error
}
```

## Edge Cases

1. **Пустой список** - новая семья без данных
2. **Большие суммы** - максимальная точность decimal
3. **Будущие даты** - транзакции с датой в будущем
4. **Конкурентный bulk delete** - race conditions
5. **Истечение сессии** - обработка ошибок
6. **Несоответствие типа категории** - income категория для expense

## Критерии приёмки

- [ ] Все 9 handler методов покрыты
- [ ] Bulk delete сценарии полностью протестированы
- [ ] HTMX responses верифицированы
- [ ] Ошибки валидации отображаются корректно
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 003: Dashboard Handler Tests
- Task 005: Categories Handler Tests

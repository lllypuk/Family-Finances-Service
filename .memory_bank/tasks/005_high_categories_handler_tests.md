# Task 005: Categories Handler Tests (HIGH)

## Priority: HIGH

## Status: Pending

## Estimated LOC: ~955

## Overview

Categories Handler (`internal/web/handlers/categories.go`) не имеет тестового файла и содержит логику управления
иерархическими категориями.

## Файлы

- **Source**: `internal/web/handlers/categories.go`
- **Test**: `internal/web/handlers/categories_test.go` (создать)

## Методы для тестирования

### CRUD операции

| Метод      | Описание             | Тест-кейсы                            |
|------------|----------------------|---------------------------------------|
| `Index()`  | Список с фильтрами   | Все категории, только родители, поиск |
| `New()`    | Форма создания       | Рендеринг, родительские опции         |
| `Create()` | Создание             | Валидные данные, дубликат имени       |
| `Edit()`   | Форма редактирования | Существующая, не найдена              |
| `Update()` | Обновление           | Валидное, циклическая ссылка          |
| `Show()`   | Детали категории     | Существующая, статистика              |
| `Delete()` | Удаление             | Пустая, с транзакциями, с детьми      |

### HTMX Endpoints

| Метод      | Описание            | Тест-кейсы                 |
|------------|---------------------|----------------------------|
| `Search()` | HTMX поиск          | По имени, пустой результат |
| `Select()` | HTMX select options | Фильтрация по типу         |

### Helper Methods

| Метод                           | Описание             |
|---------------------------------|----------------------|
| `validateUpdateRequest()`       | Валидация запроса    |
| `handleUpdateValidationError()` | Обработка ошибок     |
| `getParentOptionsForUpdate()`   | Фильтрация родителей |
| `convertToViewModels()`         | Конвертация в view   |
| `populateCategoryStats()`       | Расчёт статистики    |
| `calculateBudgetProgress()`     | Прогресс бюджета     |
| `applyNameFilter()`             | Фильтр по имени      |
| `applyParentOnlyFilter()`       | Фильтр иерархии      |
| `buildSelectOptions()`          | Построение select    |

## Ключевые сценарии

### 1. Список категорий

```go
func TestCategoriesHandler_Index(t *testing.T) {
tests := []struct {
name           string
queryParams    map[string]string
setupMock      func (*MockCategoryService)
expectedStatus int
}{
{"list all", nil, ..., http.StatusOK},
{"filter by type income", map[string]string{"type": "income"}, ...},
{"filter by type expense", map[string]string{"type": "expense"}, ...},
{"parent only", map[string]string{"parent_only": "true"}, ...},
{"search by name", map[string]string{"name": "food"}, ...},
}
}
```

### 2. Создание категории

```go
func TestCategoriesHandler_Create(t *testing.T) {
tests := []struct {
name           string
formData       url.Values
expectedStatus int
}{
{
name: "valid income category",
formData: url.Values{
"name": {"Salary"},
"type": {"income"},
},
expectedStatus: http.StatusSeeOther,
},
{
name: "valid expense with parent",
formData: url.Values{
"name":      {"Groceries"},
"type":      {"expense"},
"parent_id": {"1"},
},
expectedStatus: http.StatusSeeOther,
},
{
name: "duplicate name",
formData: url.Values{
"name": {"Existing Category"},
"type": {"expense"},
},
expectedStatus: http.StatusOK, // form with error
},
{
name: "missing name",
formData: url.Values{
"type": {"expense"},
},
expectedStatus: http.StatusOK,
},
}
}
```

### 3. Обновление категории

```go
func TestCategoriesHandler_Update(t *testing.T) {
tests := []struct {
name           string
categoryID     string
formData       url.Values
expectedStatus int
}{
{"valid update", "1", url.Values{"name": {"New Name"}}, http.StatusSeeOther},
{"not found", "999", url.Values{"name": {"Name"}}, http.StatusNotFound},
{
name:       "circular reference",
categoryID: "1",
formData:   url.Values{"parent_id": {"1"}}, // self-reference
expectedStatus: http.StatusOK, // error
},
{
name:       "child as parent",
categoryID: "1",
formData:   url.Values{"parent_id": {"2"}}, // where 2 is child of 1
expectedStatus: http.StatusOK,
},
}
}
```

### 4. Удаление категории

```go
func TestCategoriesHandler_Delete(t *testing.T) {
tests := []struct {
name           string
categoryID     string
setupMock      func (*MockCategoryService)
expectedStatus int
}{
{"empty category", "1", ..., http.StatusSeeOther},
{
name:       "with transactions",
categoryID: "2",
setupMock: func (m *MockCategoryService) {
m.DeleteFunc = func (ctx context.Context, id int64) error {
return errors.New("category has transactions")
}
},
expectedStatus: http.StatusOK, // error message
},
{
name:       "with children",
categoryID: "3",
setupMock: func (m *MockCategoryService) {
m.DeleteFunc = func (ctx context.Context, id int64) error {
return errors.New("category has children")
}
},
expectedStatus: http.StatusOK,
},
}
}
```

### 5. Статистика категории

```go
func TestCategoriesHandler_populateCategoryStats(t *testing.T) {
tests := []struct {
name       string
category   *domain.Category
stats      CategoryStats
expected   CategoryViewModel
}{
{
name: "with transactions",
stats: CategoryStats{
TransactionCount: 15,
TotalAmount:      1500.00,
},
expected: CategoryViewModel{
TransactionCount: 15,
TotalAmount:      "1,500.00",
},
},
{"no transactions", ...},
{"with budget", ...},
}
}
```

### 6. HTMX Select Options

```go
func TestCategoriesHandler_Select(t *testing.T) {
tests := []struct {
name           string
queryParams    map[string]string
expectedStatus int
checkBody      func (t *testing.T, body string)
}{
{
name:        "income categories",
queryParams: map[string]string{"type": "income"},
checkBody: func (t *testing.T, body string) {
assert.Contains(t, body, "<option")
assert.NotContains(t, body, "<!DOCTYPE")
},
},
{"expense categories", ...},
{"exclude self for parent", ...},
}
}
```

## Edge Cases

1. **Циклические ссылки** - категория не может быть своим родителем
2. **Глубокая иерархия** - категория -> подкатегория -> подподкатегория
3. **Удаление с данными** - категория с транзакциями или детьми
4. **Тип категории** - income/expense не изменяется после создания
5. **Уникальность имени** - в рамках семьи и типа

## Критерии приёмки

- [ ] Все 9 handler методов покрыты
- [ ] Helper методы протестированы
- [ ] Иерархия категорий верифицирована
- [ ] HTMX responses корректны
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 004: Transactions Handler Tests
- Task 006: Budgets Handler Tests

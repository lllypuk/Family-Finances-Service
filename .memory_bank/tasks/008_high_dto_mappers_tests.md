# Task 008: DTO Mappers Tests (HIGH)

## Priority: HIGH

## Status: Pending

## Estimated Files: 8

## Overview

DTO Mappers (`internal/services/dto/`) имеют **0% покрытия**. Эти файлы содержат логику трансформации данных между
слоями.

## Файлы для тестирования

| Файл                 | Описание               | Тест файл                 |
|----------------------|------------------------|---------------------------|
| `api_mappers.go`     | API DTO маппинг        | `api_mappers_test.go`     |
| `web_mappers.go`     | Web-to-domain маппинг  | `web_mappers_test.go`     |
| `mappers.go`         | Общие утилиты маппинга | `mappers_test.go`         |
| `category_dto.go`    | Category DTOs          | `category_dto_test.go`    |
| `budget_dto.go`      | Budget DTOs            | `budget_dto_test.go`      |
| `report_dto.go`      | Report DTOs            | `report_dto_test.go`      |
| `transaction_dto.go` | Transaction DTOs       | `transaction_dto_test.go` |
| `user_dto.go`        | User DTOs              | `user_dto_test.go`        |

## Ключевые сценарии

### 1. API Mappers

```go
func TestAPIMappers_TransactionToResponse(t *testing.T) {
tests := []struct {
name     string
domain   *domain.Transaction
expected *api.TransactionResponse
}{
{
name: "full transaction",
domain: &domain.Transaction{
ID:          1,
Amount:      1500.50,
Type:        "income",
Description: "Salary",
Date:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
CategoryID:  1,
},
expected: &api.TransactionResponse{
ID:          1,
Amount:      1500.50,
Type:        "income",
Description: "Salary",
Date:        "2024-01-15",
CategoryID:  1,
},
},
{
name: "minimal transaction",
domain: &domain.Transaction{
ID:     2,
Amount: 100,
Type:   "expense",
},
expected: &api.TransactionResponse{
ID:     2,
Amount: 100,
Type:   "expense",
},
},
{
name:     "nil input",
domain:   nil,
expected: nil,
},
}
}

func TestAPIMappers_RequestToTransaction(t *testing.T) {
tests := []struct {
name     string
request  *api.CreateTransactionRequest
expected *domain.Transaction
wantErr  bool
}{
{
name: "valid request",
request: &api.CreateTransactionRequest{
Amount:      100.00,
Type:        "expense",
CategoryID:  1,
Date:        "2024-01-15",
Description: "Groceries",
},
wantErr: false,
},
{
name: "invalid date format",
request: &api.CreateTransactionRequest{
Date: "15-01-2024",
},
wantErr: true,
},
}
}
```

### 2. Web Mappers

```go
func TestWebMappers_FormToTransaction(t *testing.T) {
tests := []struct {
name     string
form     *forms.TransactionForm
expected *domain.Transaction
}{
{
name: "income form",
form: &forms.TransactionForm{
Amount:      "1000.50",
Type:        "income",
CategoryID:  "1",
Date:        "2024-01-15",
Description: "Salary",
},
expected: &domain.Transaction{
Amount:      1000.50,
Type:        "income",
CategoryID:  1,
Description: "Salary",
},
},
{
name: "expense form",
form: &forms.TransactionForm{
Amount:     "50.00",
Type:       "expense",
CategoryID: "2",
},
expected: &domain.Transaction{
Amount:     50.00,
Type:       "expense",
CategoryID: 2,
},
},
}
}

func TestWebMappers_TransactionToViewModel(t *testing.T) {
tests := []struct {
name     string
domain   *domain.Transaction
category *domain.Category
expected *models.TransactionViewModel
}{
{
name: "with category",
domain: &domain.Transaction{
ID:     1,
Amount: 500,
Type:   "expense",
},
category: &domain.Category{
ID:   1,
Name: "Food",
},
expected: &models.TransactionViewModel{
ID:           1,
Amount:       "500.00",
Type:         "expense",
CategoryName: "Food",
},
},
}
}
```

### 3. Category DTOs

```go
func TestCategoryDTO_ToDomain(t *testing.T) {
tests := []struct {
name     string
dto      *dto.CategoryDTO
expected *domain.Category
}{
{
name: "parent category",
dto: &dto.CategoryDTO{
Name: "Food",
Type: "expense",
},
expected: &domain.Category{
Name:     "Food",
Type:     "expense",
ParentID: nil,
},
},
{
name: "child category",
dto: &dto.CategoryDTO{
Name:     "Groceries",
Type:     "expense",
ParentID: ptr(int64(1)),
},
expected: &domain.Category{
Name:     "Groceries",
Type:     "expense",
ParentID: ptr(int64(1)),
},
},
}
}
```

### 4. Budget DTOs

```go
func TestBudgetDTO_ToDomain(t *testing.T) {
tests := []struct {
name     string
dto      *dto.BudgetDTO
expected *domain.Budget
}{
{
name: "monthly budget",
dto: &dto.BudgetDTO{
CategoryID: 1,
Amount:     1000.00,
Period:     "monthly",
StartDate:  "2024-01-01",
},
expected: &domain.Budget{
CategoryID: 1,
Amount:     1000.00,
Period:     "monthly",
},
},
{
name: "yearly budget with dates",
dto: &dto.BudgetDTO{
CategoryID: 1,
Amount:     12000.00,
Period:     "yearly",
StartDate:  "2024-01-01",
EndDate:    "2024-12-31",
},
},
}
}
```

### 5. Report DTOs

```go
func TestReportDTO_ToDomain(t *testing.T) {
tests := []struct {
name     string
dto      *dto.ReportDTO
expected *domain.Report
}{
{
name: "expense report",
dto: &dto.ReportDTO{
Type:      "expense",
StartDate: "2024-01-01",
EndDate:   "2024-01-31",
},
},
{
name: "cash flow report",
dto: &dto.ReportDTO{
Type:      "cash_flow",
StartDate: "2024-01-01",
EndDate:   "2024-03-31",
},
},
}
}
```

### 6. User DTOs

```go
func TestUserDTO_ToDomain(t *testing.T) {
tests := []struct {
name     string
dto      *dto.UserDTO
expected *domain.User
}{
{
name: "admin user",
dto: &dto.UserDTO{
Email: "admin@test.com",
Name:  "Admin User",
Role:  "admin",
},
expected: &domain.User{
Email: "admin@test.com",
Name:  "Admin User",
Role:  "admin",
},
},
{
name: "member user",
dto: &dto.UserDTO{
Email: "member@test.com",
Name:  "Member User",
Role:  "member",
},
},
}
}
```

## Edge Cases

1. **Nil входные данные** - обработка nil
2. **Пустые строки** - конвертация пустых значений
3. **Невалидные даты** - обработка ошибок парсинга
4. **Decimal точность** - сохранение точности при конвертации
5. **Optional поля** - обработка указателей
6. **Slice маппинг** - пустые и большие коллекции

## Критерии приёмки

- [ ] Все 8 файлов покрыты тестами
- [ ] Все направления маппинга (Domain <-> DTO <-> API/Web)
- [ ] Edge cases покрыты
- [ ] Точность decimal верифицирована
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 007: Users Handler Tests
- Task 009: Validation Helpers Tests

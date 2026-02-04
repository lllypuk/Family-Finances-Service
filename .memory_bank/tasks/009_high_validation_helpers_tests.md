# Task 009: Validation Helpers Tests (HIGH)

## Priority: HIGH

## Status: Pending

## Overview

Validation Helpers (`internal/infrastructure/validation/validation.go`) имеют **0% покрытия**. Валидация данных критична
для безопасности.

## Файлы

- **Source**: `internal/infrastructure/validation/validation.go`
- **Test**: `internal/infrastructure/validation/validation_test.go` (создать)

## Функции для тестирования

### Валидация Email

```go
func TestValidation_ValidateEmail(t *testing.T) {
tests := []struct {
name    string
email   string
wantErr bool
}{
// Валидные
{"simple email", "user@example.com", false},
{"with subdomain", "user@mail.example.com", false},
{"with plus", "user+tag@example.com", false},
{"with dots", "first.last@example.com", false},

// Невалидные
{"empty", "", true},
{"no at sign", "userexample.com", true},
{"no domain", "user@", true},
{"no local part", "@example.com", true},
{"double at", "user@@example.com", true},
{"spaces", "user @example.com", true},
{"special chars", "user<>@example.com", true},
}
}
```

### Валидация пароля

```go
func TestValidation_ValidatePassword(t *testing.T) {
tests := []struct {
name    string
password string
wantErr bool
errMsg  string
}{
// Валидные
{"strong password", "SecurePass123!", false, ""},
{"minimum valid", "Abcd123!", false, ""},

// Невалидные
{"too short", "Ab1!", true, "at least 8 characters"},
{"no uppercase", "abcd1234!", true, "uppercase letter"},
{"no lowercase", "ABCD1234!", true, "lowercase letter"},
{"no digit", "AbcdEfgh!", true, "digit"},
{"no special", "AbcdEfgh1", true, "special character"},
{"empty", "", true, "required"},
{"only spaces", "        ", true, ""},
}
}
```

### Валидация суммы (Amount)

```go
func TestValidation_ValidateAmount(t *testing.T) {
tests := []struct {
name    string
amount  float64
wantErr bool
}{
// Валидные
{"positive integer", 100.00, false},
{"positive decimal", 99.99, false},
{"small amount", 0.01, false},
{"large amount", 999999.99, false},

// Невалидные
{"zero", 0, true},
{"negative", -100, true},
{"too many decimals", 100.999, true}, // если требуется 2 знака
}
}
```

### Валидация даты

```go
func TestValidation_ValidateDate(t *testing.T) {
tests := []struct {
name    string
date    string
wantErr bool
}{
// Валидные
{"ISO format", "2024-01-15", false},
{"first day", "2024-01-01", false},
{"last day", "2024-12-31", false},
{"leap year", "2024-02-29", false},

// Невалидные
{"empty", "", true},
{"wrong format dd-mm-yyyy", "15-01-2024", true},
{"wrong format mm/dd/yyyy", "01/15/2024", true},
{"invalid month", "2024-13-01", true},
{"invalid day", "2024-01-32", true},
{"invalid leap year", "2023-02-29", true},
}
}
```

### Валидация периода бюджета

```go
func TestValidation_ValidateBudgetPeriod(t *testing.T) {
tests := []struct {
name    string
period  string
wantErr bool
}{
{"monthly", "monthly", false},
{"yearly", "yearly", false},
{"weekly", "weekly", false},
{"custom", "custom", false},

{"empty", "", true},
{"invalid", "biweekly", true},
{"uppercase", "MONTHLY", true}, // если case-sensitive
}
}
```

### Валидация типа транзакции

```go
func TestValidation_ValidateTransactionType(t *testing.T) {
tests := []struct {
name    string
txType  string
wantErr bool
}{
{"income", "income", false},
{"expense", "expense", false},

{"empty", "", true},
{"invalid", "transfer", true},
{"uppercase", "INCOME", true},
}
}
```

### Валидация роли пользователя

```go
func TestValidation_ValidateUserRole(t *testing.T) {
tests := []struct {
name    string
role    string
wantErr bool
}{
{"admin", "admin", false},
{"member", "member", false},
{"child", "child", false},

{"empty", "", true},
{"invalid", "superadmin", true},
{"uppercase", "ADMIN", true},
}
}
```

### Валидация порога алерта

```go
func TestValidation_ValidateAlertThreshold(t *testing.T) {
tests := []struct {
name      string
threshold float64
wantErr   bool
}{
{"50%", 50, false},
{"80%", 80, false},
{"100%", 100, false},
{"1%", 1, false},

{"0%", 0, true},
{"negative", -10, true},
{"over 100%", 150, true},
}
}
```

### Валидация имени категории

```go
func TestValidation_ValidateCategoryName(t *testing.T) {
tests := []struct {
name        string
categoryName string
wantErr     bool
}{
{"simple", "Food", false},
{"with spaces", "Public Transport", false},
{"with numbers", "Category 1", false},

{"empty", "", true},
{"too long", strings.Repeat("a", 256), true},
{"only spaces", "   ", true},
{"special chars", "Food<>", true},
}
}
```

## Интеграция с go-playground/validator

```go
func TestValidation_StructValidation(t *testing.T) {
type CreateTransactionRequest struct {
Amount     float64 `validate:"required,gt=0"`
Type       string  `validate:"required,oneof=income expense"`
Date       string  `validate:"required,datetime=2006-01-02"`
CategoryID int64   `validate:"required,gt=0"`
}

tests := []struct {
name    string
request CreateTransactionRequest
wantErr bool
field   string
}{
{
name: "valid",
request: CreateTransactionRequest{
Amount:     100,
Type:       "expense",
Date:       "2024-01-15",
CategoryID: 1,
},
wantErr: false,
},
{
name: "missing amount",
request: CreateTransactionRequest{
Type:       "expense",
Date:       "2024-01-15",
CategoryID: 1,
},
wantErr: true,
field:   "Amount",
},
}
}
```

## Edge Cases

1. **Unicode символы** - имена с не-ASCII символами
2. **SQL injection** - попытки инъекции
3. **XSS** - скрипты в строках
4. **Boundary values** - граничные значения
5. **Whitespace** - пробелы в начале/конце
6. **Null bytes** - нулевые байты в строках

## Критерии приёмки

- [ ] Все функции валидации покрыты
- [ ] Позитивные и негативные сценарии
- [ ] Граничные значения протестированы
- [ ] Security edge cases покрыты
- [ ] Coverage > 90% (валидация критична)
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 008: DTO Mappers Tests
- Task 010: Web Models Tests

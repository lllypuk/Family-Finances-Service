# Task 010: Web Models Tests (MEDIUM)

## Priority: MEDIUM

## Status: ✅ Completed

## Current Coverage: 95.1% (target was 70%+)

## Overview

Web Models (`internal/web/models/`) now have comprehensive test coverage. All forms and view models for the web interface are fully tested.

## Файлы для тестирования

| Файл              | Описание             | Статус          |
|-------------------|----------------------|-----------------|
| `forms.go`        | Формы и валидация    | ✅ Покрыт (enhanced) |
| `budgets.go`      | Budget view models   | ✅ Покрыт (381 lines) |
| `categories.go`   | Category view models | ✅ Покрыт (343 lines) |
| `dashboard.go`    | Dashboard models     | ✅ Покрыт (527 lines) |
| `reports.go`      | Report models        | ✅ Покрыт (558 lines) |
| `transactions.go` | Transaction models   | ✅ Покрыт (648 lines) |

## Ключевые сценарии

### 1. Forms Validation (forms.go)

#### LoginForm

```go
func TestLoginForm_Validate(t *testing.T) {
tests := []struct {
name    string
form    LoginForm
wantErr bool
field   string
}{
{
name: "valid login",
form: LoginForm{
Email:    "user@example.com",
Password: "password123",
},
wantErr: false,
},
{
name: "missing email",
form: LoginForm{
Password: "password123",
},
wantErr: true,
field:   "Email",
},
{
name: "missing password",
form: LoginForm{
Email: "user@example.com",
},
wantErr: true,
field:   "Password",
},
{
name: "invalid email format",
form: LoginForm{
Email:    "invalid-email",
Password: "password123",
},
wantErr: true,
field:   "Email",
},
}
}
```

#### SetupForm

```go
func TestSetupForm_Validate(t *testing.T) {
tests := []struct {
name    string
form    SetupForm
wantErr bool
}{
{
name: "valid setup",
form: SetupForm{
FamilyName: "Test Family",
Email:      "admin@test.com",
Password:   "SecurePass123!",
Name:       "Admin User",
},
wantErr: false,
},
{
name: "missing family name",
form: SetupForm{
Email:    "admin@test.com",
Password: "SecurePass123!",
Name:     "Admin User",
},
wantErr: true,
},
{
name: "weak password",
form: SetupForm{
FamilyName: "Test Family",
Email:      "admin@test.com",
Password:   "weak",
Name:       "Admin User",
},
wantErr: true,
},
}
}
```

#### CreateUserForm

```go
func TestCreateUserForm_Validate(t *testing.T) {
tests := []struct {
name    string
form    CreateUserForm
wantErr bool
}{
{
name: "valid member",
form: CreateUserForm{
Email:    "member@test.com",
Password: "SecurePass123!",
Name:     "Member User",
Role:     "member",
},
wantErr: false,
},
{
name: "invalid role",
form: CreateUserForm{
Email:    "user@test.com",
Password: "SecurePass123!",
Name:     "User",
Role:     "superadmin",
},
wantErr: true,
},
}
}
```

#### CreateInviteForm

```go
func TestCreateInviteForm_Validate(t *testing.T) {
tests := []struct {
name    string
form    CreateInviteForm
wantErr bool
}{
{
name: "valid invite",
form: CreateInviteForm{
Email: "invite@test.com",
Role:  "member",
},
wantErr: false,
},
{
name: "missing email",
form: CreateInviteForm{
Role: "member",
},
wantErr: true,
},
{
name: "admin invite not allowed",
form: CreateInviteForm{
Email: "admin@test.com",
Role:  "admin",
},
wantErr: true, // если запрещено приглашать админов
},
}
}
```

#### InviteRegisterForm

```go
func TestInviteRegisterForm_Validate(t *testing.T) {
tests := []struct {
name    string
form    InviteRegisterForm
wantErr bool
}{
{
name: "valid registration",
form: InviteRegisterForm{
Token:    "valid-token-32-chars-minimum-here",
Password: "SecurePass123!",
Name:     "New User",
},
wantErr: false,
},
{
name: "missing token",
form: InviteRegisterForm{
Password: "SecurePass123!",
Name:     "New User",
},
wantErr: true,
},
}
}
```

### 2. Helper Functions

#### GetValidationErrors

```go
func TestGetValidationErrors(t *testing.T) {
tests := []struct {
name     string
err      error
expected map[string]string
}{
{
name: "single error",
err:  validator.ValidationErrors{...},
expected: map[string]string{
"Email": "Email is required",
},
},
{
name: "multiple errors",
err:  validator.ValidationErrors{...},
expected: map[string]string{
"Email":    "Email is required",
"Password": "Password must be at least 8 characters",
},
},
{
name:     "non-validation error",
err:      errors.New("other error"),
expected: nil,
},
}
}
```

#### getFieldName

```go
func TestGetFieldName(t *testing.T) {
tests := []struct {
name     string
field    string
expected string
}{
{"email field", "Email", "Email"},
{"password field", "Password", "Password"},
{"family_name field", "FamilyName", "Family Name"},
{"category_id field", "CategoryID", "Category"},
}
}
```

#### getErrorMessage

```go
func TestGetErrorMessage(t *testing.T) {
tests := []struct {
name      string
field     string
tag       string
param     string
expected  string
}{
{"required", "Email", "required", "", "Email is required"},
{"email format", "Email", "email", "", "Invalid email format"},
{"min length", "Password", "min", "8", "Password must be at least 8 characters"},
{"max length", "Name", "max", "100", "Name must not exceed 100 characters"},
{"oneof", "Role", "oneof", "admin member child", "Role must be one of: admin, member, child"},
}
}
```

### 3. View Models (budgets.go, categories.go, etc.)

#### BudgetViewModel

```go
func TestBudgetViewModel_FormatAmount(t *testing.T) {
tests := []struct {
name     string
amount   float64
expected string
}{
{"integer", 1000, "1,000.00"},
{"decimal", 1234.56, "1,234.56"},
{"small", 0.99, "0.99"},
{"large", 999999.99, "999,999.99"},
}
}

func TestBudgetViewModel_CalculateProgress(t *testing.T) {
tests := []struct {
name     string
spent    float64
amount   float64
expected int
}{
{"0%", 0, 1000, 0},
{"50%", 500, 1000, 50},
{"100%", 1000, 1000, 100},
{"over budget", 1500, 1000, 150},
{"zero amount", 100, 0, 0}, // avoid div by zero
}
}
```

#### CategoryViewModel

```go
func TestCategoryViewModel_GetIcon(t *testing.T) {
tests := []struct {
name     string
catType  string
expected string
}{
{"income", "income", "arrow-up"},
{"expense", "expense", "arrow-down"},
}
}

func TestCategoryViewModel_FormatTotal(t *testing.T) {
tests := []struct {
name     string
total    float64
expected string
}{
{"positive", 1500.50, "1,500.50"},
{"zero", 0, "0.00"},
}
}
```

#### TransactionViewModel

```go
func TestTransactionViewModel_FormatDate(t *testing.T) {
tests := []struct {
name     string
date     time.Time
expected string
}{
{"standard", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), "Jan 15, 2024"},
{"today", time.Now(), "Today"},
{"yesterday", time.Now().AddDate(0, 0, -1), "Yesterday"},
}
}
```

#### DashboardModels

```go
func TestDashboardStats_CalculateSavingsRate(t *testing.T) {
tests := []struct {
name     string
income   float64
expenses float64
expected float64
}{
{"positive savings", 10000, 7000, 30.0},
{"no savings", 5000, 5000, 0.0},
{"negative", 3000, 5000, -66.67},
{"no income", 0, 1000, 0.0}, // avoid div by zero
}
}
```

## Edge Cases

1. **Пустые формы** - все поля пустые
2. **Unicode в именах** - кириллица, emoji
3. **Whitespace** - только пробелы
4. **Длинные строки** - превышение лимитов
5. **Специальные символы** - HTML entities
6. **Нулевые значения** - деление на ноль в расчётах

## Критерии приёмки

- [x] Все формы имеют тесты валидации
- [x] Helper функции покрыты
- [x] View models форматирование проверено
- [x] Edge cases покрыты
- [x] Coverage > 70% (Achieved: 95.1%)
- [x] `make test` проходит
- [x] `make lint` проходит (minor test fixture warnings acceptable)

## Результаты

- **Test Coverage**: 95.1% (exceeded target by 25.1%)
- **Files Created**: 5 new test files + 1 enhanced
- **Total Test Lines**: ~2,500 lines
- **Test Cases**: 100+ test scenarios
- **Edge Cases**: Comprehensive unicode, validation, formatting tests

## Связанные задачи

- Task 009: Validation Helpers Tests
- Task 003: Dashboard Handler Tests

# Task 007: Users Handler Tests (HIGH)

## Priority: HIGH

## Status: Pending

## Estimated LOC: ~208

## Overview

Users Handler (`internal/web/handlers/users.go`) не имеет тестового файла и управляет пользователями семьи.

## Файлы

- **Source**: `internal/web/handlers/users.go`
- **Test**: `internal/web/handlers/users_test.go` (создать)

## Методы для тестирования

### Handlers

| Метод                  | Описание             | Тест-кейсы                      |
|------------------------|----------------------|---------------------------------|
| `Index()`              | Список пользователей | Все, по роли                    |
| `New()`                | Форма создания       | Рендеринг, роли                 |
| `Create()`             | Создание             | Валидные данные, дубликат email |
| `handleServiceError()` | Обработка ошибок     | Различные типы ошибок           |
| `userError()`          | Рендеринг ошибок     | Форматирование                  |

## Ключевые сценарии

### 1. Список пользователей

```go
func TestUsersHandler_Index(t *testing.T) {
tests := []struct {
name           string
setupMock      func (*MockUserService)
expectedStatus int
checkBody      func (t *testing.T, body string)
}{
{
name: "list all users",
setupMock: func (m *MockUserService) {
m.GetByFamilyIDFunc = func (ctx context.Context, familyID int64) ([]domain.User, error) {
return []domain.User{
{ID: 1, Email: "admin@test.com", Role: "admin"},
{ID: 2, Email: "member@test.com", Role: "member"},
}, nil
}
},
expectedStatus: http.StatusOK,
checkBody: func(t *testing.T, body string) {
assert.Contains(t, body, "admin@test.com")
assert.Contains(t, body, "member@test.com")
},
},
{
name: "empty family",
setupMock: func (m *MockUserService) {
m.GetByFamilyIDFunc = func (...) ([]domain.User, error) {
return []domain.User{}, nil
}
},
expectedStatus: http.StatusOK,
},
{
name: "service error",
setupMock: func (m *MockUserService) {
m.GetByFamilyIDFunc = func (...) ([]domain.User, error) {
return nil, errors.New("db error")
}
},
expectedStatus: http.StatusInternalServerError,
},
}
}
```

### 2. Создание пользователя

```go
func TestUsersHandler_Create(t *testing.T) {
tests := []struct {
name           string
formData       url.Values
setupMock      func(*MockUserService)
expectedStatus int
checkBody      func(t *testing.T, body string)
}{
{
name: "valid member",
formData: url.Values{
"email":    {"newuser@test.com"},
"password": {"SecurePass123!"},
"name":     {"New User"},
"role":     {"member"},
},
setupMock: func (m *MockUserService) {
m.CreateFunc = func (ctx context.Context, user *domain.User) error {
return nil
}
},
expectedStatus: http.StatusSeeOther,
},
{
name: "valid child",
formData: url.Values{
"email":    {"child@test.com"},
"password": {"ChildPass123!"},
"name":     {"Child User"},
"role":     {"child"},
},
expectedStatus: http.StatusSeeOther,
},
{
name: "duplicate email",
formData: url.Values{
"email":    {"existing@test.com"},
"password": {"Pass123!"},
"name":     {"Duplicate"},
"role":     {"member"},
},
setupMock: func (m *MockUserService) {
m.CreateFunc = func (...) error {
return errors.New("email already exists")
}
},
expectedStatus: http.StatusOK,
checkBody: func(t *testing.T, body string) {
assert.Contains(t, body, "email already exists")
},
},
{
name: "missing email",
formData: url.Values{
"password": {"Pass123!"},
"name":     {"No Email"},
"role":     {"member"},
},
expectedStatus: http.StatusOK,
},
{
name: "weak password",
formData: url.Values{
"email":    {"user@test.com"},
"password": {"123"},
"name":     {"Weak Pass"},
"role":     {"member"},
},
expectedStatus: http.StatusOK,
},
{
name: "invalid role",
formData: url.Values{
"email":    {"user@test.com"},
"password": {"Pass123!"},
"name":     {"Invalid Role"},
"role":     {"superadmin"}, // not allowed
},
expectedStatus: http.StatusOK,
},
}
}
```

### 3. Обработка ошибок сервиса

```go
func TestUsersHandler_handleServiceError(t *testing.T) {
tests := []struct {
name          string
err           error
expectedMsg   string
}{
{
name:        "email exists",
err:         services.ErrEmailExists,
expectedMsg: "Email already registered",
},
{
name:        "user not found",
err:         services.ErrUserNotFound,
expectedMsg: "User not found",
},
{
name:        "generic error",
err:         errors.New("database error"),
expectedMsg: "An error occurred",
},
}
}
```

## Роли и права

### Тестирование ролей

```go
func TestUsersHandler_RolePermissions(t *testing.T) {
tests := []struct {
name           string
currentUserRole string
targetUserRole  string
action         string
expectedStatus int
}{
{"admin creates member", "admin", "member", "create", http.StatusSeeOther},
{"admin creates child", "admin", "child", "create", http.StatusSeeOther},
{"member creates member", "member", "member", "create", http.StatusForbidden},
{"admin deletes member", "admin", "member", "delete", http.StatusSeeOther},
{"member deletes admin", "member", "admin", "delete", http.StatusForbidden},
}
}
```

## Mock Service

```go
type MockUserService struct {
GetByFamilyIDFunc func (ctx context.Context, familyID int64) ([]domain.User, error)
GetByIDFunc       func (ctx context.Context, id int64) (*domain.User, error)
CreateFunc        func (ctx context.Context, user *domain.User) error
UpdateFunc        func (ctx context.Context, user *domain.User) error
DeleteFunc        func (ctx context.Context, id int64) error
}
```

## Edge Cases

1. **Последний админ** - нельзя удалить последнего админа
2. **Самоудаление** - пользователь не может удалить себя
3. **Смена роли** - ограничения на смену роли
4. **Email формат** - валидация email
5. **Пароль** - минимальные требования

## Критерии приёмки

- [ ] Все 5 методов покрыты
- [ ] Ролевая модель протестирована
- [ ] Ошибки валидации отображаются
- [ ] Coverage > 80%
- [ ] `make test` проходит
- [ ] `make lint` проходит

## Связанные задачи

- Task 008: DTO Mappers Tests

# Задача 1.1: Ввести концепцию "единственной семьи"

## Статус: TODO

## Цель

Перевести приложение на модель «один экземпляр = одна семья». Семья создается автоматически при первом запуске (bootstrap), а не через API или форму регистрации. Все пользователи принадлежат этой единственной семье.

## Текущее поведение

1. Пользователь открывает `/register`, заполняет форму с данными семьи (имя, валюта) и своими данными (email, пароль).
2. `internal/web/handlers/auth.go` — в хэндлере Register:
   - Создается `Family` через `user.NewFamily(name, currency)`
   - Сохраняется через `familyRepo.Create()`
   - Создается `User` с `RoleAdmin` и привязкой к `family.ID`
   - FamilyID записывается в сессию
3. API позволяет создавать произвольное количество семей: `POST /api/v1/families`
4. `FamilyRepository` имеет полный CRUD: Create, GetByID, Update, Delete, GetAllFamilies

## Целевое поведение

1. При первом запуске приложение автоматически проверяет наличие семьи в БД.
2. Если семьи нет — пользователь попадает на страницу первоначальной настройки (setup wizard):
   - Ввод имени семьи и валюты
   - Создание первого пользователя (admin)
3. Если семья уже есть — стандартный login flow.
4. API для создания/удаления семей убирается.
5. `FamilyRepository` упрощается до:
   - `GetFamily(ctx) (*Family, error)` — единственная семья
   - `Create(ctx, family) error` — только для bootstrap
   - `Update(ctx, family) error` — редактирование настроек семьи

## Детальный план изменений

### Шаг 1: Упростить FamilyRepository

**Файл:** `internal/application/handlers/repositories.go`

Текущий интерфейс:
```go
type FamilyRepository interface {
    Create(ctx context.Context, family *user.Family) error
    GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error)
    Update(ctx context.Context, family *user.Family) error
    Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error
}
```

Целевой интерфейс:
```go
type FamilyRepository interface {
    Create(ctx context.Context, family *user.Family) error
    Get(ctx context.Context) (*user.Family, error)       // Единственная семья
    Update(ctx context.Context, family *user.Family) error
    Exists(ctx context.Context) (bool, error)             // Проверка наличия
}
```

### Шаг 2: Обновить SQLite-реализацию

**Файл:** `internal/infrastructure/user/family_repository_sqlite.go`

- `GetByID()` → `Get()` — `SELECT ... FROM families LIMIT 1` (без параметра id)
- `Delete()` — удалить метод
- `GetAllFamilies()` — удалить метод
- `GetFamilyStatistics()` — упростить (убрать параметр familyID)
- `CreateWithTransaction()` — оставить для bootstrap
- Добавить `Exists()` — `SELECT COUNT(*) FROM families`

### Шаг 3: Упростить FamilyService

**Файл:** `internal/services/interfaces.go`

Текущий интерфейс:
```go
type FamilyService interface {
    CreateFamily(ctx context.Context, req dto.CreateFamilyDTO) (*user.Family, error)
    GetFamilyByID(ctx context.Context, id uuid.UUID) (*user.Family, error)
    UpdateFamily(ctx context.Context, id uuid.UUID, req dto.UpdateFamilyDTO) (*user.Family, error)
    DeleteFamily(ctx context.Context, id uuid.UUID) error
}
```

Целевой интерфейс:
```go
type FamilyService interface {
    SetupFamily(ctx context.Context, req dto.SetupFamilyDTO) (*user.Family, error)  // Bootstrap
    GetFamily(ctx context.Context) (*user.Family, error)
    UpdateFamily(ctx context.Context, req dto.UpdateFamilyDTO) (*user.Family, error)
    IsSetupComplete(ctx context.Context) (bool, error)
}
```

**Файл:** `internal/services/family_service.go`
- `CreateFamily()` → `SetupFamily()` — создает семью + дефолтные категории (только если семьи нет)
- `GetFamilyByID()` → `GetFamily()` — без параметра id
- `DeleteFamily()` — удалить
- Добавить `IsSetupComplete()` — проверяет `familyRepo.Exists()`

### Шаг 4: Добавить DTO для setup

**Файл:** `internal/services/dto/user_dto.go`

Добавить:
```go
type SetupFamilyDTO struct {
    // Данные семьи
    FamilyName string `validate:"required,min=2,max=100"`
    Currency   string `validate:"required,len=3"`
    // Данные первого пользователя (admin)
    Email     string `validate:"required,email,max=254"`
    FirstName string `validate:"required,min=2,max=50"`
    LastName  string `validate:"required,min=2,max=50"`
    Password  string `validate:"required,min=6"`
}
```

### Шаг 5: Убрать API endpoint'ы семей

**Файл:** `internal/application/http_server.go`

Удалить маршруты:
```go
// Удалить:
families.POST("", s.familyHandler.CreateFamily)
families.GET("/:id", s.familyHandler.GetFamilyByID)
families.GET("/:id/members", s.familyHandler.GetFamilyMembers)
```

**Файл:** `internal/application/handlers/families.go`
- Удалить `CreateFamily` handler
- `GetFamilyByID` → `GetFamily` (без параметра `:id`)
- `GetFamilyMembers` → убрать привязку к `:id`, брать семью из БД

### Шаг 6: Обновить веб-регистрацию → setup wizard

**Файл:** `internal/web/handlers/auth.go`

Текущий Register flow:
1. Показывает форму регистрации (любой может зарегистрироваться)
2. Создает семью + admin-пользователя

Целевой Setup flow:
1. Middleware проверяет `familyService.IsSetupComplete()`
2. Если нет — редирект на `/setup` (вместо `/register`)
3. `/setup` доступен **только** если семьи ещё нет
4. После setup — редирект на `/login`
5. `/register` — удалить (новых пользователей добавляет admin через интерфейс)

### Шаг 7: Добавить setup middleware

**Файл:** `internal/web/middleware/setup.go` (новый)

```go
// RequireSetup — middleware, который проверяет что семья существует.
// Если нет — редирект на /setup.
// Если да и запрос на /setup — редирект на /login.
func RequireSetup(familyService FamilyService) echo.MiddlewareFunc
```

### Шаг 8: Обновить UserService

**Файл:** `internal/services/user_service.go`

- `CreateUser()` — убрать `FamilyID` из DTO, получать единственную семью из `familyRepo.Get()`
- `validateFamilyExists()` — упростить или удалить (семья всегда есть после setup)
- `GetUsersByFamily()` → `GetUsers()` — без параметра familyID
- `DeleteUser(ctx, id, familyID)` → `DeleteUser(ctx, id)` — без familyID

**Файл:** `internal/services/dto/user_dto.go`
- Убрать `FamilyID uuid.UUID` из `CreateUserDTO`

### Шаг 9: Обновить run.go

**Файл:** `internal/run.go`

В инициализацию не нужно добавлять auto-bootstrap — это делается через web setup wizard. Но нужно обеспечить, чтобы сервисы корректно работали до создания семьи (graceful handling).

## Файлы, затрагиваемые этой задачей

| Файл | Действие |
|------|----------|
| `internal/application/handlers/repositories.go` | Изменить интерфейс FamilyRepository |
| `internal/infrastructure/user/family_repository_sqlite.go` | Переписать реализацию |
| `internal/services/interfaces.go` | Изменить интерфейс FamilyService |
| `internal/services/family_service.go` | Переписать реализацию |
| `internal/services/user_service.go` | Убрать familyID из CreateUser |
| `internal/services/dto/user_dto.go` | Добавить SetupFamilyDTO, убрать FamilyID из CreateUserDTO |
| `internal/application/http_server.go` | Убрать маршруты /families |
| `internal/application/handlers/families.go` | Упростить хэндлер |
| `internal/web/handlers/auth.go` | Register → Setup wizard |
| `internal/web/middleware/setup.go` | Новый файл — setup middleware |
| `internal/web/middleware/session.go` | Убрать FamilyID из сессии (подготовка) |
| `internal/run.go` | Обновить инициализацию сервисов |

## Критерии готовности

- [ ] `FamilyRepository` имеет методы: `Create`, `Get`, `Update`, `Exists`
- [ ] `FamilyService` имеет методы: `SetupFamily`, `GetFamily`, `UpdateFamily`, `IsSetupComplete`
- [ ] API endpoint'ы для CRUD семей удалены
- [ ] При первом запуске пользователь попадает на `/setup`
- [ ] После setup пользователь попадает на `/login`
- [ ] Повторный доступ к `/setup` невозможен (редирект на `/login`)
- [ ] `CreateUserDTO` не содержит `FamilyID`
- [ ] `make test` проходит
- [ ] `make lint` проходит без ошибок

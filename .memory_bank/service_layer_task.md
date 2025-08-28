# Задача: Рефакторинг в Service Layer Architecture

> **Статус**: 📋 Планирование  
> **Приоритет**: 🟡 Средний  
> **Цель**: Устранение дублирования между API и Web handlers  
> **Дата создания**: 2025-08-28

## 🎯 Проблема и цели

### Обнаруженные проблемы
- **Дублирование кода** между `internal/application/handlers/users.go` и `internal/web/handlers/users.go`
- **Бизнес-логика размазана** по transport layer handlers
- **Сложность тестирования** и поддержки общей логики
- **Нарушение DRY принципа** в валидации, создании entities

### Цели рефакторинга
- ✅ Вынести общую бизнес-логику в Service Layer
- ✅ Устранить дублирование кода между handlers
- ✅ Улучшить тестируемость бизнес-логики
- ✅ Подготовить архитектуру к добавлению других transport layers (gRPC, GraphQL)
- ✅ Соблюсти принципы Clean Architecture

## 🏗️ Архитектурное решение

### Текущая архитектура
```
┌─────────────────┐    ┌─────────────────┐
│   API Handler   │    │   Web Handler   │
│   (JSON/REST)   │    │  (HTML/HTMX)    │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          │ ДУБЛИРОВАНИЕ         │
          │ БИЗНЕС-ЛОГИКИ        │
          │                      │
          └──────┬───────────────┘
                 ▼
        ┌─────────────────┐
        │   Repositories  │
        │   (Data Layer)  │
        └─────────────────┘
```

### Целевая архитектура
```
┌─────────────────┐    ┌─────────────────┐
│   API Handler   │    │   Web Handler   │
│   (JSON/REST)   │    │  (HTML/HTMX)    │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────┬───────────────┘
                 ▼
        ┌─────────────────┐
        │  Service Layer  │
        │ (Бизнес-логика) │
        └─────────┬───────┘
                  ▼
        ┌─────────────────┐
        │   Repositories  │
        │   (Data Layer)  │
        └─────────────────┘
```

## 📋 План реализации

### Этап 1: Создание Service Layer инфраструктуры (1-2 дня)

#### 1.1 Создание базовой структуры сервисов
- [ ] **Создать директорию** `internal/services/`
- [ ] **Создать интерфейс** `internal/services/interfaces.go` с UserService
- [ ] **Создать базовую реализацию** `internal/services/user_service.go`
- [ ] **Добавить DI setup** для сервисов в `internal/run.go`

#### 1.2 Определение DTO моделей
- [ ] **Создать** `internal/services/dto/user_dto.go` с общими моделями:
  - `CreateUserDTO`
  - `UpdateUserDTO` 
  - `UserFilterDTO`
  - `UserResponseDTO`

#### 1.3 Базовая структура UserService
```go
type UserService interface {
    CreateUser(ctx context.Context, req CreateUserDTO) (*user.User, error)
    GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
    GetUsersByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.User, error)
    UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserDTO) (*user.User, error)
    DeleteUser(ctx context.Context, id uuid.UUID) error
    ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error
}
```

### Этап 2: Реализация бизнес-логики в Service (2-3 дня)

#### 2.1 Миграция Create операций
- [ ] **Вынести из handlers** общую логику создания пользователя:
  - Валидация email (userRepo.ValidateEmail)
  - Проверка дублирования email
  - Валидация роли пользователя
  - Хеширование пароля (bcrypt)
  - Создание user entity
  - Сохранение в repository

#### 2.2 Реализация остальных операций
- [ ] **GetUserByID** с обработкой ошибок
- [ ] **GetUsersByFamily** с фильтрацией по семье
- [ ] **UpdateUser** с валидацией изменений
- [ ] **DeleteUser** с проверкой прав доступа
- [ ] **ChangeUserRole** с валидацией роли

#### 2.3 Добавление бизнес-логики
- [ ] **Валидация прав доступа** на операции (admin only для некоторых)
- [ ] **Проверка принадлежности к семье** при операциях
- [ ] **Аудит логика** (опционально)
- [ ] **Кеширование** (опционально)

### Этап 3: Рефакторинг API Handler (1 день)

#### 3.1 Обновление API Handler
- [ ] **Инжектить UserService** в API Handler
- [ ] **Удалить дублированную логику** из методов
- [ ] **Обновить методы** для использования сервиса:
  ```go
  func (h *APIUserHandler) CreateUser(c echo.Context) error {
      // Конвертируем API request в DTO
      dto := convertAPIRequestToDTO(apiRequest)
      
      // Вызываем сервис
      user, err := h.userService.CreateUser(ctx, dto)
      if err != nil {
          return h.handleAPIError(c, err)
      }
      
      // Конвертируем в API response
      return c.JSON(201, convertUserToAPIResponse(user))
  }
  ```

#### 3.2 Обновление моделей
- [ ] **Создать мапперы** между API models и DTOs
- [ ] **Обновить валидацию** для работы с DTOs
- [ ] **Сохранить API contracts** (не ломать существующие endpoints)

### Этап 4: Рефакторинг Web Handler (1 день)

#### 4.1 Обновление Web Handler
- [ ] **Инжектить UserService** в Web Handler
- [ ] **Удалить дублированную логику** из методов
- [ ] **Обновить методы** для использования сервиса:
  ```go
  func (h *WebUserHandler) Create(c echo.Context) error {
      // Конвертируем form в DTO
      dto := convertFormToDTO(form)
      
      // Вызываем сервис  
      user, err := h.userService.CreateUser(ctx, dto)
      if err != nil {
          return h.handleWebError(c, err) // Render HTML error
      }
      
      return c.Redirect(302, "/users")
  }
  ```

#### 4.2 Сохранение веб-специфики
- [ ] **Сохранить HTMX поддержку** в handlers
- [ ] **Сохранить template rendering** логику
- [ ] **Сохранить CSRF обработку**

### Этап 5: Тестирование и валидация (1-2 дня)

#### 5.1 Unit тестирование сервисов
- [ ] **Создать тесты** для `UserService` методов
- [ ] **Mock repositories** для изоляции тестирования
- [ ] **Покрыть edge cases** и error scenarios
- [ ] **Достичь 90%+ coverage** для Service Layer

#### 5.2 Integration тестирование
- [ ] **Обновить существующие тесты** handlers
- [ ] **Проверить API endpoints** работают корректно
- [ ] **Проверить Web endpoints** работают корректно
- [ ] **Тестировать HTMX функциональность**

#### 5.3 Регрессионное тестирование
- [ ] **Запустить полный набор тестов** `make test-fast`
- [ ] **Проверить linting** `make lint` 
- [ ] **Проверить сборку** `make build`

## 🔧 Техническая спецификация

### Структура директорий после рефакторинга
```
internal/
├── services/
│   ├── interfaces.go           # Service интерфейсы
│   ├── user_service.go        # UserService реализация
│   ├── user_service_test.go   # Unit тесты сервиса
│   └── dto/
│       ├── user_dto.go        # DTO модели
│       └── mappers.go         # Мапперы между слоями
├── application/handlers/
│   └── users.go               # API Handler (рефакторен)
├── web/handlers/
│   └── users.go               # Web Handler (рефакторен)
└── domain/user/
    └── user.go                # Domain entities (без изменений)
```

### Интерфейсы и DTOs

#### UserService Interface
```go
type UserService interface {
    // CRUD Operations
    CreateUser(ctx context.Context, req CreateUserDTO) (*user.User, error)
    GetUserByID(ctx context.Context, id uuid.UUID) (*user.User, error)
    GetUsersByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.User, error)
    UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserDTO) (*user.User, error)
    DeleteUser(ctx context.Context, id uuid.UUID) error
    
    // Business Operations
    ChangeUserRole(ctx context.Context, userID uuid.UUID, role user.Role) error
    ValidateUserAccess(ctx context.Context, userID, resourceOwnerID uuid.UUID) error
}
```

#### DTO Models
```go
type CreateUserDTO struct {
    Email     string    `validate:"required,email"`
    FirstName string    `validate:"required,min=2,max=50"`
    LastName  string    `validate:"required,min=2,max=50"`
    Password  string    `validate:"required,min=6"`
    Role      user.Role `validate:"required"`
    FamilyID  uuid.UUID `validate:"required"`
}

type UpdateUserDTO struct {
    FirstName *string `validate:"omitempty,min=2,max=50"`
    LastName  *string `validate:"omitempty,min=2,max=50"`
    Email     *string `validate:"omitempty,email"`
}
```

## ⚠️ Риски и митигации

### Технические риски
| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Поломка существующих API | Средняя | Высокое | Thorough testing, поэтапная миграция |
| Усложнение архитектуры | Высокая | Среднее | Четкая документация, code review |
| Performance деградация | Низкая | Среднее | Бенчмарки, профилирование |

### Процессные риски
| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Временные затраты | Высокая | Среднее | Поэтапное выполнение, приоритизация |
| Merge conflicts | Средняя | Среднее | Частые коммиты, координация команды |

## 🎯 Критерии готовности

### Definition of Done
- [ ] **Все тесты проходят** (unit, integration, e2e)
- [ ] **Linting чистый** (0 issues)
- [ ] **Code coverage ≥ 85%** для Service Layer
- [ ] **API endpoints работают** без изменения контрактов
- [ ] **Web interface работает** без потери функциональности
- [ ] **HTMX функциональность сохранена**
- [ ] **Performance не деградировала** (бенчмарки)

### Приемочные критерии
- [ ] **Дублирование устранено** - общая логика в сервисе
- [ ] **Handlers упрощены** - только transport layer логика
- [ ] **Бизнес-логика изолирована** - легко тестируется отдельно
- [ ] **Архитектура готова** к добавлению новых transport layers

## 📅 Временные рамки

### Общие временные затраты: 6-9 дней
- **Этап 1**: 1-2 дня (инфраструктура)
- **Этап 2**: 2-3 дня (бизнес-логика) 
- **Этап 3**: 1 день (API рефакторинг)
- **Этап 4**: 1 день (Web рефакторинг)
- **Этап 5**: 1-2 дня (тестирование)

### Критический путь
1. Service interfaces → Service implementation
2. API Handler refactoring → Web Handler refactoring  
3. Unit tests → Integration tests → Regression tests

## 🔄 План отката

### Rollback стратегия
1. **Git branches**: Каждый этап в отдельной ветке
2. **Feature flags**: Включение/выключение Service Layer
3. **A/B testing**: Постепенный переход handlers
4. **Backup**: Сохранение старых handlers до полного тестирования

### Rollback triggers
- **Критические баги** в production
- **Performance деградация** > 20%
- **Функциональная регрессия** в API/Web

---

**Владелец**: Backend Team  
**Reviewer**: Tech Lead  
**Следующий checkpoint**: После завершения Этапа 1
# Текущая задача: Фаза 1.6 — Упростить доменные модели

## Статус предыдущих задач

### Фаза 1.1 — Концепция единственной семьи (ВЫПОЛНЕНО)

Реализовано в коммите `5499c8e`:

- [x] Добавлено автоматическое создание семьи при первом запуске (setup wizard)
- [x] Family создаётся через страницу `/setup` при первом запуске
- [x] Убраны API endpoint'ы создания/удаления/редактирования семей (оставлен только `GetFamily`)
- [x] `FamilyRepository` сведён к минимуму (`GetFamily()`, `GetOrCreateFamily()`)
- [x] `FamilyService` упрощён (убраны `CreateFamily`, `UpdateFamily`, `DeleteFamily`, `ListFamilies`)
- [x] Добавлен middleware `SetupCheck` — перенаправляет на `/setup` если семья не создана
- [x] Обновлены тесты families, users, integration

Затронутые файлы (32 файла):
- `internal/application/handlers/families.go` — упрощён до GetFamily
- `internal/application/handlers/repositories.go` — упрощён интерфейс FamilyRepository
- `internal/infrastructure/user/family_repository_sqlite.go` — упрощён
- `internal/services/family_service.go` — упрощён
- `internal/services/interfaces.go` — упрощён FamilyService
- `internal/web/handlers/auth.go` — setup flow
- `internal/web/middleware/setup.go` — новый middleware
- `internal/web/templates/pages/setup.html` — страница первоначальной настройки
- И другие (тесты, DTO, маршруты)

---

## Текущая задача: Фаза 1.6 — Упростить доменные модели ✅ ВЫПОЛНЕНО

### Цель

Убрать поле `FamilyID` из доменных сущностей. В self-hosted модели одна семья = один экземпляр, поэтому `FamilyID` в доменных объектах избыточен. Семья определяется автоматически на уровне инфраструктуры.

**Статус:** ✅ Задача выполнена на 95%. Осталось обновить веб-хэндлеры (~10 файлов).

### Что нужно сделать

#### 1. Убрать `FamilyID` из структуры `Transaction`
**Файл:** `internal/domain/transaction/transaction.go`
- Убрать поле `FamilyID uuid.UUID` из структуры `Transaction` (строка 17)
- Убрать поле `FamilyID uuid.UUID` из структуры `Filter` (строка 32)
- Убрать параметр `familyID uuid.UUID` из конструктора `NewTransaction()` (строка 50)
- Убрать присвоение `FamilyID: familyID` (строка 60)
- Убрать метод `GetFamilyID()` (строки 86-89)
- Обновить валидацию — не требовать `FamilyID`

#### 2. Убрать `FamilyID` из структуры `Category`
**Файл:** `internal/domain/category/category.go`
- Убрать поле `FamilyID uuid.UUID` из структуры `Category` (строка 16)
- Убрать параметр `familyID uuid.UUID` из конструктора `NewCategory()` (строка 58)
- Убрать присвоение `FamilyID: familyID` (строка 65)
- Обновить валидацию

#### 3. Убрать `FamilyID` из структуры `Budget`
**Файл:** `internal/domain/budget/budget.go`
- Убрать поле `FamilyID uuid.UUID` из структуры `Budget` (строка 21)
- Убрать параметр `familyID uuid.UUID` из конструктора `NewBudget()` (строка 51)
- Убрать присвоение `FamilyID: familyID` (строка 60)
- Убрать метод `GetFamilyID()` (строки 89-92)
- Обновить валидацию

#### 4. Убрать `FamilyID` из структуры `Report`
**Файл:** `internal/domain/report/report.go`
- Убрать поле `FamilyID uuid.UUID` из структуры `Report` (строка 14)
- Убрать параметр `familyID uuid.UUID` из конструктора `NewReport()` (строка 88)
- Убрать присвоение `FamilyID: familyID` (строка 96)
- Убрать метод `GetFamilyID()` (строки 105-108)
- Обновить валидацию

#### 5. Убрать `FamilyID` из структуры `User`
**Файл:** `internal/domain/user/user.go`
- Убрать поле `FamilyID uuid.UUID` из структуры `User` (строка 16)
- Убрать параметр `familyID uuid.UUID` из конструктора `NewUser()` (строка 37)
- Убрать присвоение `FamilyID: familyID` (строка 44)
- Обновить валидацию

### Каскадные изменения (обязательно обновить после изменения доменных моделей)

#### 6. Обновить тесты доменных моделей
- `internal/domain/transaction/transaction_test.go` — убрать familyID из вызовов NewTransaction
- `internal/domain/category/category_test.go` — убрать familyID из вызовов NewCategory
- `internal/domain/budget/budget_test.go` — убрать familyID из вызовов NewBudget
- `internal/domain/report/report_test.go` — убрать familyID из вызовов NewReport
- `internal/domain/user/user_test.go` — убрать familyID из вызовов NewUser

#### 7. Обновить DTO и мапперы
- `internal/services/dto/` — убрать FamilyID из DTO, обновить мапперы (api_mappers.go, mappers.go, web_mappers.go)
- Убрать FamilyID из всех DTO-структур которые его содержат

#### 8. Обновить сервисный слой (вызовы конструкторов)
- `internal/services/category_service.go` — вызовы `NewCategory()` без familyID
- `internal/services/user_service.go` — вызовы `NewUser()` без familyID
- Другие сервисы, вызывающие конструкторы доменных объектов

#### 9. Обновить инфраструктурный слой (репозитории)
- SQLite-репозитории при чтении из БД заполняют FamilyID — нужно убрать маппинг
- Репозитории при записи в БД передают FamilyID — нужно использовать единственный family_id из конфигурации
- **Важно:** в SQL-таблицах `family_id` остаётся (для совместимости), но заполняется автоматически на уровне репозитория

#### 10. Обновить хэндлеры
- API хэндлеры, которые читают FamilyID из доменных объектов
- Веб-хэндлеры, которые передают FamilyID при создании сущностей

### Порядок выполнения внутри задачи

1. Изменить доменные модели (пункты 1-5)
2. Обновить тесты доменных моделей (пункт 6) — чтобы они компилировались
3. Обновить DTO и мапперы (пункт 7)
4. Обновить сервисы (пункт 8)
5. Обновить репозитории (пункт 9)
6. Обновить хэндлеры (пункт 10)
7. Запустить `make test` и `make lint` — исправить все ошибки

### Критерии завершения

- [x] Поле `FamilyID` отсутствует во всех доменных структурах ✅
- [x] Методы `GetFamilyID()` удалены ✅
- [x] Конструкторы `New*()` не принимают `familyID` ✅
- [x] Тесты доменных моделей обновлены ✅
- [x] Репозитории обновлены (helper-методы, Create/Update/Delete) ✅
- [x] Интерфейсы репозиториев обновлены ✅
- [x] Сервисный слой частично обновлен (BudgetService, CategoryService) ✅
- [ ] Веб-хэндлеры обновлены ⏸️ **ОСТАЛОСЬ**
- [ ] Все тесты проходят (`make test`) ⏸️
- [ ] Линтер проходит без ошибок (`make lint`) ⏸️
- [x] SQL-таблицы сохраняют `family_id` (заполняется на уровне инфраструктуры) ✅

---

## Что выполнено

### ✅ Доменные модели (5 файлов)
- Удалено поле `FamilyID` из: Transaction, Category, Budget, Report, User
- Удалены методы `GetFamilyID()` из: Transaction, Budget, Report
- Обновлены конструкторы: `NewTransaction()`, `NewCategory()`, `NewBudget()`, `NewReport()`, `NewUser()`
- Обновлена структура `Filter` в Transaction (убрано поле FamilyID)

### ✅ Тесты доменных моделей (5 файлов)
- `internal/domain/transaction/transaction_test.go`
- `internal/domain/category/category_test.go`
- `internal/domain/budget/budget_test.go`
- `internal/domain/report/report_test.go`
- `internal/domain/user/user_test.go`

### ✅ Репозитории (5 файлов)
- Добавлены helper-методы `getSingleFamilyID()` и `getSingleFamilyIDWithTx()`
- Обновлены методы `Create()` - автоматическое получение familyID
- Обновлены методы `Update()` - автоматическое получение familyID
- Обновлены методы `Delete()` - автоматическое получение familyID, убран параметр
- Переименованы методы:
  - `GetByFamilyID()` → `GetAll()`
  - `GetActiveBudgets(ctx, familyID)` → `GetActiveBudgets(ctx)`
  - `GetByFamilyAndCategory()` → `GetByCategory()`
- Убран маппинг FamilyID при чтении из БД (комментарии "unused - single family model")

**Файлы:**
- `internal/infrastructure/user/user_repository_sqlite.go`
- `internal/infrastructure/category/category_repository_sqlite.go`
- `internal/infrastructure/transaction/transaction_repository_sqlite.go`
- `internal/infrastructure/budget/budget_repository_sqlite.go`
- `internal/infrastructure/report/report_repository_sqlite.go`

### ✅ Интерфейсы репозиториев
- `internal/application/handlers/repositories.go` - обновлены все интерфейсы:
  - `UserRepository`: `GetAll()`, `Delete(ctx, id)`
  - `CategoryRepository`: `GetAll()`, `Delete(ctx, id)`
  - `TransactionRepository`: `GetAll()`, `Delete(ctx, id)`, `GetTotalByDateRange()`
  - `BudgetRepository`: `GetAll()`, `GetActiveBudgets()`, `Delete(ctx, id)`, `GetByCategory()`
  - `ReportRepository`: `GetAll()`, `Delete(ctx, id)`

### ✅ Сервисный слой (частично)
- `internal/services/budget_service.go`:
  - Обновлены локальные интерфейсы (BudgetRepository, TransactionRepositoryForBudgets)
  - Обновлены методы: `GetActiveBudgets()`, `DeleteBudget()`, `GetBudgetsByCategory()`, `ValidateBudgetPeriod()`
  - Обновлен метод `recalculateAndUpdateSpent()` - использует `GetTotalByDateRange()`
- `internal/services/category_service.go` - уже обновлен (линтером)
- `internal/services/interfaces.go`:
  - Обновлен интерфейс `BudgetService`

### ✅ DTO мапперы (частично)
- `internal/services/dto/api_mappers.go`:
  - `ToUserAPIResponse()` - возвращает `uuid.Nil` для FamilyID
  - `ToTransactionAPIResponse()` - возвращает `uuid.Nil` для FamilyID
- `internal/services/dto/mappers.go`:
  - `ToUserResponseDTO()` - возвращает `uuid.Nil` для FamilyID

---

## Что осталось сделать

### ⏸️ Веб-хэндлеры (~10 файлов с ошибками компиляции)

**Файлы с известными ошибками:**
1. `internal/web/handlers/auth.go:85` - `foundUser.FamilyID` не существует
2. `internal/web/handlers/budgets.go`:
   - `GetBudgetsByFamily()` не существует → использовать `GetAllBudgets()`
   - `GetCategoriesByFamily()` не существует → использовать `GetAllCategories()`
   - `budgetEntity.FamilyID` не существует
   - `DeleteBudget()` - убрать второй параметр (familyID)
3. Другие веб-хэндлеры (categories, transactions, reports, dashboard)

**Необходимые изменения:**
- Обновить вызовы сервисов (убрать familyID параметры)
- Убрать чтение `budgetEntity.FamilyID`, `foundUser.FamilyID` и т.д.
- Убрать передачу familyID при создании сущностей

### ⏸️ Финальные проверки
- Запустить `make test` - убедиться что все тесты проходят
- Запустить `make lint` - исправить все ошибки линтера
- Проверить веб-интерфейс локально

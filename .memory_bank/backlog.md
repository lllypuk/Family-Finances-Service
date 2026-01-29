# Бэклог задач для проекта "Family Finances Service"

## Контекст

Проект переходит на модель **self-hosted решения для одной семьи**: один Docker-образ = один экземпляр = одна семья. Необходимо убрать мультитенантность (family_id как параметр фильтрации во всех слоях), упростить observability и привести код в соответствие с новой архитектурой.

---

## Фаза 1: Убрать мультитенантность (family_id)

Сейчас `familyID` используется как ключ изоляции данных во всех слоях: домен, репозитории, сервисы, хэндлеры, веб-middleware. В self-hosted модели один экземпляр обслуживает одну семью, поэтому family создается при первом запуске и привязывается ко всем данным автоматически.

### 1.1 Ввести концепцию "единственной семьи"
- [ ] Добавить автоматическое создание семьи при первом запуске (seed/bootstrap)
- [ ] Хранить ID семьи в конфигурации или получать из БД (единственная запись в таблице `families`)
- [ ] Убрать API endpoint'ы создания/удаления/редактирования семей
- [ ] Убрать `FamilyRepository` из интерфейсов (или свести к `GetFamily()`)

**Затрагивает:**
- `internal/application/handlers/repositories.go` — интерфейс `FamilyRepository`
- `internal/infrastructure/user/family_repository_sqlite.go` — реализация
- `internal/services/interfaces.go` — `FamilyService`

### 1.2 Упростить интерфейсы репозиториев
- [ ] Убрать параметр `familyID` из всех методов репозиториев
- [ ] `GetByFamilyID(ctx, familyID)` → `GetAll(ctx)` или аналог
- [ ] `Delete(ctx, id, familyID)` → `Delete(ctx, id)`
- [ ] `GetTotalByFamilyAndDateRange(ctx, familyID, ...)` → `GetTotalByDateRange(ctx, ...)`
- [ ] `GetByFamilyAndCategory(ctx, familyID, ...)` → `GetByCategory(ctx, ...)`
- [ ] `GetActiveBudgets(ctx, familyID)` → `GetActiveBudgets(ctx)`
- [ ] `GetByPeriod(ctx, familyID, ...)` → `GetByPeriod(ctx, ...)`

**Затрагивает:**
- `internal/application/handlers/repositories.go` — все интерфейсы
- `internal/infrastructure/*/` — все SQLite-реализации (6 файлов)

### 1.3 Упростить сервисный слой
- [ ] Убрать параметр `familyID` из всех методов сервисов
- [ ] Методы вида `GetTransactionsByFamily(ctx, familyID, ...)` → `GetTransactions(ctx, ...)`
- [ ] `CreateDefaultCategories(familyID)` → выполняется при bootstrap
- [ ] `GetCategoryHierarchy(familyID)` → `GetCategoryHierarchy()`

**Затрагивает:**
- `internal/services/interfaces.go` — 22+ методов с `familyID`
- `internal/services/*.go` — все реализации сервисов

### 1.4 Упростить API хэндлеры
- [ ] Убрать `family_id` как query parameter из всех endpoint'ов
- [ ] Убрать `FamilyID` из request-типов (`CreateTransactionRequest`, `CreateCategoryRequest`, `CreateUserRequest`)
- [ ] Убрать валидацию `family_id` из хэндлеров

**Затрагивает:**
- `internal/application/handlers/transactions.go`
- `internal/application/handlers/categories.go`
- `internal/application/handlers/budgets.go`
- `internal/application/handlers/reports.go`
- `internal/application/handlers/users.go`
- `internal/application/handlers/types.go` — request/response типы

### 1.5 Упростить веб-слой
- [ ] Убрать `FamilyID` из `SessionData` (`internal/web/middleware/session.go`)
- [ ] Убрать `SessionFamilyKey = "family_id"` из констант сессии
- [ ] Упростить `GetUserFromContext()` — не извлекать family из сессии
- [ ] Обновить веб-хэндлеры (dashboard, transactions, categories, budgets, reports) — не читать `sessionData.FamilyID`

**Затрагивает:**
- `internal/web/middleware/session.go`
- `internal/web/middleware/auth.go`
- `internal/web/models.go` — `SessionData`, `PageData`
- `internal/web/handlers/*.go` — все веб-хэндлеры

### 1.6 Упростить доменные модели
- [ ] Убрать поле `FamilyID` из доменных сущностей или сделать его внутренним (автозаполнение)
- [ ] Убрать метод `GetFamilyID()` у `Transaction`
- [ ] Упростить валидацию — не требовать `FamilyID` при создании сущностей

**Затрагивает:**
- `internal/domain/transaction/transaction.go`
- `internal/domain/category/category.go`
- `internal/domain/budget/budget.go`
- `internal/domain/report/report.go`
- `internal/domain/user/user.go`

### 1.7 Упростить миграции БД
- [ ] Обновить SQL-схему: оставить `family_id` в таблицах (для совместимости), но не требовать в бизнес-логике
- [ ] Альтернатива: создать единственную запись family при миграции

**Затрагивает:**
- `migrations/001_initial_schema.up.sql`

---

## Фаза 2: Упростить observability

Стек Prometheus/Grafana/Jaeger/OpenTelemetry уже удален из зависимостей. Осталось упростить код в `internal/observability/`.

### 2.1 Упростить пакет observability
- [ ] Убрать `BusinessLogger` с методами `LogTransactionEvent`, `LogBudgetEvent` и т.д. (избыточно для self-hosted)
- [ ] Оставить базовый structured logger (slog)
- [ ] Оставить HTTP logging middleware (`middleware.go`)
- [ ] Оставить health check (`health.go`) — упростить до одного endpoint'а `/health`
- [ ] Убрать `ReadinessHandler` и `LivenessHandler` (не нужны без Kubernetes)

**Затрагивает:**
- `internal/observability/observability.go`
- `internal/observability/logging.go`
- `internal/observability/middleware.go`
- `internal/observability/health.go`
- `internal/run.go` — инициализация observability
- `internal/application/http_server.go` — регистрация middleware и health endpoint'ов

### 2.2 Удалить каталог monitoring/
- [ ] Удалить `monitoring/` целиком (Grafana dashboards, Prometheus config, alertmanager, postgres_exporter)

**Затрагивает:**
- `monitoring/` — весь каталог (если ещё существует)

---

## Фаза 3: Обновить тесты

### 3.1 Обновить unit-тесты
- [ ] Обновить тесты репозиториев — убрать `familyID` из вызовов
- [ ] Обновить тесты хэндлеров — убрать `family_id` из запросов
- [ ] Обновить тесты сервисов — убрать `familyID` из вызовов

**Затрагивает:**
- `internal/infrastructure/*/` — тесты репозиториев
- `internal/application/handlers/*_test.go`
- `internal/services/*_test.go`
- `internal/domain/*_test.go`

### 3.2 Обновить интеграционные тесты
- [ ] Обновить `internal/testhelpers/sqlite.go` — упростить setup (одна семья)
- [ ] Обновить `tests/integration/` — убрать мультисемейные сценарии
- [ ] Обновить бенчмарки `tests/benchmarks/`

### 3.3 Обновить веб-тесты
- [ ] Обновить тесты middleware — убрать family из сессии
- [ ] Обновить тесты веб-хэндлеров

---

## Фаза 4: Доработки для self-hosted

### 4.1 Первоначальная настройка (onboarding)
- [ ] Реализовать wizard первого запуска: создание семьи + первого пользователя (admin)
- [ ] Если в БД нет семьи — перенаправлять на страницу настройки
- [ ] После создания семьи — перенаправлять на login

### 4.2 Управление пользователями
- [ ] Admin может добавлять/удалять участников семьи через веб-интерфейс
- [ ] Регистрация новых пользователей только через invite от admin (без открытой регистрации)

### 4.3 Резервное копирование через интерфейс
- [ ] Добавить endpoint для скачивания файла БД (SQLite backup)
- [ ] Добавить endpoint для загрузки файла БД (restore)
- [ ] Страница в веб-интерфейсе для управления бэкапами

---

## Порядок выполнения

Рекомендуемый порядок для минимизации конфликтов:

1. **Фаза 1.1** — Концепция единственной семьи (bootstrap)
2. **Фаза 1.6** — Доменные модели (сверху вниз)
3. **Фаза 1.2** — Интерфейсы репозиториев
4. **Фаза 1.3** — Сервисный слой
5. **Фаза 1.4** — API хэндлеры
6. **Фаза 1.5** — Веб-слой
7. **Фаза 1.7** — Миграции БД
8. **Фаза 3** — Тесты (параллельно с каждой задачей из фазы 1)
9. **Фаза 2** — Observability
10. **Фаза 4** — Self-hosted доработки

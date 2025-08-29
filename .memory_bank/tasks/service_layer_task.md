# Задача: Расширение Service Layer Architecture на все домены

> **Статус**: 📋 В ПЛАНИРОВАНИИ
> **Приоритет**: 🟡 Средний
> **Цель**: Создание Service Layer для всех доменов приложения
> **Дата создания**: 2025-08-29

## 🎯 Обзор

После успешного создания **UserService** и **FamilyService**, необходимо расширить Service Layer Architecture на все остальные домены приложения для достижения полной архитектурной консистентности.

### 📋 ПЛАН РЕАЛИЗАЦИИ

## 🎯 Цели расширения

### Текущая ситуация
- ✅ **UserService** и **FamilyService** реализованы
- ❌ **CategoryService** - прямые repository вызовы в handlers
- ❌ **TransactionService** - бизнес-логика размазана по handlers
- ❌ **BudgetService** - сложная логика подсчетов в handlers
- ❌ **ReportService** - логика генерации отчетов в handlers

### Цели расширения
- 🎯 **Архитектурная консистентность** - все домены используют Service Layer
- 🎯 **Устранение дублирования** между API и Web handlers
- 🎯 **Централизация бизнес-логики** в сервисах
- 🎯 **Улучшение тестируемости** сложной доменной логики
- 🎯 **Подготовка к масштабированию** - легкое добавление новых transport layers

## 🏗️ Архитектурное решение

### Целевая архитектура сервисов
```
┌─────────────────┐    ┌─────────────────┐
│   API Handler   │    │   Web Handler   │
│   (JSON/REST)   │    │  (HTML/HTMX)    │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────┬───────────────┘
                 ▼
        ┌─────────────────────────────────┐
        │         Service Layer           │
        │  ┌─────────────────────────────┐ │
        │  │ UserService  │ FamilyService│ │  ✅ Готово
        │  │ (Auth, RBAC) │ (Management) │ │
        │  └─────────────────────────────┘ │
        │  ┌─────────────────────────────┐ │
        │  │CategoryService│TransactionSrv│ │  🔄 В планах
        │  │(Hierarchies)  │(Calculations)│ │
        │  └─────────────────────────────┘ │
        │  ┌─────────────────────────────┐ │
        │  │ BudgetService │ ReportService│ │  🔄 В планах
        │  │(Limits&Alert) │ (Analytics)  │ │
        │  └─────────────────────────────┘ │
        └─────────┬───────────────────────┘
                  ▼
        ┌─────────────────┐
        │   Repositories  │
        │   (Data Layer)  │
        └─────────────────┘
```

## 📋 План реализации сервисов

### 🎯 Этап 1: CategoryService (2-3 дня)

#### 1.1 Анализ существующих handlers
- [x] **Изучить** `internal/application/handlers/categories.go` - API handler
- [x] **Найти дублирование** с потенциальным Web handler (если есть)
- [x] **Выявить бизнес-логику** которую нужно вынести в сервис

#### 1.2 Создание CategoryService
- [x] **Создать интерфейс** `CategoryService` в `internal/services/interfaces.go`:
  ```go
  type CategoryService interface {
      CreateCategory(ctx context.Context, req dto.CreateCategoryDTO) (*category.Category, error)
      GetCategoryByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
      GetCategoriesByFamily(ctx context.Context, familyID uuid.UUID, typeFilter *category.Type) ([]*category.Category, error)
      UpdateCategory(ctx context.Context, id uuid.UUID, req dto.UpdateCategoryDTO) (*category.Category, error)
      DeleteCategory(ctx context.Context, id uuid.UUID) error
      GetCategoryHierarchy(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error)
      ValidateCategoryHierarchy(ctx context.Context, categoryID, parentID uuid.UUID) error
      CheckCategoryUsage(ctx context.Context, categoryID uuid.UUID) (bool, error)
  }
  ```

#### 1.3 Создание DTO моделей
- [x] **Создать** `internal/services/dto/category_dto.go`:
  - `CreateCategoryDTO` - валидация имени, типа, родительской категории
  - `UpdateCategoryDTO` - поля для обновления
  - `CategoryFilterDTO` - фильтры по типу, активности
  - `CategoryResponseDTO` - для ответов API
  - `CategoryHierarchyDTO` - для иерархической структуры
- [x] **Создать мапперы** для конвертации между API/Web моделями и DTOs в `api_mappers.go`

#### 1.4 Реализация бизнес-логики
- [x] **Создать** `internal/services/category_service.go`:
  - Валидация иерархии категорий (предотвращение циклов)
  - Проверка принадлежности к семье
  - Валидация типов (Income/Expense)
  - Soft delete с проверкой использования в транзакциях
  - Предотвращение дублирования имен в одной области
  - Ограничение глубины иерархии (максимум 2 уровня)
- [x] **Создать** `internal/services/category_usage_checker.go` для проверки использования
- [x] **Добавить unit тесты** с полным покрытием (16 тестов, все проходят)

#### 1.5 Рефакторинг handlers
- [x] **Обновить API handler** для использования CategoryService
- [x] **Добавить новый endpoint** `/api/v1/categories/hierarchy` для иерархической структуры
- [x] **Обновить error handling** для маппинга service ошибок
- [x] **Обновить container и DI** для интеграции CategoryService
- [ ] **Создать Web handler** (если нужен) с HTMX поддержкой (отложено)

**✅ ЭТАП 1 ЗАВЕРШЕН**

**Итоги реализации:**
- ✅ Полная реализация CategoryService с продвинутой бизнес-логикой
- ✅ Комплексные DTO модели с валидацией
- ✅ 16 unit тестов с полным покрытием основных сценариев
- ✅ Интеграция с существующими handlers
- ✅ Обработка ошибок с типизированными exception'ами
- ✅ Проверка использования категорий в транзакциях для безопасного удаления
- ✅ Валидация иерархии с предотвращением циклов и ограничением глубины

**Архитектурные решения:**
- Использован паттерн Dependency Injection для чистой архитектуры
- Создан отдельный CategoryUsageChecker для декупления от TransactionRepository
- Применены типизированные ошибки для лучшего error handling
- Реализованы mapper'ы для чистого разделения между слоями

### 🎯 Этап 2: TransactionService (3-4 дня)

#### 2.1 Анализ сложной бизнес-логики
- [x] **Изучить** `internal/application/handlers/transactions.go` - текущие операции
- [x] **Выявить расчетные операции**:
  - Обновление бюджетов при создании транзакций
  - Валидация лимитов и остатков
  - Конвертация валют (если есть)
  - Категоризация и теги

#### 2.2 Создание TransactionService
- [x] **Создать интерфейс** `TransactionService` в `internal/services/interfaces.go`:
  ```go
  type TransactionService interface {
      CreateTransaction(ctx context.Context, req dto.CreateTransactionDTO) (*transaction.Transaction, error)
      GetTransactionByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error)
      GetTransactionsByFamily(ctx context.Context, familyID uuid.UUID, filter dto.TransactionFilterDTO) ([]*transaction.Transaction, error)
      UpdateTransaction(ctx context.Context, id uuid.UUID, req dto.UpdateTransactionDTO) (*transaction.Transaction, error)
      DeleteTransaction(ctx context.Context, id uuid.UUID) error
      GetTransactionsByCategory(ctx context.Context, categoryID uuid.UUID, filter dto.TransactionFilterDTO) ([]*transaction.Transaction, error)
      GetTransactionsByDateRange(ctx context.Context, familyID uuid.UUID, from, to time.Time) ([]*transaction.Transaction, error)
      BulkCategorizeTransactions(ctx context.Context, transactionIDs []uuid.UUID, categoryID uuid.UUID) error
      ValidateTransactionLimits(ctx context.Context, familyID uuid.UUID, categoryID uuid.UUID, amount float64, transactionType transaction.Type) error
  }
  ```

#### 2.3 Создание DTO моделей
- [x] **Создать** `internal/services/dto/transaction_dto.go`:
  - `CreateTransactionDTO` - с валидацией и бизнес-правилами
  - `UpdateTransactionDTO` - поля для частичного обновления
  - `TransactionFilterDTO` - с комплексными фильтрами, pagination и sorting
  - `TransactionResponseDTO` - для ответов API
  - `BulkCategorizeDTO` - для массовых операций
  - `TransactionStatsDTO` - для аналитики
- [x] **Создать API мапперы** в `api_mappers.go` для конвертации между слоями

#### 2.4 Реализация бизнес-логики
- [x] **Создать** `internal/services/transaction_service.go`:
  - Валидация amounts, дат и категорий
  - Автоматическое обновление связанных бюджетов
  - Проверка лимитов перед созданием expense транзакций
  - Валидация принадлежности пользователей и категорий к семье
  - Обработка bulk operations с индивидуальными обновлениями
  - Комплексная фильтрация и pagination
  - Типизированные ошибки и proper error handling
- [x] **Интеграция с Budget operations** для автоматического обновления

#### 2.5 Unit тестирование 
- [x] **Создать** `internal/services/transaction_service_test.go`:
  - 18 comprehensive тестов с полным покрытием
  - Mock repositories для изоляции unit тестов
  - Тестирование success и error сценариев
  - Валидация бизнес-логики и edge cases
  - Проверка интеграции с budget operations

#### 2.6 Интеграция в DI контейнер
- [x] **Обновить** `internal/services/container.go` для включения TransactionService
- [x] **Обновить** `internal/run.go` для передачи Budget repository
- [x] **Проверить** успешную сборку приложения

**✅ ЭТАП 2 ПОЛНОСТЬЮ ЗАВЕРШЕН**

**Итоги реализации:**
- ✅ Полная реализация TransactionService с продвинутой бизнес-логикой
- ✅ 9 методов интерфейса + дополнительные helper методы 
- ✅ Comprehensive DTO модели с валидацией и константами
- ✅ 18 unit тестов с полным покрытием основных и edge сценариев
- ✅ Интеграция с DI контейнером и успешная сборка
- ✅ Автоматическое управление бюджетами при транзакциях
- ✅ Валидация лимитов и принадлежности к семье
- ✅ Bulk операции с individual error handling
- ✅ Код полностью соответствует linter стандартам (0 issues)

**Архитектурные решения:**
- Использован Dependency Injection с minimal интерфейсами
- Создана декомпозиция сложных операций на helper методы
- Применены типизированные ошибки с nolint аннотациями
- Реализованы mapper'ы для чистого разделения между слоями
- Добавлена поддержка констант для избежания magic numbers

### 🎯 Этап 3: BudgetService (3-4 дня)

#### 3.1 Анализ комплексной логики
- [ ] **Изучить** `internal/application/handlers/budgets.go` - текущие расчеты
- [ ] **Выявить комплексные операции**:
  - Расчет потраченных сумм по категориям
  - Проверка превышений бюджета
  - Алгоритмы предупреждений и алертов
  - Периодические бюджеты (месячные, годовые)

#### 3.2 Создание BudgetService
- [ ] **Создать интерфейс** `BudgetService`:
  ```go
  type BudgetService interface {
      CreateBudget(ctx context.Context, req dto.CreateBudgetDTO) (*budget.Budget, error)
      GetBudgetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error)
      GetBudgetsByFamily(ctx context.Context, familyID uuid.UUID, filter dto.BudgetFilterDTO) ([]*budget.Budget, error)
      UpdateBudget(ctx context.Context, id uuid.UUID, req dto.UpdateBudgetDTO) (*budget.Budget, error)
      DeleteBudget(ctx context.Context, id uuid.UUID) error
      GetActiveBudgets(ctx context.Context, familyID uuid.UUID, date time.Time) ([]*budget.Budget, error)
      UpdateBudgetSpent(ctx context.Context, budgetID uuid.UUID, amount float64) error
      CheckBudgetLimits(ctx context.Context, familyID uuid.UUID, categoryID uuid.UUID, amount float64) error
      GetBudgetStatus(ctx context.Context, budgetID uuid.UUID) (*dto.BudgetStatusDTO, error)
  }
  ```

#### 3.3 Реализация алгоритмов
- [ ] **Создать** `internal/services/budget_service.go`:
  - Автоматические расчеты потраченных сумм
  - Алгоритмы проверки лимитов
  - Генерация уведомлений о превышении
  - Поддержка периодических бюджетов
  - Rollover неиспользованных средств

### 🎯 Этап 4: ReportService (4-5 дней)

#### 4.1 Анализ аналитической логики
- [ ] **Изучить** `internal/application/handlers/reports.go` - текущая генерация отчетов
- [ ] **Выявить сложные вычисления**:
  - Агрегация данных по периодам
  - Группировка по категориям
  - Расчеты трендов и прогнозов
  - Сравнение с бюджетами
  - Экспорт в различные форматы

#### 4.2 Создание ReportService
- [ ] **Создать интерфейс** `ReportService`:
  ```go
  type ReportService interface {
      GenerateExpenseReport(ctx context.Context, req dto.ReportRequestDTO) (*dto.ExpenseReportDTO, error)
      GenerateIncomeReport(ctx context.Context, req dto.ReportRequestDTO) (*dto.IncomeReportDTO, error)
      GenerateBudgetComparisonReport(ctx context.Context, familyID uuid.UUID, period dto.ReportPeriod) (*dto.BudgetComparisonDTO, error)
      GenerateCashFlowReport(ctx context.Context, familyID uuid.UUID, from, to time.Time) (*dto.CashFlowReportDTO, error)
      GenerateCategoryBreakdownReport(ctx context.Context, familyID uuid.UUID, period dto.ReportPeriod) (*dto.CategoryBreakdownDTO, error)
      ExportReport(ctx context.Context, reportID uuid.UUID, format string) ([]byte, error)
      ScheduleReport(ctx context.Context, req dto.ScheduleReportDTO) (*dto.ScheduledReportDTO, error)
  }
  ```

#### 4.3 Реализация аналитики
- [ ] **Создать** `internal/services/report_service.go`:
  - Комплексные SQL/MongoDB агрегации
  - Кеширование тяжелых расчетов
  - Интеграция с TransactionService и BudgetService
  - Алгоритмы прогнозирования трендов
  - Генерация PDF/CSV/Excel отчетов

### 🎯 Этап 5: Интеграция сервисов (2-3 дня)

#### 5.1 Междоменные зависимости
- [ ] **Интеграция TransactionService + BudgetService**:
  - Автоматическое обновление бюджетов при создании транзакций
  - Проверка лимитов перед созданием expense транзакций
  - Event-driven updates через channels или пабликация событий

#### 5.2 Интеграция с CategoryService
- [ ] **Валидация категорий** во всех связанных сервисах
- [ ] **Каскадные операции** при удалении категорий
- [ ] **Поддержка иерархии** в отчетах и бюджетах

#### 5.3 Рефакторинг Handlers
- [ ] **Обновить все API handlers** для использования соответствующих сервисов
- [ ] **Создать Web handlers** для всех доменов с HTMX поддержкой
- [ ] **Унифицировать error handling** между всеми сервисами

### 🎯 Этап 6: Тестирование и валидация (3-4 дня)

#### 6.1 Unit тестирование всех сервисов
- [ ] **CategoryService тесты** - 100% покрытие бизнес-логики
- [ ] **TransactionService тесты** - включая интеграцию с BudgetService
- [ ] **BudgetService тесты** - сложные алгоритмы расчетов
- [ ] **ReportService тесты** - аналитические функции с mock data

#### 6.2 Integration тестирование
- [ ] **End-to-end workflow тесты**:
  - Создание семьи → категории → бюджеты → транзакции → отчеты
  - Проверка междоменных зависимостей
  - Валидация RBAC во всех сервисах

#### 6.3 Performance тестирование
- [ ] **Тестирование производительности** report generation
- [ ] **Бенчмарки** для heavy aggregation queries
- [ ] **Memory leak testing** для long-running calculations

## 🔧 Техническая спецификация

### Целевая структура директорий
```
internal/
├── services/
│   ├── interfaces.go              # Все Service интерфейсы
│   ├── container.go              # DI контейнер сервисов
│   ├── user_service.go           # ✅ UserService (готово)
│   ├── family_service.go         # ✅ FamilyService (готово)
│   ├── category_service.go       # 🔄 CategoryService
│   ├── transaction_service.go    # 🔄 TransactionService
│   ├── budget_service.go         # 🔄 BudgetService
│   ├── report_service.go         # 🔄 ReportService
│   ├── *_service_test.go         # Unit тесты всех сервисов
│   └── dto/
│       ├── user_dto.go           # ✅ User DTOs (готово)
│       ├── category_dto.go       # 🔄 Category DTOs
│       ├── transaction_dto.go    # 🔄 Transaction DTOs
│       ├── budget_dto.go         # 🔄 Budget DTOs
│       ├── report_dto.go         # 🔄 Report DTOs
│       ├── web_mappers.go        # ✅ Web мапперы (готово)
│       └── api_mappers.go        # ✅ API мапперы (готово)
├── application/handlers/
│   ├── users.go                  # ✅ API Handler (рефакторен)
│   ├── categories.go             # 🔄 Needs refactoring
│   ├── transactions.go           # 🔄 Needs refactoring
│   ├── budgets.go               # 🔄 Needs refactoring
│   └── reports.go               # 🔄 Needs refactoring
├── web/handlers/
│   ├── users.go                 # ✅ Web Handler (рефакторен)
│   ├── categories.go            # 🔄 To be created
│   ├── transactions.go          # 🔄 To be created
│   ├── budgets.go              # 🔄 To be created
│   └── reports.go              # 🔄 To be created
└── domain/
    ├── user/                    # ✅ Без изменений
    ├── category/                # ✅ Без изменений
    ├── transaction/             # ✅ Без изменений
    ├── budget/                  # ✅ Без изменений
    └── report/                  # ✅ Без изменений
```

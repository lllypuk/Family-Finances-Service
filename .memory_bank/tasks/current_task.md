# Текущая задача: Дополнение интеграционных тестов

## Анализ существующих интеграционных тестов

В каталоге `tests/integration` уже существуют следующие файлы:
- ✅ `categories_test.go` - полностью покрыт (CRUD операции + валидация)
- ✅ `families_test.go` - полностью покрыт (CRUD операции + валидация + получение участников)
- ✅ `users_test.go` - полностью покрыт (CRUD операции + валидация)

## Недостающие интеграционные тесты

### 1. Транзакции (`transactions_test.go`) - ОТСУТСТВУЕТ
**Приоритет: ВЫСОКИЙ**

Необходимо покрыть следующие сценарии (API методы: CreateTransaction, GetTransactions, GetTransactionByID, UpdateTransaction, DeleteTransaction):
- ✗ `CreateTransaction_Success` - создание транзакции с валидными данными
- ✗ `CreateTransaction_ValidationError` - валидация полей (amount > 0, type in [income, expense], description не пустое)
- ✗ `CreateTransaction_ForeignKeyValidation` - проверка существования category_id, user_id, family_id
- ✗ `GetTransactionByID_Success` - получение транзакции по ID
- ✗ `GetTransactionByID_NotFound` - обработка несуществующего ID
- ✗ `GetTransactionByID_InvalidUUID` - обработка некорректного UUID
- ✗ `GetTransactions_ByFamily` - получение транзакций семьи (обязательный family_id)
- ✗ `GetTransactions_ByDateRange` - фильтрация по диапазону дат
- ✗ `GetTransactions_ByCategory` - фильтрация по категории
- ✗ `GetTransactions_ByType` - фильтрация по типу (income/expense)
- ✗ `GetTransactions_MissingFamilyID` - ошибка при отсутствии family_id
- ✗ `UpdateTransaction_Success` - обновление полей транзакции
- ✗ `UpdateTransaction_PartialUpdate` - частичное обновление через указатели
- ✗ `DeleteTransaction_Success` - удаление транзакции

### 2. Бюджеты (`budgets_test.go`) - ОТСУТСТВУЕТ
**Приоритет: ВЫСОКИЙ**

Необходимо покрыть следующие сценарии (API методы: CreateBudget, GetBudgets, GetBudgetByID, UpdateBudget, DeleteBudget):
- ✗ `CreateBudget_Success` - создание бюджета с валидными данными
- ✗ `CreateBudget_ValidationError` - валидация полей (amount > 0, period in [weekly,monthly,yearly,custom], даты корректные)
- ✗ `CreateBudget_DateValidation` - проверка что start_date < end_date
- ✗ `GetBudgetByID_Success` - получение бюджета по ID
- ✗ `GetBudgetByID_NotFound` - обработка несуществующего ID
- ✗ `GetBudgetByID_InvalidUUID` - обработка некорректного UUID
- ✗ `GetBudgets_ByFamily` - получение бюджетов семьи (обязательный family_id)
- ✗ `GetBudgets_ActiveOnly` - фильтрация активных бюджетов через active_only=true
- ✗ `GetBudgets_MissingFamilyID` - ошибка при отсутствии family_id
- ✗ `UpdateBudget_Success` - обновление бюджета
- ✗ `UpdateBudget_PartialUpdate` - частичное обновление через указатели
- ✗ `UpdateBudget_ToggleActive` - активация/деактивация через is_active
- ✗ `DeleteBudget_Success` - удаление бюджета

### 3. Отчеты (`reports_test.go`) - ОТСУТСТВУЕТ
**Приоритет: СРЕДНИЙ**

Необходимо покрыть следующие сценарии (API методы: CreateReport, GetReports, GetReportByID, DeleteReport):
- ✗ `CreateReport_Success` - создание отчета с валидными данными
- ✗ `CreateReport_ValidationError` - валидация полей отчета
- ✗ `GetReportByID_Success` - получение отчета по ID
- ✗ `GetReportByID_NotFound` - обработка несуществующего ID
- ✗ `GetReportByID_InvalidUUID` - обработка некорректного UUID
- ✗ `GetReports_ByFamily` - получение отчетов семьи (обязательный family_id)
- ✗ `GetReports_ByUser` - фильтрация отчетов по пользователю
- ✗ `GetReports_MissingFamilyID` - ошибка при отсутствии family_id
- ✗ `DeleteReport_Success` - удаление отчета

### 4. Интеграционные сценарии (`integration_scenarios_test.go`) - ОТСУТСТВУЕТ
**Приоритет: СРЕДНИЙ**

Комплексные сценарии взаимодействия между доменами:
- ✗ `CompleteUserJourney_NewFamily` - полный сценарий создания семьи с пользователем
- ✗ `CompleteUserJourney_BudgetTracking` - создание категорий, бюджета, транзакций, отчетов
- ✗ `MultiUserScenario` - работа нескольких пользователей в одной семье
- ✗ `CrossDomainValidation` - проверка консистентности данных между доменами
- ✗ `FamilyIsolation_DataLeakage` - проверка изоляции данных между семьями
- ✗ `RoleBasedAccess` - проверка ролевой модели (admin, member, child)

### 5. Дополнения к существующим тестам

#### `categories_test.go`
- ✗ `GetCategories_FilterByType` - фильтрация по типу (income/expense)
- ✗ `CategoryUsage_WithTransactions` - проверка связи с транзакциями
- ✗ `DeleteCategory_WithTransactions` - удаление категории с существующими транзакциями

#### `families_test.go`
- ✗ `UpdateFamily_Success` - обновление данных семьи
- ✗ `DeleteFamily_Success` - удаление семьи
- ✗ `FamilyStats_Overview` - получение общей статистики семьи

#### `users_test.go`
- ✗ `UserAuthentication_Success` - проверка аутентификации
- ✗ `UserPermissions_RoleBasedAccess` - проверка прав доступа по ролям
- ✗ `ChangePassword_Success` - смена пароля

## План реализации

### Этап 1: Основные домены (1-2 дня)
1. Создать `transactions_test.go` с основными CRUD операциями
2. Создать `budgets_test.go` с основными CRUD операциями
3. Добавить базовую валидацию и error handling

### Этап 2: Расширенная функциональность (2-3 дня)
1. Добавить фильтрацию и поиск в транзакциях и бюджетах
2. Создать `reports_test.go` с основными отчетами
3. Дополнить существующие тесты недостающими сценариями

### Этап 3: Комплексные сценарии (1-2 дня)
1. Создать `integration_scenarios_test.go`
2. Реализовать сценарии взаимодействия между доменами
3. Добавить тесты изоляции данных и безопасности

### Этап 4: Оптимизация и покрытие (1 день)
1. Проверить покрытие интеграционных тестов
2. Оптимизировать производительность тестов
3. Добавить недостающие edge cases

## Технические требования

### Структура тестов
- Использовать `testhelpers.SetupHTTPServer(t)` для настройки сервера
- Следовать паттерну AAA (Arrange, Act, Assert)
- Использовать table-driven tests где применимо
- Обеспечить очистку данных между тестами

### Валидация данных
- Проверять HTTP статус коды
- Валидировать структуру ответов
- Тестировать граничные случаи
- Проверять обработку ошибок

### Производительность
- Использовать `testcontainers-go` для изоляции
- Оптимизировать создание тестовых данных
- Применять параллельное выполнение где возможно

## Критерии готовности

- [ ] Все основные CRUD операции покрыты интеграционными тестами
- [ ] Покрытие интеграционными тестами > 80% для новых модулей  
- [ ] Все тесты проходят в CI/CD pipeline
- [ ] Добавлены тесты для основных бизнес-сценариев
- [ ] Проверена изоляция данных между семьями
- [ ] Документированы сложные тестовые сценарии

## Готовая инфраструктура

### Тестовые хелперы (`internal/testhelpers`)
Уже доступны следующие фабрики для создания тестовых данных:
- ✅ `CreateTestFamily()` - создание тестовой семьи
- ✅ `CreateTestUser(familyID)` - создание пользователя
- ✅ `CreateTestCategory(familyID, type)` - создание категории
- ✅ `CreateTestTransaction(familyID, userID, categoryID, type)` - создание транзакции
- ✅ `CreateTestBudget(familyID, categoryID)` - создание бюджета
- ✅ `CreateTestReport(familyID, userID)` - создание отчета
- ✅ `SetupHTTPServer(t)` - настройка HTTP сервера для тестов

### Константы для тестов
- `TestTransactionAmount = 100.50` - стандартная сумма транзакции
- `TestBudgetAmount = 1000.0` - стандартная сумма бюджета
- `TestReportExpenses = 500.0` - стандартная сумма расходов в отчете

### Структура API endpoints
Из анализа кода найдены следующие эндпоинты:

**Транзакции:**
- ✅ `POST /api/v1/transactions` - CreateTransaction
- ✅ `GET /api/v1/transactions` - GetTransactions (с фильтрами)
- ✅ `GET /api/v1/transactions/{id}` - GetTransactionByID
- ✅ `PUT /api/v1/transactions/{id}` - UpdateTransaction
- ✅ `DELETE /api/v1/transactions/{id}` - DeleteTransaction

**Бюджеты:**
- ✅ `POST /api/v1/budgets` - CreateBudget
- ✅ `GET /api/v1/budgets` - GetBudgets (с фильтрами)
- ✅ `GET /api/v1/budgets/{id}` - GetBudgetByID
- ✅ `PUT /api/v1/budgets/{id}` - UpdateBudget
- ✅ `DELETE /api/v1/budgets/{id}` - DeleteBudget

**Отчеты:**
- ✅ `POST /api/v1/reports` - CreateReport
- ✅ `GET /api/v1/reports` - GetReports (с фильтрами)
- ✅ `GET /api/v1/reports/{id}` - GetReportByID
- ✅ `DELETE /api/v1/reports/{id}` - DeleteReport

### Типы запросов (готовы к использованию)
**Транзакции:**
- `CreateTransactionRequest` - amount(float64), type(string), description(string), category_id, user_id, family_id, date, tags
- `UpdateTransactionRequest` - все поля опциональные через указатели

**Бюджеты:**
- `CreateBudgetRequest` - name(string), amount(float64), period(string), category_id(optional), family_id, start_date, end_date
- `UpdateBudgetRequest` - все поля опциональные через указатели, включая is_active

**Отчеты:**
- `CreateReportRequest` - name(string), type(string: expenses|income|budget|cash_flow|category_break), period(string: daily|weekly|monthly|yearly|custom), family_id, user_id, start_date, end_date
- Фильтры через query parameters: family_id (обязательный), user_id (опциональный)

## Примеры использования

### Базовый паттерн теста
```go
func TestTransactionHandler_Integration(t *testing.T) {
    testServer := testhelpers.SetupHTTPServer(t)
    
    // Создание тестовых данных
    family := testhelpers.CreateTestFamily()
    err := testServer.Repos.Family.Create(context.Background(), family)
    require.NoError(t, err)
    
    user := testhelpers.CreateTestUser(family.ID)
    err = testServer.Repos.User.Create(context.Background(), user)
    require.NoError(t, err)
    
    // HTTP запрос
    req := httptest.NewRequest(http.MethodPost, "/api/v1/transactions", body)
    rec := httptest.NewRecorder()
    testServer.Server.Echo().ServeHTTP(rec, req)
    
    // Проверки
    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

## Примечания

- Приоритизировать тесты транзакций и бюджетов как основную функциональность
- Использовать существующие тестовые хелперы из `internal/testhelpers`
- Следовать стилю кода существующих тестов
- Обеспечить совместимость с `make test-fast` для быстрого выполнения
- Все необходимые фабрики данных уже готовы к использованию
# Testing Strategy - Стратегия тестирования

## 🎯 Философия тестирования

### Принципы
- **Test-Driven Development (TDD)**: Тесты пишутся до кода
- **Пирамида тестирования**: Больше unit тестов, меньше интеграционных и E2E
- **Fail Fast**: Тесты должны быстро обнаруживать проблемы
- **Независимость**: Тесты не должны зависеть друг от друга
- **Читаемость**: Тесты как живая документация
- **Автоматизация**: Все тесты запускаются автоматически

### Цели тестирования
1. **Качество кода** - предотвращение багов
2. **Рефакторинг** - безопасные изменения кода
3. **Документация** - понимание поведения системы
4. **Уверенность** - в работоспособности системы

## 🏗️ Пирамида тестирования

### Структура тестов
```
        🔺 E2E Tests (5%)
       🔺🔺 Integration Tests (15%)
    🔺🔺🔺🔺 Unit Tests (80%)
```

### Распределение усилий
- **Unit Tests (80%)**: Быстрые, изолированные, множество сценариев
- **Integration Tests (15%)**: Взаимодействие компонентов
- **E2E Tests (5%)**: Критические пользовательские сценарии

## 🧪 Unit Testing

### Структура unit тестов
```go
func TestFunctionName_Scenario_ExpectedResult(t *testing.T) {
    // Arrange - подготовка данных
    // Act - выполнение тестируемого кода
    // Assert - проверка результата
}
```

### Пример unit теста
```go
func TestFamilyService_CreateFamily_Success(t *testing.T) {
    // Arrange
    mockRepo := &MockFamilyRepository{}
    mockValidator := &MockValidator{}
    mockLogger := &MockLogger{}
    
    service := NewFamilyService(mockRepo, mockValidator, mockLogger)
    
    request := CreateFamilyRequest{
        Name:    "Test Family",
        OwnerID: "user-123",
    }
    
    expectedFamily := &Family{
        ID:       "family-456",
        Name:     "Test Family",
        OwnerID:  "user-123",
        IsActive: true,
    }
    
    mockValidator.On("Validate", request).Return(nil)
    mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*Family")).Return(nil)
    mockLogger.On("Info", mock.Anything).Return()
    
    // Act
    family, err := service.CreateFamily(context.Background(), request)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, family)
    assert.Equal(t, expectedFamily.Name, family.Name)
    assert.Equal(t, expectedFamily.OwnerID, family.OwnerID)
    assert.True(t, family.IsActive)
    assert.NotEmpty(t, family.ID)
    
    mockRepo.AssertExpectations(t)
    mockValidator.AssertExpectations(t)
}
```

### Тестирование ошибок
```go
func TestFamilyService_CreateFamily_ValidationError(t *testing.T) {
    // Arrange
    mockRepo := &MockFamilyRepository{}
    mockValidator := &MockValidator{}
    service := NewFamilyService(mockRepo, mockValidator, nil)
    
    request := CreateFamilyRequest{
        Name:    "", // Невалидное имя
        OwnerID: "user-123",
    }
    
    validationErr := errors.ValidationError("Name is required", "name")
    mockValidator.On("Validate", request).Return(validationErr)
    
    // Act
    family, err := service.CreateFamily(context.Background(), request)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, family)
    
    var appErr *errors.AppError
    assert.True(t, errors.As(err, &appErr))
    assert.Equal(t, "VALIDATION_ERROR", appErr.Code())
    
    mockValidator.AssertExpectations(t)
    mockRepo.AssertNotCalled(t, "Save")
}
```

### Table-driven тесты
```go
func TestTransactionService_CalculateMonthlyTotal(t *testing.T) {
    tests := []struct {
        name         string
        transactions []Transaction
        expected     decimal.Decimal
    }{
        {
            name:         "empty transactions",
            transactions: []Transaction{},
            expected:     decimal.Zero,
        },
        {
            name: "single transaction",
            transactions: []Transaction{
                {Amount: decimal.NewFromFloat(100.50), Type: "income"},
            },
            expected: decimal.NewFromFloat(100.50),
        },
        {
            name: "mixed transactions",
            transactions: []Transaction{
                {Amount: decimal.NewFromFloat(1000), Type: "income"},
                {Amount: decimal.NewFromFloat(200), Type: "expense"},
                {Amount: decimal.NewFromFloat(300), Type: "expense"},
            },
            expected: decimal.NewFromFloat(500), // 1000 - 200 - 300
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            service := NewTransactionService()
            
            result := service.CalculateMonthlyTotal(tt.transactions)
            
            assert.True(t, tt.expected.Equal(result))
        })
    }
}
```

## 🔗 Integration Testing

### Тестирование репозиториев
```go
func TestFamilyRepository_Integration(t *testing.T) {
    // Подключение к тестовой БД
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    repo := NewFamilyRepository(db)
    
    t.Run("CreateAndGet", func(t *testing.T) {
        // Arrange
        family := &Family{
            ID:       "test-family-1",
            Name:     "Test Family",
            OwnerID:  "user-123",
            IsActive: true,
        }
        
        // Act - Save
        err := repo.Save(context.Background(), family)
        assert.NoError(t, err)
        
        // Act - Get
        retrieved, err := repo.GetByID(context.Background(), family.ID)
        
        // Assert
        assert.NoError(t, err)
        assert.NotNil(t, retrieved)
        assert.Equal(t, family.ID, retrieved.ID)
        assert.Equal(t, family.Name, retrieved.Name)
        assert.Equal(t, family.OwnerID, retrieved.OwnerID)
    })
    
    t.Run("GetNonExistent", func(t *testing.T) {
        // Act
        family, err := repo.GetByID(context.Background(), "non-existent-id")
        
        // Assert
        assert.Error(t, err)
        assert.Nil(t, family)
        
        var appErr *errors.AppError
        assert.True(t, errors.As(err, &appErr))
        assert.Equal(t, "RESOURCE_NOT_FOUND", appErr.Code())
    })
}

func setupTestDB(t *testing.T) *gorm.DB {
    // Используем in-memory SQLite для тестов
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    
    // Автомиграция схемы
    err = db.AutoMigrate(&Family{}, &FamilyMember{}, &Transaction{})
    require.NoError(t, err)
    
    return db
}

func cleanupTestDB(t *testing.T, db *gorm.DB) {
    sqlDB, err := db.DB()
    require.NoError(t, err)
    sqlDB.Close()
}
```

### Testcontainers для реальной БД
```go
func TestFamilyRepository_WithPostgres(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Запуск PostgreSQL в контейнере
    ctx := context.Background()
    container, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(5*time.Second)),
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)
    
    // Получение connection string
    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    // Подключение к БД
    db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
    require.NoError(t, err)
    
    // Миграции
    err = db.AutoMigrate(&Family{}, &FamilyMember{}, &Transaction{})
    require.NoError(t, err)
    
    // Тесты
    repo := NewFamilyRepository(db)
    
    t.Run("ComplexQueries", func(t *testing.T) {
        // Тестирование сложных запросов на реальной БД
    })
}
```

## 🌐 API Testing

### Тестирование HTTP handlers
```go
func TestFamilyHandler_CreateFamily(t *testing.T) {
    // Arrange
    mockService := &MockFamilyService{}
    handler := NewFamilyHandler(mockService)
    
    router := gin.New()
    router.POST("/families", handler.CreateFamily)
    
    requestBody := CreateFamilyRequest{
        Name: "Test Family",
    }
    
    expectedFamily := &Family{
        ID:   "family-123",
        Name: "Test Family",
    }
    
    mockService.On("CreateFamily", mock.Anything, mock.AnythingOfType("CreateFamilyRequest")).
        Return(expectedFamily, nil)
    
    // Act
    w := httptest.NewRecorder()
    body, _ := json.Marshal(requestBody)
    req, _ := http.NewRequest("POST", "/families", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer valid-token")
    
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, http.StatusCreated, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    
    data := response["data"].(map[string]interface{})
    assert.Equal(t, expectedFamily.ID, data["id"])
    assert.Equal(t, expectedFamily.Name, data["name"])
    
    mockService.AssertExpectations(t)
}
```

### Contract тестирование
```go
func TestAPI_ContractCompliance(t *testing.T) {
    // Загружаем OpenAPI спецификацию
    swagger, err := loads.Spec("../../api/openapi.yaml")
    require.NoError(t, err)
    
    // Создаем тестовый сервер
    server := setupTestServer()
    defer server.Close()
    
    t.Run("CreateFamily_MatchesSchema", func(t *testing.T) {
        // Prepare request
        reqBody := map[string]interface{}{
            "name": "Test Family",
        }
        
        // Make request
        resp := makeAPIRequest(t, server.URL, "POST", "/api/v1/families", reqBody)
        
        // Validate response against schema
        err := validateResponseSchema(swagger, "/families", "post", resp)
        assert.NoError(t, err)
    })
}

func validateResponseSchema(swagger *loads.Document, path, method string, resp *http.Response) error {
    // Validate response against OpenAPI schema
    // Implementation depends on chosen validation library
    return nil
}
```

## 🎭 Test Doubles (Моки и стабы)

### Использование testify/mock
```go
//go:generate mockery --name=FamilyRepository --case=underscore

type MockFamilyRepository struct {
    mock.Mock
}

func (m *MockFamilyRepository) GetByID(ctx context.Context, id string) (*Family, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Family), args.Error(1)
}

func (m *MockFamilyRepository) Save(ctx context.Context, family *Family) error {
    args := m.Called(ctx, family)
    return args.Error(0)
}

func (m *MockFamilyRepository) Delete(ctx context.Context, id string) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}
```

### Фикстуры и тестовые данные
```go
package fixtures

func CreateTestFamily() *Family {
    return &Family{
        ID:        "family-test-1",
        Name:      "Test Family",
        OwnerID:   "user-test-1",
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
        IsActive:  true,
        Members: []FamilyMember{
            {
                UserID:   "user-test-1",
                Role:     RoleOwner,
                JoinedAt: time.Now().UTC(),
            },
        },
    }
}

func CreateTestTransaction(familyID string) *Transaction {
    return &Transaction{
        ID:          "tx-test-1",
        FamilyID:    familyID,
        Amount:      decimal.NewFromFloat(100.50),
        Type:        TransactionTypeExpense,
        Category:    "food",
        Description: "Grocery shopping",
        CreatedAt:   time.Now().UTC(),
    }
}

func CreateTestUser() *User {
    return &User{
        ID:       "user-test-1",
        Email:    "test@example.com",
        Name:     "Test User",
        IsActive: true,
    }
}
```

## 🎪 End-to-End Testing

### Структура E2E тестов
```go
package e2e

func TestFamilyManagement_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    
    // Setup
    server := setupE2EServer(t)
    defer server.Close()
    
    client := NewAPIClient(server.URL)
    
    t.Run("CompleteUserJourney", func(t *testing.T) {
        // 1. Create user account
        user := createTestUser(t, client)
        
        // 2. Authenticate
        token := authenticateUser(t, client, user)
        client.SetAuth(token)
        
        // 3. Create family
        family := createFamily(t, client, "My Family")
        
        // 4. Add family member
        member := addFamilyMember(t, client, family.ID, "member@example.com")
        
        // 5. Create transactions
        tx1 := createTransaction(t, client, family.ID, TransactionRequest{
            Amount:      1000.00,
            Type:        "income",
            Description: "Salary",
        })
        
        tx2 := createTransaction(t, client, family.ID, TransactionRequest{
            Amount:      200.00,
            Type:        "expense",
            Description: "Groceries",
        })
        
        // 6. Get family report
        report := getFamilyReport(t, client, family.ID, "2024-12")
        
        // 7. Verify report data
        assert.Equal(t, 1000.00, report.TotalIncome)
        assert.Equal(t, 200.00, report.TotalExpenses)
        assert.Equal(t, 800.00, report.Balance)
        assert.Len(t, report.Transactions, 2)
        
        // 8. Delete family
        deleteFamily(t, client, family.ID)
        
        // 9. Verify family is deleted
        _, err := client.GetFamily(family.ID)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "404")
    })
}

func setupE2EServer(t *testing.T) *httptest.Server {
    // Setup database
    db := setupTestDatabase(t)
    
    // Setup services
    familyRepo := NewFamilyRepository(db)
    familyService := NewFamilyService(familyRepo, NewValidator(), NewLogger())
    
    // Setup router
    router := setupRouter(familyService)
    
    return httptest.NewServer(router)
}
```

## 📊 Test Coverage

### Настройка coverage
```bash
# Запуск тестов с coverage
go test -v -cover ./...

# Детальный отчет
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Покрытие по функциям
go tool cover -func=coverage.out
```

### Цели по покрытию
- **Unit Tests**: > 80% для бизнес-логики
- **Integration Tests**: > 60% для репозиториев
- **Overall Coverage**: > 75%

### Исключения из coverage
```go
//go:build !test
// +build !test

// Файлы, исключаемые из тестирования
```

## 🚀 Performance Testing

### Benchmark тесты
```go
func BenchmarkTransactionService_CalculateBalance(b *testing.B) {
    service := NewTransactionService()
    transactions := generateTestTransactions(1000) // 1000 транзакций
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _ = service.CalculateBalance(transactions)
    }
}

func BenchmarkFamilyRepository_GetByID(b *testing.B) {
    db := setupBenchmarkDB(b)
    repo := NewFamilyRepository(db)
    
    // Подготовка данных
    family := createTestFamily()
    repo.Save(context.Background(), family)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, _ = repo.GetByID(context.Background(), family.ID)
    }
}
```

### Load Testing с Go
```go
func TestAPI_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test")
    }
    
    server := setupLoadTestServer(t)
    defer server.Close()
    
    const (
        concurrency = 50
        requests    = 1000
    )
    
    var wg sync.WaitGroup
    requestChan := make(chan struct{}, requests)
    results := make(chan time.Duration, requests)
    
    // Заполняем канал запросами
    for i := 0; i < requests; i++ {
        requestChan <- struct{}{}
    }
    close(requestChan)
    
    // Запускаем воркеров
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for range requestChan {
                start := time.Now()
                makeLoadTestRequest(server.URL)
                results <- time.Since(start)
            }
        }()
    }
    
    wg.Wait()
    close(results)
    
    // Анализ результатов
    var durations []time.Duration
    for duration := range results {
        durations = append(durations, duration)
    }
    
    avg := calculateAverage(durations)
    p95 := calculatePercentile(durations, 95)
    
    assert.Less(t, avg, 100*time.Millisecond, "Average response time too high")
    assert.Less(t, p95, 200*time.Millisecond, "95th percentile too high")
}
```

## 🔧 Test Automation

### Makefile команды
```makefile
# Makefile
.PHONY: test test-unit test-integration test-e2e test-coverage test-bench

test: test-unit test-integration

test-unit:
	go test -v -short ./...

test-integration:
	go test -v -run Integration ./...

test-e2e:
	go test -v -run E2E ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

test-bench:
	go test -v -bench=. -benchmem ./...

test-race:
	go test -v -race ./...

test-clean:
	go clean -testcache
```

### CI/CD Pipeline
```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run unit tests
      run: make test-unit
    
    - name: Run integration tests
      run: make test-integration
      env:
        DB_HOST: localhost
        DB_PORT: 5432
        DB_NAME: testdb
        DB_USER: postgres
        DB_PASSWORD: postgres
    
    - name: Generate coverage
      run: make test-coverage
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## 📋 Testing Checklist

### Перед коммитом
- [ ] Все unit тесты проходят
- [ ] Новые функции покрыты тестами
- [ ] Тесты читаемы и понятны
- [ ] Моки используются правильно
- [ ] Нет зависимостей между тестами

### При создании новой функции
- [ ] TDD подход (тест -> код -> рефакторинг)
- [ ] Тестируются все граничные случаи
- [ ] Тестируются ошибочные сценарии
- [ ] Добавлены integration тесты при необходимости
- [ ] Обновлена документация

### При рефакторинге
- [ ] Все существующие тесты проходят
- [ ] Поведение системы не изменилось
- [ ] Тесты адаптированы к новой структуре
- [ ] Performance не ухудшился

## 🎯 Best Practices

### DO ✅
- **Пишите тесты до кода** (TDD)
- **Тестируйте поведение**, а не реализацию
- **Используйте понятные имена** для тестов
- **Изолируйте тесты** друг от друга
- **Тестируйте граничные случаи**
- **Используйте фикстуры** для тестовых данных
- **Мокайте внешние зависимости**

### DON'T ❌
- **Не игнорируйте упавшие тесты**
- **Не тестируйте приватные методы** напрямую
- **Не создавайте флакающие тесты**
- **Не пишите слишком сложные тесты**
- **Не забывайте чистить ресурсы**
- **Не смешивайте unit и integration тесты**

---

*Документ создан: 2024*  
*Владелец: QA Team*  
*Регулярность обновлений: ежемесячно*
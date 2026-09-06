# Coding Standards - Стандарты кодирования

## 🎯 Общие принципы

### Философия кода
- **Читаемость превыше всего**: Код читается чаще, чем пишется
- **Простота и ясность**: Избегайте излишней сложности
- **Консистентность**: Единый стиль во всем проекте
- **Самодокументируемый код**: Код должен объяснять себя сам
- **DRY (Don't Repeat Yourself)**: Избегайте дублирования
- **SOLID принципы**: Следуйте принципам объектно-ориентированного дизайна

### Приоритеты
1. **Корректность** - код должен работать правильно
2. **Читаемость** - код должен быть понятным
3. **Производительность** - оптимизируйте только там, где нужно
4. **Красота** - эстетика кода тоже важна

## 🔧 Go Code Style

### Именование

#### Переменные
```go
// ✅ Хорошо
var userID string
var familyMembers []Member
var isActive bool
var httpClient *http.Client

// ❌ Плохо
var userId string      // snake_case в Go не используется
var family_members []Member
var is_active bool
var HTTPClient *http.Client // Избыточные заглавные
```

#### Функции и методы
```go
// ✅ Хорошо
func GetUserByID(id string) (*User, error)
func (s *FamilyService) CreateFamily(ctx context.Context, req CreateFamilyRequest) (*Family, error)
func validateEmail(email string) bool

// ❌ Плохо
func getUserById(id string) (*User, error)  // camelCase для публичных
func (s *FamilyService) create_family(ctx context.Context, req CreateFamilyRequest) (*Family, error)
```

#### Константы
```go
// ✅ Хорошо
const (
    DefaultPageSize = 20
    MaxPageSize     = 100
    DatabaseTimeout = 30 * time.Second
)

// Группировка по смыслу
const (
    // Transaction types
    TransactionTypeIncome  = "income"
    TransactionTypeExpense = "expense"

    // Transaction statuses
    TransactionStatusPending   = "pending"
    TransactionStatusCompleted = "completed"
)

// ❌ Плохо
const DEFAULT_PAGE_SIZE = 20  // snake_case не используется
const database_timeout = 30   // приватная константа должна начинаться с маленькой буквы
```

#### Интерфейсы
```go
// ✅ Хорошо
type FamilyRepository interface {
    GetByID(ctx context.Context, id string) (*Family, error)
    Create(ctx context.Context, family *Family) error
    Update(ctx context.Context, family *Family) error
    Delete(ctx context.Context, id string) error
}

type Validator interface {
    Validate(data any) error
}

// ❌ Плохо
type IFamilyRepository interface { // Не используем I префикс
    // ...
}

type FamilyRepositoryInterface interface { // Избыточное название
    // ...
}
```

### Структуры

#### Определение структур
```go
// ✅ Хорошо
type Family struct {
    ID          string           `json:"id" db:"id"`
    Name        string           `json:"name" db:"name" validate:"required,min=1,max=100"`
    CreatedAt   time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
    Members     []FamilyMember   `json:"members,omitempty"`
    Settings    FamilySettings   `json:"settings"`
    IsActive    bool             `json:"is_active" db:"is_active"`
}

type FamilySettings struct {
    Currency                string `json:"currency" validate:"required,iso4217"`
    NotificationsEnabled    bool   `json:"notifications_enabled"`
    AutoCategorizationEnabled bool `json:"auto_categorization_enabled"`
}

// ❌ Плохо
type family struct { // Публичная структура должна начинаться с заглавной
    id        string    // Поля должны быть публичными если структура экспортируется
    name      string
    createdAt time.Time
}
```

#### Методы структур
```go
// ✅ Хорошо
func (f *Family) AddMember(member FamilyMember) error {
    if f.Members == nil {
        f.Members = make([]FamilyMember, 0)
    }

    // Проверяем, что член семьи еще не добавлен
    for _, existingMember := range f.Members {
        if existingMember.UserID == member.UserID {
            return errors.BusinessError("MEMBER_ALREADY_EXISTS", "Member already exists in family")
        }
    }

    f.Members = append(f.Members, member)
    f.UpdatedAt = time.Now().UTC()

    return nil
}

func (f *Family) IsOwner(userID string) bool {
    for _, member := range f.Members {
        if member.UserID == userID && member.Role == RoleOwner {
            return true
        }
    }
    return false
}

// ❌ Плохо
func (f *Family) addMember(member FamilyMember) { // Должен быть публичным
    f.Members = append(f.Members, member) // Нет валидации
}
```

### Функции

#### Сигнатуры функций
```go
// ✅ Хорошо
func CreateFamily(ctx context.Context, name string, ownerID string) (*Family, error) {
    if strings.TrimSpace(name) == "" {
        return nil, errors.ValidationError("Family name is required", "name")
    }

    family := &Family{
        ID:        generateID(),
        Name:      strings.TrimSpace(name),
        CreatedAt: time.Now().UTC(),
        UpdatedAt: time.Now().UTC(),
        IsActive:  true,
        Members: []FamilyMember{
            {
                UserID: ownerID,
                Role:   RoleOwner,
                JoinedAt: time.Now().UTC(),
            },
        },
    }

    return family, nil
}

// ❌ Плохо
func createFamily(name string, ownerId string) *Family { // Нет контекста и обработки ошибок
    return &Family{
        Name: name, // Нет валидации
    }
}
```

#### Возврат ошибок
```go
// ✅ Хорошо
func (r *familyRepository) GetByID(ctx context.Context, id string) (*Family, error) {
    var family Family

    err := r.db.GetContext(ctx, &family, "SELECT * FROM families WHERE id = $1", id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, errors.NotFoundError("family", id)
        }
        return nil, errors.InternalError("Failed to get family from database", err)
    }

    return &family, nil
}

// ❌ Плохо
func (r *familyRepository) GetByID(ctx context.Context, id string) (*Family, error) {
    var family Family
    err := r.db.GetContext(ctx, &family, "SELECT * FROM families WHERE id = $1", id)
    return &family, err // Не обрабатываем разные типы ошибок
}
```

### Комментарии

#### Документирование функций
```go
// ✅ Хорошо

// CreateTransaction создает новую финансовую транзакцию для указанной семьи.
// Функция валидирует входные данные и проверяет права доступа пользователя.
//
// Возвращает ошибку если:
// - семья не найдена
// - пользователь не является членом семьи
// - данные транзакции невалидны
func CreateTransaction(ctx context.Context, familyID string, userID string, req CreateTransactionRequest) (*Transaction, error) {
    // Реализация...
}

// GetFamilyMonthlyReport возвращает отчет о доходах и расходах семьи за указанный месяц.
// Параметр month должен быть в формате "2006-01" (YYYY-MM).
func GetFamilyMonthlyReport(ctx context.Context, familyID string, month string) (*MonthlyReport, error) {
    // Реализация...
}

// ❌ Плохо
// Создает транзакцию
func CreateTransaction(ctx context.Context, familyID string, userID string, req CreateTransactionRequest) (*Transaction, error) {
    // Реализация...
}
```

#### Комментарии в коде
```go
// ✅ Хорошо
func (s *TransactionService) CategorizeTransaction(transaction *Transaction) error {
    // Если категория уже установлена, ничего не делаем
    if transaction.CategoryID != "" {
        return nil
    }

    // Пытаемся определить категорию по описанию и сумме
    suggestedCategory, confidence := s.categorizer.SuggestCategory(
        transaction.Description,
        transaction.AmountMinor,
    )

    // Устанавливаем категорию только если уверенность > 80%
    if confidence > 0.8 {
        transaction.CategoryID = suggestedCategory.ID
        transaction.AutoCategorized = true
    }

    return nil
}

// ❌ Плохо
func (s *TransactionService) CategorizeTransaction(transaction *Transaction) error {
    // проверяем категорию
    if transaction.CategoryID != "" {
        return nil
    }

    // получаем категорию
    cat, conf := s.categorizer.SuggestCategory(transaction.Description, transaction.AmountMinor)

    // устанавливаем если хорошая
    if conf > 0.8 {
        transaction.CategoryID = cat.ID
        transaction.AutoCategorized = true
    }

    return nil
}
```

## 🏗️ Архитектурные принципы

### Clean Architecture

#### Слои приложения
```go
// Domain Layer - Бизнес-логика
package domain

type Family struct {
    ID       string
    Name     string
    Members  []FamilyMember
    Settings FamilySettings
}

type FamilyRepository interface {
    GetByID(ctx context.Context, id string) (*Family, error)
    Save(ctx context.Context, family *Family) error
}

// Use Cases Layer - Прикладная логика
package usecases

type FamilyUseCase struct {
    familyRepo domain.FamilyRepository
    validator  Validator
}

func (uc *FamilyUseCase) CreateFamily(ctx context.Context, req CreateFamilyRequest) (*Family, error) {
    // Валидация
    if err := uc.validator.Validate(req); err != nil {
        return nil, err
    }

    // Бизнес-логика
    family := domain.NewFamily(req.Name, req.OwnerID)

    // Сохранение
    if err := uc.familyRepo.Save(ctx, family); err != nil {
        return nil, err
    }

    return family, nil
}
```

#### Dependency Injection
```go
// ✅ Хорошо - Внедрение зависимостей через конструктор
type FamilyService struct {
    repo      FamilyRepository
    validator Validator
    logger    Logger
}

func NewFamilyService(repo FamilyRepository, validator Validator, logger Logger) *FamilyService {
    return &FamilyService{
        repo:      repo,
        validator: validator,
        logger:    logger,
    }
}

// ❌ Плохо - Прямые зависимости
type FamilyService struct {
    db *sql.DB
}

func (s *FamilyService) GetFamily(id string) (*Family, error) {
    // Прямое обращение к базе данных
    row := s.db.QueryRow("SELECT * FROM families WHERE id = ?", id)
    // ...
}
```

### Обработка ошибок

#### Типизированные ошибки
```go
// ✅ Хорошо
var (
    ErrFamilyNotFound     = errors.New("family not found")
    ErrInsufficientFunds  = errors.New("insufficient funds")
    ErrBudgetExceeded     = errors.New("budget exceeded")
)

func (s *TransactionService) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*Transaction, error) {
    family, err := s.familyRepo.GetByID(ctx, req.FamilyID)
    if err != nil {
        if errors.Is(err, ErrFamilyNotFound) {
            return nil, errors.NotFoundError("family", req.FamilyID)
        }
        return nil, errors.InternalError("Failed to get family", err)
    }

    // Остальная логика...
}

// ❌ Плохо
func (s *TransactionService) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*Transaction, error) {
    family, err := s.familyRepo.GetByID(ctx, req.FamilyID)
    if err != nil {
        return nil, err // Не обрабатываем специфичные ошибки
    }

    // Остальная логика...
}
```

## 📊 Database & SQL

### Работа с базой данных

#### SQL запросы
```go
// ✅ Хорошо - используем параметризованные запросы
func (r *TransactionRepository) GetByDateRange(ctx context.Context, familyID string, from, to time.Time) ([]Transaction, error) {
    var transactions []Transaction

    query := `
        SELECT t.*, c.name as category_name
        FROM transactions t
        LEFT JOIN categories c ON t.category_id = c.id
        WHERE t.family_id = $1
        AND t.created_at BETWEEN $2 AND $3
        ORDER BY t.created_at DESC
    `

    err := r.db.WithContext(ctx).Raw(query, familyID, from, to).Scan(&transactions).Error
    if err != nil {
        return nil, errors.InternalError("Failed to get transactions", err)
    }

    return transactions, nil
}

// ❌ Плохо - SQL инъекции
func (r *TransactionRepository) GetByDateRange(familyID string, from, to string) ([]Transaction, error) {
    query := fmt.Sprintf("SELECT * FROM transactions WHERE family_id = '%s' AND created_at BETWEEN '%s' AND '%s'",
        familyID, from, to) // Уязвимость к SQL инъекциям!

    // ...
}
```

## 🧪 Тестирование

### Unit тесты

#### Структура тестов
```go
// ✅ Хорошо
func TestFamilyService_CreateFamily(t *testing.T) {
    tests := []struct {
        name          string
        request       CreateFamilyRequest
        mockSetup     func(*MockFamilyRepository)
        expectedError string
        expectedName  string
    }{
        {
            name: "success - valid family creation",
            request: CreateFamilyRequest{
                Name:    "Test Family",
                OwnerID: "user-123",
            },
            mockSetup: func(mockRepo *MockFamilyRepository) {
                mockRepo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Family")).
                    Return(nil)
            },
            expectedName: "Test Family",
        },
        {
            name: "error - empty family name",
            request: CreateFamilyRequest{
                Name:    "",
                OwnerID: "user-123",
            },
            mockSetup:     func(mockRepo *MockFamilyRepository) {},
            expectedError: "VALIDATION_ERROR",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockRepo := &MockFamilyRepository{}
            tt.mockSetup(mockRepo)

            service := NewFamilyService(mockRepo, NewValidator(), NewLogger())

            // Act
            family, err := service.CreateFamily(context.Background(), tt.request)

            // Assert
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
                assert.Nil(t, family)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, family)
                assert.Equal(t, tt.expectedName, family.Name)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}

// ❌ Плохо
func TestCreateFamily(t *testing.T) {
    service := NewFamilyService(nil, nil, nil) // Нет моков
    family, err := service.CreateFamily(context.Background(), CreateFamilyRequest{
        Name: "Test",
    })

    if err != nil {
        t.Error(err) // Не проверяем конкретные ошибки
    }

    if family.Name != "Test" {
        t.Error("Wrong name") // Нет детального сообщения
    }
}
```

#### Использование моков
```go
// ✅ Хорошо - моки с testify
//go:generate mockery --name=FamilyRepository --case=underscore
type MockFamilyRepository struct {
    mock.Mock
}

func (m *MockFamilyRepository) GetByID(ctx context.Context, id string) (*Family, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*Family), args.Error(1)
}

func (m *MockFamilyRepository) Save(ctx context.Context, family *Family) error {
    args := m.Called(ctx, family)
    return args.Error(0)
}
```

## 📐 Форматирование и линтинг

При написании кода в проекте Family Finances Service обязательно проверяй написанное на соответствие следующим стандартам.

### gofmt и goimports
```bash
# Форматирование кода
gofmt -w .
goimports -w .
```

### golangci-lint конфигурация
```yaml
# .golangci.yml
run:
  timeout: 5m
  modules-download-mode: readonly

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true

  govet:
    check-shadowing: true

  golint:
    min-confidence: 0.8

  gocyclo:
    min-complexity: 10

  dupl:
    threshold: 100

  goconst:
    min-len: 3
    min-occurrences: 3

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - varcheck
    - deadcode
    - golint
    - gocyclo
    - dupl
    - goconst
    - misspell
    - unparam
    - nakedret
    - prealloc
```

## 📝 Документация кода

### Godoc комментарии
```go
// ✅ Хорошо
// Package usecases содержит бизнес-логику приложения Family Finances Service.
// Этот пакет реализует все use cases согласно принципам Clean Architecture.
package usecases

// FamilyUseCase обрабатывает все операции, связанные с семьями.
// Содержит бизнес-логику для создания, обновления и управления семьями.
type FamilyUseCase struct {
    familyRepo domain.FamilyRepository
    validator  Validator
    logger     Logger
}

// CreateFamily создает новую семью с указанным владельцем.
//
// Функция выполняет следующие операции:
//   - Валидирует входные данные
//   - Создает новую семью с владельцем
//   - Сохраняет семью в репозитории
//   - Отправляет уведомление о создании
//
// Возвращает ошибку если:
//   - Данные невалидны
//   - Семья с таким именем уже существует
//   - Произошла ошибка при сохранении
func (uc *FamilyUseCase) CreateFamily(ctx context.Context, req CreateFamilyRequest) (*Family, error) {
    // Реализация...
}
```

## 🚀 Performance & Optimization

### Эффективная работа с памятью
```go
// ✅ Хорошо - переиспользование слайсов
func (s *TransactionService) ProcessTransactions(transactions []Transaction) []ProcessedTransaction {
    // Предварительно выделяем память
    result := make([]ProcessedTransaction, 0, len(transactions))

    for _, tx := range transactions {
        processed := s.processTransaction(tx)
        result = append(result, processed)
    }

    return result
}

// ✅ Хорошо - пулы объектов для часто создаваемых структур
var transactionPool = sync.Pool{
    New: func() any {
        return &Transaction{}
    },
}

func (s *TransactionService) CreateTransaction(req CreateTransactionRequest) *Transaction {
    tx := transactionPool.Get().(*Transaction)
    defer transactionPool.Put(tx)

    // Инициализация...
    return tx
}

// ❌ Плохо - неэффективная работа с памятью
func (s *TransactionService) ProcessTransactions(transactions []Transaction) []ProcessedTransaction {
    var result []ProcessedTransaction // Не предварительное выделение памяти

    for _, tx := range transactions {
        result = append(result, s.processTransaction(tx)) // Множественные аллокации
    }

    return result
}
```

### Работа с горутинами
```go
// ✅ Хорошо - контролируемый параллелизм
func (s *ReportService) GenerateMonthlyReports(ctx context.Context, familyIDs []string) error {
    const maxWorkers = 10

    jobs := make(chan string, len(familyIDs))
    results := make(chan error, len(familyIDs))

    // Запускаем воркеров
    for i := 0; i < maxWorkers; i++ {
        go s.reportWorker(ctx, jobs, results)
    }

    // Отправляем задачи
    for _, familyID := range familyIDs {
        jobs <- familyID
    }
    close(jobs)

    // Собираем результаты
    for i := 0; i < len(familyIDs); i++ {
        if err := <-results; err != nil {
            return err
        }
    }

    return nil
}

func (s *ReportService) reportWorker(ctx context.Context, jobs <-chan string, results chan<- error) {
    for familyID := range jobs {
        err := s.generateFamilyReport(ctx, familyID)
        results <- err
    }
}
```

## ✅ Чек-лист для Code Review

### Перед коммитом
- [ ] Код отформатирован с помощью `gofmt`
- [ ] Импорты упорядочены с помощью `goimports`
- [ ] Все линтеры проходят без ошибок
- [ ] Тесты написаны и проходят
- [ ] Покрытие тестами не снизилось
- [ ] Документация обновлена при необходимости

### Во время ревью
- [ ] Код соответствует архитектурным принципам
- [ ] Ошибки обрабатываются корректно
- [ ] Именование переменных и функций понятно
- [ ] Нет дублирования кода
- [ ] Производительность кода приемлема
- [ ] Безопасность учтена (SQL инъекции, XSS и т.д.)

---

*Документ создан: 2025*
*Владелец: Development Team*
*Регулярность обновлений: при изменении стандартов*

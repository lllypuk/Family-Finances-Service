# New Feature Workflow - Процесс разработки новых функций

## 🎯 Цель процесса

Обеспечить структурированный, качественный и контролируемый процесс разработки новых функций в Family Finances Service от идеи до релиза в продакшен.

## 🔄 Общий обзор процесса

```
Идея → Исследование → Планирование → Разработка → Тестирование → Релиз → Мониторинг
```

### Этапы и временные рамки
- **📋 Планирование**: 1-2 недели
- **🏗️ Разработка**: 2-4 недели
- **🧪 Тестирование**: 1 неделя
- **🚀 Релиз**: 1-3 дня
- **📊 Мониторинг**: Непрерывно

## 📋 Этап 1: Планирование и исследование

### 1.1 Создание Feature Specification
```bash
# Создать новую спецификацию
cp .memory_bank/specs/feature_xyz.md .memory_bank/specs/feature_[name].md
```

#### Checklist для спецификации
- [ ] Бизнес-контекст и проблема описаны
- [ ] Пользовательские сценарии детализированы
- [ ] Функциональные требования определены
- [ ] Нефункциональные требования установлены
- [ ] API контракт спроектирован
- [ ] Модель данных продумана
- [ ] Риски и митигации выявлены

### 1.2 Technical Design Review
#### Участники
- **Tech Lead** (обязательно)
- **Senior Engineers** (2-3 человека)
- **Product Owner** (для валидации требований)
- **DevOps Engineer** (для инфраструктурных вопросов)

#### Agenda для Design Review
1. **Обзор требований** (10 мин)
2. **Архитектурное решение** (15 мин)
3. **API дизайн** (10 мин)
4. **Модель данных** (10 мин)
5. **Интеграции** (5 мин)
6. **Риски и митигации** (5 мин)
7. **Questions & Answers** (5 мин)

#### Design Review Checklist
- [ ] Решение соответствует архитектурным принципам
- [ ] API следует установленным стандартам
- [ ] Безопасность учтена на всех уровнях
- [ ] Производительность будет соответствовать требованиям
- [ ] Решение масштабируемо
- [ ] Есть план для мониторинга и наблюдаемости
- [ ] Совместимость с существующими системами
- [ ] План миграции данных (если нужен)

### 1.3 Story Points Estimation
```
Fibonacci sequence: 1, 2, 3, 5, 8, 13, 21

1 point  = ~2-4 часа работы
2 points = ~1 день работы
3 points = ~1.5 дня работы
5 points = ~2-3 дня работы
8 points = ~1 неделя работы
13+ points = требует разбиения на более мелкие задачи
```

#### Planning Poker Session
1. **Product Owner** объясняет требования
2. **Команда** задает уточняющие вопросы
3. **Индивидуальная оценка** (без обсуждения)
4. **Обсуждение** расхождений в оценках
5. **Повторная оценка** до консенсуса

## 🏗️ Этап 2: Разработка

### 2.1 Создание Feature Branch
```bash
# Создание ветки для новой фичи
git checkout develop
git pull origin develop
git checkout -b feature/JIRA-123-feature-name

# Naming conventions:
# feature/JIRA-123-short-description
# feature/family-budgets-api
# feature/transaction-categorization
```

### 2.2 Development Process

#### Test-Driven Development (TDD)
```bash
# 1. Red: Написать failing test
func TestCreateBudget_Success(t *testing.T) {
    // Arrange
    service := NewBudgetService(mockRepo, mockValidator)
    request := CreateBudgetRequest{
        FamilyID: "family-123",
        Amount:   1000.00,
        Period:   "monthly",
    }

    // Act
    budget, err := service.CreateBudget(context.Background(), request)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, budget)
    assert.Equal(t, request.Amount, budget.Amount)
}

# 2. Green: Написать минимальный код для прохождения теста
func (s *BudgetService) CreateBudget(ctx context.Context, req CreateBudgetRequest) (*Budget, error) {
    budget := &Budget{
        ID:       generateID(),
        FamilyID: req.FamilyID,
        Amount:   req.Amount,
        Period:   req.Period,
    }
    return budget, nil
}

# 3. Refactor: Улучшить код без изменения функциональности
```

#### Development Checklist (per task)
- [ ] Тесты написаны до кода (TDD)
- [ ] Код следует стандартам проекта
- [ ] Обработка ошибок реализована
- [ ] Логирование добавлено где необходимо
- [ ] Валидация входных данных
- [ ] Документация в коде (godoc)
- [ ] Performance considerations учтены

### 2.3 Incremental Development
```
Sprint Planning → Daily Development → Mini-Demo → Retrospective
     ↓                    ↓               ↓           ↓
  Task Breakdown → Code + Test → Show Progress → Adjust Plan
```

#### Daily Development Routine
```bash
# Утром
git pull origin develop
git rebase develop  # Убедиться что ветка актуальна
make test          # Запустить тесты
make lint          # Проверить качество кода

# В течение дня
git add .
git commit -m "feat: implement basic budget creation"  # Conventional commits
git push origin feature/budget-creation

# Вечером
# Создать/обновить PR если готово к ревью
```

### 2.4 API First Development

#### OpenAPI Specification
```yaml
# api/openapi.yaml - обновить спецификацию
paths:
  /api/v1/families/{familyId}/budgets:
    post:
      summary: Create family budget
      tags: [Budgets]
      parameters:
        - name: familyId
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateBudgetRequest'
      responses:
        201:
          description: Budget created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/BudgetResponse'
        400:
          $ref: '#/components/responses/ValidationError'
        404:
          $ref: '#/components/responses/NotFound'
```

#### Contract Testing
```bash
# Генерация моков из OpenAPI
make generate-mocks

# Валидация API против спецификации
make validate-api
```

## 🧪 Этап 3: Тестирование

### 3.1 Testing Strategy

#### Testing Pyramid
```
        🔺 E2E Tests (5%)
       🔺🔺 Integration Tests (15%)
    🔺🔺🔺🔺 Unit Tests (80%)
```

#### Unit Testing
```bash
# Запуск unit тестов
go test -v -short ./internal/usecases/budget_test.go
go test -v -short ./internal/domain/budget_test.go

# Coverage для новой функциональности
go test -v -coverprofile=coverage.out ./internal/usecases/
go tool cover -html=coverage.out

# Target: >80% coverage для новой логики
```

#### Integration Testing
```bash
# Тестирование с реальной БД
go test -v -run Integration ./internal/infrastructure/

# Тестирование API endpoints
go test -v -run TestBudget ./internal/interfaces/http/
```

#### E2E Testing
```go
func TestBudgetCreation_E2E(t *testing.T) {
    // Setup test environment
    server := setupE2EServer(t)
    client := NewAPIClient(server.URL)

    // Create test user and family
    user := createTestUser(t, client)
    family := createTestFamily(t, client, user.ID)

    // Test budget creation
    budget := CreateBudgetRequest{
        FamilyID: family.ID,
        Amount:   1500.00,
        Period:   "monthly",
        Category: "groceries",
    }

    createdBudget, err := client.CreateBudget(budget)
    assert.NoError(t, err)
    assert.Equal(t, budget.Amount, createdBudget.Amount)

    // Verify budget in database
    storedBudget, err := client.GetBudget(createdBudget.ID)
    assert.NoError(t, err)
    assert.Equal(t, createdBudget.ID, storedBudget.ID)
}
```

### 3.2 Quality Gates

#### Automated Checks (CI/CD)
```yaml
# .github/workflows/feature-check.yml
name: Feature Quality Check
on:
  pull_request:
    branches: [develop]

jobs:
  quality-check:
    steps:
      - name: Run tests
        run: |
          make test
          make test-coverage

      - name: Lint check
        run: make lint

      - name: Security scan
        run: make security-scan

      - name: Performance check
        run: make benchmark
```

#### Manual QA Checklist
- [ ] Happy path scenarios работают
- [ ] Error cases обрабатываются корректно
- [ ] API responses соответствуют спецификации
- [ ] Валидация входных данных работает
- [ ] Авторизация проверяется
- [ ] Performance в пределах нормы
- [ ] Logs содержат необходимую информацию

## 🔍 Этап 4: Code Review

### 4.1 Pull Request Creation

#### PR Template
```markdown
## Description
Brief description of the changes and the problem they solve.

## Type of Change
- [ ] 🚀 New feature
- [ ] 🐛 Bug fix
- [ ] 📚 Documentation update
- [ ] 🔧 Refactoring
- [ ] ⚡ Performance improvement

## Related Issues
- Closes #123
- Related to #456

## Changes Made
- Added budget creation API endpoint
- Implemented budget validation logic
- Added comprehensive test coverage
- Updated API documentation

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed
- [ ] API documentation updated

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests cover the changes
- [ ] No breaking changes (or breaking changes documented)
```

#### PR Best Practices
- **Размер**: < 400 строк изменений (исключая тесты)
- **Scope**: Одна логическая функция
- **Description**: Понятное объяснение изменений
- **Tests**: Обязательны для нового кода
- **Documentation**: Обновлена при необходимости

### 4.2 Review Process

#### Reviewers Assignment
- **Mandatory**: Tech Lead или Senior Engineer
- **Optional**: Domain expert (если есть)
- **Automatic**: CodeOwners file определяет ревьюеров

#### Review Checklist
##### Architecture & Design
- [ ] Решение соответствует архитектурным принципам
- [ ] Нет нарушений SOLID принципов
- [ ] Dependency injection используется правильно
- [ ] Интерфейсы определены корректно

##### Code Quality
- [ ] Код читаемый и понятный
- [ ] Именование переменных и функций meaningful
- [ ] Нет дублирования кода
- [ ] Error handling реализован правильно
- [ ] Логирование добавлено где нужно

##### Security
- [ ] Input validation присутствует
- [ ] SQL injection protection
- [ ] Authorization checks в месте
- [ ] Sensitive data не логируется

##### Performance
- [ ] Нет N+1 запросов к БД
- [ ] Эффективное использование памяти
- [ ] Pagination для больших наборов данных
- [ ] Appropriate indexing в БД

##### Testing
- [ ] Unit tests покрывают новую логику
- [ ] Edge cases протестированы
- [ ] Error scenarios покрыты тестами
- [ ] Моки используются правильно

### 4.3 Review Comments

#### Comment Types
```markdown
**💡 Suggestion**: Предложение по улучшению
**❓ Question**: Вопрос для понимания
**🐛 Issue**: Проблема, которую нужно исправить
**💭 Nitpick**: Мелкие замечания (не блокирующие)
**🚨 Blocker**: Критичная проблема (блокирует merge)
```

#### Example Comments
```go
// 💡 Suggestion: Consider using a constant for this magic number
const DefaultBudgetPeriod = "monthly"

// ❓ Question: What happens if familyID is empty? Should we validate this?

// 🐛 Issue: This could cause a race condition. Consider using mutex.
var budgetCache = make(map[string]*Budget)

// 🚨 Blocker: This is vulnerable to SQL injection
query := fmt.Sprintf("SELECT * FROM budgets WHERE id = '%s'", id)
```

## 🚀 Этап 5: Релиз

### 5.1 Pre-Release Checklist
- [ ] Все тесты проходят на CI/CD
- [ ] Code review завершен и одобрен
- [ ] Feature flag настроен (если нужен)
- [ ] Database migration готова (если нужна)
- [ ] Monitoring и alerting настроены
- [ ] Rollback plan подготовлен
- [ ] Documentation обновлена

### 5.2 Feature Flags Strategy

#### Когда использовать Feature Flags
- **Новая критичная функциональность**
- **Изменения в существующем API**
- **Эксперименты и A/B тесты**
- **Постепенный rollout**

#### Configuration Example
```json
{
  "budget_management": {
    "enabled": false,
    "rollout_percentage": 0,
    "user_whitelist": ["beta-user-1", "beta-user-2"],
    "family_whitelist": ["family-test-1"]
  }
}
```

#### Implementation
```go
func (h *BudgetHandler) CreateBudget(c *gin.Context) {
    if !h.featureFlags.IsEnabled("budget_management", c.GetString("user_id")) {
        c.JSON(http.StatusNotFound, gin.H{"error": "Feature not available"})
        return
    }

    // Feature implementation
}
```

### 5.3 Deployment Strategy

#### Staged Rollout
```
1. 🧪 Development → Deploy to dev environment
2. 🔬 Staging → Deploy to staging, run E2E tests
3. 🎭 Canary → Deploy to 5% of production traffic
4. 📈 Production → Full rollout if metrics are good
```

#### Deployment Commands
```bash
# Deploy to staging
make deploy-staging

# Run E2E tests on staging
make test-e2e-staging

# Deploy canary (5% traffic)
make deploy-canary

# Monitor canary metrics
make monitor-canary

# Full production rollout
make deploy-production

# Or rollback if issues
make rollback-production
```

### 5.4 Post-Deployment Monitoring

#### Key Metrics to Monitor
- **Response Time**: P50, P95, P99 latency
- **Error Rate**: 4xx and 5xx errors percentage
- **Throughput**: Requests per second
- **Resource Usage**: CPU, Memory, DB connections
- **Business Metrics**: Feature adoption, usage patterns

#### Monitoring Dashboard
```yaml
# Grafana dashboard queries
- Response Time:
    query: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

- Error Rate:
    query: rate(http_requests_total{status=~"4..|5.."}[5m]) / rate(http_requests_total[5m])

- Feature Usage:
    query: rate(budget_creation_total[5m])
```

#### Alerting Rules
```yaml
- name: budget_feature_alerts
  rules:
    - alert: HighErrorRate
      expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.01
      for: 2m
      labels:
        severity: critical
      annotations:
        summary: "High error rate detected for budget feature"

    - alert: SlowResponse
      expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Slow response time for budget endpoints"
```

## 📊 Этап 6: Мониторинг и итерации

### 6.1 Success Metrics

#### Technical Metrics
- **Performance**: < 200ms P95 response time
- **Reliability**: > 99.9% uptime
- **Error Rate**: < 0.1% for new endpoints
- **Coverage**: > 80% test coverage

#### Business Metrics
- **Adoption**: % of families using new feature
- **Engagement**: Daily/Weekly active usage
- **Satisfaction**: User feedback and ratings
- **Conversion**: Impact on key business metrics

### 6.2 Feedback Collection

#### User Feedback Channels
- **In-app feedback**: Feature-specific feedback forms
- **Support tickets**: Monitor for feature-related issues
- **Analytics**: Usage patterns and drop-off points
- **User interviews**: Direct feedback from key users

#### Internal Feedback
- **Team retrospective**: What went well/wrong in development
- **Performance review**: Technical metrics analysis
- **Process improvement**: Lessons learned for next features

### 6.3 Iteration Planning

#### Feature Enhancement Backlog
```markdown
## Enhancement Ideas
- [ ] **[HIGH]** Add budget notifications when limits exceeded
- [ ] **[MED]** Implement budget categories breakdown
- [ ] **[LOW]** Add budget comparison with previous periods

## Bug Fixes
- [ ] **[P1]** Fix budget calculation edge case for leap years
- [ ] **[P2]** Improve error messages for validation failures

## Technical Debt
- [ ] Refactor budget calculation logic for better performance
- [ ] Add more comprehensive integration tests
- [ ] Improve API documentation with more examples
```

## 🛠️ Tools and Templates

### 6.1 Development Tools
```bash
# Code generation
make generate          # Generate mocks, swagger docs
make migrate-create     # Create new database migration
make api-docs          # Update API documentation

# Quality checks
make lint              # Run golangci-lint
make test              # Run all tests
make test-coverage     # Generate coverage report
make security-scan     # Run security analysis

# Local development
make dev               # Start local development environment
make db-reset          # Reset local database
make mock-data         # Load test data
```

### 6.2 Useful Commands
```bash
# Feature branch management
git checkout -b feature/new-feature
git rebase develop     # Keep feature branch up to date
git push origin feature/new-feature

# Testing specific packages
go test -v ./internal/usecases/budget/...
go test -v -run TestCreateBudget
go test -bench=. ./internal/usecases/budget/

# Database operations
migrate -path migrations -database postgres://... up
migrate -path migrations -database postgres://... down 1

# API testing
curl -X POST http://localhost:8080/api/v1/families/123/budgets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '{"amount": 1000, "period": "monthly"}'
```

## ✅ Definition of Done

### Feature Development Complete When:
- [ ] **Requirements**: All acceptance criteria met
- [ ] **Code**: Implemented and follows coding standards
- [ ] **Tests**: Unit, integration, and E2E tests pass
- [ ] **Review**: Code reviewed and approved
- [ ] **Documentation**: API docs and guides updated
- [ ] **Security**: Security review completed
- [ ] **Performance**: Meets performance requirements
- [ ] **Deployment**: Successfully deployed to production
- [ ] **Monitoring**: Metrics and alerts configured
- [ ] **Validation**: Feature works as expected in production

### Quality Gates
1. **Code Quality**: All lints pass, coverage > 80%
2. **Security**: No critical vulnerabilities
3. **Performance**: Response time < 200ms P95
4. **Reliability**: Error rate < 0.1%
5. **Documentation**: Complete and accurate
6. **Testing**: All test suites pass

## 📚 Resources and References

### Documentation
- [API Standards](../patterns/api_standards.md)
- [Error Handling](../patterns/error_handling.md)
- [Coding Standards](../guides/coding_standards.md)
- [Testing Strategy](../guides/testing_strategy.md)

### External Resources
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [REST API Design](https://restfulapi.net/)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Test-Driven Development](https://martinfowler.com/bliki/TestDrivenDevelopment.html)

### Training Materials
- [Go Advanced Patterns Workshop](internal-link)
- [API Design Masterclass](internal-link)
- [Security Best Practices](internal-link)
- [Performance Optimization](internal-link)

---

*Документ создан: 2025*
*Владелец: Engineering Team*
*Регулярность обновлений: после каждого крупного релиза*
*Следующий ревью: После завершения текущего спринта*

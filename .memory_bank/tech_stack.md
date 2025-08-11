# Технологический паспорт - Family Finances Service

## 🏗️ Архитектурный обзор

### Общая архитектура
- **Тип**: Микросервис
- **Стиль**: RESTful API + Clean Architecture
- **Развертывание**: Docker + Docker Compose
- **Масштабирование**: Горизонтальное

### Архитектурные принципы
- **Clean Architecture**: Разделение на слои (Domain, Use Cases, Interface Adapters, Infrastructure)
- **Dependency Inversion**: Зависимости направлены внутрь к бизнес-логике
- **Single Responsibility**: Каждый компонент имеет одну ответственность
- **API First**: API проектируется до реализации

## 💻 Основной технологический стек

### Backend
- **Язык**: Go 1.21+
- **Framework**: Gin Web Framework
- **База данных**: PostgreSQL 15+
- **ORM**: GORM v2
- **Миграции**: golang-migrate
- **Валидация**: go-playground/validator

### Инфраструктура
- **Контейнеризация**: Docker & Docker Compose
- **Веб-сервер**: Nginx (в продакшене)
- **Процесс-менеджер**: systemd
- **CI/CD**: GitHub Actions (планируется)

### Документация API
- **Спецификация**: OpenAPI 3.0
- **Генерация**: swaggo/swag
- **UI**: Swagger UI
- **Тестирование**: Postman Collections

## 🗂️ Структура проекта

```
Family-Finances-Service/
├── cmd/                    # Точки входа приложения
│   └── server/            # HTTP сервер
├── internal/              # Приватный код приложения
│   ├── domain/           # Бизнес-логика и сущности
│   ├── usecases/         # Прикладная логика
│   ├── interfaces/       # Интерфейсы (HTTP handlers, repos)
│   └── infrastructure/   # Внешние зависимости (DB, APIs)
├── api/                   # API спецификации
├── generated/             # Автогенерированный код
├── .memory_bank/         # Документация проекта
└── Makefile              # Автоматизация задач
```

## 🔧 Инструменты разработки

### Сборка и зависимости
- **Менеджер пакетов**: Go Modules
- **Сборка**: Make + Dockerfile
- **Линтеры**: golangci-lint
- **Форматирование**: gofmt, goimports

### Тестирование
- **Unit тесты**: testing пакет Go
- **Mocking**: gomock
- **Integration тесты**: testcontainers-go
- **Coverage**: go test -cover

### Отладка и мониторинг
- **Логирование**: logrus/zap
- **Метрики**: Prometheus (планируется)
- **Трейсинг**: OpenTelemetry (планируется)
- **Health checks**: Встроенные эндпоинты

## 🗄️ База данных

### PostgreSQL конфигурация
- **Версия**: 15+
- **Пулл соединений**: pgxpool
- **Миграции**: golang-migrate/migrate
- **Backup**: pg_dump (автоматизировано)

### Дизайн БД
- **Подход**: Database First
- **Нормализация**: 3NF
- **Индексы**: Оптимизированы под запросы
- **Constraints**: Foreign keys, checks, unique

### Основные таблицы
```sql
families          # Семейные профили
family_members    # Члены семьи
transactions      # Финансовые транзакции
categories        # Категории транзакций
budgets          # Бюджеты и планы
financial_goals  # Финансовые цели
```

## 🌐 API Design

### REST принципы
- **Ресурсно-ориентированный**: /families/{id}/transactions
- **HTTP методы**: GET, POST, PUT, DELETE
- **Статус-коды**: Стандартные HTTP коды
- **Content-Type**: application/json

### Аутентификация и авторизация
- **Схема**: JWT Bearer tokens
- **Refresh tokens**: Да
- **Роли**: Family Admin, Family Member
- **Permissions**: RBAC модель

### Версионирование
- **Подход**: URI versioning (/api/v1/)
- **Backward compatibility**: Минимум 2 версии
- **Deprecation**: Уведомления в headers

## 🚀 DevOps и развертывание

### Локальная разработка
```bash
# Запуск всех сервисов
make dev

# Только база данных
make db-up

# Миграции
make migrate-up
```

### Среды
- **Development**: Docker Compose
- **Staging**: Планируется (Docker + CI/CD)
- **Production**: Планируется (Kubernetes)

### Мониторинг
- **Healthcheck**: /health эндпоинт
- **Metrics**: /metrics эндпоинт (Prometheus format)
- **Logging**: Structured JSON logs
- **Alerting**: Планируется

## 📦 Зависимости

### Основные Go модули
```go
github.com/gin-gonic/gin           # Web framework
github.com/lib/pq                  # PostgreSQL driver
gorm.io/gorm                       # ORM
github.com/golang-jwt/jwt/v5       # JWT tokens
github.com/go-playground/validator # Validation
github.com/joho/godotenv          # Environment variables
```

### Dev зависимости
```go
github.com/stretchr/testify       # Testing utilities
github.com/golang/mock            # Mocking
github.com/swaggo/swag           # Swagger generation
```

## 🔒 Безопасность

### Принципы
- **Defense in Depth**: Многоуровневая защита
- **Least Privilege**: Минимальные права доступа
- **Data Encryption**: Шифрование в покое и в движении
- **Input Validation**: Валидация всех входных данных

### Реализация
- **SQL Injection**: Параметризованные запросы
- **XSS**: Content Security Policy
- **CORS**: Настроенные CORS политики
- **Rate Limiting**: Ограничение запросов

## 📈 Производительность

### Целевые метрики
- **Response Time**: < 200ms (95th percentile)
- **Throughput**: > 1000 RPS
- **Availability**: 99.9%
- **Recovery Time**: < 1 минута

### Оптимизации
- **Database**: Индексы, connection pooling
- **Caching**: Redis (планируется)
- **Compression**: gzip для HTTP
- **Profiling**: pprof интеграция

## 🔄 Планы развития

### Ближайшие обновления (1-3 месяца)
- [ ] Интеграция с Redis для кеширования
- [ ] Prometheus метрики
- [ ] CI/CD pipeline
- [ ] Docker многоэтапная сборка

### Среднесрочные планы (3-6 месяцев)
- [ ] Kubernetes развертывание
- [ ] OpenTelemetry трейсинг
- [ ] GraphQL API
- [ ] Event-driven архитектура

### Долгосрочная перспектива (6-12 месяцев)
- [ ] Микросервисное разбиение
- [ ] Message queues (RabbitMQ/Kafka)
- [ ] Machine Learning интеграция
- [ ] Multi-region deployment

## 📚 Полезные ресурсы

### Документация
- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [GORM Guide](https://gorm.io/docs/)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)

### Лучшие практики
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

*Последнее обновление: 2024*  
*Технический лидер: Development Team*  
*Частота ревизий: ежемесячно*
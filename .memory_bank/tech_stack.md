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
- **Язык**: Go 1.24+
- **Framework**: Echo Web Framework v4.13.4+
- **База данных**: MongoDB 7.0+
- **Driver**: Official MongoDB Go Driver v1.17.4+
- **Валидация**: go-playground/validator v10.27.0
- **UUID**: google/uuid v1.6.0 для идентификаторов
- **Sessions**: gorilla/sessions v1.4.0
- **Password Hashing**: golang.org/x/crypto/bcrypt
- **Testing**: testify v1.10.0 + testcontainers-go v0.38.0

### Frontend (Web Interface)
- **Framework**: HTMX v1.9+ для dynamic updates
- **CSS**: PicoCSS v1.5+ minimalist framework
- **Templates**: Go Templates с layout system
- **Static Assets**: Echo static middleware
- **PWA**: Service Worker ready

### Инфраструктура
- **Контейнеризация**: Docker & Docker Compose
- **Multi-platform**: linux/amd64, linux/arm64
- **CI/CD**: GitHub Actions (ci, docker, security, release)
- **Registry**: GitHub Container Registry
- **Security Scanning**: CodeQL, Semgrep, TruffleHog, OSV Scanner

### Документация API
- **Спецификация**: OpenAPI 3.0 (планируется)
- **Генерация**: go generate (в development)
- **UI**: Swagger UI (планируется)
- **Тестирование**: HTTP тесты

## 🗂️ Структура проекта

```
Family-Finances-Service/
├── cmd/                    # Точки входа приложения
│   └── server/            # HTTP сервер
├── internal/              # Приватный код приложения
│   ├── domain/           # Domain entities и бизнес-логика
│   ├── application/      # Application layer с интерфейсами
│   ├── infrastructure/   # Реализация репозиториев (MongoDB)
│   ├── config.go         # Конфигурация приложения
│   └── run.go           # Bootstrap приложения
├── generated/             # Автогенерированный код (OpenAPI)
├── .memory_bank/         # Документация проекта
├── docker-compose.yml    # Docker окружение
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

### Observability
- **Логирование**: slog (structured logging)
- **Метрики**: Prometheus v1.23.0 с custom metrics
- **Трейсинг**: OpenTelemetry v1.37.0 с Jaeger
- **Health checks**: Liveness/Readiness probes
- **Monitoring**: Grafana dashboards для visualization

## 🗄️ База данных

### MongoDB конфигурация
- **Версия**: 7.0+
- **Driver**: Official MongoDB Go Driver v1.13+
- **Connection Pool**: Встроенное управление соединениями
- **Миграции**: Программные миграции или скрипты

### Дизайн БД
- **Подход**: Document-oriented
- **Schema**: Flexible schema с validation
- **Индексы**: Составные индексы для оптимизации запросов
- **Aggregation Pipeline**: Для сложной аналитики

### Основные коллекции
```javascript
families       // Семейные профили
users          // Пользователи (члены семей)
transactions   // Финансовые транзакции
categories     // Категории доходов/расходов
budgets        // Бюджеты и планы
reports        // Сгенерированные отчеты
```

### Особенности MongoDB
- **BSON типы**: ObjectId, UUID для идентификаторов
- **Embedded documents**: Для связанных данных
- **Array fields**: Для списков и коллекций
- **Multi-tenancy**: Фильтрация по family_id

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
make docker-up

# Запуск приложения локально
make run-local

# Форматирование и линтинг
make fmt && make lint

# Тестирование
make test
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
github.com/labstack/echo/v4        # Web framework
go.mongodb.org/mongo-driver        # MongoDB driver
github.com/google/uuid             # UUID generation
github.com/golang-jwt/jwt          # JWT tokens (indirect)
```

### Dev зависимости
```go
github.com/stretchr/testify       # Testing utilities (планируется)
github.com/golang/mock            # Mocking (планируется)
```

## 🔒 Безопасность

### Принципы
- **Defense in Depth**: Многоуровневая защита
- **Least Privilege**: Минимальные права доступа
- **Data Encryption**: Шифрование в покое и в движении
- **Input Validation**: Валидация всех входных данных

### Реализация
- **NoSQL Injection**: Валидация и санитизация входных данных
- **XSS**: Content Security Policy
- **CORS**: Настроенные CORS политики Echo
- **Rate Limiting**: Middleware для ограничения запросов

## 📈 Производительность

### Целевые метрики
- **Response Time**: < 200ms (95th percentile)
- **Throughput**: > 1000 RPS
- **Availability**: 99.9%
- **Recovery Time**: < 1 минута

### Оптимизации
- **MongoDB**: Индексы, connection pooling, aggregation pipeline
- **Caching**: Redis (в docker-compose.yml)
- **Compression**: gzip middleware Echo
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
- [Echo Framework](https://echo.labstack.com/guide/)
- [MongoDB Go Driver](https://www.mongodb.com/docs/drivers/go/current/)
- [MongoDB Docs](https://www.mongodb.com/docs/)

### Лучшие практики
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)

---

*Последнее обновление: 2025*
*Технический лидер: Development Team*
*Частота ревизий: ежемесячно*

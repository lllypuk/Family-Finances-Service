# Текущая задача

## **[INFRA-004]** Мониторинг и observability

### 📋 Описание
Реализовать комплексную систему мониторинга, логирования и observability для Family Finances Service с использованием современных инструментов и практик для обеспечения надежности, производительности и простоты диагностики в production среде.

### 🎯 Цели
- Обеспечить полную видимость состояния системы и производительности
- Настроить проактивное обнаружение проблем через алерты и мониторинг
- Реализовать structured logging для эффективной диагностики
- Добавить трейсинг запросов для анализа производительности
- Интегрировать метрики бизнес-логики и инфраструктуры
- Создать дашборды для мониторинга ключевых показателей

### 📊 Метрики задачи
- **Приоритет**: Высокий
- **Оценка времени**: 3-4 дня
- **Сложность**: Высокая
- **Команда**: DevOps/Backend/SRE

### 🔧 Техническая спецификация

#### Компоненты observability стека:

1. **Metrics (Метрики)** - Prometheus + Grafana
   - **Application metrics**: HTTP requests, response times, error rates
   - **Business metrics**: Transactions count, budget usage, user activity
   - **System metrics**: Memory, CPU, DB connections, goroutines
   - **Custom metrics**: Domain-specific KPIs для семейного бюджета

2. **Logging (Логирование)** - Structured JSON logging
   - **Framework**: `log/slog` (Go 1.21+) для structured logging
   - **Log levels**: DEBUG, INFO, WARN, ERROR с правильной конфигурацией
   - **Correlation IDs**: Трейсинг запросов через request ID
   - **Security logging**: Audit trail для аутентификации и авторизации

3. **Tracing (Трейсинг)** - OpenTelemetry
   - **Distributed tracing**: Полный путь запроса через компоненты
   - **Database tracing**: Время выполнения MongoDB запросов
   - **External calls**: Трейсинг вызовов внешних API
   - **Custom spans**: Бизнес-логика и критичные операции

4. **Health Checks** - Проверки состояния сервиса
   - **Readiness probe**: Готовность принимать трафик
   - **Liveness probe**: Состояние сервиса для Kubernetes
   - **Dependency checks**: MongoDB, Redis, внешние сервисы
   - **Custom health indicators**: Бизнес-метрики здоровья

#### Архитектура мониторинга:

5. **Prometheus Configuration**
   - **Scrape configs**: Endpoints для сбора метрик
   - **Recording rules**: Предвычисленные метрики
   - **Alerting rules**: Правила для критичных событий
   - **Service discovery**: Автоматическое обнаружение сервисов

6. **Grafana Dashboards**
   - **Application Dashboard**: HTTP metrics, errors, latency
   - **Business Dashboard**: Пользователи, транзакции, семьи
   - **Infrastructure Dashboard**: System resources, DB performance
   - **SLI/SLO Dashboard**: Service Level Indicators/Objectives

7. **Alerting (Алертинг)**
   - **AlertManager**: Центральное управление алертами
   - **Notification channels**: Slack, email, PagerDuty интеграции
   - **Alert routing**: Эскалация по критичности и времени
   - **Silence management**: Управление отключением алертов

#### Конкретные метрики для отслеживания:

8. **HTTP/API Metrics**
   - `http_requests_total{method, endpoint, status}` - Количество запросов
   - `http_request_duration_seconds{method, endpoint}` - Время ответа
   - `http_requests_errors_total{method, endpoint, type}` - Ошибки по типам

9. **Business Metrics**
   - `families_total` - Общее количество семей
   - `users_total{role}` - Пользователи по ролям
   - `transactions_total{type, family_id}` - Транзакции по типам
   - `budgets_active{family_id}` - Активные бюджеты
   - `revenue_total{currency}` - Финансовые показатели

10. **System Metrics**
    - `go_memstats_*` - Память Go приложения
    - `go_goroutines` - Количество горутин
    - `mongodb_connections_*` - MongoDB метрики
    - `redis_connections_*` - Redis метрики (если используется)

#### Технологический стек:

11. **Instrumentation Libraries**
    - `github.com/prometheus/client_golang` - Prometheus metrics
    - `go.opentelemetry.io/otel` - OpenTelemetry tracing
    - `log/slog` - Structured logging
    - `github.com/labstack/echo/v4/middleware` - HTTP middleware

12. **Infrastructure Components**
    - **Prometheus** - Сбор и хранение метрик
    - **Grafana** - Визуализация и дашборды
    - **AlertManager** - Управление алертами
    - **Jaeger/Tempo** - Distributed tracing backend

#### Docker Compose Integration:

13. **Observability Services**
    - Prometheus (порт 9090)
    - Grafana (порт 3000)
    - AlertManager (порт 9093)
    - Jaeger (порт 16686)
    - Node Exporter (порт 9100)

### ✅ Критерии готовности (Definition of Done)

1. ✅ Добавлены Prometheus метрики в Go приложение
2. ✅ Реализован structured logging с `log/slog`
3. ✅ Настроен OpenTelemetry трейсинг
4. ✅ Созданы health check endpoints
5. ✅ Настроена конфигурация Prometheus
6. ✅ Созданы Grafana дашборды для всех компонентов
7. ✅ Настроены алерты для критичных метрик
8. ✅ Добавлены observability сервисы в docker-compose.yml
9. ✅ Протестирована работа всего мониторинг стека
10. ✅ Обновлена документация с описанием мониторинга
11. ✅ Созданы SLI/SLO определения для сервиса

### 🛡️ Security и Privacy требования

- **Sensitive Data**: Исключение PII из логов и метрик
- **Access Control**: Ограниченный доступ к мониторинг системам
- **Data Retention**: Политики хранения логов и метрик
- **Encryption**: TLS для передачи метрик и логов
- **Audit Logging**: Трейсинг доступа к финансовым данным

### 📊 Метрики успеха

- **MTTR (Mean Time To Recovery)**: < 15 минут
- **Alert Precision**: > 95% relevant alerts (low false positives)
- **Coverage**: 100% критичных компонентов под мониторингом
- **Dashboard Response Time**: < 2 секунды загрузка дашбордов
- **Log Search Performance**: < 5 секунд поиск в логах за день

### 🚨 Потенциальные риски

- **Performance Impact**: Overhead от инструментации (< 5%)
- **Storage Costs**: Объем метрик и логов в production
- **Complexity**: Сложность настройки и поддержки стека
- **Alert Fatigue**: Слишком много ложных срабатываний
- **Privacy Leaks**: Случайное логирование чувствительных данных

### 📝 Связанные задачи

- **INFRA-001** ✅ (завершена) - Docker оптимизация и настройка
- **INFRA-002** ✅ (завершена) - Линтинг и code quality
- **INFRA-003** ✅ (завершена) - GitHub Actions CI/CD
- **AUTH-001** (планируется) - JWT аутентификация (потребует audit logging)
- **PERF-001** (планируется) - Performance optimization

### 🔄 Workflow мониторинга

1. **Development**: Локальное тестирование с docker-compose observability стеком
2. **CI/CD**: Проверка metrics endpoints в GitHub Actions
3. **Staging**: Полноценный мониторинг для тестирования алертов
4. **Production**: 24/7 мониторинг с on-call алертингом

### 📈 SLI/SLO определения

- **Availability**: 99.9% uptime (< 8.77 часов downtime в год)
- **Latency**: 95% запросов < 500ms, 99% < 1s
- **Error Rate**: < 0.1% HTTP 5xx errors
- **Throughput**: Поддержка до 1000 RPS на инстанс

---

## 🎉 ЗАДАЧА ВЫПОЛНЕНА ✅

**Дата завершения**: 13.08.2025
**Статус**: COMPLETED

### 📋 Что реализовано:

#### 🔧 Основная инфраструктура
- ✅ **Observability package** (`internal/observability/`) с полным набором функций
- ✅ **Prometheus metrics** - HTTP метрики, бизнес-метрики, системные метрики
- ✅ **Structured logging** - JSON логирование с `log/slog` и корреляционными ID
- ✅ **OpenTelemetry tracing** - распределенный трейсинг с Jaeger интеграцией
- ✅ **Health checks** - `/health` и `/ready` endpoints с MongoDB проверками

#### 📊 Мониторинг стек  
- ✅ **Prometheus** (порт 9090) - сбор метрик с alerting rules
- ✅ **Grafana** (порт 3000) - 3 дашборда: application, business, SLI/SLO
- ✅ **Jaeger** (порт 16686) - UI для distributed tracing
- ✅ **AlertManager** (порт 9093) - управление алертами
- ✅ **Exporters** - Node, MongoDB, Redis exporters

#### 🐳 Docker интеграция
- ✅ **Docker Compose profiles** - observability, production профили
- ✅ **Makefile команды** - `observability-up`, `dev-up`, `full-up`
- ✅ **Environment variables** - гибкая конфигурация портов и настроек

#### 🧪 Тестирование
- ✅ **Functional testing** - все endpoints работают корректно
- ✅ **Metrics collection** - HTTP метрики собираются и отображаются
- ✅ **Logging verification** - structured logs с корреляционными ID
- ✅ **Health monitoring** - MongoDB health checks работают

### 🌟 Ключевые особенности реализации:

1. **Production-ready архитектура** с полным observability стеком
2. **Автоматическое instrumentation** через middleware для всех HTTP запросов  
3. **Корреляционные ID** для трейсинга запросов через всю систему
4. **Бизнес-метрики** специфичные для семейных финансов (families, users, transactions)
5. **Гибкая конфигурация** через environment variables
6. **Security-первый подход** с исключением PII из логов

### 📈 Доступные сервисы:

| Сервис | URL | Описание |
|--------|-----|----------|
| Application | http://localhost:8082 | Основное приложение |
| Health Check | http://localhost:8082/health | Проверка состояния |
| Metrics | http://localhost:8082/metrics | Prometheus метрики |
| Prometheus | http://localhost:9090 | Сбор метрик |
| Grafana | http://localhost:3000 | Дашборды (admin/admin) |
| Jaeger | http://localhost:16686 | Distributed tracing |
| AlertManager | http://localhost:9093 | Управление алертами |

### 🚀 Следующие шаги:
Задача **INFRA-004** полностью завершена. Система готова для production deployment с полным мониторингом, логированием и алертингом.

# Memory Bank - Навигационная карта проекта Family Finances Service

Добро пожаловать в центральную систему документации проекта! Этот файл служит главной точкой входа и навигационной картой для всей технической документации.

## 🎯 Текущий статус проекта: READY FOR PRODUCTION

**Проект полностью готов к production deployment** с завершенными веб-интерфейсом, API, системами безопасности и comprehensive test coverage (59.5%+).

## 🚀 Быстрый старт

### Для новых разработчиков
1. **[Описание продукта](product_brief.md)** - Узнайте ЗАЧЕМ мы создаем этот сервис
2. **[Технологический стек](tech_stack.md)** - Изучите архитектуру и технологии
3. **[Стандарты кодирования](guides/coding_standards.md)** - Правила написания кода
4. **[Стратегия тестирования](guides/testing_strategy.md)** - Как мы тестируем код

### Для работы с задачами
- **[Текущие задачи](current_task.md)** - Статус проекта и следующие шаги
- **[Новая фича](workflows/new_feature.md)** - Пошаговый процесс добавления функций
- **[Исправление багов](workflows/bug_fix.md)** - Алгоритм работы с ошибками

## 🏆 Завершенные компоненты

### ✅ Core Application
- **Clean Architecture** реализована
- **Domain models** с полным business logic
- **Repository pattern** с MongoDB интеграцией
- **HTTP server** с Echo framework
- **Configuration management** через переменные окружения

### ✅ Web Interface (HTMX + PicoCSS)
- **Authentication & Authorization** с role-based access
- **Dashboard** с family financial overview
- **CRUD interfaces** для всех entities
- **Forms validation** и error handling
- **Responsive design** для mobile/desktop
- **HTMX dynamic updates** без page reload

### ✅ Security
- **Session management** с CSRF protection
- **Password hashing** (bcrypt)
- **Input validation** и sanitization
- **Authorization middleware** с role checks
- **Security headers** и best practices

### ✅ Testing & Quality (59.5% coverage)
- **450+ comprehensive tests** across all components
- **Unit tests** с mocking и table-driven patterns
- **Integration tests** с testcontainers
- **Performance tests** и load testing
- **E2E tests** для user workflows
- **Security tests** для vulnerability detection

### ✅ Observability
- **Structured logging** (slog)
- **Prometheus metrics** с business и technical metrics
- **Health checks** (liveness/readiness)
- **Distributed tracing** (OpenTelemetry)
- **Grafana dashboards** для monitoring

### ✅ CI/CD & DevOps
- **GitHub Actions** workflows (ci.yml, docker.yml, security.yml, release.yml)
- **Multi-platform Docker builds** (linux/amd64, linux/arm64)
- **Security scanning** (CodeQL, Semgrep, TruffleHog)
- **Dependency management** (Dependabot)
- **Automated releases** с semantic versioning

## 📁 Структура документации

### 📊 Основные документы
- **[product_brief.md](product_brief.md)** - Бизнес-контекст и цели проекта
- **[tech_stack.md](tech_stack.md)** - Технологический паспорт
- **[current_tasks.md](current_tasks.md)** - Активные задачи и их статусы

### 🏗️ patterns/ - Архитектурные решения
- **[api_standards.md](patterns/api_standards.md)** - Стандарты проектирования API
- **[error_handling.md](patterns/error_handling.md)** - Единая система обработки ошибок

### 📚 guides/ - Практические руководства
- **[coding_standards.md](guides/coding_standards.md)** - Стиль кода и соглашения
- **[testing_strategy.md](guides/testing_strategy.md)** - Подходы к тестированию

### 📋 specs/ - Технические задания
- **[feature_xyz.md](specs/feature_xyz.md)** - Шаблон для новых функций
- *Здесь будут появляться спецификации новых фич*

### ⚙️ workflows/ - Рабочие процессы
- **[new_feature.md](workflows/new_feature.md)** - Жизненный цикл новой функции
- **[bug_fix.md](workflows/bug_fix.md)** - Процесс исправления ошибок

## 🔄 Как поддерживать актуальность

### При добавлении новой функции:
1. Создайте спецификацию в `specs/`
2. Обновите `current_tasks.md`
3. Следуйте процессу из `workflows/new_feature.md`

### При изменении архитектуры:
1. Обновите соответствующий файл в `patterns/`
2. Проверьте актуальность `tech_stack.md`
3. Уведомите команду об изменениях

### При обнаружении бага:
1. Следуйте `workflows/bug_fix.md`
2. Обновите `current_tasks.md`
3. Рассмотрите необходимость обновления `guides/testing_strategy.md`

## 🤝 Принципы работы с документацией

- **Живая документация** - Обновляем по мере развития проекта
- **Краткость и ясность** - Пишем то, что действительно нужно
- **Практичность** - Каждый документ должен решать конкретную задачу
- **Актуальность** - Устаревшая информация хуже отсутствующей

## 📞 Контакты и помощь

Если вы не можете найти нужную информацию или хотите предложить улучшения:
- Создайте issue в репозитории
- Обратитесь к тех-лиду команды
- Предложите правки через PR

---

*Последнее обновление: $(date)*
*Поддерживается командой разработки Family Finances Service*

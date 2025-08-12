# Текущая задача

## **[INFRA-003]** Добавить GitHub Actions CI/CD

### 📋 Описание
Настроить автоматизированный pipeline CI/CD с использованием GitHub Actions для обеспечения качества кода, автоматического тестирования и безопасного деплоя Family Finances Service.

### 🎯 Цели
- Автоматизировать процесс интеграции и развертывания кода
- Обеспечить запуск тестов и линтинга при каждом push/PR
- Настроить автоматическую сборку и публикацию Docker образов
- Внедрить безопасность через проверку уязвимостей
- Улучшить скорость и надежность релизов

### 📊 Метрики задачи
- **Приоритет**: Высокий
- **Оценка времени**: 2-3 дня
- **Сложность**: Средняя-высокая
- **Команда**: DevOps/Backend

### 🔧 Техническая спецификация

#### GitHub Actions Workflows для создания:

1. **`.github/workflows/ci.yml`** - Основной CI pipeline
   - **Triggers**: push на main/develop, pull requests
   - **Матрица тестирования**: Go 1.24
   - **Шаги**:
     - Checkout кода
     - Setup Go environment
     - Кэширование зависимостей
     - Запуск `make deps`
     - Запуск `make lint` (golangci-lint)
     - Запуск `make fmt` с проверкой форматирования
     - Запуск `make test` с покрытием
     - Upload coverage отчетов в Codecov
     - Проверка security с `govulncheck`

2. **`.github/workflows/docker.yml`** - Сборка и публикация Docker образов
   - **Triggers**: push тегов, релизы
   - **Шаги**:
     - Multi-platform build (linux/amd64, linux/arm64)
     - Сборка и пуш в GitHub Container Registry
     - Semantic versioning для тегов
     - Vulnerability scanning с Trivy
     - Проверка размера образа

3. **`.github/workflows/release.yml`** - Автоматизация релизов
   - **Triggers**: push тегов версий (v*.*.*)
   - **Шаги**:
     - Генерация release notes
     - Создание GitHub Release
     - Attach бинарных файлов
     - Уведомления в Slack/Discord

#### Дополнительные конфигурации:

4. **`.github/dependabot.yml`** - Автоматическое обновление зависимостей
   - Go modules обновления
   - GitHub Actions обновления
   - Docker base images обновления

5. **`.github/CODEOWNERS`** - Настройка code review
   - Обязательные reviewers для критичных файлов
   - Разделение ответственности по компонентам

6. **`.github/workflows/security.yml`** - Security сканирование
   - CodeQL analysis для Go
   - Dependency vulnerability scanning
   - Secret scanning validation

#### Интеграции и сервисы:

- **Codecov** - отчеты о покрытии кода тестами
- **GitHub Container Registry** - хранение Docker образов
- **Trivy** - сканирование уязвимостей в образах
- **govulncheck** - проверка Go уязвимостей

### ✅ Критерии готовности (Definition of Done)

1. ✅ Создан и настроен CI workflow (.github/workflows/ci.yml)
2. ✅ Создан Docker build workflow (.github/workflows/docker.yml)
3. ✅ Настроен dependabot для автоматических обновлений (.github/dependabot.yml)
4. ✅ Добавлена интеграция с Codecov для coverage отчетов
5. ✅ Настроено security сканирование (CodeQL, govulncheck, Semgrep, OSV Scanner)
6. ✅ Протестированы все workflows (YAML валидация, синтаксис, тестирование)
7. ✅ Создан CODEOWNERS файл для автоматических code review
8. ✅ Обновлена документация с описанием CI/CD процесса в CLAUDE.md
9. ✅ Создан release workflow для автоматизации релизов

### 🎉 Задача завершена успешно!

**Реализованные компоненты:**
- **CI Pipeline**: Полный CI процесс с тестированием, линтингом, coverage отчетами
- **Security Pipeline**: Комплексное security сканирование (CodeQL, govulncheck, Semgrep, TruffleHog, OSV)
- **Docker Pipeline**: Multi-platform сборка и публикация образов в GHCR
- **Release Pipeline**: Автоматизированные релизы с бинарными файлами и Docker образами
- **Dependabot**: Автоматическое обновление Go modules, GitHub Actions, Docker образов  
- **CODEOWNERS**: Автоматическое назначение reviewers для code review

### 🛡️ Security требования

- Использование GitHub secrets для чувствительных данных
- Minimal permissions для GitHub Actions tokens
- Signed commits verification (опционально)
- Vulnerability scanning всех dependencies
- Security-первый подход к Docker образам

### 📊 Метрики успеха

- **Build time**: < 5 минут для полного CI pipeline
- **Test coverage**: Мониторинг и отчетность
- **Security score**: 0 critical/high уязвимостей
- **Reliability**: 99%+ успешность CI builds

### 🚨 Потенциальные риски

- **Сложность настройки**: Множество moving parts в CI/CD
- **Performance**: Длительное время выполнения тестов с контейнерами
- **Security**: Неправильная настройка secrets и permissions
- **Dependencies**: Проблемы с внешними сервисами (Codecov, registries)

### 📝 Связанные задачи

- **INFRA-001** ✅ (завершена) - Docker оптимизация и настройка
- **INFRA-002** ✅ (завершена) - Линтинг и code quality
- **INFRA-004** (планируется) - Мониторинг и observability
- **AUTH-001** (планируется) - JWT аутентификация

### 🔄 Workflow последовательность

1. **Developer** создает PR
2. **GitHub Actions** автоматически запускает CI
3. **CI pipeline** запускает: lint → test → security scan
4. **Code review** после успешного CI
5. **Merge to main** запускает build и deploy workflows
6. **Release** создается автоматически при push тегов

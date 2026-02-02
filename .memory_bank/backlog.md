# Бэклог задач для проекта "Family Finances Service"

## Критичные (блокируют CI)

### SEC-1: Path Traversal в backup_service.go

- **Файл**: `internal/services/backup_service.go` (строки 164, 187, 208, 218)
- **Файл**: `internal/web/handlers/backup.go` (строка 131)
- **Проблема**: CodeQL детектирует uncontrolled data in path expression. Функция `validateFilename` недостаточна для
  CodeQL.
- **Решение**: После regex-валидации применять `filepath.Base()` + проверку `strings.HasPrefix(cleanPath, s.backupDir)`
  для всех операций с пользовательским filename.
- **Приоритет**: CRITICAL — CodeQL check падает

### SEC-2: SQL Injection в VACUUM INTO

- **Файл**: `internal/services/backup_service.go:84`
- **Проблема**: `fmt.Sprintf("VACUUM INTO '%s'", backupPath)` — строковая интерполяция в SQL-запросе.
- **Решение**: VACUUM INTO не поддерживает параметризацию в SQLite. Убедиться, что `backupPath` формируется только из
  контролируемых данных (timestamp + backupDir), без внешнего ввода. Добавить дополнительную валидацию символов в пути.
- **Приоритет**: CRITICAL — Semgrep alert

### SEC-3: Open URL Redirect в auth.go

- **Файл**: `internal/web/handlers/auth.go:113`
- **Проблема**: Параметр redirect URL не валидируется — возможно перенаправление на внешний сайт.
- **Решение**: Проверять, что URL начинается с `/` и не содержит `//` (предотвращение protocol-relative URL).
- **Приоритет**: CRITICAL — CodeQL alert (severity: medium)

## Важные (code quality)

### CQ-1: Дублирование requireAdmin

- **Файлы**: `internal/web/handlers/admin.go:36-53`, `internal/web/handlers/backup.go:29-45`
- **Проблема**: Почти идентичные методы `requireAdmin` в двух хендлерах.
- **Решение**: Вынести в общий middleware или базовый хендлер.

### CQ-2: Игнорирование ошибок в invite_service.go

- **Файл**: `internal/services/invite_service.go` (строки 140, 202, 270)
- **Проблема**: `_ = updateErr` — ошибки обновления молча игнорируются.
- **Решение**: Использовать structured logging через пакет observability.

### CQ-3: Неиспользуемые параметры в BaseHandler

- **Файл**: `internal/web/handlers/base.go` (строки 166, 173)
- **Проблема**: `redirectWithError` и `redirectWithSuccess` принимают параметр message, но игнорируют его (`_`).
- **Решение**: Реализовать flash messages или убрать неиспользуемый параметр.

### CQ-4: Context не прокидывается в репозитории

- **Файл**: `internal/services/invite_service.go` (строки 128, 257, 279)
- **Проблема**: `context.Context` принимается, но не передаётся в вызовы репозитория.
- **Решение**: Прокинуть context до слоя репозитория для корректной обработки timeout/cancellation.

### CQ-5: N+1 запросы при обновлении просроченных приглашений

- **Файл**: `internal/services/invite_service.go:264-272`
- **Проблема**: Обновление приглашений по одному в цикле.
- **Решение**: Реализовать bulk update или вынести в scheduled job.

## Тестирование

### TEST-1: Тесты для web handlers

- **Файлы**: `internal/web/handlers/admin.go`, `internal/web/handlers/backup.go`
- **Проблема**: Нет тестового покрытия для новых web-хендлеров.
- **Решение**: Написать table-driven тесты с моками репозиториев.

### TEST-2: Интеграционные тесты для invite flow

- **Проблема**: Нет end-to-end тестов для полного цикла приглашений.
- **Решение**: Добавить интеграционные тесты: создание invite -> переход по токену -> регистрация.

### TEST-3: Security тесты

- **Проблема**: Нет тестов на path traversal, невалидные токены, SQL injection attempts.
- **Решение**: Добавить негативные security test cases для backup и invite сервисов.

## Улучшения (nice to have)

### IMP-1: Асинхронное создание бэкапов

- **Проблема**: VACUUM INTO выполняется синхронно в HTTP-запросе, блокируя ответ на больших БД.
- **Решение**: Вынести в фоновую задачу с индикатором прогресса.

### IMP-3: Flash messages

- **Проблема**: Нет механизма показа пользователю сообщений после redirect (успех/ошибка).
- **Решение**: Реализовать flash messages через session storage.

# SEC-2: SQL Injection в VACUUM INTO

## Статус: DONE ✅

## Приоритет: CRITICAL (Semgrep alert)
## Дата завершения: 2026-02-03

## Проблема

В `internal/services/backup_service.go:84`:

```go
_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
```

Semgrep детектирует строковую интерполяцию в SQL-запросе (`string-formatted-query`). SQLite не поддерживает
параметризацию для `VACUUM INTO`, поэтому стандартный `?` placeholder не работает.

## Анализ

В текущем коде `backupPath` формируется в `CreateBackup()`:

```go
timestamp := now.Format("20060102_150405")
milliseconds := now.UnixNano() / nanosToMillis % millisInSecond
filename := fmt.Sprintf("backup_%s%03d.db", timestamp, milliseconds)
backupPath := filepath.Join(s.backupDir, filename)
```

Данные полностью контролируемые (timestamp), внешний пользовательский ввод не участвует. Реальный риск SQL injection
минимален.

## Решение

### Вариант: Дополнительная валидация + nolint

Поскольку VACUUM INTO не поддерживает параметры, нужно:

1. Добавить явную валидацию `backupPath` перед использованием
2. Убедить Semgrep, что путь безопасен (комментарий или аннотация)

```go
// CreateBackup creates a new backup using SQLite VACUUM INTO
func (s *backupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
if err := s.ensureBackupDir(); err != nil {
return nil, fmt.Errorf("failed to create backup directory: %w", err)
}

now := time.Now()
timestamp := now.Format("20060102_150405")
const (
nanosToMillis = 1e6
millisInSecond = 1000
)
milliseconds := now.UnixNano() / nanosToMillis % millisInSecond
filename := fmt.Sprintf("backup_%s%03d.db", timestamp, milliseconds)

// Validate the generated filename matches expected pattern
if err := validateFilename(filename); err != nil {
return nil, fmt.Errorf("generated invalid backup filename: %w", err)
}

backupPath := filepath.Join(s.backupDir, filepath.Base(filename))

// Validate path contains only safe characters (alphanumeric, underscores, dots, slashes)
if !isValidBackupPath(backupPath) {
return nil, fmt.Errorf("unsafe backup path detected: %s", backupPath)
}

// VACUUM INTO does not support parameterized queries in SQLite.
// The backupPath is constructed entirely from controlled data (timestamp + backupDir)
// and validated above. No user input reaches this query.
//nolint:gosec // backupPath is generated from timestamp, not user input
query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
_, err := s.db.ExecContext(ctx, query)
if err != nil {
return nil, fmt.Errorf("failed to create backup: %w", err)
}

// ... rest of method unchanged
}
```

### Добавить функцию валидации пути

```go
// isValidBackupPath checks that a backup path contains only safe characters.
// This is defense-in-depth for the VACUUM INTO query which cannot use parameterized queries.
var validPathRegex = regexp.MustCompile(`^[a-zA-Z0-9_.\-/\\]+$`)

func isValidBackupPath(path string) bool {
return validPathRegex.MatchString(path)
}
```

### Добавить nosemgrep комментарий (если nolint недостаточно)

Если Semgrep продолжает жаловаться, можно добавить inline suppression:

```go
query := fmt.Sprintf("VACUUM INTO '%s'", backupPath) // nosemgrep: go.lang.security.audit.database.string-formatted-query
```

## Файлы для изменения

1. `internal/services/backup_service.go` — добавить `isValidBackupPath()`, обновить `CreateBackup()`

## Тестирование

- ✅ `make test` — все тесты проходят
- ✅ `make lint` — 0 issues
- ✅ Добавлен тест `TestIsValidBackupPath` с 8 сценариями
- ⏳ Проверить, что Semgrep alert пропадает (проверится в следующем CI run)

## Реализованные изменения

### 1. Добавлена функция `isValidBackupPath()`
- Валидирует, что путь содержит только безопасные символы
- Использует regex `^[a-zA-Z0-9_.\-/\\:]+$`
- Защита от SQL injection через специальные символы

### 2. Обновлен метод `CreateBackup()`
- Добавлена валидация сгенерированного filename через `validateFilename()`
- Применяется `filepath.Base()` для удаления компонентов директорий
- Валидация полного пути через `isValidBackupPath()`
- Добавлен детальный комментарий с `//nolint:gosec`
- SQL query выделен в отдельную переменную для ясности

### 3. Тестовое покрытие
- Новый тест `TestIsValidBackupPath` с 8 сценариями
- Проверка валидных путей (Unix, Windows, relative)
- Проверка защиты от SQL injection (спецсимволы, кавычки, null byte, newline, пробелы)
- Все существующие тесты проходят

### 4. Defense in depth
- Filename генерируется из timestamp (контролируемые данные)
- Двойная валидация: filename + full path
- Использование `filepath.Base()` для безопасности
- Подробный комментарий объясняет, почему параметризация невозможна

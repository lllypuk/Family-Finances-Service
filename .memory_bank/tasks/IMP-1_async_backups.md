# IMP-1: Асинхронное создание бэкапов

## Статус: TODO

## Приоритет: NICE TO HAVE

## Проблема

`VACUUM INTO` в `CreateBackup()` выполняется синхронно в HTTP-запросе. На больших базах данных это может занять
значительное время и заблокировать HTTP-ответ.

Текущий код (`internal/services/backup_service.go:84`):

```go
_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
```

## Решение

### Подход: Фоновая горутина с каналом статуса

#### 1. Добавить статус бэкапа

```go
type BackupStatus string

const (
BackupStatusPending   BackupStatus = "pending"
BackupStatusRunning   BackupStatus = "running"
BackupStatusCompleted BackupStatus = "completed"
BackupStatusFailed    BackupStatus = "failed"
)

type BackupInfo struct {
Filename  string       `json:"filename"`
Size      int64        `json:"size"`
CreatedAt time.Time    `json:"created_at"`
Status    BackupStatus `json:"status"`
Error     string       `json:"error,omitempty"`
}
```

#### 2. Обновить BackupService

```go
type backupService struct {
db         *sql.DB
dbPath     string
backupDir  string
logger     *slog.Logger
mu         sync.Mutex
inProgress bool
}

func (s *backupService) CreateBackupAsync(ctx context.Context) (*BackupInfo, error) {
s.mu.Lock()
if s.inProgress {
s.mu.Unlock()
return nil, errors.New("backup already in progress")
}
s.inProgress = true
s.mu.Unlock()

// Generate filename
info := &BackupInfo{
Filename: filename,
Status:   BackupStatusPending,
}

go func () {
defer func () {
s.mu.Lock()
s.inProgress = false
s.mu.Unlock()
}()

bgCtx := context.Background()
_, err := s.CreateBackup(bgCtx)
if err != nil {
s.logger.Error("async backup failed", slog.String("error", err.Error()))
}
}()

return info, nil
}
```

#### 3. HTMX polling для статуса

В шаблоне backup page добавить polling:

```html

<div hx-get="/admin/backup/status" hx-trigger="every 2s" hx-swap="innerHTML">
    Creating backup...
</div>
```

#### 4. Добавить endpoint для статуса

В backup handler:

```go
func (h *BackupHandler) BackupStatus(c echo.Context) error {
// Return current backup status
}
```

## Файлы для изменения

1. `internal/services/backup_service.go` — добавить async метод
2. `internal/web/handlers/backup.go` — добавить status endpoint
3. `internal/web/templates/admin/backup.html` — добавить HTMX polling
4. Роутинг — добавить `/admin/backup/status`

## Примечание

Это улучшение имеет смысл только при реально больших базах данных. Для текущего размера (семейный бюджет) синхронный
бэкап, скорее всего, достаточно быстр.

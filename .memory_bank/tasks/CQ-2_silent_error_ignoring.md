# CQ-2: Игнорирование ошибок в invite_service.go

## Статус: DONE ✅

## Приоритет: IMPORTANT

## Проблема

В `internal/services/invite_service.go` ошибки обновления invite молча игнорируются:

- **Строка 140:** `_ = updateErr` в `GetInviteByToken()` — при обновлении статуса expired
- **Строка 203:** `_ = updateErr` в `AcceptInvite()` — при маркировке invite как accepted
- **Строка 270:** `_ = updateErr` в `ListFamilyInvites()` — при обновлении expired invites

## Решение

### 1. Добавить logger в inviteService

Обновить структуру и конструктор:

```go
type inviteService struct {
inviteRepo user.InviteRepository
userRepo   UserRepository
familyRepo FamilyRepository
logger     *slog.Logger
}

func NewInviteService(
inviteRepo user.InviteRepository,
userRepo UserRepository,
familyRepo FamilyRepository,
logger *slog.Logger,
) InviteService {
return &inviteService{
inviteRepo: inviteRepo,
userRepo:   userRepo,
familyRepo: familyRepo,
logger:     logger,
}
}
```

Добавить import `"log/slog"`.

### 2. Заменить `_ = updateErr` на логирование

**GetInviteByToken (строка 137-140):**

```go
if updateErr := s.inviteRepo.Update(invite); updateErr != nil {
s.logger.Warn("failed to update expired invite status",
slog.String("invite_id", invite.ID.String()),
slog.String("error", updateErr.Error()),
)
}
```

**AcceptInvite (строка 200-203):**

```go
if updateErr := s.inviteRepo.Update(invite); updateErr != nil {
s.logger.Error("failed to mark invite as accepted",
slog.String("invite_id", invite.ID.String()),
slog.String("user_id", newUser.ID.String()),
slog.String("error", updateErr.Error()),
)
}
```

**ListFamilyInvites (строка 267-270):**

```go
if updateErr := s.inviteRepo.Update(inv); updateErr != nil {
s.logger.Warn("failed to update expired invite status",
slog.String("invite_id", inv.ID.String()),
slog.String("error", updateErr.Error()),
)
}
```

### 3. Обновить место создания сервиса

Найти где вызывается `NewInviteService` и передать logger. Обычно это в `internal/run.go` или аналогичном
bootstrap-файле:

```go
inviteService := services.NewInviteService(inviteRepo, userRepo, familyRepo, logger)
```

### 4. Аналогично для backup_service.go

В `CreateBackup()` строка 105 тоже есть `_ = cleanupErr`. Применить тот же паттерн:

```go
if cleanupErr := s.cleanupOldBackups(ctx); cleanupErr != nil {
s.logger.Warn("failed to cleanup old backups",
slog.String("error", cleanupErr.Error()),
)
}
```

## Файлы для изменения

1. `internal/services/invite_service.go` — добавить logger, заменить `_ = err`
2. `internal/services/backup_service.go` — добавить logger, заменить `_ = err`
3. Файл bootstrap (где создаются сервисы) — передать logger в конструкторы

## Тестирование

- Обновить тесты invite_service_test.go — передать `slog.Default()` в конструктор
- Обновить тесты backup_service_test.go — передать `slog.Default()` в конструктор
- `make test`
- `make lint`

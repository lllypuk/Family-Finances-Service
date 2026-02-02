# CQ-5: N+1 запросы при обновлении просроченных приглашений

## Статус: TODO

## Приоритет: IMPORTANT

## Проблема

В `internal/services/invite_service.go:264-272`:

```go
for _, inv := range invites {
if inv.Status == user.InviteStatusPending && inv.IsExpired() {
inv.MarkExpired()
if updateErr := s.inviteRepo.Update(inv); updateErr != nil {
_ = updateErr
}
}
}
```

Каждый expired invite обновляется отдельным запросом в БД. При большом количестве приглашений это создаёт N+1 проблему.

## Решение

### Вариант A: Добавить bulk-метод в репозиторий (рекомендуется)

#### 1. Обновить интерфейс InviteRepository

В `internal/domain/user/invite.go`:

```go
type InviteRepository interface {
// ... existing methods ...

// MarkExpiredBulk marks all pending invites past their expiration as expired
MarkExpiredBulk(ctx context.Context) (int64, error)
}
```

#### 2. Реализация в репозитории

В `internal/infrastructure/invite_repository.go`:

```go
func (r *inviteRepository) MarkExpiredBulk(ctx context.Context) (int64, error) {
query := `UPDATE invites SET status = ?, updated_at = ?
			  WHERE status = ? AND expires_at < ?`
result, err := r.db.ExecContext(ctx, query,
user.InviteStatusExpired, time.Now(),
user.InviteStatusPending, time.Now(),
)
if err != nil {
return 0, fmt.Errorf("failed to bulk expire invites: %w", err)
}
return result.RowsAffected()
}
```

#### 3. Обновить ListFamilyInvites

```go
func (s *inviteService) ListFamilyInvites(ctx context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
// Bulk-update expired invites in a single query
if _, err := s.inviteRepo.MarkExpiredBulk(ctx); err != nil {
s.logger.Warn("failed to bulk expire invites", slog.String("error", err.Error()))
}

invites, err := s.inviteRepo.GetByFamily(ctx, familyID)
if err != nil {
return nil, fmt.Errorf("failed to get family invites: %w", err)
}

return invites, nil
}
```

### Вариант B: Scheduled job (дополнительно)

Добавить периодическую задачу, которая чистит expired invites, чтобы не делать это при каждом запросе списка.

## Файлы для изменения

1. `internal/domain/user/invite.go` — добавить `MarkExpiredBulk` в интерфейс
2. `internal/infrastructure/invite_repository.go` — реализовать `MarkExpiredBulk`
3. `internal/services/invite_service.go` — обновить `ListFamilyInvites`

## Тестирование

- Тест для `MarkExpiredBulk` с несколькими expired invites
- Тест что `ListFamilyInvites` больше не вызывает `Update` в цикле
- `make test`
- `make lint`

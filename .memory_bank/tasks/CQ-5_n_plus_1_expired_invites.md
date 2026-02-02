# CQ-5: N+1 запросы при обновлении просроченных приглашений

## Статус: DONE

## Приоритет: IMPORTANT

## Дата выполнения: 2026-02-02

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

## Реализация

### Выполнено:

1. ✅ Добавлен метод `MarkExpiredBulk(ctx context.Context) (int64, error)` в интерфейс `InviteRepository` (`internal/domain/user/invite.go`)

2. ✅ Реализован метод `MarkExpiredBulk` в `InviteSQLiteRepository` (`internal/infrastructure/user/invite_repository_sqlite.go`):
   - Обновляет все pending приглашения с истекшим сроком одним SQL-запросом
   - Возвращает количество обновленных записей
   - Использует тот же паттерн timeout и обработки ошибок, что и другие методы

3. ✅ Обновлен метод `ListFamilyInvites` в `InviteService` (`internal/services/invite_service.go`):
   - Удален N+1 цикл с индивидуальными `Update` запросами
   - Добавлен вызов `MarkExpiredBulk` перед получением списка приглашений
   - При ошибке bulk-операции логируется предупреждение, но выполнение продолжается

4. ✅ Добавлен тест `MarkExpiredBulk_Success` в `invite_repository_test.go`:
   - Проверяет корректное обновление нескольких expired invites
   - Проверяет, что valid invites не затронуты
   - Проверяет, что уже expired invites не считаются дважды
   - Проверяет правильное количество затронутых строк

5. ✅ Обновлен мок `MockInviteRepository` (`internal/services/invite_service_test.go`):
   - Добавлен метод `MarkExpiredBulk`

6. ✅ Обновлены тесты `TestInviteService_ListFamilyInvites`:
   - Заменены моки с `Update` на `MarkExpiredBulk`
   - Добавлен тест-кейс для случая, когда bulk-операция падает (проверяется graceful degradation)

### Результаты тестирования:

```bash
# Все тесты пройдены успешно
make test  # ✅ PASS
make lint  # ✅ 0 issues
```

### Производительность:

**До**: N+1 запросов (1 SELECT + N UPDATE для каждого expired invite)
**После**: 2 запроса (1 UPDATE для всех expired invites + 1 SELECT)

**Улучшение**: При наличии N просроченных приглашений сокращение с O(N) UPDATE запросов до O(1) UPDATE запроса.

### Примечания:

- Метод устойчив к ошибкам: если bulk-update падает, логируется предупреждение, но список приглашений все равно возвращается
- Bulk-операция выполняется на уровне БД, что безопасно для конкурентного доступа
- Количество обновленных строк возвращается для потенциального мониторинга/метрик
````

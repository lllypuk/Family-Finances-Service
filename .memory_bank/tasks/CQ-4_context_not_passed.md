# CQ-4: Context не прокидывается в репозитории

## Статус: TODO

## Приоритет: IMPORTANT

## Проблема

В `internal/services/invite_service.go` `context.Context` принимается в методах, но не передаётся в вызовы репозитория:

**GetInviteByToken (строка 128):**

```go
func (s *inviteService) GetInviteByToken(_ context.Context, token string) (*user.Invite, error) {
invite, err := s.inviteRepo.GetByToken(token) // context не передаётся
```

**ListFamilyInvites (строка 257):**

```go
func (s *inviteService) ListFamilyInvites(_ context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
invites, err := s.inviteRepo.GetByFamily(familyID) // context не передаётся
```

**DeleteExpiredInvites (строка 279):**

```go
func (s *inviteService) DeleteExpiredInvites(_ context.Context) error {
if err := s.inviteRepo.DeleteExpired(); err != nil { // context не передаётся
```

Также внутри других методов (`CreateInvite`, `AcceptInvite`, `RevokeInvite`) вызовы `inviteRepo` не передают context.

## Причина

Интерфейс `InviteRepository` (`internal/domain/user/invite.go:113-135`) не принимает `context.Context`:

```go
type InviteRepository interface {
Create(invite *Invite) error
GetByToken(token string) (*Invite, error)
GetByID(id uuid.UUID) (*Invite, error)
GetByFamily(familyID uuid.UUID) ([]*Invite, error)
GetPendingByEmail(email string) ([]*Invite, error)
Update(invite *Invite) error
Delete(id uuid.UUID) error
DeleteExpired() error
}
```

## Решение

### 1. Обновить интерфейс InviteRepository

`internal/domain/user/invite.go`:

```go
type InviteRepository interface {
Create(ctx context.Context, invite *Invite) error
GetByToken(ctx context.Context, token string) (*Invite, error)
GetByID(ctx context.Context, id uuid.UUID) (*Invite, error)
GetByFamily(ctx context.Context, familyID uuid.UUID) ([]*Invite, error)
GetPendingByEmail(ctx context.Context, email string) ([]*Invite, error)
Update(ctx context.Context, invite *Invite) error
Delete(ctx context.Context, id uuid.UUID) error
DeleteExpired(ctx context.Context) error
}
```

Добавить import `"context"`.

### 2. Обновить реализацию репозитория

Найти реализацию в `internal/infrastructure/` (предположительно `invite_repository.go`). Обновить все методы:

```go
func (r *inviteRepository) Create(ctx context.Context, invite *user.Invite) error {
_, err := r.db.ExecContext(ctx, query, args...)
// ...
}

func (r *inviteRepository) GetByToken(ctx context.Context, token string) (*user.Invite, error) {
row := r.db.QueryRowContext(ctx, query, token)
// ...
}
// ... и так далее для всех методов
```

### 3. Обновить invite_service.go

Прокинуть context во все вызовы repo:

```go
func (s *inviteService) GetInviteByToken(ctx context.Context, token string) (*user.Invite, error) {
invite, err := s.inviteRepo.GetByToken(ctx, token)
// ...
if updateErr := s.inviteRepo.Update(ctx, invite); updateErr != nil {
// ...
}

func (s *inviteService) CreateInvite(ctx context.Context, creatorID uuid.UUID, req dto.CreateInviteDTO) (*user.Invite, error) {
// ...
pendingInvites, err := s.inviteRepo.GetPendingByEmail(ctx, email)
// ...
if createErr := s.inviteRepo.Create(ctx, invite); createErr != nil {
// ...
}

func (s *inviteService) AcceptInvite(ctx context.Context, token string, req dto.AcceptInviteDTO) (*user.User, error) {
// ...
if updateErr := s.inviteRepo.Update(ctx, invite); updateErr != nil {
// ...
}

func (s *inviteService) RevokeInvite(ctx context.Context, inviteID, revokerID uuid.UUID) error {
// ...
invite, err := s.inviteRepo.GetByID(ctx, inviteID)
// ...
if updateErr := s.inviteRepo.Update(ctx, invite); updateErr != nil {
// ...
}

func (s *inviteService) ListFamilyInvites(ctx context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
invites, err := s.inviteRepo.GetByFamily(ctx, familyID)
// ...
if updateErr := s.inviteRepo.Update(ctx, inv); updateErr != nil {
// ...
}

func (s *inviteService) DeleteExpiredInvites(ctx context.Context) error {
if err := s.inviteRepo.DeleteExpired(ctx); err != nil {
// ...
}
```

### 4. Обновить тесты

Все моки InviteRepository нужно обновить, чтобы принимать `context.Context` как первый аргумент.

## Файлы для изменения

1. `internal/domain/user/invite.go` — обновить интерфейс InviteRepository
2. `internal/infrastructure/invite_repository.go` (или аналогичный) — обновить реализацию
3. `internal/services/invite_service.go` — прокинуть ctx во все вызовы
4. Все тестовые файлы, использующие InviteRepository — обновить моки

## Тестирование

- `make test` — все тесты проходят
- `make lint` — 0 issues

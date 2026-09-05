package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

// Срок сессии: продлевается активностью на IdleTTL, но не дольше AbsoluteTTL с момента выдачи.
const (
	IdleTTL     = 30 * 24 * time.Hour
	AbsoluteTTL = 180 * 24 * time.Hour
)

// ErrSessionNotFound — сессии с таким id или хешем токена нет.
var ErrSessionNotFound = errors.New("session not found")

// Session — bearer-сессия; в БД хранится только хеш токена.
type Session struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	DeviceName string
	CreatedAt  time.Time
	LastUsedAt time.Time
	ExpiresAt  time.Time
}

// NewSession — сессия, выданная в момент now.
func NewSession(userID uuid.UUID, tokenHash, deviceName string, now time.Time) *Session {
	s := &Session{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceName: deviceName,
		CreatedAt:  now,
		LastUsedAt: now,
	}
	s.ExpiresAt = s.ExpiryAfter(now)
	return s
}

// ExpiryAfter — срок сессии после активности в момент at.
func (s *Session) ExpiryAfter(at time.Time) time.Time {
	idle := at.Add(IdleTTL)
	absolute := s.CreatedAt.Add(AbsoluteTTL)
	if idle.After(absolute) {
		return absolute
	}
	return idle
}

// SessionRepository — хранилище сессий; реализация в internal/infrastructure/auth.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	// FindByTokenHash возвращает сессию и её активного владельца одним запросом.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, *user.User, error)
	Touch(ctx context.Context, id uuid.UUID, at time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteByUser отзывает все сессии пользователя, кроме exceptID (uuid.Nil — все).
	DeleteByUser(ctx context.Context, userID, exceptID uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	DeleteExpired(ctx context.Context, now time.Time) error
}

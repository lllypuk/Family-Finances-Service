package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
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

// NoExpiry — expires_at сессии: она живёт до отзыва, но поле обязательно для клиентов.
func NoExpiry() time.Time {
	return time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
}

// NewSession — сессия, выданная в момент now.
func NewSession(userID uuid.UUID, tokenHash, deviceName string, now time.Time) *Session {
	return &Session{
		ID:         uuid.New(),
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceName: deviceName,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  NoExpiry(),
	}
}

// SessionRepository — хранилище сессий; реализация в internal/infrastructure/auth.
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	// FindByTokenHash возвращает сессию и её активного владельца одним запросом.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, *user.User, error)
	// Touch записывает last_used_at; исчезнувшая сессия → ErrSessionNotFound.
	Touch(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteOwned удаляет сессию только её владельца; чужая или неизвестная → ErrSessionNotFound.
	DeleteOwned(ctx context.Context, userID, id uuid.UUID) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*Session, error)
}

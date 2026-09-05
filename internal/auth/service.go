package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

var (
	// ErrInvalidCredentials — неверные email или пароль; ответ одинаков для неизвестного email.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrSetupRequired — семья ещё не создана, логин невозможен.
	ErrSetupRequired = errors.New("setup required")
	// ErrUnauthorized — токен неизвестен или истёк.
	ErrUnauthorized = errors.New("unauthorized")
)

// dummyHash — bcrypt cost 12 от случайной строки, прообраз выброшен. Сравнивается с паролем
// при неизвестном email, чтобы ответ занимал столько же времени, сколько с настоящим хешем.
const dummyHash = "$2a$12$wFOQnJ9KhOwd9WIJUrpp8ObKeCKCr/1xA9YA2i6HiWBSqN5G4T4/S"

// UserLookup — доступ к пользователям; реализация в internal/infrastructure/user.
type UserLookup interface {
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
}

// SetupChecker сообщает, создана ли семья.
type SetupChecker interface {
	Exists(ctx context.Context) (bool, error)
}

// Principal — владелец сессии, попадает в контекст запроса.
type Principal struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	Email     string
	Role      user.Role
}

// LoginResult — выданный токен вместе с сессией и её владельцем.
type LoginResult struct {
	Token   string
	Session *Session
	User    *user.User
}

// Service выдаёт, проверяет и отзывает bearer-токены.
type Service struct {
	sessions SessionRepository
	users    UserLookup
	setup    SetupChecker
	now      func() time.Time
}

// NewService создаёт сервис.
func NewService(sessions SessionRepository, users UserLookup, setup SetupChecker) *Service {
	return &Service{
		sessions: sessions,
		users:    users,
		setup:    setup,
		now:      time.Now,
	}
}

// SetClock подменяет источник времени (для тестов).
func (s *Service) SetClock(now func() time.Time) {
	s.now = now
}

// Login проверяет пароль и выдаёт токен новой сессии; попутно чистит истёкшие сессии.
func (s *Service) Login(ctx context.Context, email, password, device string) (*LoginResult, error) {
	exists, err := s.setup.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check setup: %w", err)
	}
	if !exists {
		return nil, ErrSetupRequired
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			ComparePassword(dummyHash, password)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !ComparePassword(u.Password, password) || !u.IsActive {
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	if err = s.sessions.DeleteExpired(ctx, now); err != nil {
		return nil, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	plain, hash := GenerateToken()
	sess := NewSession(u.ID, hash, device, now)
	if err = s.sessions.Create(ctx, sess); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return &LoginResult{Token: plain, Session: sess, User: u}, nil
}

// Authenticate проверяет токен; просроченная сессия удаляется.
func (s *Service) Authenticate(ctx context.Context, token string) (*Principal, error) {
	sess, u, err := s.sessions.FindByTokenHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if !u.IsActive {
		return nil, ErrUnauthorized
	}

	now := s.now()
	if !now.Before(sess.ExpiresAt) {
		if err = s.sessions.Delete(ctx, sess.ID); err != nil && !errors.Is(err, ErrSessionNotFound) {
			return nil, fmt.Errorf("failed to delete expired session: %w", err)
		}
		return nil, ErrUnauthorized
	}
	if now.Sub(sess.LastUsedAt) > TouchInterval {
		// Отзыв между FindByTokenHash и Touch — сессии уже нет, это 401, а не сбой.
		if err = s.sessions.Touch(ctx, sess.ID, now, sess.ExpiryAfter(now)); err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				return nil, ErrUnauthorized
			}
			return nil, fmt.Errorf("failed to touch session: %w", err)
		}
	}

	return &Principal{SessionID: sess.ID, UserID: u.ID, Email: u.Email, Role: u.Role}, nil
}

// Logout удаляет сессию.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.sessions.Delete(ctx, sessionID)
}

// RevokeAllSessions удаляет все сессии пользователя (деактивация админом).
func (s *Service) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return s.sessions.DeleteByUser(ctx, userID, uuid.Nil)
}

// ListSessions — живые сессии пользователя, новые первыми; истёкшие лежат в БД до ближайшего логина.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	all, err := s.sessions.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	live := make([]*Session, 0, len(all))
	for _, sess := range all {
		if now.Before(sess.ExpiresAt) {
			live = append(live, sess)
		}
	}
	return live, nil
}

// RevokeSession удаляет сессию пользователя; чужая или неизвестная → ErrSessionNotFound.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	return s.sessions.DeleteOwned(ctx, userID, sessionID)
}

// ChangePassword меняет пароль по текущему и отзывает все сессии, кроме keepSessionID.
func (s *Service) ChangePassword(
	ctx context.Context,
	userID uuid.UUID,
	current, newPassword string,
	keepSessionID uuid.UUID,
) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if !ComparePassword(u.Password, current) {
		return ErrInvalidCredentials
	}
	return s.setPassword(ctx, userID, newPassword, keepSessionID)
}

// AdminSetPassword задаёт пароль без проверки текущего и отзывает все сессии пользователя.
func (s *Service) AdminSetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	return s.setPassword(ctx, userID, newPassword, uuid.Nil)
}

func (s *Service) setPassword(ctx context.Context, userID uuid.UUID, newPassword string, keep uuid.UUID) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err = s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if err = s.sessions.DeleteByUser(ctx, userID, keep); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}

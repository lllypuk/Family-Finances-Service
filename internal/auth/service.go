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

// Option настраивает Service.
type Option func(*Service)

// WithClock подменяет источник времени (для тестов).
func WithClock(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

// Service выдаёт, проверяет и отзывает bearer-токены.
type Service struct {
	sessions SessionRepository
	users    UserLookup
	setup    SetupChecker
	now      func() time.Time
	// dummyHash сравнивается с паролем при неизвестном email, чтобы ответ занимал столько же времени.
	dummyHash string
}

// NewService создаёт сервис; считает dummyHash (cost 12), поэтому вызывается один раз при старте.
func NewService(sessions SessionRepository, users UserLookup, setup SetupChecker, opts ...Option) (*Service, error) {
	dummy, err := HashPassword(uuid.NewString())
	if err != nil {
		return nil, err
	}
	s := &Service{
		sessions:  sessions,
		users:     users,
		setup:     setup,
		now:       time.Now,
		dummyHash: dummy,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Login проверяет пароль и выдаёт токен новой сессии.
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
			ComparePassword(s.dummyHash, password)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !ComparePassword(u.Password, password) || !u.IsActive {
		return nil, ErrInvalidCredentials
	}

	plain, hash := GenerateToken()
	sess := NewSession(u.ID, hash, device, s.now())
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
		if err = s.sessions.Touch(ctx, sess.ID, now); err != nil {
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

// ListSessions — сессии пользователя, новые первыми.
func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	return s.sessions.ListByUser(ctx, userID)
}

// RevokeSession удаляет сессию пользователя; чужая или неизвестная → ErrSessionNotFound.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	list, err := s.sessions.ListByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, sess := range list {
		if sess.ID == sessionID {
			return s.sessions.Delete(ctx, sessionID)
		}
	}
	return ErrSessionNotFound
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

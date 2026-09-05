package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

// SessionSQLiteRepository реализует auth.SessionRepository на SQLite.
type SessionSQLiteRepository struct {
	db *sql.DB
}

// NewSessionSQLiteRepository создаёт репозиторий сессий.
func NewSessionSQLiteRepository(db *sql.DB) *SessionSQLiteRepository {
	return &SessionSQLiteRepository{db: db}
}

// Create сохраняет сессию.
func (r *SessionSQLiteRepository) Create(ctx context.Context, s *auth.Session) error {
	if err := validation.ValidateUUID(s.ID); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}
	if err := validation.ValidateUUID(s.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		INSERT INTO sessions (id, user_id, token_hash, device_name, created_at, last_used_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		s.ID.String(), s.UserID.String(), s.TokenHash, s.DeviceName,
		formatTime(s.CreatedAt), formatTime(s.LastUsedAt), formatTime(s.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// FindByTokenHash ищет сессию по хешу токена вместе с владельцем; активность решает auth.Service.
func (r *SessionSQLiteRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*auth.Session, *user.User, error) {
	query := `
		SELECT s.id, s.user_id, s.token_hash, s.device_name, s.created_at, s.last_used_at, s.expires_at,
		       u.email, u.first_name, u.last_name, u.role, u.is_active
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ?`

	var (
		idStr, userIDStr, createdAt, lastUsedAt, expiresAt string
		s                                                  auth.Session
		u                                                  user.User
		roleStr                                            string
		isActive                                           int
	)

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&idStr, &userIDStr, &s.TokenHash, &s.DeviceName, &createdAt, &lastUsedAt, &expiresAt,
		&u.Email, &u.FirstName, &u.LastName, &roleStr, &isActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, auth.ErrSessionNotFound
		}
		return nil, nil, fmt.Errorf("failed to find session: %w", err)
	}

	if err = fillSession(&s, idStr, userIDStr, createdAt, lastUsedAt, expiresAt); err != nil {
		return nil, nil, err
	}
	u.ID = s.UserID
	u.Role = user.Role(roleStr)
	u.IsActive = isActive == 1
	return &s, &u, nil
}

// Touch продлевает сессию активностью в момент at.
func (r *SessionSQLiteRepository) Touch(ctx context.Context, id uuid.UUID, at time.Time) error {
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	var createdAt string
	err := r.db.QueryRowContext(ctx, `SELECT created_at FROM sessions WHERE id = ?`, id.String()).Scan(&createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrSessionNotFound
		}
		return fmt.Errorf("failed to read session: %w", err)
	}

	s := auth.Session{}
	if s.CreatedAt, err = parseTime(createdAt, "created_at"); err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE sessions SET last_used_at = ?, expires_at = ? WHERE id = ?`,
		formatTime(at), formatTime(s.ExpiryAfter(at)), id.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

// Delete удаляет сессию по id.
func (r *SessionSQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid session ID: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affected == 0 {
		return auth.ErrSessionNotFound
	}
	return nil
}

// DeleteByUser удаляет все сессии пользователя, кроме exceptID.
func (r *SessionSQLiteRepository) DeleteByUser(ctx context.Context, userID, exceptID uuid.UUID) error {
	if err := validation.ValidateUUID(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err := r.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ? AND id != ?`,
		userID.String(), exceptID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// ListByUser возвращает сессии пользователя, новые первыми.
func (r *SessionSQLiteRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*auth.Session, error) {
	if err := validation.ValidateUUID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT id, user_id, token_hash, device_name, created_at, last_used_at, expires_at
		FROM sessions
		WHERE user_id = ?
		ORDER BY created_at DESC, id`

	rows, err := r.db.QueryContext(ctx, query, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*auth.Session
	for rows.Next() {
		var (
			idStr, userIDStr, createdAt, lastUsedAt, expiresAt string
			s                                                  auth.Session
		)
		if err = rows.Scan(
			&idStr, &userIDStr, &s.TokenHash, &s.DeviceName, &createdAt, &lastUsedAt, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		if err = fillSession(&s, idStr, userIDStr, createdAt, lastUsedAt, expiresAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return sessions, nil
}

// DeleteExpired удаляет сессии, истёкшие к моменту now.
func (r *SessionSQLiteRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, formatTime(now)); err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

func fillSession(s *auth.Session, idStr, userIDStr, createdAt, lastUsedAt, expiresAt string) error {
	var err error
	if s.ID, err = uuid.Parse(idStr); err != nil {
		return fmt.Errorf("failed to parse session ID: %w", err)
	}
	if s.UserID, err = uuid.Parse(userIDStr); err != nil {
		return fmt.Errorf("failed to parse user ID: %w", err)
	}
	if s.CreatedAt, err = parseTime(createdAt, "created_at"); err != nil {
		return err
	}
	if s.LastUsedAt, err = parseTime(lastUsedAt, "last_used_at"); err != nil {
		return err
	}
	if s.ExpiresAt, err = parseTime(expiresAt, "expires_at"); err != nil {
		return err
	}
	return nil
}

// Время хранится строкой RFC3339 в UTC: так `expires_at < ?` сравнивается лексикографически.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(value, column string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %s: %w", column, err)
	}
	return t, nil
}

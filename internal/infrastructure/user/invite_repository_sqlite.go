package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

const (
	// dbTimeout is the default timeout for database operations
	dbTimeout = 5 * time.Second
)

// InviteSQLiteRepository implements invite repository using SQLite
type InviteSQLiteRepository struct {
	db *sql.DB
}

// NewInviteSQLiteRepository creates a new SQLite invite repository
func NewInviteSQLiteRepository(db *sql.DB) *InviteSQLiteRepository {
	return &InviteSQLiteRepository{
		db: db,
	}
}

// Create creates a new invite
func (r *InviteSQLiteRepository) Create(ctx context.Context, invite *user.Invite) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		INSERT INTO invites (
			id, family_id, created_by, email, role, token,
			status, expires_at, accepted_at, accepted_by,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var acceptedAt *string
	if invite.AcceptedAt != nil {
		at := invite.AcceptedAt.Format(time.RFC3339)
		acceptedAt = &at
	}

	var acceptedBy *string
	if invite.AcceptedBy != nil {
		by := invite.AcceptedBy.String()
		acceptedBy = &by
	}

	_, err := r.db.ExecContext(ctx, query,
		invite.ID.String(),
		invite.FamilyID.String(),
		invite.CreatedBy.String(),
		invite.Email,
		string(invite.Role),
		invite.Token,
		string(invite.Status),
		invite.ExpiresAt.Format(time.RFC3339),
		acceptedAt,
		acceptedBy,
		invite.CreatedAt.Format(time.RFC3339),
		invite.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create invite: %w", err)
	}

	return nil
}

// GetByToken retrieves an invite by its token
func (r *InviteSQLiteRepository) GetByToken(ctx context.Context, token string) (*user.Invite, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		SELECT id, family_id, created_by, email, role, token,
		       status, expires_at, accepted_at, accepted_by,
		       created_at, updated_at
		FROM invites
		WHERE token = ?
	`

	invite, err := r.scanInvite(r.db.QueryRowContext(ctx, query, token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invite not found")
		}
		return nil, fmt.Errorf("failed to get invite by token: %w", err)
	}

	return invite, nil
}

// GetByID retrieves an invite by its ID
func (r *InviteSQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Invite, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		SELECT id, family_id, created_by, email, role, token,
		       status, expires_at, accepted_at, accepted_by,
		       created_at, updated_at
		FROM invites
		WHERE id = ?
	`

	invite, err := r.scanInvite(r.db.QueryRowContext(ctx, query, id.String()))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invite not found")
		}
		return nil, fmt.Errorf("failed to get invite by ID: %w", err)
	}

	return invite, nil
}

// GetByFamily retrieves all invites for a family
func (r *InviteSQLiteRepository) GetByFamily(ctx context.Context, familyID uuid.UUID) ([]*user.Invite, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		SELECT id, family_id, created_by, email, role, token,
		       status, expires_at, accepted_at, accepted_by,
		       created_at, updated_at
		FROM invites
		WHERE family_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, familyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query invites by family: %w", err)
	}
	defer rows.Close()

	var invites []*user.Invite
	for rows.Next() {
		inv, scanErr := r.scanInvite(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan invite: %w", scanErr)
		}
		invites = append(invites, inv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invites: %w", err)
	}

	return invites, nil
}

// GetPendingByEmail retrieves pending invites for an email
func (r *InviteSQLiteRepository) GetPendingByEmail(ctx context.Context, email string) ([]*user.Invite, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		SELECT id, family_id, created_by, email, role, token,
		       status, expires_at, accepted_at, accepted_by,
		       created_at, updated_at
		FROM invites
		WHERE email = ? AND status = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, email, string(user.InviteStatusPending))
	if err != nil {
		return nil, fmt.Errorf("failed to query pending invites: %w", err)
	}
	defer rows.Close()

	var invites []*user.Invite
	for rows.Next() {
		inv, scanErr := r.scanInvite(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan invite: %w", scanErr)
		}
		invites = append(invites, inv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invites: %w", err)
	}

	return invites, nil
}

// Update updates an invite
func (r *InviteSQLiteRepository) Update(ctx context.Context, invite *user.Invite) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		UPDATE invites
		SET status = ?, accepted_at = ?, accepted_by = ?, updated_at = ?
		WHERE id = ?
	`

	var acceptedAt *string
	if invite.AcceptedAt != nil {
		at := invite.AcceptedAt.Format(time.RFC3339)
		acceptedAt = &at
	}

	var acceptedBy *string
	if invite.AcceptedBy != nil {
		by := invite.AcceptedBy.String()
		acceptedBy = &by
	}

	result, err := r.db.ExecContext(ctx, query,
		string(invite.Status),
		acceptedAt,
		acceptedBy,
		time.Now().Format(time.RFC3339),
		invite.ID.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to update invite: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite not found")
	}

	return nil
}

// Delete deletes an invite
func (r *InviteSQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `DELETE FROM invites WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete invite: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("invite not found")
	}

	return nil
}

// DeleteExpired deletes all expired invites
func (r *InviteSQLiteRepository) DeleteExpired(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		DELETE FROM invites
		WHERE status = ? AND expires_at < ?
	`

	_, err := r.db.ExecContext(ctx, query,
		string(user.InviteStatusPending),
		time.Now().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to delete expired invites: %w", err)
	}

	return nil
}

// MarkExpiredBulk marks all pending invites past their expiration as expired
func (r *InviteSQLiteRepository) MarkExpiredBulk(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	query := `
		UPDATE invites 
		SET status = ?, updated_at = ?
		WHERE status = ? AND expires_at < ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		string(user.InviteStatusExpired),
		now.Format(time.RFC3339),
		string(user.InviteStatusPending),
		now.Format(time.RFC3339),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to bulk expire invites: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// scanInvite scans a single invite from a row or rows
func (r *InviteSQLiteRepository) scanInvite(scanner interface {
	Scan(dest ...interface{}) error
}) (*user.Invite, error) {
	var (
		id, familyID, createdBy, email, role, token, status string
		expiresAt, createdAt, updatedAt                     string
		acceptedAt, acceptedBy                              *string
	)

	err := scanner.Scan(
		&id, &familyID, &createdBy, &email, &role, &token,
		&status, &expiresAt, &acceptedAt, &acceptedBy,
		&createdAt, &updatedAt,
	)

	if err != nil {
		return nil, err
	}

	inviteID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse invite ID: %w", err)
	}

	familyUUID, err := uuid.Parse(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	createdByUUID, err := uuid.Parse(createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_by ID: %w", err)
	}

	expiresAtTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_at: %w", err)
	}

	createdAtTime, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAtTime, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	invite := &user.Invite{
		ID:        inviteID,
		FamilyID:  familyUUID,
		CreatedBy: createdByUUID,
		Email:     email,
		Role:      user.Role(role),
		Token:     token,
		Status:    user.InviteStatus(status),
		ExpiresAt: expiresAtTime,
		CreatedAt: createdAtTime,
		UpdatedAt: updatedAtTime,
	}

	if acceptedAt != nil {
		acceptedAtTime, parseErr := time.Parse(time.RFC3339, *acceptedAt)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse accepted_at: %w", parseErr)
		}
		invite.AcceptedAt = &acceptedAtTime
	}

	if acceptedBy != nil {
		acceptedByUUID, parseErr := uuid.Parse(*acceptedBy)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse accepted_by: %w", parseErr)
		}
		invite.AcceptedBy = &acceptedByUUID
	}

	return invite, nil
}

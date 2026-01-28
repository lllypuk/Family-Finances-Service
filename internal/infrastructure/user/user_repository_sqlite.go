package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements user repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite user repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// scanUser is a helper function to scan user from rows and handle UUID parsing
func scanUser(rows *sql.Rows) (*user.User, error) {
	var u user.User
	var idStr, familyIDStr string
	var roleStr string
	var isActive int
	var lastLogin sql.NullTime

	err := rows.Scan(
		&idStr, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&roleStr, &familyIDStr, &isActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	// Parse UUIDs
	u.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	u.FamilyID, err = uuid.Parse(familyIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	u.Role = user.Role(roleStr)
	return &u, nil
}

// Create creates a new user in the database
func (r *SQLiteRepository) Create(ctx context.Context, u *user.User) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(u.ID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateUUID(u.FamilyID); err != nil {
		return fmt.Errorf("invalid user familyID: %w", err)
	}

	// Validate email to prevent injection attacks
	if err := validation.ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("invalid user email: %w", err)
	}

	// Sanitize email before storing
	u.Email = validation.SanitizeEmail(u.Email)

	// Set timestamps
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, role, family_id,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID.String(), u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID.String(), 1, u.CreatedAt, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM users
		WHERE id = ? AND is_active = 1`

	var u user.User
	var idStr, familyIDStr string
	var roleStr string
	var isActive int
	var lastLogin sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&idStr, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&roleStr, &familyIDStr, &isActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	// Parse UUIDs
	u.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	u.FamilyID, err = uuid.Parse(familyIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	u.Role = user.Role(roleStr)
	return &u, nil
}

// GetByEmail retrieves a user by their email address
func (r *SQLiteRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	// Validate email to prevent injection attacks
	if err := validation.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email parameter: %w", err)
	}

	// Sanitize email for consistent querying
	sanitizedEmail := validation.SanitizeEmail(email)

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM users
		WHERE email = ? AND is_active = 1`

	var u user.User
	var idStr, familyIDStr string
	var roleStr string
	var isActive int
	var lastLogin sql.NullTime

	err := r.db.QueryRowContext(ctx, query, sanitizedEmail).Scan(
		&idStr, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&roleStr, &familyIDStr, &isActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user with email %s not found", sanitizedEmail)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Parse UUIDs
	u.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	u.FamilyID, err = uuid.Parse(familyIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse family ID: %w", err)
	}

	u.Role = user.Role(roleStr)
	return &u, nil
}

// GetByFamilyID retrieves all users belonging to a specific family
func (r *SQLiteRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM users
		WHERE family_id = ? AND is_active = 1
		ORDER BY role, first_name, last_name`

	rows, err := r.db.QueryContext(ctx, query, familyID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get users by family id: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

// Update updates an existing user
func (r *SQLiteRepository) Update(ctx context.Context, u *user.User) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(u.ID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateUUID(u.FamilyID); err != nil {
		return fmt.Errorf("invalid user familyID: %w", err)
	}

	// Validate email to prevent injection attacks
	if err := validation.ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("invalid user email: %w", err)
	}

	// Sanitize email before updating
	u.Email = validation.SanitizeEmail(u.Email)

	// Update timestamp
	u.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = ?, password_hash = ?, first_name = ?, last_name = ?,
			role = ?, family_id = ?, updated_at = ?
		WHERE id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query,
		u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID.String(), u.UpdatedAt, u.ID.String(),
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %s not found", u.ID)
	}

	return nil
}

// Delete soft deletes a user (sets is_active to false)
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		UPDATE users
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND family_id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, id.String(), familyID.String())
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *SQLiteRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		UPDATE users
		SET last_login = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}

	return nil
}

// GetUsersByRole retrieves all users with a specific role in a family
func (r *SQLiteRepository) GetUsersByRole(
	ctx context.Context,
	familyID uuid.UUID,
	role user.Role,
) ([]*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM users
		WHERE family_id = ? AND role = ? AND is_active = 1
		ORDER BY first_name, last_name`

	rows, err := r.db.QueryContext(ctx, query, familyID.String(), string(role))
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		u, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

// CreateWithTransaction creates a user within a database transaction
func (r *SQLiteRepository) CreateWithTransaction(ctx context.Context, tx *sql.Tx, u *user.User) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(u.ID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateUUID(u.FamilyID); err != nil {
		return fmt.Errorf("invalid user familyID: %w", err)
	}

	// Validate email to prevent injection attacks
	if err := validation.ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("invalid user email: %w", err)
	}

	// Sanitize email before storing
	u.Email = validation.SanitizeEmail(u.Email)

	// Set timestamps
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, role, family_id,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := tx.ExecContext(ctx, query,
		u.ID.String(), u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID.String(), 1, u.CreatedAt, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

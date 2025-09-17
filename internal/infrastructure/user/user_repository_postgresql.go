package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

// PostgreSQLRepository implements user repository using PostgreSQL
type PostgreSQLRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLRepository creates a new PostgreSQL user repository
func NewPostgreSQLRepository(db *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// Create creates a new user in the database
func (r *PostgreSQLRepository) Create(ctx context.Context, u *user.User) error {
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
		INSERT INTO family_budget.users (
			id, email, password_hash, first_name, last_name, role, family_id,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID, true, u.CreatedAt, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *PostgreSQLRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM family_budget.users
		WHERE id = $1 AND is_active = true`

	var u user.User
	var roleStr string
	var lastLogin *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&roleStr, &u.FamilyID, new(bool), &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	u.Role = user.Role(roleStr)
	return &u, nil
}

// GetByEmail retrieves a user by their email address
func (r *PostgreSQLRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	// Validate email to prevent injection attacks
	if err := validation.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email parameter: %w", err)
	}

	// Sanitize email for consistent querying
	sanitizedEmail := validation.SanitizeEmail(email)

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM family_budget.users
		WHERE email = $1 AND is_active = true`

	var u user.User
	var roleStr string
	var lastLogin *time.Time

	err := r.db.QueryRow(ctx, query, sanitizedEmail).Scan(
		&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
		&roleStr, &u.FamilyID, new(bool), &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user with email %s not found", sanitizedEmail)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	u.Role = user.Role(roleStr)
	return &u, nil
}

// GetByFamilyID retrieves all users belonging to a specific family
func (r *PostgreSQLRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM family_budget.users
		WHERE family_id = $1 AND is_active = true
		ORDER BY role, first_name, last_name`

	rows, err := r.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by family id: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		var u user.User
		var roleStr string
		var lastLogin *time.Time

		err = rows.Scan(
			&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
			&roleStr, &u.FamilyID, new(bool), &lastLogin, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		u.Role = user.Role(roleStr)
		users = append(users, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

// Update updates an existing user
func (r *PostgreSQLRepository) Update(ctx context.Context, u *user.User) error {
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
		UPDATE family_budget.users
		SET email = $2, password_hash = $3, first_name = $4, last_name = $5,
			role = $6, family_id = $7, updated_at = $8
		WHERE id = $1 AND is_active = true`

	result, err := r.db.Exec(ctx, query,
		u.ID, u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with id %s not found", u.ID)
	}

	return nil
}

// Delete soft deletes a user (sets is_active to false)
func (r *PostgreSQLRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		UPDATE family_budget.users
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND family_id = $2 AND is_active = true`

	result, err := r.db.Exec(ctx, query, id, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *PostgreSQLRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		UPDATE family_budget.users
		SET last_login = NOW(), updated_at = NOW()
		WHERE id = $1 AND is_active = true`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}

	return nil
}

// GetUsersByRole retrieves all users with a specific role in a family
func (r *PostgreSQLRepository) GetUsersByRole(ctx context.Context, familyID uuid.UUID, role user.Role) ([]*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, email, password_hash, first_name, last_name, role, family_id,
			   is_active, last_login, created_at, updated_at
		FROM family_budget.users
		WHERE family_id = $1 AND role = $2 AND is_active = true
		ORDER BY first_name, last_name`

	rows, err := r.db.Query(ctx, query, familyID, string(role))
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer rows.Close()

	var users []*user.User
	for rows.Next() {
		var u user.User
		var roleStr string
		var lastLogin *time.Time

		err = rows.Scan(
			&u.ID, &u.Email, &u.Password, &u.FirstName, &u.LastName,
			&roleStr, &u.FamilyID, new(bool), &lastLogin, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		u.Role = user.Role(roleStr)
		users = append(users, &u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

// CreateWithTransaction creates a user within a database transaction
func (r *PostgreSQLRepository) CreateWithTransaction(ctx context.Context, tx pgx.Tx, u *user.User) error {
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
		INSERT INTO family_budget.users (
			id, email, password_hash, first_name, last_name, role, family_id,
			is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := tx.Exec(ctx, query,
		u.ID, u.Email, u.Password, u.FirstName, u.LastName,
		string(u.Role), u.FamilyID, true, u.CreatedAt, u.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (email already exists)
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

package category

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements category repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// Node represents a category node in the hierarchy
type Node struct {
	ID    uuid.UUID     `json:"id"`
	Name  string        `json:"name"`
	Type  category.Type `json:"type"`
	Level int           `json:"level"`
}

// NewSQLiteRepository creates a new SQLite category repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// Create creates a new category in the database
func (r *SQLiteRepository) Create(ctx context.Context, c *category.Category) error {
	// Validate category parameters before creating
	if err := validation.ValidateUUID(c.ID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(c.FamilyID); err != nil {
		return fmt.Errorf("invalid category familyID: %w", err)
	}
	if err := validation.ValidateCategoryType(c.Type); err != nil {
		return fmt.Errorf("invalid category type: %w", err)
	}
	if err := validation.ValidateCategoryName(c.Name); err != nil {
		return fmt.Errorf("invalid category name: %w", err)
	}

	// Validate parent ID if provided
	if c.ParentID != nil {
		if err := validation.ValidateUUID(*c.ParentID); err != nil {
			return fmt.Errorf("invalid parent ID: %w", err)
		}

		// Check that parent exists and belongs to the same family
		if err := r.validateParentCategory(ctx, *c.ParentID, c.FamilyID, c.Type); err != nil {
			return fmt.Errorf("invalid parent category: %w", err)
		}
	}

	// Set timestamps
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	query := `
		INSERT INTO categories (
			id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(c.ID),
		c.Name,
		string(c.Type),
		"",
		sqlitehelpers.UUIDPtrToString(c.ParentID),
		sqlitehelpers.UUIDToString(c.FamilyID),
		sqlitehelpers.BoolToInt(c.IsActive),
		c.CreatedAt,
		c.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("category with name '%s' already exists in this family", c.Name)
		}
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// GetByID retrieves a category by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM categories
		WHERE id = ?`

	var c category.Category
	var idStr, typeStr, familyIDStr string
	var description, parentIDStr *string
	var isActiveInt int

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &c.Name, &typeStr, &description, &parentIDStr, &familyIDStr,
		&isActiveInt, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("category with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}

	// Parse UUID fields
	c.ID, _ = uuid.Parse(idStr)
	c.FamilyID, _ = uuid.Parse(familyIDStr)
	if parentIDStr != nil && *parentIDStr != "" {
		parentID, _ := uuid.Parse(*parentIDStr)
		c.ParentID = &parentID
	}

	c.Type = category.Type(typeStr)
	c.IsActive = sqlitehelpers.IntToBool(isActiveInt)

	return &c, nil
}

// scanCategories is a helper method to scan multiple categories from query results
func (r *SQLiteRepository) scanCategories(rows *sql.Rows, errorContext string) ([]*category.Category, error) {
	var categories []*category.Category
	for rows.Next() {
		var c category.Category
		var idStr, typeStr, familyIDStr string
		var description, parentIDStr *string
		var isActiveInt int

		err := rows.Scan(
			&idStr, &c.Name, &typeStr, &description, &parentIDStr, &familyIDStr,
			&isActiveInt, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		// Parse UUID fields
		c.ID, _ = uuid.Parse(idStr)
		c.FamilyID, _ = uuid.Parse(familyIDStr)
		if parentIDStr != nil && *parentIDStr != "" {
			parentID, _ := uuid.Parse(*parentIDStr)
			c.ParentID = &parentID
		}

		c.Type = category.Type(typeStr)
		c.IsActive = sqlitehelpers.IntToBool(isActiveInt)
		categories = append(categories, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s rows iteration error: %w", errorContext, err)
	}

	return categories, nil
}

// GetByFamilyID retrieves all categories belonging to a specific family
func (r *SQLiteRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM categories
		WHERE family_id = ? AND is_active = 1
		ORDER BY type, parent_id NULLS FIRST, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by family id: %w", err)
	}
	defer rows.Close()

	return r.scanCategories(rows, "get categories by family id")
}

// GetByFamilyIDAndType retrieves categories by family ID and type
func (r *SQLiteRepository) GetByFamilyIDAndType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.Type,
) ([]*category.Category, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}
	if err := validation.ValidateCategoryType(categoryType); err != nil {
		return nil, fmt.Errorf("invalid category type: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM categories
		WHERE family_id = ? AND type = ? AND is_active = 1
		ORDER BY parent_id NULLS FIRST, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), string(categoryType))
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by family and type: %w", err)
	}
	defer rows.Close()

	return r.scanCategories(rows, "get categories by family and type")
}

// scanNodesSQLite scans rows into Node slice for SQLite
func scanNodesSQLite(rows *sql.Rows) ([]*Node, error) {
	var nodes []*Node
	for rows.Next() {
		var node Node
		var idStr, typeStr string

		err := rows.Scan(&idStr, &node.Name, &typeStr, &node.Level)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category node: %w", err)
		}

		node.ID, _ = uuid.Parse(idStr)
		node.Type = category.Type(typeStr)
		nodes = append(nodes, &node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return nodes, nil
}

// GetCategoryChildren returns all children of a category using recursive CTE
// SQLite supports WITH RECURSIVE since version 3.8.3
func (r *SQLiteRepository) GetCategoryChildren(ctx context.Context, parentID uuid.UUID) ([]*Node, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(parentID); err != nil {
		return nil, fmt.Errorf("invalid parentID parameter: %w", err)
	}

	query := `
		WITH RECURSIVE category_tree AS (
			-- Base case: start with the parent category
			SELECT c.id, c.name, c.type, c.parent_id, c.family_id, 0 as level
			FROM categories c
			WHERE c.id = ? AND c.is_active = 1

			UNION ALL

			-- Recursive case: find children
			SELECT c.id, c.name, c.type, c.parent_id, c.family_id, ct.level + 1
			FROM categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
			WHERE c.is_active = 1
		)
		SELECT id, name, type, level
		FROM category_tree
		ORDER BY level, name`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(parentID))
	if err != nil {
		return nil, fmt.Errorf("failed to get category children: %w", err)
	}
	defer rows.Close()

	return scanNodesSQLite(rows)
}

// GetCategoryPath returns the path from root to a specific category
func (r *SQLiteRepository) GetCategoryPath(ctx context.Context, categoryID uuid.UUID) ([]*Node, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(categoryID); err != nil {
		return nil, fmt.Errorf("invalid categoryID parameter: %w", err)
	}

	query := `
		WITH RECURSIVE category_path AS (
			-- Base case: start with the target category
			SELECT c.id, c.name, c.type, c.parent_id, 0 as level
			FROM categories c
			WHERE c.id = ? AND c.is_active = 1

			UNION ALL

			-- Recursive case: go up to parents
			SELECT c.id, c.name, c.type, c.parent_id, cp.level + 1
			FROM categories c
			INNER JOIN category_path cp ON c.id = cp.parent_id
			WHERE c.is_active = 1
		)
		SELECT id, name, type, level
		FROM category_path
		ORDER BY level DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(categoryID))
	if err != nil {
		return nil, fmt.Errorf("failed to get category path: %w", err)
	}
	defer rows.Close()

	return scanNodesSQLite(rows)
}

// Update updates an existing category
func (r *SQLiteRepository) Update(ctx context.Context, c *category.Category) error {
	// Validate category parameters
	if err := validation.ValidateUUID(c.ID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(c.FamilyID); err != nil {
		return fmt.Errorf("invalid category familyID: %w", err)
	}
	if err := validation.ValidateCategoryType(c.Type); err != nil {
		return fmt.Errorf("invalid category type: %w", err)
	}
	if err := validation.ValidateCategoryName(c.Name); err != nil {
		return fmt.Errorf("invalid category name: %w", err)
	}

	// Validate parent ID if provided and prevent circular references
	if c.ParentID != nil {
		if err := validation.ValidateUUID(*c.ParentID); err != nil {
			return fmt.Errorf("invalid parent ID: %w", err)
		}

		// Prevent self-reference
		if *c.ParentID == c.ID {
			return errors.New("category cannot be its own parent")
		}

		// Check for circular reference
		if err := r.checkCircularReference(ctx, c.ID, *c.ParentID); err != nil {
			return fmt.Errorf("circular reference detected: %w", err)
		}
	}

	// Update timestamp
	c.UpdatedAt = time.Now()

	query := `
		UPDATE categories
		SET name = ?, type = ?, parent_id = ?, updated_at = ?
		WHERE id = ? AND family_id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query,
		c.Name,
		string(c.Type),
		sqlitehelpers.UUIDPtrToString(c.ParentID),
		c.UpdatedAt,
		sqlitehelpers.UUIDToString(c.ID),
		sqlitehelpers.UUIDToString(c.FamilyID),
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("category with name '%s' already exists in this family", c.Name)
		}
		return fmt.Errorf("failed to update category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category with id %s not found", c.ID)
	}

	return nil
}

// Delete soft deletes a category (sets is_active to false)
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Check if category has children
	hasChildren, err := r.hasChildren(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check for children: %w", err)
	}
	if hasChildren {
		return errors.New("cannot delete category with subcategories")
	}

	query := `
		UPDATE categories
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND family_id = ? AND is_active = 1`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category with id %s not found", id)
	}

	return nil
}

// validateParentCategory validates that parent category exists and is compatible
func (r *SQLiteRepository) validateParentCategory(
	ctx context.Context,
	parentID, familyID uuid.UUID,
	categoryType category.Type,
) error {
	query := `
		SELECT type, family_id, is_active
		FROM categories
		WHERE id = ?`

	var parentType, parentFamilyIDStr string
	var isActiveInt int

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(parentID)).
		Scan(&parentType, &parentFamilyIDStr, &isActiveInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("parent category not found")
		}
		return fmt.Errorf("failed to validate parent category: %w", err)
	}

	isActive := sqlitehelpers.IntToBool(isActiveInt)
	if !isActive {
		return errors.New("parent category is not active")
	}

	parentFamilyID, _ := uuid.Parse(parentFamilyIDStr)
	if parentFamilyID != familyID {
		return errors.New("parent category belongs to different family")
	}

	if category.Type(parentType) != categoryType {
		return errors.New("parent category has different type")
	}

	return nil
}

// checkCircularReference checks for circular references in category hierarchy
func (r *SQLiteRepository) checkCircularReference(ctx context.Context, categoryID, parentID uuid.UUID) error {
	query := `
		WITH RECURSIVE category_path AS (
			SELECT id, parent_id, 0 as level
			FROM categories
			WHERE id = ? AND is_active = 1

			UNION ALL

			SELECT c.id, c.parent_id, cp.level + 1
			FROM categories c
			INNER JOIN category_path cp ON c.id = cp.parent_id
			WHERE c.is_active = 1 AND cp.level < 10 -- Prevent infinite recursion
		)
		SELECT EXISTS(SELECT 1 FROM category_path WHERE id = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(parentID), sqlitehelpers.UUIDToString(categoryID)).
		Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check circular reference: %w", err)
	}

	if exists {
		return errors.New("circular reference would be created")
	}

	return nil
}

// hasChildren checks if a category has any child categories
func (r *SQLiteRepository) hasChildren(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM categories
			WHERE parent_id = ? AND is_active = 1
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(categoryID)).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check for children: %w", err)
	}

	return exists, nil
}

// GetByType возвращает категории по типу для совместимости с интерфейсом
func (r *SQLiteRepository) GetByType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.Type,
) ([]*category.Category, error) {
	return r.GetByFamilyIDAndType(ctx, familyID, categoryType)
}

package category

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/infrastructure/validation"
)

// PostgreSQLRepository implements category repository using PostgreSQL
type PostgreSQLRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLRepository creates a new PostgreSQL category repository
func NewPostgreSQLRepository(db *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// ValidateCategoryName validates category name
func ValidateCategoryName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("category name cannot be empty")
	}
	if len(name) > 255 {
		return errors.New("category name too long")
	}
	return nil
}

// ValidateCategoryType validates category type
func ValidateCategoryType(categoryType category.Type) error {
	if categoryType != category.TypeIncome && categoryType != category.TypeExpense {
		return errors.New("invalid category type")
	}
	return nil
}

// Create creates a new category in the database
func (r *PostgreSQLRepository) Create(ctx context.Context, c *category.Category) error {
	// Validate category parameters before creating
	if err := validation.ValidateUUID(c.ID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(c.FamilyID); err != nil {
		return fmt.Errorf("invalid category familyID: %w", err)
	}
	if err := ValidateCategoryType(c.Type); err != nil {
		return fmt.Errorf("invalid category type: %w", err)
	}
	if err := ValidateCategoryName(c.Name); err != nil {
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
		INSERT INTO family_budget.categories (
			id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		c.ID, c.Name, string(c.Type), "", c.ParentID, c.FamilyID,
		c.IsActive, c.CreatedAt, c.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("category with name '%s' already exists in this family", c.Name)
		}
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// GetByID retrieves a category by their ID
func (r *PostgreSQLRepository) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM family_budget.categories
		WHERE id = $1`

	var c category.Category
	var typeStr string
	var description *string

	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &typeStr, &description, &c.ParentID, &c.FamilyID,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("category with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}

	c.Type = category.Type(typeStr)
	return &c, nil
}

// GetByFamilyID retrieves all categories belonging to a specific family
func (r *PostgreSQLRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM family_budget.categories
		WHERE family_id = $1 AND is_active = true
		ORDER BY type, parent_id NULLS FIRST, name`

	rows, err := r.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by family id: %w", err)
	}
	defer rows.Close()

	var categories []*category.Category
	for rows.Next() {
		var c category.Category
		var typeStr string
		var description *string

		err = rows.Scan(
			&c.ID, &c.Name, &typeStr, &description, &c.ParentID, &c.FamilyID,
			&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		c.Type = category.Type(typeStr)
		categories = append(categories, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return categories, nil
}

// GetByFamilyIDAndType retrieves categories by family ID and type
func (r *PostgreSQLRepository) GetByFamilyIDAndType(ctx context.Context, familyID uuid.UUID, categoryType category.Type) ([]*category.Category, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}
	if err := ValidateCategoryType(categoryType); err != nil {
		return nil, fmt.Errorf("invalid category type: %w", err)
	}

	query := `
		SELECT id, name, type, description, parent_id, family_id, is_active, created_at, updated_at
		FROM family_budget.categories
		WHERE family_id = $1 AND type = $2 AND is_active = true
		ORDER BY parent_id NULLS FIRST, name`

	rows, err := r.db.Query(ctx, query, familyID, string(categoryType))
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by family and type: %w", err)
	}
	defer rows.Close()

	var categories []*category.Category
	for rows.Next() {
		var c category.Category
		var typeStr string
		var description *string

		err = rows.Scan(
			&c.ID, &c.Name, &typeStr, &description, &c.ParentID, &c.FamilyID,
			&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		c.Type = category.Type(typeStr)
		categories = append(categories, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return categories, nil
}

// GetCategoryChildren returns all children of a category using recursive CTE
func (r *PostgreSQLRepository) GetCategoryChildren(ctx context.Context, parentID uuid.UUID) ([]*CategoryNode, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(parentID); err != nil {
		return nil, fmt.Errorf("invalid parentID parameter: %w", err)
	}

	query := `
		WITH RECURSIVE category_tree AS (
			-- Base case: start with the parent category
			SELECT c.id, c.name, c.type, c.parent_id, c.family_id, 0 as level
			FROM family_budget.categories c
			WHERE c.id = $1 AND c.is_active = true

			UNION ALL

			-- Recursive case: find children
			SELECT c.id, c.name, c.type, c.parent_id, c.family_id, ct.level + 1
			FROM family_budget.categories c
			INNER JOIN category_tree ct ON c.parent_id = ct.id
			WHERE c.is_active = true
		)
		SELECT id, name, type, level
		FROM category_tree
		ORDER BY level, name`

	rows, err := r.db.Query(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category children: %w", err)
	}
	defer rows.Close()

	var nodes []*CategoryNode
	for rows.Next() {
		var node CategoryNode
		var typeStr string

		err = rows.Scan(&node.ID, &node.Name, &typeStr, &node.Level)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category node: %w", err)
		}

		node.Type = category.Type(typeStr)
		nodes = append(nodes, &node)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return nodes, nil
}

// GetCategoryPath returns the path from root to a specific category
func (r *PostgreSQLRepository) GetCategoryPath(ctx context.Context, categoryID uuid.UUID) ([]*CategoryNode, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(categoryID); err != nil {
		return nil, fmt.Errorf("invalid categoryID parameter: %w", err)
	}

	query := `
		WITH RECURSIVE category_path AS (
			-- Base case: start with the target category
			SELECT c.id, c.name, c.type, c.parent_id, 0 as level
			FROM family_budget.categories c
			WHERE c.id = $1 AND c.is_active = true

			UNION ALL

			-- Recursive case: go up to parents
			SELECT c.id, c.name, c.type, c.parent_id, cp.level + 1
			FROM family_budget.categories c
			INNER JOIN category_path cp ON c.id = cp.parent_id
			WHERE c.is_active = true
		)
		SELECT id, name, type, level
		FROM category_path
		ORDER BY level DESC`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category path: %w", err)
	}
	defer rows.Close()

	var nodes []*CategoryNode
	for rows.Next() {
		var node CategoryNode
		var typeStr string

		err = rows.Scan(&node.ID, &node.Name, &typeStr, &node.Level)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category node: %w", err)
		}

		node.Type = category.Type(typeStr)
		nodes = append(nodes, &node)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return nodes, nil
}

// Update updates an existing category
func (r *PostgreSQLRepository) Update(ctx context.Context, c *category.Category) error {
	// Validate category parameters
	if err := validation.ValidateUUID(c.ID); err != nil {
		return fmt.Errorf("invalid category ID: %w", err)
	}
	if err := validation.ValidateUUID(c.FamilyID); err != nil {
		return fmt.Errorf("invalid category familyID: %w", err)
	}
	if err := ValidateCategoryType(c.Type); err != nil {
		return fmt.Errorf("invalid category type: %w", err)
	}
	if err := ValidateCategoryName(c.Name); err != nil {
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
		UPDATE family_budget.categories
		SET name = $2, type = $3, parent_id = $4, updated_at = $5
		WHERE id = $1 AND family_id = $6 AND is_active = true`

	result, err := r.db.Exec(ctx, query,
		c.ID, c.Name, string(c.Type), c.ParentID, c.UpdatedAt, c.FamilyID,
	)

	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("category with name '%s' already exists in this family", c.Name)
		}
		return fmt.Errorf("failed to update category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("category with id %s not found", c.ID)
	}

	return nil
}

// Delete soft deletes a category (sets is_active to false)
func (r *PostgreSQLRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
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
		UPDATE family_budget.categories
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND family_id = $2 AND is_active = true`

	result, err := r.db.Exec(ctx, query, id, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("category with id %s not found", id)
	}

	return nil
}

// validateParentCategory validates that parent category exists and is compatible
func (r *PostgreSQLRepository) validateParentCategory(ctx context.Context, parentID, familyID uuid.UUID, categoryType category.Type) error {
	query := `
		SELECT type, family_id, is_active
		FROM family_budget.categories
		WHERE id = $1`

	var parentType string
	var parentFamilyID uuid.UUID
	var isActive bool

	err := r.db.QueryRow(ctx, query, parentID).Scan(&parentType, &parentFamilyID, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("parent category not found")
		}
		return fmt.Errorf("failed to validate parent category: %w", err)
	}

	if !isActive {
		return errors.New("parent category is not active")
	}

	if parentFamilyID != familyID {
		return errors.New("parent category belongs to different family")
	}

	if category.Type(parentType) != categoryType {
		return errors.New("parent category has different type")
	}

	return nil
}

// checkCircularReference checks for circular references in category hierarchy
func (r *PostgreSQLRepository) checkCircularReference(ctx context.Context, categoryID, parentID uuid.UUID) error {
	query := `
		WITH RECURSIVE category_path AS (
			SELECT id, parent_id, 0 as level
			FROM family_budget.categories
			WHERE id = $1 AND is_active = true

			UNION ALL

			SELECT c.id, c.parent_id, cp.level + 1
			FROM family_budget.categories c
			INNER JOIN category_path cp ON c.id = cp.parent_id
			WHERE c.is_active = true AND cp.level < 10 -- Prevent infinite recursion
		)
		SELECT EXISTS(SELECT 1 FROM category_path WHERE id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, parentID, categoryID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check circular reference: %w", err)
	}

	if exists {
		return errors.New("circular reference would be created")
	}

	return nil
}

// hasChildren checks if a category has any child categories
func (r *PostgreSQLRepository) hasChildren(ctx context.Context, categoryID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM family_budget.categories
			WHERE parent_id = $1 AND is_active = true
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, categoryID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check for children: %w", err)
	}

	return exists, nil
}

// CategoryNode represents a category in a hierarchical structure
type CategoryNode struct {
	ID    uuid.UUID     `json:"id"`
	Name  string        `json:"name"`
	Type  category.Type `json:"type"`
	Level int           `json:"level"`
}

// GetByType возвращает категории по типу для совместимости с интерфейсом
func (r *PostgreSQLRepository) GetByType(ctx context.Context, familyID uuid.UUID, categoryType category.Type) ([]*category.Category, error) {
	return r.GetByFamilyIDAndType(ctx, familyID, categoryType)
}

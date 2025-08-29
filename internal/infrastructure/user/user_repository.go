package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

const (
	// MaxEmailLength defines the maximum allowed length for email addresses (RFC 5321)
	MaxEmailLength = 254
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongo.Database) *Repository {
	return &Repository{
		collection: database.Collection("users"),
	}
}

// ValidateEmail performs comprehensive email validation to prevent injection attacks
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}

	// Trim whitespace and convert to lowercase for consistency
	email = strings.TrimSpace(strings.ToLower(email))

	// Check for MongoDB injection patterns and dangerous characters
	if strings.ContainsAny(email, "${}[]()\"'\\;") {
		return errors.New("email contains invalid characters")
	}

	// Check for control characters that could be used in attacks
	for _, char := range email {
		if char < 32 || char == 127 {
			return errors.New("email contains control characters")
		}
	}

	// Use Go's built-in email validation
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	// Additional length check to prevent excessively long emails
	if len(email) > MaxEmailLength {
		return errors.New("email too long")
	}

	// Ensure email doesn't start or end with potentially dangerous characters
	if strings.HasPrefix(email, ".") || strings.HasSuffix(email, ".") {
		return errors.New("email cannot start or end with a dot")
	}

	// Basic domain validation - must contain at least one dot after @
	atIndex := strings.LastIndex(email, "@")
	if atIndex == -1 || !strings.Contains(email[atIndex:], ".") {
		return errors.New("email must have a valid domain")
	}

	return nil
}

// SanitizeEmail safely prepares email for database query
func SanitizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func (r *Repository) Create(ctx context.Context, u *user.User) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(u.ID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateUUID(u.FamilyID); err != nil {
		return fmt.Errorf("invalid user familyID: %w", err)
	}

	// Validate email to prevent injection attacks
	if err := ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("invalid user email: %w", err)
	}

	// Sanitize email before storing
	u.Email = SanitizeEmail(u.Email)

	_, err := r.collection.InsertOne(ctx, u)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("user with email %s already exists", u.Email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	var u user.User
	err := r.collection.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	// Validate email to prevent injection attacks
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email parameter: %w", err)
	}

	// Sanitize email for consistent querying
	sanitizedEmail := SanitizeEmail(email)

	// Use explicit field matching with sanitized input
	filter := bson.D{
		{Key: "email", Value: sanitizedEmail},
	}

	var u user.User
	err := r.collection.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("user with email %s not found", sanitizedEmail)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*user.User, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "family_id", Value: familyID}}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by family id: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*user.User
	for cursor.Next(ctx) {
		var u user.User
		err = cursor.Decode(&u)
		if err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &u)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return users, nil
}

func (r *Repository) Update(ctx context.Context, u *user.User) error {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(u.ID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateUUID(u.FamilyID); err != nil {
		return fmt.Errorf("invalid user familyID: %w", err)
	}

	// Validate email to prevent injection attacks
	if err := ValidateEmail(u.Email); err != nil {
		return fmt.Errorf("invalid user email: %w", err)
	}

	// Sanitize email before updating
	u.Email = SanitizeEmail(u.Email)

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: u.ID}}
	update := bson.D{{Key: "$set", Value: u}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user with id %s not found", u.ID)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user with id %s not found", id)
	}

	return nil
}

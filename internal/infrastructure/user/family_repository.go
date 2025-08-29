package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/infrastructure/validation"
)

type FamilyRepository struct {
	collection *mongo.Collection
}

func NewFamilyRepository(database *mongo.Database) *FamilyRepository {
	return &FamilyRepository{
		collection: database.Collection("families"),
	}
}

func (r *FamilyRepository) Create(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before creating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	_, err := r.collection.InsertOne(ctx, family)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}
	return nil
}

func (r *FamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	var family user.Family
	err := r.collection.FindOne(ctx, filter).Decode(&family)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("family with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get family by id: %w", err)
	}
	return &family, nil
}

func (r *FamilyRepository) Update(ctx context.Context, family *user.Family) error {
	// Validate family ID parameter before updating
	if err := validation.ValidateUUID(family.ID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: family.ID}}
	update := bson.D{{Key: "$set", Value: family}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update family: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("family with id %s not found", family.ID)
	}

	return nil
}

func (r *FamilyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete family: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("family with id %s not found", id)
	}

	return nil
}

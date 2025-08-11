package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"family-budget-service/internal/domain/user"
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
	_, err := r.collection.InsertOne(ctx, family)
	if err != nil {
		return fmt.Errorf("failed to create family: %w", err)
	}
	return nil
}

func (r *FamilyRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.Family, error) {
	var family user.Family
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&family)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("family with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get family by id: %w", err)
	}
	return &family, nil
}

func (r *FamilyRepository) Update(ctx context.Context, family *user.Family) error {
	filter := bson.M{"_id": family.ID}
	update := bson.M{"$set": family}
	
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
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete family: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("family with id %s not found", id)
	}
	
	return nil
}
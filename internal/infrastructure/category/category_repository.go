package category

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/category"
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongo.Database) *Repository {
	return &Repository{
		collection: database.Collection("categories"),
	}
}

func (r *Repository) Create(ctx context.Context, c *category.Category) error {
	_, err := r.collection.InsertOne(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	var c category.Category
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("category with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get category by id: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*category.Category, error) {
	filter := bson.M{
		"family_id": familyID,
		"is_active": true,
	}

	opts := options.Find().SetSort(bson.M{"name": 1})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by family id: %w", err)
	}
	defer cursor.Close(ctx)

	var categories []*category.Category
	for cursor.Next(ctx) {
		var c category.Category
		err = cursor.Decode(&c)
		if err != nil {
			return nil, fmt.Errorf("failed to decode category: %w", err)
		}
		categories = append(categories, &c)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return categories, nil
}

func (r *Repository) GetByType(
	ctx context.Context,
	familyID uuid.UUID,
	categoryType category.Type,
) ([]*category.Category, error) {
	filter := bson.M{
		"family_id": familyID,
		"type":      categoryType,
		"is_active": true,
	}

	opts := options.Find().SetSort(bson.M{"name": 1})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories by type: %w", err)
	}
	defer cursor.Close(ctx)

	var categories []*category.Category
	for cursor.Next(ctx) {
		var c category.Category
		err = cursor.Decode(&c)
		if err != nil {
			return nil, fmt.Errorf("failed to decode category: %w", err)
		}
		categories = append(categories, &c)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return categories, nil
}

func (r *Repository) Update(ctx context.Context, c *category.Category) error {
	filter := bson.M{"_id": c.ID}
	update := bson.M{"$set": c}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("category with id %s not found", c.ID)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft delete - устанавливаем is_active в false
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"is_active": false}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("category with id %s not found", id)
	}

	return nil
}

package budget

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/budget"
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongo.Database) *Repository {
	return &Repository{
		collection: database.Collection("budgets"),
	}
}

func (r *Repository) Create(ctx context.Context, b *budget.Budget) error {
	_, err := r.collection.InsertOne(ctx, b)
	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	var b budget.Budget
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&b)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get budget by id: %w", err)
	}
	return &b, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	filter := bson.M{"family_id": familyID}
	opts := options.Find().SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by family id: %w", err)
	}
	defer cursor.Close(ctx)

	var budgets []*budget.Budget
	for cursor.Next(ctx) {
		var b budget.Budget
		err = cursor.Decode(&b)
		if err != nil {
			return nil, fmt.Errorf("failed to decode budget: %w", err)
		}
		budgets = append(budgets, &b)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return budgets, nil
}

func (r *Repository) GetActiveBudgets(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	now := time.Now()
	filter := bson.M{
		"family_id":  familyID,
		"is_active":  true,
		"start_date": bson.M{"$lte": now},
		"end_date":   bson.M{"$gte": now},
	}

	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get active budgets: %w", err)
	}
	defer cursor.Close(ctx)

	var budgets []*budget.Budget
	for cursor.Next(ctx) {
		var b budget.Budget
		err = cursor.Decode(&b)
		if err != nil {
			return nil, fmt.Errorf("failed to decode budget: %w", err)
		}
		budgets = append(budgets, &b)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return budgets, nil
}

func (r *Repository) Update(ctx context.Context, b *budget.Budget) error {
	filter := bson.M{"_id": b.ID}
	update := bson.M{"$set": b}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("budget with id %s not found", b.ID)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("budget with id %s not found", id)
	}

	return nil
}

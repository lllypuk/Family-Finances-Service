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
	"family-budget-service/internal/infrastructure/validation"
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
	// Validate budget parameters before creating
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid budget familyID: %w", err)
	}
	if b.CategoryID != nil {
		if err := validation.ValidateUUID(*b.CategoryID); err != nil {
			return fmt.Errorf("invalid budget categoryID: %w", err)
		}
	}

	_, err := r.collection.InsertOne(ctx, b)
	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*budget.Budget, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	var b budget.Budget
	err := r.collection.FindOne(ctx, filter).Decode(&b)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("budget with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get budget by id: %w", err)
	}
	return &b, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*budget.Budget, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "family_id", Value: familyID}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

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
	// Validate budget parameters before updating
	if err := validation.ValidateUUID(b.ID); err != nil {
		return fmt.Errorf("invalid budget ID: %w", err)
	}
	if err := validation.ValidateUUID(b.FamilyID); err != nil {
		return fmt.Errorf("invalid budget familyID: %w", err)
	}
	if b.CategoryID != nil {
		if err := validation.ValidateUUID(*b.CategoryID); err != nil {
			return fmt.Errorf("invalid budget categoryID: %w", err)
		}
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: b.ID}}
	update := bson.D{{Key: "$set", Value: b}}

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
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("budget with id %s not found", id)
	}

	return nil
}

func (r *Repository) GetByFamilyAndCategory(
	ctx context.Context,
	familyID uuid.UUID,
	categoryID *uuid.UUID,
) ([]*budget.Budget, error) {
	// Validate UUID parameters to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}
	if categoryID != nil {
		if err := validation.ValidateUUID(*categoryID); err != nil {
			return nil, fmt.Errorf("invalid categoryID parameter: %w", err)
		}
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "family_id", Value: familyID}}

	if categoryID != nil {
		filter = append(filter, bson.E{Key: "category_id", Value: *categoryID})
	} else {
		// Find budgets without a specific category (family-wide budgets)
		filter = append(filter, bson.E{Key: "category_id", Value: bson.D{{Key: "$exists", Value: false}}})
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by family and category: %w", err)
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

func (r *Repository) GetByPeriod(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) ([]*budget.Budget, error) {
	// Find budgets that overlap with the specified period
	filter := bson.M{
		"family_id": familyID,
		"$or": []bson.M{
			{
				// Budget starts within the period
				"start_date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
			{
				// Budget ends within the period
				"end_date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
			{
				// Budget contains the entire period
				"start_date": bson.M{"$lte": startDate},
				"end_date":   bson.M{"$gte": endDate},
			},
		},
	}

	opts := options.Find().SetSort(bson.M{"start_date": 1, "created_at": -1})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets by period: %w", err)
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

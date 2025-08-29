package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/infrastructure/validation"
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongo.Database) *Repository {
	return &Repository{
		collection: database.Collection("transactions"),
	}
}

func (r *Repository) Create(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters before creating
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.FamilyID); err != nil {
		return fmt.Errorf("invalid transaction familyID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid transaction userID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid transaction categoryID: %w", err)
	}
	if err := validation.ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}

	_, err := r.collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	var t transaction.Transaction
	err := r.collection.FindOne(ctx, filter).Decode(&t)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("transaction with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get transaction by id: %w", err)
	}
	return &t, nil
}

func (r *Repository) GetByFilter(
	ctx context.Context,
	filter transaction.Filter,
) ([]*transaction.Transaction, error) {
	mongoFilter := r.buildFilterQuery(filter)

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "date", Value: -1}, {Key: "created_at", Value: -1}})

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}

	cursor, err := r.collection.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by filter: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*transaction.Transaction
	for cursor.Next(ctx) {
		var t transaction.Transaction
		err = cursor.Decode(&t)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return transactions, nil
}

func (r *Repository) GetByFamilyID(
	ctx context.Context,
	familyID uuid.UUID,
	limit, offset int,
) ([]*transaction.Transaction, error) {
	filter := bson.M{"family_id": familyID}

	opts := options.Find()
	opts.SetSort(bson.D{{Key: "date", Value: -1}, {Key: "created_at", Value: -1}})

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by family id: %w", err)
	}
	defer cursor.Close(ctx)

	var transactions []*transaction.Transaction
	for cursor.Next(ctx) {
		var t transaction.Transaction
		err = cursor.Decode(&t)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return transactions, nil
}

func (r *Repository) Update(ctx context.Context, t *transaction.Transaction) error {
	// Validate transaction parameters before updating
	if err := validation.ValidateUUID(t.ID); err != nil {
		return fmt.Errorf("invalid transaction ID: %w", err)
	}
	if err := validation.ValidateUUID(t.FamilyID); err != nil {
		return fmt.Errorf("invalid transaction familyID: %w", err)
	}
	if err := validation.ValidateUUID(t.UserID); err != nil {
		return fmt.Errorf("invalid transaction userID: %w", err)
	}
	if err := validation.ValidateUUID(t.CategoryID); err != nil {
		return fmt.Errorf("invalid transaction categoryID: %w", err)
	}
	if err := validation.ValidateTransactionType(t.Type); err != nil {
		return fmt.Errorf("invalid transaction type: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: t.ID}}
	update := bson.D{{Key: "$set", Value: t}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("transaction with id %s not found", t.ID)
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
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("transaction with id %s not found", id)
	}

	return nil
}

func (r *Repository) GetTotalByCategory(
	ctx context.Context,
	categoryID uuid.UUID,
	transactionType transaction.Type,
) (float64, error) {
	matchFilter := bson.M{
		"category_id": categoryID,
		"type":        transactionType,
	}

	return r.getTotalWithFilter(ctx, matchFilter, "category")
}

func (r *Repository) GetTotalByFamilyAndDateRange(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	matchFilter := bson.M{
		"family_id": familyID,
		"type":      transactionType,
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	return r.getTotalWithFilter(ctx, matchFilter, "family and date range")
}

func (r *Repository) GetTotalByCategoryAndDateRange(
	ctx context.Context,
	categoryID uuid.UUID,
	startDate, endDate time.Time,
	transactionType transaction.Type,
) (float64, error) {
	matchFilter := bson.M{
		"category_id": categoryID,
		"type":        transactionType,
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	return r.getTotalWithFilter(ctx, matchFilter, "category and date range")
}

// getTotalWithFilter is a helper function to reduce code duplication in total calculation methods
func (r *Repository) getTotalWithFilter(ctx context.Context, matchFilter bson.M, description string) (float64, error) {
	pipeline := []bson.M{
		{"$match": matchFilter},
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$amount"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate transactions by %s: %w", description, err)
	}
	defer cursor.Close(ctx)

	var result struct {
		Total float64 `bson:"total"`
	}

	if cursor.Next(ctx) {
		err = cursor.Decode(&result)
		if err != nil {
			return 0, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
	}

	return result.Total, nil
}

func (r *Repository) buildFilterQuery(filter transaction.Filter) bson.D {
	// Use explicit field specification to prevent injection
	query := bson.D{{Key: "family_id", Value: filter.FamilyID}}

	if filter.UserID != nil {
		query = append(query, bson.E{Key: "user_id", Value: *filter.UserID})
	}

	if filter.CategoryID != nil {
		query = append(query, bson.E{Key: "category_id", Value: *filter.CategoryID})
	}

	if filter.Type != nil {
		query = append(query, bson.E{Key: "type", Value: string(*filter.Type)})
	}

	if filter.DateFrom != nil || filter.DateTo != nil {
		dateFilter := bson.D{}
		if filter.DateFrom != nil {
			dateFilter = append(dateFilter, bson.E{Key: "$gte", Value: *filter.DateFrom})
		}
		if filter.DateTo != nil {
			dateFilter = append(dateFilter, bson.E{Key: "$lte", Value: *filter.DateTo})
		}
		query = append(query, bson.E{Key: "date", Value: dateFilter})
	}

	if filter.AmountFrom != nil || filter.AmountTo != nil {
		amountFilter := bson.D{}
		if filter.AmountFrom != nil {
			amountFilter = append(amountFilter, bson.E{Key: "$gte", Value: *filter.AmountFrom})
		}
		if filter.AmountTo != nil {
			amountFilter = append(amountFilter, bson.E{Key: "$lte", Value: *filter.AmountTo})
		}
		query = append(query, bson.E{Key: "amount", Value: amountFilter})
	}

	if filter.Description != "" {
		regexFilter := bson.D{
			{Key: "$regex", Value: filter.Description},
			{Key: "$options", Value: "i"},
		}
		query = append(query, bson.E{Key: "description", Value: regexFilter})
	}

	return query
}

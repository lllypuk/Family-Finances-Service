package transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/transaction"
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
	_, err := r.collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*transaction.Transaction, error) {
	var t transaction.Transaction
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
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
	filter transaction.TransactionFilter,
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
		if err := cursor.Decode(&t); err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}

	if err := cursor.Err(); err != nil {
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
		if err := cursor.Decode(&t); err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}
		transactions = append(transactions, &t)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return transactions, nil
}

func (r *Repository) Update(ctx context.Context, t *transaction.Transaction) error {
	filter := bson.M{"_id": t.ID}
	update := bson.M{"$set": t}

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
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
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
	transactionType transaction.TransactionType,
) (float64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"category_id": categoryID,
				"type":        transactionType,
			},
		},
		{
			"$group": bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$amount"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		Total float64 `bson:"total"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode aggregation result: %w", err)
		}
	}

	return result.Total, nil
}

func (r *Repository) buildFilterQuery(filter transaction.TransactionFilter) bson.M {
	query := bson.M{"family_id": filter.FamilyID}

	if filter.UserID != nil {
		query["user_id"] = *filter.UserID
	}

	if filter.CategoryID != nil {
		query["category_id"] = *filter.CategoryID
	}

	if filter.Type != nil {
		query["type"] = *filter.Type
	}

	if filter.DateFrom != nil || filter.DateTo != nil {
		dateFilter := bson.M{}
		if filter.DateFrom != nil {
			dateFilter["$gte"] = *filter.DateFrom
		}
		if filter.DateTo != nil {
			dateFilter["$lte"] = *filter.DateTo
		}
		query["date"] = dateFilter
	}

	if filter.AmountFrom != nil || filter.AmountTo != nil {
		amountFilter := bson.M{}
		if filter.AmountFrom != nil {
			amountFilter["$gte"] = *filter.AmountFrom
		}
		if filter.AmountTo != nil {
			amountFilter["$lte"] = *filter.AmountTo
		}
		query["amount"] = amountFilter
	}

	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}

	if filter.Description != "" {
		query["description"] = bson.M{
			"$regex":   filter.Description,
			"$options": "i", // case insensitive
		}
	}

	return query
}

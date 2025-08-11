package report

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/report"
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongo.Database) *Repository {
	return &Repository{
		collection: database.Collection("reports"),
	}
}

func (r *Repository) Create(ctx context.Context, rep *report.Report) error {
	_, err := r.collection.InsertOne(ctx, rep)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	var rep report.Report
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rep)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("report with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get report by id: %w", err)
	}
	return &rep, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	filter := bson.M{"family_id": familyID}
	opts := options.Find().SetSort(bson.M{"generated_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports by family id: %w", err)
	}
	defer cursor.Close(ctx)

	var reports []*report.Report
	for cursor.Next(ctx) {
		var rep report.Report
		if err := cursor.Decode(&rep); err != nil {
			return nil, fmt.Errorf("failed to decode report: %w", err)
		}
		reports = append(reports, &rep)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return reports, nil
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	filter := bson.M{"user_id": userID}
	opts := options.Find().SetSort(bson.M{"generated_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports by user id: %w", err)
	}
	defer cursor.Close(ctx)

	var reports []*report.Report
	for cursor.Next(ctx) {
		var rep report.Report
		if err := cursor.Decode(&rep); err != nil {
			return nil, fmt.Errorf("failed to decode report: %w", err)
		}
		reports = append(reports, &rep)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return reports, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("report with id %s not found", id)
	}
	
	return nil
}
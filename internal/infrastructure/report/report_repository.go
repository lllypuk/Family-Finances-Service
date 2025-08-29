package report

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/infrastructure/validation"
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
	// Validate report parameters before creating
	if err := validation.ValidateUUID(rep.ID); err != nil {
		return fmt.Errorf("invalid report ID: %w", err)
	}
	if err := validation.ValidateUUID(rep.FamilyID); err != nil {
		return fmt.Errorf("invalid report familyID: %w", err)
	}
	if err := validation.ValidateUUID(rep.UserID); err != nil {
		return fmt.Errorf("invalid report userID: %w", err)
	}

	_, err := r.collection.InsertOne(ctx, rep)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "_id", Value: id}}

	var rep report.Report
	err := r.collection.FindOne(ctx, filter).Decode(&rep)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("report with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get report by id: %w", err)
	}
	return &rep, nil
}

// getReportsByFilter is a helper function to get reports by filter with sorting
func (r *Repository) getReportsByFilter(ctx context.Context, filter bson.D, errorMsg string) ([]*report.Report, error) {
	opts := options.Find().SetSort(bson.D{{Key: "generated_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorMsg, err)
	}
	defer cursor.Close(ctx)

	var reports []*report.Report
	for cursor.Next(ctx) {
		var rep report.Report
		err = cursor.Decode(&rep)
		if err != nil {
			return nil, fmt.Errorf("failed to decode report: %w", err)
		}
		reports = append(reports, &rep)
	}

	err = cursor.Err()
	if err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return reports, nil
}

func (r *Repository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "family_id", Value: familyID}}
	return r.getReportsByFilter(ctx, filter, "failed to get reports by family id")
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter to prevent injection attacks
	if err := validation.ValidateUUID(userID); err != nil {
		return nil, fmt.Errorf("invalid userID parameter: %w", err)
	}

	// Use explicit field specification to prevent injection
	filter := bson.D{{Key: "user_id", Value: userID}}
	return r.getReportsByFilter(ctx, filter, "failed to get reports by user id")
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
		return fmt.Errorf("failed to delete report: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("report with id %s not found", id)
	}

	return nil
}

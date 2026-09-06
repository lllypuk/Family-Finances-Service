package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/infrastructure/sqlitehelpers"
	"family-budget-service/internal/infrastructure/validation"
)

// SQLiteRepository implements report repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// Summary holds report summary statistics
type Summary struct {
	FamilyID        uuid.UUID  `json:"family_id"`
	TotalReports    int        `json:"total_reports"`
	ExpenseReports  int        `json:"expense_reports"`
	IncomeReports   int        `json:"income_reports"`
	BudgetReports   int        `json:"budget_reports"`
	CashFlowReports int        `json:"cash_flow_reports"`
	LastGenerated   *time.Time `json:"last_generated,omitempty"`
}

// NewSQLiteRepository creates a new SQLite report repository
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

// getSingleFamilyID retrieves the ID of the single family from the database
func (r *SQLiteRepository) getSingleFamilyID(ctx context.Context) (uuid.UUID, error) {
	query := `SELECT id FROM families LIMIT 1`
	var idStr string
	err := r.db.QueryRowContext(ctx, query).Scan(&idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get family ID: %w", err)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse family ID: %w", err)
	}
	return id, nil
}

// scanReportRow scans a single row from SQL query into a Report struct
func scanReportRow(rows *sql.Rows) (*report.Report, error) {
	var rep report.Report
	var idStr, typeStr, periodStr, familyIDStr, userIDStr string
	var dataJSON string
	var generatedAtStr string

	err := rows.Scan(
		&idStr, &rep.Name, &typeStr, &periodStr,
		&rep.StartDate, &rep.EndDate, &dataJSON,
		&familyIDStr, &userIDStr, &generatedAtStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan report: %w", err)
	}

	// Parse UUID fields
	rep.ID, _ = uuid.Parse(idStr)
	// familyIDStr unused - single family model
	rep.UserID, _ = uuid.Parse(userIDStr)

	rep.Type = report.Type(typeStr)
	rep.Period = report.Period(periodStr)

	rep.GeneratedAt, _ = time.Parse(time.RFC3339, generatedAtStr)

	// Parse data from JSON
	if err = json.Unmarshal([]byte(dataJSON), &rep.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
	}

	return &rep, nil
}

// Create creates a new report in the database
func (r *SQLiteRepository) Create(ctx context.Context, rep *report.Report) error {
	// Validate report parameters
	if err := validation.ValidateUUID(rep.ID); err != nil {
		return fmt.Errorf("invalid report ID: %w", err)
	}

	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	if validationErr := validation.ValidateUUID(rep.UserID); validationErr != nil {
		return fmt.Errorf("invalid user ID: %w", validationErr)
	}
	if validationErr := validation.ValidateReportType(rep.Type); validationErr != nil {
		return fmt.Errorf("invalid report type: %w", validationErr)
	}
	if validationErr := validation.ValidateReportPeriod(rep.Period); validationErr != nil {
		return fmt.Errorf("invalid report period: %w", validationErr)
	}
	if validationErr := validation.ValidateReportName(rep.Name); validationErr != nil {
		return fmt.Errorf("invalid report name: %w", validationErr)
	}

	// Validate date range
	if rep.EndDate.Before(rep.StartDate) {
		return errors.New("end date must be after or equal to start date")
	}

	// Set generation timestamp
	rep.GeneratedAt = time.Now()

	// Convert data to JSON
	dataJSON, err := json.Marshal(rep.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	query := `
		INSERT INTO reports (
			id, name, type, period, start_date, end_date, data,
			family_id, generated_by, generated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		sqlitehelpers.UUIDToString(rep.ID),
		rep.Name,
		string(rep.Type),
		string(rep.Period),
		rep.StartDate,
		rep.EndDate,
		string(dataJSON),
		familyID.String(),
		sqlitehelpers.UUIDToString(rep.UserID),
		rep.GeneratedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

// GetByID retrieves a report by their ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, generated_by, generated_at
		FROM reports
		WHERE id = ?`

	var rep report.Report
	var idStr, typeStr, periodStr, familyIDStr, userIDStr string
	var dataJSON string
	var generatedAtStr string

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &rep.Name, &typeStr, &periodStr,
		&rep.StartDate, &rep.EndDate, &dataJSON,
		&familyIDStr, &userIDStr, &generatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("report with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get report by id: %w", err)
	}

	// Parse UUID fields
	rep.ID, _ = uuid.Parse(idStr)
	// familyIDStr unused - single family model
	rep.UserID, _ = uuid.Parse(userIDStr)

	rep.Type = report.Type(typeStr)
	rep.Period = report.Period(periodStr)

	rep.GeneratedAt, _ = time.Parse(time.RFC3339, generatedAtStr)

	// Parse data from JSON
	if err = json.Unmarshal([]byte(dataJSON), &rep.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
	}

	return &rep, nil
}

// GetAll retrieves all reports for the family
func (r *SQLiteRepository) GetAll(ctx context.Context) ([]*report.Report, error) {
	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get family ID: %w", err)
	}

	// Без LIMIT: окно режет вызывающий, а усечение здесь занижало бы
	// meta.pagination.total и делало недостижимой любую страницу после сотой.
	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, generated_by, generated_at
		FROM reports
		WHERE family_id = ?
		ORDER BY generated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID))
	if err != nil {
		return nil, fmt.Errorf("failed to get reports by family id: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep *report.Report
		rep, err = scanReportRow(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// GetByFamilyIDWithPagination retrieves reports by family ID with custom pagination
func (r *SQLiteRepository) GetByFamilyIDWithPagination(
	ctx context.Context,
	familyID uuid.UUID,
	limit, offset int,
) ([]*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, generated_by, generated_at
		FROM reports
		WHERE family_id = ?		ORDER BY generated_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports by family: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep *report.Report
		rep, err = scanReportRow(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// GetByUserID retrieves reports by user ID
func (r *SQLiteRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, generated_by, generated_at
		FROM reports
		WHERE generated_by = ?		ORDER BY generated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to query reports by user: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep *report.Report
		rep, err = scanReportRow(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// Delete deletes a report
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}

	// Get single family ID
	familyID, err := r.getSingleFamilyID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get family ID: %w", err)
	}

	query := `DELETE FROM reports WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), familyID.String())
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("report with id %s not found", id)
	}

	return nil
}

// GetSummary returns summary statistics for reports
func (r *SQLiteRepository) GetSummary(ctx context.Context, familyID uuid.UUID) (*Summary, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	query := `
		SELECT
			COUNT(*) as total_reports,
			COUNT(CASE WHEN type = 'expenses' THEN 1 END) as expense_reports,
			COUNT(CASE WHEN type = 'income' THEN 1 END) as income_reports,
			COUNT(CASE WHEN type = 'budget' THEN 1 END) as budget_reports,
			COUNT(CASE WHEN type = 'cash_flow' THEN 1 END) as cash_flow_reports,
			MAX(generated_at) as last_generated
		FROM reports
		WHERE family_id = ?`

	var summary Summary
	var lastGeneratedStr *string
	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(familyID)).Scan(
		&summary.TotalReports, &summary.ExpenseReports, &summary.IncomeReports,
		&summary.BudgetReports, &summary.CashFlowReports, &lastGeneratedStr,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get report summary: %w", err)
	}

	// Parse last_generated timestamp if present
	if lastGeneratedStr != nil && *lastGeneratedStr != "" {
		lastGenerated, parseErr := time.Parse(time.RFC3339, *lastGeneratedStr)
		if parseErr == nil {
			summary.LastGenerated = &lastGenerated
		}
	}

	summary.FamilyID = familyID
	return &summary, nil
}

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

const (
	percentageMultiplier = 100
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

// scanReportRow scans a single row from SQL query into a Report struct
func scanReportRow(rows *sql.Rows) (*report.Report, error) {
	var rep report.Report
	var idStr, typeStr, periodStr, familyIDStr, userIDStr string
	var dataJSON string

	err := rows.Scan(
		&idStr, &rep.Name, &typeStr, &periodStr,
		&rep.StartDate, &rep.EndDate, &dataJSON,
		&familyIDStr, &userIDStr, &rep.GeneratedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan report: %w", err)
	}

	// Parse UUID fields
	rep.ID, _ = uuid.Parse(idStr)
	rep.FamilyID, _ = uuid.Parse(familyIDStr)
	rep.UserID, _ = uuid.Parse(userIDStr)

	rep.Type = report.Type(typeStr)
	rep.Period = report.Period(periodStr)

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
	if err := validation.ValidateUUID(rep.FamilyID); err != nil {
		return fmt.Errorf("invalid family ID: %w", err)
	}
	if err := validation.ValidateUUID(rep.UserID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	if err := validation.ValidateReportType(rep.Type); err != nil {
		return fmt.Errorf("invalid report type: %w", err)
	}
	if err := validation.ValidateReportPeriod(rep.Period); err != nil {
		return fmt.Errorf("invalid report period: %w", err)
	}
	if err := validation.ValidateReportName(rep.Name); err != nil {
		return fmt.Errorf("invalid report name: %w", err)
	}

	// Validate date range
	if !rep.EndDate.After(rep.StartDate) && !rep.EndDate.Equal(rep.StartDate) {
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
		sqlitehelpers.UUIDToString(rep.FamilyID),
		sqlitehelpers.UUIDToString(rep.UserID),
		rep.GeneratedAt,
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

	err := r.db.QueryRowContext(ctx, query, sqlitehelpers.UUIDToString(id)).Scan(
		&idStr, &rep.Name, &typeStr, &periodStr,
		&rep.StartDate, &rep.EndDate, &dataJSON,
		&familyIDStr, &userIDStr, &rep.GeneratedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("report with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get report by id: %w", err)
	}

	// Parse UUID fields
	rep.ID, _ = uuid.Parse(idStr)
	rep.FamilyID, _ = uuid.Parse(familyIDStr)
	rep.UserID, _ = uuid.Parse(userIDStr)

	rep.Type = report.Type(typeStr)
	rep.Period = report.Period(periodStr)

	// Parse data from JSON
	if err = json.Unmarshal([]byte(dataJSON), &rep.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
	}

	return &rep, nil
}

// GetByFamilyID retrieves reports by family ID with pagination
func (r *SQLiteRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Set default pagination values
	limit := 100 // Default limit for reports
	offset := 0

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, generated_by, generated_at
		FROM reports
		WHERE family_id = ?
		ORDER BY generated_at DESC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, sqlitehelpers.UUIDToString(familyID), limit, offset)
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

// GenerateExpenseReport generates a comprehensive expense report
//
//nolint:funlen // Complex report generation requires multiple queries and data processing
func (r *SQLiteRepository) GenerateExpenseReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*report.Data, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	var data report.Data

	// Get total expenses
	expenseQuery := `
		SELECT COALESCE(SUM(amount), 0) as total_expenses
		FROM transactions
		WHERE family_id = ? AND type = 'expense'
		AND date BETWEEN ? AND ?`

	err := r.db.QueryRowContext(ctx, expenseQuery, sqlitehelpers.UUIDToString(familyID), startDate, endDate).
		Scan(&data.TotalExpenses)
	if err != nil {
		return nil, fmt.Errorf("failed to get total expenses: %w", err)
	}

	// Get category breakdown
	categoryQuery := `
		SELECT
			c.id, c.name,
			COALESCE(SUM(t.amount), 0) as total_amount,
			COUNT(t.id) as transaction_count
		FROM categories c
		LEFT JOIN transactions t ON c.id = t.category_id
			AND t.family_id = ? AND t.type = 'expense'
			AND t.date BETWEEN ? AND ?
		WHERE c.family_id = ? AND c.type = 'expense' AND c.is_active = 1
		GROUP BY c.id, c.name
		HAVING COALESCE(SUM(t.amount), 0) > 0
		ORDER BY total_amount DESC`

	rows, err := r.db.QueryContext(
		ctx,
		categoryQuery,
		sqlitehelpers.UUIDToString(familyID),
		startDate,
		endDate,
		sqlitehelpers.UUIDToString(familyID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}
	defer rows.Close()

	var categoryBreakdown []report.CategoryReportItem
	for rows.Next() {
		var item report.CategoryReportItem
		var categoryIDStr string
		err = rows.Scan(&categoryIDStr, &item.CategoryName, &item.Amount, &item.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category item: %w", err)
		}

		item.CategoryID, _ = uuid.Parse(categoryIDStr)

		// Calculate percentage
		if data.TotalExpenses > 0 {
			item.Percentage = (item.Amount / data.TotalExpenses) * percentageMultiplier
		}

		categoryBreakdown = append(categoryBreakdown, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("category breakdown rows iteration error: %w", err)
	}

	data.CategoryBreakdown = categoryBreakdown

	// Get top expenses
	topExpensesQuery := `
		SELECT t.id, t.amount, t.description, c.name, t.date
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.family_id = ? AND t.type = 'expense'
		AND t.date BETWEEN ? AND ?
		ORDER BY t.amount DESC
		LIMIT 10`

	topRows, err := r.db.QueryContext(ctx, topExpensesQuery, sqlitehelpers.UUIDToString(familyID), startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get top expenses: %w", err)
	}
	defer topRows.Close()

	var topExpenses []report.TransactionReportItem
	for topRows.Next() {
		var item report.TransactionReportItem
		var idStr string
		err = topRows.Scan(&idStr, &item.Amount, &item.Description, &item.Category, &item.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top expense item: %w", err)
		}
		item.ID, _ = uuid.Parse(idStr)
		topExpenses = append(topExpenses, item)
	}

	if err = topRows.Err(); err != nil {
		return nil, fmt.Errorf("top expenses rows iteration error: %w", err)
	}

	data.TopExpenses = topExpenses

	return &data, nil
}

// GenerateIncomeReport generates a comprehensive income report
func (r *SQLiteRepository) GenerateIncomeReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*report.Data, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	var data report.Data

	// Get total income
	incomeQuery := `
		SELECT COALESCE(SUM(amount), 0) as total_income
		FROM transactions
		WHERE family_id = ? AND type = 'income'
		AND date BETWEEN ? AND ?`

	err := r.db.QueryRowContext(ctx, incomeQuery, sqlitehelpers.UUIDToString(familyID), startDate, endDate).
		Scan(&data.TotalIncome)
	if err != nil {
		return nil, fmt.Errorf("failed to get total income: %w", err)
	}

	// Get category breakdown for income
	categoryQuery := `
		SELECT
			c.id, c.name,
			COALESCE(SUM(t.amount), 0) as total_amount,
			COUNT(t.id) as transaction_count
		FROM categories c
		LEFT JOIN transactions t ON c.id = t.category_id
			AND t.family_id = ? AND t.type = 'income'
			AND t.date BETWEEN ? AND ?
		WHERE c.family_id = ? AND c.type = 'income' AND c.is_active = 1
		GROUP BY c.id, c.name
		HAVING COALESCE(SUM(t.amount), 0) > 0
		ORDER BY total_amount DESC`

	rows, err := r.db.QueryContext(
		ctx,
		categoryQuery,
		sqlitehelpers.UUIDToString(familyID),
		startDate,
		endDate,
		sqlitehelpers.UUIDToString(familyID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get income category breakdown: %w", err)
	}
	defer rows.Close()

	var categoryBreakdown []report.CategoryReportItem
	for rows.Next() {
		var item report.CategoryReportItem
		var categoryIDStr string
		err = rows.Scan(&categoryIDStr, &item.CategoryName, &item.Amount, &item.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan income category item: %w", err)
		}

		item.CategoryID, _ = uuid.Parse(categoryIDStr)

		// Calculate percentage
		if data.TotalIncome > 0 {
			item.Percentage = (item.Amount / data.TotalIncome) * percentageMultiplier
		}

		categoryBreakdown = append(categoryBreakdown, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("income category breakdown rows iteration error: %w", err)
	}

	data.CategoryBreakdown = categoryBreakdown

	return &data, nil
}

// GenerateCashFlowReport generates a cash flow report with daily breakdown
func (r *SQLiteRepository) GenerateCashFlowReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*report.Data, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	var data report.Data

	// Get totals
	totalsQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as total_expenses
		FROM transactions
		WHERE family_id = ? AND date BETWEEN ? AND ?`

	err := r.db.QueryRowContext(ctx, totalsQuery, sqlitehelpers.UUIDToString(familyID), startDate, endDate).
		Scan(&data.TotalIncome, &data.TotalExpenses)
	if err != nil {
		return nil, fmt.Errorf("failed to get totals: %w", err)
	}

	data.NetIncome = data.TotalIncome - data.TotalExpenses

	// Get daily breakdown
	dailyQuery := `
		SELECT
			date,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) as daily_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) as daily_expenses
		FROM transactions
		WHERE family_id = ? AND date BETWEEN ? AND ?
		GROUP BY date
		ORDER BY date`

	rows, err := r.db.QueryContext(ctx, dailyQuery, sqlitehelpers.UUIDToString(familyID), startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily breakdown: %w", err)
	}
	defer rows.Close()

	var dailyBreakdown []report.DailyReportItem
	for rows.Next() {
		var item report.DailyReportItem
		err = rows.Scan(&item.Date, &item.Income, &item.Expenses)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily item: %w", err)
		}
		item.Balance = item.Income - item.Expenses
		dailyBreakdown = append(dailyBreakdown, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("daily breakdown rows iteration error: %w", err)
	}

	data.DailyBreakdown = dailyBreakdown

	return &data, nil
}

// GenerateBudgetComparisonReport generates budget vs actual spending comparison
func (r *SQLiteRepository) GenerateBudgetComparisonReport(
	ctx context.Context,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) (*report.Data, error) {
	// Validate parameters
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid family ID: %w", err)
	}

	var data report.Data

	// Get budget comparison
	budgetQuery := `
		SELECT
			b.id,
			b.name,
			b.amount as planned,
			b.spent as actual
		FROM budgets b
		WHERE b.family_id = ? AND b.is_active = 1
		AND (
			(b.start_date <= ? AND b.end_date >= ?) OR
			(b.start_date <= ? AND b.end_date >= ?) OR
			(b.start_date >= ? AND b.end_date <= ?)
		)
		ORDER BY b.name`

	rows, err := r.db.QueryContext(
		ctx,
		budgetQuery,
		sqlitehelpers.UUIDToString(familyID),
		startDate,
		startDate,
		endDate,
		endDate,
		startDate,
		endDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget comparison: %w", err)
	}
	defer rows.Close()

	var budgetComparison []report.BudgetComparisonItem
	for rows.Next() {
		var item report.BudgetComparisonItem
		var budgetIDStr string
		err = rows.Scan(&budgetIDStr, &item.BudgetName, &item.Planned, &item.Actual)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget comparison item: %w", err)
		}

		item.BudgetID, _ = uuid.Parse(budgetIDStr)
		item.Difference = item.Actual - item.Planned
		if item.Planned > 0 {
			item.Percentage = (item.Actual / item.Planned) * percentageMultiplier
		}

		budgetComparison = append(budgetComparison, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("budget comparison rows iteration error: %w", err)
	}

	data.BudgetComparison = budgetComparison

	return &data, nil
}

// Delete deletes a report
func (r *SQLiteRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid family ID parameter: %w", err)
	}

	query := `DELETE FROM reports WHERE id = ? AND family_id = ?`

	result, err := r.db.ExecContext(ctx, query, sqlitehelpers.UUIDToString(id), sqlitehelpers.UUIDToString(familyID))
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

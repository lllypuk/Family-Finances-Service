package report

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/infrastructure/validation"
)

// PostgreSQLRepository implements report repository using PostgreSQL
type PostgreSQLRepository struct {
	db *pgxpool.Pool
}

// NewPostgreSQLRepository creates a new PostgreSQL report repository
func NewPostgreSQLRepository(db *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// ValidateReportType validates report type
func ValidateReportType(reportType report.Type) error {
	switch reportType {
	case report.TypeExpenses, report.TypeIncome, report.TypeBudget, report.TypeCashFlow, report.TypeCategoryBreak:
		return nil
	default:
		return errors.New("invalid report type")
	}
}

// ValidateReportPeriod validates report period
func ValidateReportPeriod(period report.Period) error {
	switch period {
	case report.PeriodDaily, report.PeriodWeekly, report.PeriodMonthly, report.PeriodYearly, report.PeriodCustom:
		return nil
	default:
		return errors.New("invalid report period")
	}
}

// ValidateReportName validates report name
func ValidateReportName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("report name cannot be empty")
	}
	if len(name) > 255 {
		return errors.New("report name too long")
	}
	return nil
}

// Create creates a new report in the database
func (r *PostgreSQLRepository) Create(ctx context.Context, rep *report.Report) error {
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
	if err := ValidateReportType(rep.Type); err != nil {
		return fmt.Errorf("invalid report type: %w", err)
	}
	if err := ValidateReportPeriod(rep.Period); err != nil {
		return fmt.Errorf("invalid report period: %w", err)
	}
	if err := ValidateReportName(rep.Name); err != nil {
		return fmt.Errorf("invalid report name: %w", err)
	}

	// Validate date range
	if !rep.EndDate.After(rep.StartDate) && !rep.EndDate.Equal(rep.StartDate) {
		return errors.New("end date must be after or equal to start date")
	}

	// Set generation timestamp
	rep.GeneratedAt = time.Now()

	// Convert data to JSONB
	dataJSON, err := json.Marshal(rep.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal report data: %w", err)
	}

	query := `
		INSERT INTO family_budget.reports (
			id, name, type, period, start_date, end_date, data,
			family_id, user_id, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = r.db.Exec(ctx, query,
		rep.ID, rep.Name, string(rep.Type), string(rep.Period),
		rep.StartDate, rep.EndDate, dataJSON, rep.FamilyID,
		rep.UserID, rep.GeneratedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

// GetByID retrieves a report by their ID
func (r *PostgreSQLRepository) GetByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(id); err != nil {
		return nil, fmt.Errorf("invalid id parameter: %w", err)
	}

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, user_id, generated_at
		FROM family_budget.reports
		WHERE id = $1`

	var rep report.Report
	var typeStr, periodStr string
	var dataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&rep.ID, &rep.Name, &typeStr, &periodStr,
		&rep.StartDate, &rep.EndDate, &dataJSON,
		&rep.FamilyID, &rep.UserID, &rep.GeneratedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("report with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get report by id: %w", err)
	}

	rep.Type = report.Type(typeStr)
	rep.Period = report.Period(periodStr)

	// Parse data from JSONB
	if err := json.Unmarshal(dataJSON, &rep.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
	}

	return &rep, nil
}

// GetByFamilyID retrieves reports by family ID with pagination
func (r *PostgreSQLRepository) GetByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(familyID); err != nil {
		return nil, fmt.Errorf("invalid familyID parameter: %w", err)
	}

	// Set default pagination values
	limit := 100 // Default limit for reports
	offset := 0

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, user_id, generated_at
		FROM family_budget.reports
		WHERE family_id = $1
		ORDER BY generated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, familyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports by family id: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep report.Report
		var typeStr, periodStr string
		var dataJSON []byte

		err = rows.Scan(
			&rep.ID, &rep.Name, &typeStr, &periodStr,
			&rep.StartDate, &rep.EndDate, &dataJSON,
			&rep.FamilyID, &rep.UserID, &rep.GeneratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		rep.Type = report.Type(typeStr)
		rep.Period = report.Period(periodStr)

		// Parse data from JSONB
		if err := json.Unmarshal(dataJSON, &rep.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
		}

		reports = append(reports, &rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// GetByFamilyIDWithPagination retrieves reports by family ID with custom pagination
func (r *PostgreSQLRepository) GetByFamilyIDWithPagination(
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
			   family_id, user_id, generated_at
		FROM family_budget.reports
		WHERE family_id = $1 AND deleted_at IS NULL
		ORDER BY generated_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, familyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports by family: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep report.Report
		var dataJSON []byte
		var periodStr string

		err := rows.Scan(
			&rep.ID,
			&rep.Name,
			&rep.Type,
			&periodStr,
			&rep.StartDate,
			&rep.EndDate,
			&dataJSON,
			&rep.FamilyID,
			&rep.UserID,
			&rep.GeneratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		// Convert period string to enum
		rep.Period = report.Period(periodStr)

		// Parse data from JSONB
		if err := json.Unmarshal(dataJSON, &rep.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
		}

		reports = append(reports, &rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// GetByUserID retrieves reports by user ID
func (r *PostgreSQLRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	// Validate UUID parameter
	if err := validation.ValidateUUID(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT id, name, type, period, start_date, end_date, data,
			   family_id, user_id, generated_at
		FROM family_budget.reports
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY generated_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports by user: %w", err)
	}
	defer rows.Close()

	var reports []*report.Report
	for rows.Next() {
		var rep report.Report
		var dataJSON []byte
		var periodStr string

		err := rows.Scan(
			&rep.ID,
			&rep.Name,
			&rep.Type,
			&periodStr,
			&rep.StartDate,
			&rep.EndDate,
			&dataJSON,
			&rep.FamilyID,
			&rep.UserID,
			&rep.GeneratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}

		// Convert period string to enum
		rep.Period = report.Period(periodStr)

		// Parse data from JSONB
		if err := json.Unmarshal(dataJSON, &rep.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report data: %w", err)
		}

		reports = append(reports, &rep)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return reports, nil
}

// GenerateExpenseReport generates a comprehensive expense report
func (r *PostgreSQLRepository) GenerateExpenseReport(
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
		FROM family_budget.transactions
		WHERE family_id = $1 AND type = 'expense'
		AND date BETWEEN $2 AND $3`

	err := r.db.QueryRow(ctx, expenseQuery, familyID, startDate, endDate).Scan(&data.TotalExpenses)
	if err != nil {
		return nil, fmt.Errorf("failed to get total expenses: %w", err)
	}

	// Get category breakdown
	categoryQuery := `
		SELECT
			c.id, c.name,
			COALESCE(SUM(t.amount), 0) as total_amount,
			COUNT(t.id) as transaction_count
		FROM family_budget.categories c
		LEFT JOIN family_budget.transactions t ON c.id = t.category_id
			AND t.family_id = $1 AND t.type = 'expense'
			AND t.date BETWEEN $2 AND $3
		WHERE c.family_id = $1 AND c.type = 'expense' AND c.is_active = true
		GROUP BY c.id, c.name
		HAVING COALESCE(SUM(t.amount), 0) > 0
		ORDER BY total_amount DESC`

	rows, err := r.db.Query(ctx, categoryQuery, familyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get category breakdown: %w", err)
	}
	defer rows.Close()

	var categoryBreakdown []report.CategoryReportItem
	for rows.Next() {
		var item report.CategoryReportItem
		err = rows.Scan(&item.CategoryID, &item.CategoryName, &item.Amount, &item.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category item: %w", err)
		}

		// Calculate percentage
		if data.TotalExpenses > 0 {
			item.Percentage = (item.Amount / data.TotalExpenses) * 100
		}

		categoryBreakdown = append(categoryBreakdown, item)
	}
	data.CategoryBreakdown = categoryBreakdown

	// Get top expenses
	topExpensesQuery := `
		SELECT t.id, t.amount, t.description, c.name, t.date
		FROM family_budget.transactions t
		JOIN family_budget.categories c ON t.category_id = c.id
		WHERE t.family_id = $1 AND t.type = 'expense'
		AND t.date BETWEEN $2 AND $3
		ORDER BY t.amount DESC
		LIMIT 10`

	topRows, err := r.db.Query(ctx, topExpensesQuery, familyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get top expenses: %w", err)
	}
	defer topRows.Close()

	var topExpenses []report.TransactionReportItem
	for topRows.Next() {
		var item report.TransactionReportItem
		err = topRows.Scan(&item.ID, &item.Amount, &item.Description, &item.Category, &item.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top expense item: %w", err)
		}
		topExpenses = append(topExpenses, item)
	}
	data.TopExpenses = topExpenses

	return &data, nil
}

// GenerateIncomeReport generates a comprehensive income report
func (r *PostgreSQLRepository) GenerateIncomeReport(
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
		FROM family_budget.transactions
		WHERE family_id = $1 AND type = 'income'
		AND date BETWEEN $2 AND $3`

	err := r.db.QueryRow(ctx, incomeQuery, familyID, startDate, endDate).Scan(&data.TotalIncome)
	if err != nil {
		return nil, fmt.Errorf("failed to get total income: %w", err)
	}

	// Get category breakdown for income
	categoryQuery := `
		SELECT
			c.id, c.name,
			COALESCE(SUM(t.amount), 0) as total_amount,
			COUNT(t.id) as transaction_count
		FROM family_budget.categories c
		LEFT JOIN family_budget.transactions t ON c.id = t.category_id
			AND t.family_id = $1 AND t.type = 'income'
			AND t.date BETWEEN $2 AND $3
		WHERE c.family_id = $1 AND c.type = 'income' AND c.is_active = true
		GROUP BY c.id, c.name
		HAVING COALESCE(SUM(t.amount), 0) > 0
		ORDER BY total_amount DESC`

	rows, err := r.db.Query(ctx, categoryQuery, familyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get income category breakdown: %w", err)
	}
	defer rows.Close()

	var categoryBreakdown []report.CategoryReportItem
	for rows.Next() {
		var item report.CategoryReportItem
		err = rows.Scan(&item.CategoryID, &item.CategoryName, &item.Amount, &item.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan income category item: %w", err)
		}

		// Calculate percentage
		if data.TotalIncome > 0 {
			item.Percentage = (item.Amount / data.TotalIncome) * 100
		}

		categoryBreakdown = append(categoryBreakdown, item)
	}
	data.CategoryBreakdown = categoryBreakdown

	return &data, nil
}

// GenerateCashFlowReport generates a cash flow report with daily breakdown
func (r *PostgreSQLRepository) GenerateCashFlowReport(
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
		FROM family_budget.transactions
		WHERE family_id = $1 AND date BETWEEN $2 AND $3`

	err := r.db.QueryRow(ctx, totalsQuery, familyID, startDate, endDate).Scan(&data.TotalIncome, &data.TotalExpenses)
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
		FROM family_budget.transactions
		WHERE family_id = $1 AND date BETWEEN $2 AND $3
		GROUP BY date
		ORDER BY date`

	rows, err := r.db.Query(ctx, dailyQuery, familyID, startDate, endDate)
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
	data.DailyBreakdown = dailyBreakdown

	return &data, nil
}

// GenerateBudgetComparisonReport generates budget vs actual spending comparison
func (r *PostgreSQLRepository) GenerateBudgetComparisonReport(
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
		FROM family_budget.budgets b
		WHERE b.family_id = $1 AND b.is_active = true
		AND (
			(b.start_date <= $2 AND b.end_date >= $2) OR
			(b.start_date <= $3 AND b.end_date >= $3) OR
			(b.start_date >= $2 AND b.end_date <= $3)
		)
		ORDER BY b.name`

	rows, err := r.db.Query(ctx, budgetQuery, familyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get budget comparison: %w", err)
	}
	defer rows.Close()

	var budgetComparison []report.BudgetComparisonItem
	for rows.Next() {
		var item report.BudgetComparisonItem
		err = rows.Scan(&item.BudgetID, &item.BudgetName, &item.Planned, &item.Actual)
		if err != nil {
			return nil, fmt.Errorf("failed to scan budget comparison item: %w", err)
		}

		item.Difference = item.Actual - item.Planned
		if item.Planned > 0 {
			item.Percentage = (item.Actual / item.Planned) * 100
		}

		budgetComparison = append(budgetComparison, item)
	}
	data.BudgetComparison = budgetComparison

	return &data, nil
}

// Delete deletes a report
func (r *PostgreSQLRepository) Delete(ctx context.Context, id uuid.UUID, familyID uuid.UUID) error {
	// Validate UUID parameters
	if err := validation.ValidateUUID(id); err != nil {
		return fmt.Errorf("invalid id parameter: %w", err)
	}
	if err := validation.ValidateUUID(familyID); err != nil {
		return fmt.Errorf("invalid family ID parameter: %w", err)
	}

	query := `DELETE FROM family_budget.reports WHERE id = $1 AND family_id = $2`

	result, err := r.db.Exec(ctx, query, id, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("report with id %s not found", id)
	}

	return nil
}

// GetReportSummary returns summary statistics for reports
func (r *PostgreSQLRepository) GetReportSummary(ctx context.Context, familyID uuid.UUID) (*ReportSummary, error) {
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
		FROM family_budget.reports
		WHERE family_id = $1`

	var summary ReportSummary
	err := r.db.QueryRow(ctx, query, familyID).Scan(
		&summary.TotalReports, &summary.ExpenseReports, &summary.IncomeReports,
		&summary.BudgetReports, &summary.CashFlowReports, &summary.LastGenerated,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get report summary: %w", err)
	}

	summary.FamilyID = familyID
	return &summary, nil
}

// ReportSummary holds report summary statistics
type ReportSummary struct {
	FamilyID        uuid.UUID  `json:"family_id"`
	TotalReports    int        `json:"total_reports"`
	ExpenseReports  int        `json:"expense_reports"`
	IncomeReports   int        `json:"income_reports"`
	BudgetReports   int        `json:"budget_reports"`
	CashFlowReports int        `json:"cash_flow_reports"`
	LastGenerated   *time.Time `json:"last_generated,omitempty"`
}

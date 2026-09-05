package handlers_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services/dto"
)

// MockReportService is a mock implementation of ReportService
type MockReportService struct {
	mock.Mock
}

func (m *MockReportService) GenerateExpenseReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.ExpenseReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ExpenseReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateIncomeReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*dto.IncomeReportDTO, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.IncomeReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateBudgetComparisonReport(
	ctx context.Context,
	period report.Period,
) (*dto.BudgetComparisonDTO, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.BudgetComparisonDTO), args.Error(1)
}

func (m *MockReportService) GenerateCashFlowReport(
	ctx context.Context,
	from, to time.Time,
) (*dto.CashFlowReportDTO, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CashFlowReportDTO), args.Error(1)
}

func (m *MockReportService) GenerateCategoryBreakdownReport(
	ctx context.Context,
	period report.Period,
) (*dto.CategoryBreakdownDTO, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.CategoryBreakdownDTO), args.Error(1)
}

func (m *MockReportService) GenerateReport(
	ctx context.Context,
	req dto.ReportRequestDTO,
) (*report.Report, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) SaveReport(ctx context.Context, reportEntity *report.Report) error {
	args := m.Called(ctx, reportEntity)
	return args.Error(0)
}

func (m *MockReportService) GetReportByID(ctx context.Context, id uuid.UUID) (*report.Report, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*report.Report), args.Error(1)
}

func (m *MockReportService) GetReports(
	ctx context.Context,
	typeFilter *report.Type,
) ([]*report.Report, error) {
	args := m.Called(ctx, typeFilter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportService) GetReportsByUserID(ctx context.Context, userID uuid.UUID) ([]*report.Report, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*report.Report), args.Error(1)
}

func (m *MockReportService) DeleteReport(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockReportService) ExportReport(
	ctx context.Context,
	reportID uuid.UUID,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportID, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) ExportReportData(
	ctx context.Context,
	reportData any,
	format string,
	options dto.ExportOptionsDTO,
) ([]byte, error) {
	args := m.Called(ctx, reportData, format, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReportService) GenerateTrendAnalysis(
	ctx context.Context,
	categoryID *uuid.UUID,
	period report.Period,
) (*dto.TrendAnalysisDTO, error) {
	args := m.Called(ctx, categoryID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.TrendAnalysisDTO), args.Error(1)
}

func (m *MockReportService) GenerateFinancialInsights(ctx context.Context) ([]dto.RecommendationDTO, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.RecommendationDTO), args.Error(1)
}

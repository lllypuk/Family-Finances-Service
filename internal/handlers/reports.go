package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/report"
)

type ReportHandler struct {
	repositories *Repositories
	validator    *validator.Validate
}

func NewReportHandler(repositories *Repositories) *ReportHandler {
	return &ReportHandler{
		repositories: repositories,
		validator:    validator.New(),
	}
}

func (h *ReportHandler) CreateReport(c echo.Context) error {
	var req CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
				Details: err.Error(),
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	if err := h.validator.Struct(req); err != nil {
		var validationErrors []ValidationError
		for _, err := range func() validator.ValidationErrors {
			var target validator.ValidationErrors
			_ = errors.As(err, &target)
			return target
		}() {
			validationErrors = append(validationErrors, ValidationError{
				Field:   err.Field(),
				Message: err.Tag(),
				Code:    "VALIDATION_ERROR",
			})
		}

		return c.JSON(http.StatusBadRequest, APIResponse[interface{}]{
			Data: nil,
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
			Errors: validationErrors,
		})
	}

	// Создаем новый отчет
	newReport := &report.Report{
		ID:          uuid.New(),
		Name:        req.Name,
		Type:        report.ReportType(req.Type),
		Period:      report.ReportPeriod(req.Period),
		FamilyID:    req.FamilyID,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Data:        report.ReportData{}, // Пока пустые данные
		GeneratedAt: time.Now(),
	}

	// TODO: Здесь должна быть логика генерации данных отчета
	// в зависимости от типа отчета (expenses, income, budget, cash_flow, category_break)
	newReport.Data = h.generateReportData(
		c.Request().Context(),
		report.ReportType(req.Type),
		req.FamilyID,
		req.StartDate,
		req.EndDate,
	)

	if err := h.repositories.Report.Create(c.Request().Context(), newReport); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "CREATE_FAILED",
				Message: "Failed to create report",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := ReportResponse{
		ID:          newReport.ID,
		Name:        newReport.Name,
		Type:        string(newReport.Type),
		Period:      string(newReport.Period),
		FamilyID:    newReport.FamilyID,
		UserID:      newReport.UserID,
		StartDate:   newReport.StartDate,
		EndDate:     newReport.EndDate,
		Data:        newReport.Data,
		GeneratedAt: newReport.GeneratedAt,
	}

	return c.JSON(http.StatusCreated, APIResponse[ReportResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *ReportHandler) GetReports(c echo.Context) error {
	// Получаем параметры запроса
	familyIDParam := c.QueryParam("family_id")
	userIDParam := c.QueryParam("user_id")

	if familyIDParam == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "MISSING_FAMILY_ID",
				Message: "family_id query parameter is required",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	familyID, err := uuid.Parse(familyIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_FAMILY_ID",
				Message: "Invalid family ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var reports []*report.Report

	// Если указан пользователь, получаем отчеты для конкретного пользователя
	if userIDParam != "" {
		userID, err := uuid.Parse(userIDParam)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: ErrorDetail{
					Code:    "INVALID_USER_ID",
					Message: "Invalid user ID format",
				},
				Meta: ResponseMeta{
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
					Timestamp: time.Now(),
					Version:   "v1",
				},
			})
		}
		reports, err = h.repositories.Report.GetByUserID(c.Request().Context(), userID)
	} else {
		// Иначе получаем все отчеты семьи
		reports, err = h.repositories.Report.GetByFamilyID(c.Request().Context(), familyID)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: ErrorDetail{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch reports",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	var response []ReportResponse
	for _, r := range reports {
		response = append(response, ReportResponse{
			ID:          r.ID,
			Name:        r.Name,
			Type:        string(r.Type),
			Period:      string(r.Period),
			FamilyID:    r.FamilyID,
			UserID:      r.UserID,
			StartDate:   r.StartDate,
			EndDate:     r.EndDate,
			Data:        r.Data,
			GeneratedAt: r.GeneratedAt,
		})
	}

	return c.JSON(http.StatusOK, APIResponse[[]ReportResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *ReportHandler) GetReportByID(c echo.Context) error {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid report ID format",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	foundReport, err := h.repositories.Report.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:    "REPORT_NOT_FOUND",
				Message: "Report not found",
			},
			Meta: ResponseMeta{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Timestamp: time.Now(),
				Version:   "v1",
			},
		})
	}

	response := ReportResponse{
		ID:          foundReport.ID,
		Name:        foundReport.Name,
		Type:        string(foundReport.Type),
		Period:      string(foundReport.Period),
		FamilyID:    foundReport.FamilyID,
		UserID:      foundReport.UserID,
		StartDate:   foundReport.StartDate,
		EndDate:     foundReport.EndDate,
		Data:        foundReport.Data,
		GeneratedAt: foundReport.GeneratedAt,
	}

	return c.JSON(http.StatusOK, APIResponse[ReportResponse]{
		Data: response,
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

func (h *ReportHandler) DeleteReport(c echo.Context) error {
	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		return h.repositories.Report.Delete(c.Request().Context(), id)
	}, "Report")
}

// generateReportData генерирует данные отчета в зависимости от типа
func (h *ReportHandler) generateReportData(
	ctx context.Context,
	reportType report.ReportType,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) report.ReportData {
	data := report.ReportData{}

	// Получаем базовые данные для всех типов отчетов
	h.populateBasicReportData(ctx, &data, familyID, startDate, endDate)

	switch reportType {
	case report.ReportTypeExpenses:
		h.generateExpensesReportData(ctx, &data, familyID, startDate, endDate)
	case report.ReportTypeIncome:
		h.generateIncomeReportData(ctx, &data, familyID, startDate, endDate)
	case report.ReportTypeBudget:
		h.generateBudgetReportData(ctx, &data, familyID, startDate, endDate)
	case report.ReportTypeCashFlow:
		h.generateCashFlowReportData(ctx, &data, familyID, startDate, endDate)
	case report.ReportTypeCategoryBreak:
		h.generateCategoryBreakdownReportData(ctx, &data, familyID, startDate, endDate)
	}

	return data
}

// populateBasicReportData заполняет базовые данные для всех типов отчетов
func (h *ReportHandler) populateBasicReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// Базовые расчеты для всех отчетов
	data.TotalIncome = 0
	data.TotalExpenses = 0
	data.NetIncome = 0
	data.CategoryBreakdown = []report.CategoryReportItem{}
	data.DailyBreakdown = []report.DailyReportItem{}
	data.TopExpenses = []report.TransactionReportItem{}
	data.BudgetComparison = []report.BudgetComparisonItem{}
}

// generateExpensesReportData генерирует данные для отчета по расходам
func (h *ReportHandler) generateExpensesReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// TODO: Реализовать получение расходов из транзакций
	// Пример структуры данных для расходов
	data.TotalExpenses = 0 // Будет рассчитано из транзакций
}

// generateIncomeReportData генерирует данные для отчета по доходам
func (h *ReportHandler) generateIncomeReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// TODO: Реализовать получение доходов из транзакций
	data.TotalIncome = 0 // Будет рассчитано из транзакций
}

// generateBudgetReportData генерирует данные для отчета по бюджету
func (h *ReportHandler) generateBudgetReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// TODO: Реализовать сравнение бюджета с фактическими тратами
	// Получение активных бюджетов и сравнение с транзакциями
}

// generateCashFlowReportData генерирует данные для отчета по денежному потоку
func (h *ReportHandler) generateCashFlowReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// TODO: Реализовать расчет денежного потока по дням
	// Заполнение DailyBreakdown с доходами и расходами по дням
}

// generateCategoryBreakdownReportData генерирует данные для разбивки по категориям
func (h *ReportHandler) generateCategoryBreakdownReportData(
	ctx context.Context,
	data *report.ReportData,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) {
	// TODO: Реализовать группировку транзакций по категориям
	// Заполнение CategoryBreakdown с суммами по категориям
}

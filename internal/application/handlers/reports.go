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

		return c.JSON(http.StatusBadRequest, APIResponse[any]{
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
		Type:        report.Type(req.Type),
		Period:      report.Period(req.Period),
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Data:        report.Data{}, // Пока пустые данные
		GeneratedAt: time.Now(),
	}

	// TODO: Здесь должна быть логика генерации данных отчета
	// в зависимости от типа отчета (expenses, income, budget, cash_flow, category_breakdown)
	newReport.Data = h.generateReportData(
		c.Request().Context(),
		report.Type(req.Type),
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
	userIDParam := c.QueryParam("user_id")

	var reports []*report.Report
	var err error

	// Если указан пользователь, получаем отчеты для конкретного пользователя
	if userIDParam != "" {
		userID, parseErr := uuid.Parse(userIDParam)
		if parseErr != nil {
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
		reports, err = h.repositories.Report.GetAll(c.Request().Context())
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
		// In single-family model, repository handles family ID internally
		return h.repositories.Report.Delete(c.Request().Context(), id)
	}, "Report")
}

// generateReportData генерирует данные отчета в зависимости от типа
func (h *ReportHandler) generateReportData(
	ctx context.Context,
	reportType report.Type,
	startDate, endDate time.Time,
) report.Data {
	data := report.Data{}

	// Получаем базовые данные для всех типов отчетов
	h.populateBasicReportData(ctx, &data, startDate, endDate)

	switch reportType {
	case report.TypeExpenses:
		h.generateExpensesReportData(ctx, &data, startDate, endDate)
	case report.TypeIncome:
		h.generateIncomeReportData(ctx, &data, startDate, endDate)
	case report.TypeBudget:
		h.generateBudgetReportData(ctx, &data, startDate, endDate)
	case report.TypeCashFlow:
		h.generateCashFlowReportData(ctx, &data, startDate, endDate)
	case report.TypeCategoryBreak:
		h.generateCategoryBreakdownReportData(ctx, &data, startDate, endDate)
	}

	return data
}

// populateBasicReportData заполняет базовые данные для всех типов отчетов
func (h *ReportHandler) populateBasicReportData(
	_ context.Context,
	data *report.Data,
	_, _ time.Time,
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
	_ context.Context,
	data *report.Data,
	_, _ time.Time,
) {
	// TODO: Реализовать получение расходов из транзакций
	// Пример структуры данных для расходов
	data.TotalExpenses = 0 // Будет рассчитано из транзакций
}

// generateIncomeReportData генерирует данные для отчета по доходам
func (h *ReportHandler) generateIncomeReportData(
	_ context.Context,
	data *report.Data,
	_, _ time.Time,
) {
	// TODO: Реализовать получение доходов из транзакций
	data.TotalIncome = 0 // Будет рассчитано из транзакций
}

// generateBudgetReportData генерирует данные для отчета по бюджету
func (h *ReportHandler) generateBudgetReportData(
	_ context.Context,
	_ *report.Data,
	_, _ time.Time,
) {
	// TODO: Реализовать сравнение бюджета с фактическими тратами
	// Получение активных бюджетов и сравнение с транзакциями
}

// generateCashFlowReportData генерирует данные для отчета по денежному потоку
func (h *ReportHandler) generateCashFlowReportData(
	_ context.Context,
	_ *report.Data,
	_, _ time.Time,
) {
	// TODO: Реализовать расчет денежного потока по дням
	// Заполнение DailyBreakdown с доходами и расходами по дням
}

// generateCategoryBreakdownReportData генерирует данные для разбивки по категориям
func (h *ReportHandler) generateCategoryBreakdownReportData(
	_ context.Context,
	_ *report.Data,
	_, _ time.Time,
) {
	// TODO: Реализовать группировку транзакций по категориям
	// Заполнение CategoryBreakdown с суммами по категориям
}

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

	if err := h.repositories.Report.Delete(c.Request().Context(), id); err != nil {
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

	return c.JSON(http.StatusOK, APIResponse[interface{}]{
		Data: map[string]string{"message": "Report deleted successfully"},
		Meta: ResponseMeta{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Timestamp: time.Now(),
			Version:   "v1",
		},
	})
}

// generateReportData генерирует данные отчета в зависимости от типа
func (h *ReportHandler) generateReportData(
	ctx context.Context,
	reportType report.ReportType,
	familyID uuid.UUID,
	startDate, endDate time.Time,
) report.ReportData {
	// TODO: Реализовать генерацию данных для каждого типа отчета
	// Пока возвращаем заглушку со значениями по умолчанию
	data := report.ReportData{}

	switch reportType {
	case report.ReportTypeExpenses:
		// заполните поля, когда будет реализована логика
	case report.ReportTypeIncome:
	case report.ReportTypeBudget:
	case report.ReportTypeCashFlow:
	case report.ReportTypeCategoryBreak:
	}

	return data
}

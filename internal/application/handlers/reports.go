package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
)

type ReportHandler struct {
	repositories  *Repositories
	validator     *validator.Validate
	reportService services.ReportService
}

func NewReportHandler(
	repositories *Repositories,
	reportServices ...services.ReportService,
) *ReportHandler {
	var reportService services.ReportService
	if len(reportServices) > 0 {
		reportService = reportServices[0]
	}

	return &ReportHandler{
		repositories:  repositories,
		validator:     validator.New(),
		reportService: reportService,
	}
}

func (h *ReportHandler) CreateReport(c echo.Context) error {
	var req CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err.Error())
	}

	if err := h.validator.Struct(req); err != nil {
		return HandleValidationError(c, err)
	}

	return respondError(
		c,
		http.StatusNotImplemented,
		"NOT_IMPLEMENTED",
		"Report generation API is not implemented yet",
		"Use stored reports endpoints only until report generation is completed",
	)
}

func (h *ReportHandler) GetReports(c echo.Context) error {
	if h.reportService != nil {
		return h.getReportsViaService(c)
	}

	// Получаем параметры запроса
	userIDParam := c.QueryParam("user_id")

	var reports []*report.Report
	var err error

	// Если указан пользователь, получаем отчеты для конкретного пользователя
	if userIDParam != "" {
		userID, parseErr := uuid.Parse(userIDParam)
		if parseErr != nil {
			return respondError(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID format")
		}
		reports, err = h.repositories.Report.GetByUserID(c.Request().Context(), userID)
	} else {
		// Иначе получаем все отчеты семьи
		reports, err = h.repositories.Report.GetAll(c.Request().Context())
	}

	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch reports")
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

	return respondAPI(c, http.StatusOK, response)
}

func (h *ReportHandler) GetReportByID(c echo.Context) error {
	if h.reportService != nil {
		return h.getReportByIDViaService(c)
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return HandleIDParseError(c, "report")
	}

	foundReport, err := h.repositories.Report.GetByID(c.Request().Context(), id)
	if err != nil {
		return HandleNotFoundError(c, "Report")
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

	return respondAPI(c, http.StatusOK, response)
}

func (h *ReportHandler) DeleteReport(c echo.Context) error {
	if h.reportService != nil {
		return DeleteEntityHelper(c, func(id uuid.UUID) error {
			return h.reportService.DeleteReport(c.Request().Context(), id)
		}, "Report")
	}

	return DeleteEntityHelper(c, func(id uuid.UUID) error {
		// In single-family model, repository handles family ID internally
		return h.repositories.Report.Delete(c.Request().Context(), id)
	}, "Report")
}

func (h *ReportHandler) getReportsViaService(c echo.Context) error {
	userIDParam := c.QueryParam("user_id")

	var reports []*report.Report
	var err error

	if userIDParam != "" {
		userID, parseErr := uuid.Parse(userIDParam)
		if parseErr != nil {
			return respondError(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID format")
		}
		reports, err = h.reportService.GetReportsByUserID(c.Request().Context(), userID)
	} else {
		reports, err = h.reportService.GetReports(c.Request().Context(), nil)
	}
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch reports")
	}

	response := make([]ReportResponse, 0, len(reports))
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

	return respondAPI(c, http.StatusOK, response)
}

func (h *ReportHandler) getReportByIDViaService(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "report")
	if err != nil {
		var idParseErr *IDParseError
		if errors.As(err, &idParseErr) {
			return HandleIDParseError(c, "report")
		}
		return err
	}

	foundReport, err := h.reportService.GetReportByID(c.Request().Context(), id)
	if err != nil {
		return HandleNotFoundError(c, "Report")
	}

	return respondAPI(c, http.StatusOK, ReportResponse{
		ID:          foundReport.ID,
		Name:        foundReport.Name,
		Type:        string(foundReport.Type),
		Period:      string(foundReport.Period),
		UserID:      foundReport.UserID,
		StartDate:   foundReport.StartDate,
		EndDate:     foundReport.EndDate,
		Data:        foundReport.Data,
		GeneratedAt: foundReport.GeneratedAt,
	})
}

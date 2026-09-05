package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
)

const (
	exportFormatCSV = "csv"
	csvContentType  = "text/csv; charset=utf-8"
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

// CreateReport генерирует отчёт и сохраняет его. Автор берётся из сессии:
// user_id в теле запроса нет намеренно (S-01).
func (h *ReportHandler) CreateReport(c echo.Context) error {
	sessionData, sessionErr := middleware.GetUserFromContext(c)
	if sessionErr != nil {
		return respondUnauthorized(c)
	}

	var req CreateReportRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
			bodyDetail(ErrCodeInvalidRequest, err.Error()))
	}

	if err := h.validator.Struct(req); err != nil {
		return respondValidationErrors(c, err)
	}

	if h.reportService == nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	ctx := c.Request().Context()
	generated, err := h.reportService.GenerateReport(ctx, dto.ReportRequestDTO{
		Name:      req.Name,
		Type:      report.Type(req.Type),
		Period:    report.Period(req.Period),
		UserID:    sessionData.UserID,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	})
	if err != nil {
		if errors.Is(err, services.ErrUnsupportedReportType) {
			return respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "Unsupported report type")
		}
		return respondError(c, http.StatusInternalServerError, ErrCodeGenerationFailed, "Failed to generate report")
	}

	if saveErr := h.reportService.SaveReport(ctx, generated); saveErr != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeSaveFailed, "Failed to save report")
	}

	return respondAPI(c, http.StatusCreated, newReportResponse(generated))
}

// ExportReport отдаёт отчёт в CSV; другие форматы не поддерживаются.
func (h *ReportHandler) ExportReport(c echo.Context) error {
	id, err := ParseIDParamWithError(c, "report")
	if err != nil {
		return HandleIDParseError(c, "report")
	}

	if h.reportService == nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeInternal, ErrMessageInternal)
	}

	ctx := c.Request().Context()
	if _, getErr := h.reportService.GetReportByID(ctx, id); getErr != nil {
		return HandleNotFoundError(c, "Report")
	}

	body, err := h.reportService.ExportReport(ctx, id, exportFormatCSV, dto.ExportOptionsDTO{})
	if err != nil {
		return respondError(c, http.StatusInternalServerError, ErrCodeExportFailed, "Failed to export report")
	}

	c.Response().Header().Set(echo.HeaderContentDisposition,
		fmt.Sprintf("attachment; filename=%q", "report-"+id.String()+".csv"))

	return c.Blob(http.StatusOK, csvContentType, body)
}

func (h *ReportHandler) GetReports(c echo.Context) error {
	page, err := parsePagination(c)
	if err != nil {
		return ignoreWritten(err)
	}

	if h.reportService != nil {
		return h.getReportsViaService(c, page)
	}

	userIDParam := c.QueryParam("user_id")

	var reports []*report.Report
	if userIDParam != "" {
		userID, parseErr := uuid.Parse(userIDParam)
		if parseErr != nil {
			return respondError(c, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID format")
		}
		reports, err = h.repositories.Report.GetByUserID(c.Request().Context(), userID)
	} else {
		reports, err = h.repositories.Report.GetAll(c.Request().Context())
	}

	if err != nil {
		return respondError(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch reports")
	}

	return respondList(c, buildReportListResponse(pageSlice(reports, page)), page, len(reports))
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

	return respondAPI(c, http.StatusOK, newReportResponse(foundReport))
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

func (h *ReportHandler) getReportsViaService(c echo.Context, page pageParams) error {
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

	return respondList(c, buildReportListResponse(pageSlice(reports, page)), page, len(reports))
}

func buildReportListResponse(reports []*report.Report) []ReportResponse {
	response := make([]ReportResponse, 0, len(reports))
	for _, r := range reports {
		response = append(response, newReportResponse(r))
	}

	return response
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

	return respondAPI(c, http.StatusOK, newReportResponse(foundReport))
}

func newReportResponse(r *report.Report) ReportResponse {
	return ReportResponse{
		ID:          r.ID,
		Name:        r.Name,
		Type:        string(r.Type),
		Period:      string(r.Period),
		UserID:      r.UserID,
		StartDate:   r.StartDate,
		EndDate:     r.EndDate,
		Data:        r.Data,
		GeneratedAt: r.GeneratedAt,
	}
}

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/report"
	"family-budget-service/internal/services"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

// errHTMXResponseSent is a sentinel error indicating that an HTMX response has already been sent.
// This is used to signal that the handler should return nil to complete the request.
var errHTMXResponseSent = errors.New("HTMX response already sent")

const (
	// MockFoodPercentage represents demo food category percentage
	MockFoodPercentage = 34.3
	// MockTransportPercentage represents demo transport category percentage
	MockTransportPercentage = 22.9
)

// ReportHandler обрабатывает HTTP запросы для отчетов
type ReportHandler struct {
	*BaseHandler

	validator *validator.Validate
}

// NewReportHandler создает новый обработчик отчетов
func NewReportHandler(
	repositories *handlers.Repositories,
	services *services.Services,
	cookieSecure bool,
) *ReportHandler {
	return &ReportHandler{
		BaseHandler: NewBaseHandler(repositories, services, cookieSecure),
		validator:   validator.New(),
	}
}

// reportIndexData — контракт данных страницы списка отчётов.
//
// Встроенный *PageData отдаёт шаблону `.CurrentUser` и `.CSRFToken` из корня
// контекста (шапка страницы ветвится на `{{if .CurrentUser}}`), а
// `{{.PageData.X}}` продолжает работать благодаря имени встроенного поля.
type reportIndexData struct {
	*PageData

	Reports           []webModels.ReportDataVM
	ReportTypeOptions []webModels.ReportTypeOption
	DefaultForm       webModels.ReportForm
}

// reportFormData — контракт данных формы отчёта (pages/reports/new).
//
// Form — значение, а не указатель: pages/reports/new.html читает `.Form.Name`
// и остальные поля без всяких `{{if}}`, а у структуры (в отличие от map)
// обращение к отсутствующему полю — ошибка исполнения шаблона.
type reportFormData struct {
	*PageData

	Form webModels.ReportForm
}

// reportShowData — контракт данных страницы просмотра отчёта.
type reportShowData struct {
	*PageData

	Report webModels.ReportDataVM
}

// Index отображает список отчетов и форму создания
func (h *ReportHandler) Index(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем список существующих отчетов
	reports, err := h.services.Report.GetReports(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get reports")
	}

	// Конвертируем в view модели
	reportVMs := make([]webModels.ReportDataVM, len(reports))
	for i, r := range reports {
		reportVMs[i].FromDomain(r)
	}

	// CSRF-токен приходит из PageData и промотируется в корень контекста.
	data := reportIndexData{
		PageData:          h.buildPageData(c, titleReports),
		Reports:           reportVMs,
		ReportTypeOptions: webModels.GetReportTypeOptions(),
		DefaultForm:       defaultReportForm(),
	}

	return h.renderPage(c, "pages/reports/index", data)
}

// New отображает форму создания нового отчета
func (h *ReportHandler) New(c echo.Context) error {
	data := reportFormData{
		PageData: h.buildPageData(c, titleNewReport),
		Form:     defaultReportForm(),
	}

	return h.renderPage(c, "pages/reports/new", data)
}

// defaultReportForm возвращает форму отчёта, предзаполненную текущим месяцем.
func defaultReportForm() webModels.ReportForm {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	return webModels.ReportForm{
		Type:      "expenses",
		Period:    "monthly",
		StartDate: startOfMonth.Format("2006-01-02"),
		EndDate:   endOfMonth.Format("2006-01-02"),
	}
}

// Create создает и генерирует новый отчет
func (h *ReportHandler) Create(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим и валидируем форму
	form, err := h.parseAndValidateReportForm(c)
	if err != nil {
		return err
	}

	// Проверяем, что форма не nil перед использованием
	if form == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Form validation failed")
	}

	// Создаем DTO для запроса отчета
	createDTO, err := h.buildReportRequestDTO(*form, sessionData)
	if err != nil {
		return err
	}

	// Генерируем отчет
	reportEntity, err := h.generateReport(c, createDTO)
	if err != nil {
		// Если HTMX ответ уже был отправлен, возвращаем nil чтобы завершить обработку
		if errors.Is(err, errHTMXResponseSent) {
			return nil
		}
		return err
	}

	// Проверяем, что отчет был создан (дополнительная защита от nil pointer)
	if reportEntity == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create report")
	}

	// Успешное создание - редирект на просмотр отчета
	reportURL := fmt.Sprintf("/reports/%s", reportEntity.ID)
	if h.IsHTMXRequest(c) {
		// Для HTMX запросов используем Hx-Redirect
		c.Response().Header().Set("Hx-Redirect", reportURL)
		return c.NoContent(http.StatusOK)
	}

	// Для обычных запросов - стандартный редирект
	return h.redirect(c, reportURL)
}

// parseAndValidateReportForm парсит и валидирует форму отчета
func (h *ReportHandler) parseAndValidateReportForm(c echo.Context) (*webModels.ReportForm, error) {
	var form webModels.ReportForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	if validationErr := h.validator.Struct(form); validationErr != nil {
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			return nil, h.renderPartial(c, "components/form_errors", map[string]any{
				tplKeyErrors: validationErrors,
			})
		}

		return nil, h.renderReportFormWithErrors(c, form, validationErrors)
	}

	return &form, nil
}

// buildReportRequestDTO создает DTO для запроса отчета
func (h *ReportHandler) buildReportRequestDTO(
	form webModels.ReportForm,
	sessionData *middleware.SessionData,
) (dto.ReportRequestDTO, error) {
	startDate, err := form.GetStartDate()
	if err != nil {
		return dto.ReportRequestDTO{}, fmt.Errorf("invalid start date: %w", err)
	}

	endDate, err := form.GetEndDate()
	if err != nil {
		return dto.ReportRequestDTO{}, fmt.Errorf("invalid end date: %w", err)
	}

	return dto.ReportRequestDTO{
		Name:      form.Name,
		Type:      form.ToReportType(),
		Period:    form.ToReportPeriod(),
		UserID:    sessionData.UserID,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

// generateReport генерирует и сохраняет отчет через сервис
func (h *ReportHandler) generateReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	ctx := c.Request().Context()

	reportEntity, err := h.services.Report.GenerateReport(ctx, createDTO)
	if err != nil {
		if errors.Is(err, services.ErrUnsupportedReportType) {
			return h.handleUnsupportedReportType(c)
		}
		return nil, h.handleReportGenerationError(c, err)
	}

	if saveErr := h.services.Report.SaveReport(ctx, reportEntity); saveErr != nil {
		return nil, h.handleReportGenerationError(c, saveErr)
	}

	return reportEntity, nil
}

// handleUnsupportedReportType обрабатывает неподдерживаемый тип отчета
func (h *ReportHandler) handleUnsupportedReportType(c echo.Context) (*report.Report, error) {
	errorMsg := "Unsupported report type"
	if h.IsHTMXRequest(c) {
		if renderErr := h.renderPartial(c, "components/form_errors", map[string]any{
			tplKeyErrors: map[string]string{tplKeyForm: errorMsg},
		}); renderErr != nil {
			return nil, renderErr
		}
		return nil, errHTMXResponseSent
	}
	return nil, echo.NewHTTPError(http.StatusBadRequest, errorMsg)
}

// handleReportGenerationError обрабатывает ошибки генерации отчетов
func (h *ReportHandler) handleReportGenerationError(c echo.Context, err error) error {
	errorMsg := h.getReportServiceErrorMessage(err)
	if h.IsHTMXRequest(c) {
		if renderErr := h.renderPartial(c, "components/form_errors", map[string]any{
			tplKeyErrors: map[string]string{tplKeyForm: errorMsg},
		}); renderErr != nil {
			return renderErr
		}
		return errHTMXResponseSent
	}
	return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
}

// Show отображает сгенерированный отчет
func (h *ReportHandler) Show(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID отчета
	id := c.Param("id")
	reportID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report ID")
	}

	// Получаем отчет
	report, err := h.services.Report.GetReportByID(c.Request().Context(), reportID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Report not found")
	}

	// In single-family model, all reports belong to the family
	// No additional access check needed

	// Конвертируем в view модель
	reportVM := webModels.ReportDataVM{}
	reportVM.FromDomain(report)

	data := reportShowData{
		PageData: h.buildPageData(c, titleReportPrefix+report.Name),
		Report:   reportVM,
	}

	return h.renderPage(c, "pages/reports/show", data)
}

// Delete удаляет отчет
func (h *ReportHandler) Delete(c echo.Context) error {
	return h.handleDelete(c, DeleteEntityParams{
		EntityName: "report",
		GetEntityFunc: func(ctx echo.Context, entityID uuid.UUID) (any, error) {
			return h.services.Report.GetReportByID(ctx.Request().Context(), entityID)
		},
		DeleteEntityFunc: func(ctx echo.Context, entityID uuid.UUID) error {
			return h.services.Report.DeleteReport(ctx.Request().Context(), entityID)
		},
		GetErrorMsgFunc: h.getReportServiceErrorMessage,
		RedirectURL:     "/reports",
	})
}

// Export экспортирует отчет в указанном формате (CSV)
func (h *ReportHandler) Export(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID отчета
	id := c.Param("id")
	reportID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid report ID")
	}

	format := c.QueryParam("format")
	if format != "csv" {
		return echo.NewHTTPError(http.StatusBadRequest, "Unsupported export format")
	}

	// Получаем отчет
	report, err := h.services.Report.GetReportByID(c.Request().Context(), reportID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Report not found")
	}

	// In single-family model, all reports belong to the family
	// No additional access check needed

	// Экспортируем в CSV
	return h.exportReportAsCSV(c, report)
}

// Generate генерирует отчет по параметрам (HTMX)
func (h *ReportHandler) Generate(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим данные формы
	var form webModels.ReportForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		validationErrors := webModels.GetValidationErrors(validationErr)
		return h.renderPartial(c, "components/form_errors", map[string]any{
			tplKeyErrors: validationErrors,
		})
	}

	// Парсим даты
	startDate, err := form.GetStartDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start date")
	}

	endDate, err := form.GetEndDate()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end date")
	}

	// Создание DTO для генерации отчета
	generateDTO := dto.ReportRequestDTO{
		Name:      form.Name,
		Type:      form.ToReportType(),
		Period:    form.ToReportPeriod(),
		UserID:    sessionData.UserID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Генерация отчета через сервис (без сохранения — это предпросмотр)
	tempReport, err := h.services.Report.GenerateReport(c.Request().Context(), generateDTO)
	if err != nil {
		if errors.Is(err, services.ErrUnsupportedReportType) {
			return h.renderPartial(c, "components/form_errors", map[string]any{
				tplKeyErrors: map[string]string{tplKeyForm: "Unsupported report type"},
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate report")
	}

	reportVM := webModels.ReportDataVM{}
	reportVM.FromDomain(tempReport)

	data := map[string]any{
		"Report": reportVM,
	}

	return h.renderPartial(c, "components/report_data", data)
}

// exportReportAsCSV отдаёт CSV, собранный сервисом отчётов.
func (h *ReportHandler) exportReportAsCSV(c echo.Context, r *report.Report) error {
	body, err := h.services.Report.ExportReport(c.Request().Context(), r.ID, "csv", dto.ExportOptionsDTO{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to export report")
	}

	filename := fmt.Sprintf("%s_%s.csv",
		strings.ReplaceAll(r.Name, " ", "_"),
		r.GeneratedAt.Format("2006-01-02"))
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Blob(http.StatusOK, "text/csv", body)
}

// renderReportFormWithErrors отображает форму создания отчёта с ошибками
// валидации. Страницы редактирования отчёта нет ни в маршрутах, ни в шаблонах,
// поэтому ветка на "pages/reports/edit" убрана вместе с параметром title.
func (h *ReportHandler) renderReportFormWithErrors(
	c echo.Context,
	form webModels.ReportForm,
	errors map[string]string,
) error {
	data := reportFormData{
		PageData: h.formPageData(c, titleNewReport, errors),
		Form:     form,
	}

	return h.renderPage(c, "pages/reports/new", data)
}

// getReportServiceErrorMessage возвращает пользовательское сообщение об ошибке.
//
// Как и у бюджетов, исходный текст ошибки клиенту не показывается: обёрнутая
// ошибка репозитория раскрывает схему БД. Наружу идёт только распознанная
// формулировка либо общая.
func (h *ReportHandler) getReportServiceErrorMessage(err error) string {
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "report not found"):
		return "Report not found"
	case strings.Contains(errMsg, "invalid date range"):
		return "Invalid date range"
	case strings.Contains(errMsg, "no data available"):
		return "No data available for the specified period"
	case strings.Contains(errMsg, "generation failed"):
		return "Failed to generate report"
	default:
		return "Failed to process report"
	}
}

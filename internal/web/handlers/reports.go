package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
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
func NewReportHandler(repositories *handlers.Repositories, services *services.Services) *ReportHandler {
	return &ReportHandler{
		BaseHandler: NewBaseHandler(repositories, services),
		validator:   validator.New(),
	}
}

// Index отображает список отчетов и форму создания
func (h *ReportHandler) Index(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем список существующих отчетов семьи
	reports, err := h.services.Report.GetReportsByFamily(c.Request().Context(), sessionData.FamilyID, nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get reports")
	}

	// Конвертируем в view модели
	reportVMs := make([]webModels.ReportDataVM, len(reports))
	for i, r := range reports {
		reportVMs[i].FromDomain(r)
	}

	// Подготавливаем опции типов отчетов
	reportTypeOptions := webModels.GetReportTypeOptions()

	// Предзаполняем форму с текущим месяцем
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	defaultForm := webModels.ReportForm{
		Type:      "expenses",
		Period:    "monthly",
		StartDate: startOfMonth.Format("2006-01-02"),
		EndDate:   endOfMonth.Format("2006-01-02"),
	}

	pageData := &PageData{
		Title: "Reports",
	}

	data := map[string]any{
		"PageData":          pageData,
		"Reports":           reportVMs,
		"ReportTypeOptions": reportTypeOptions,
		"DefaultForm":       defaultForm,
	}

	return h.renderPage(c, "pages/reports/index", data)
}

// New отображает форму создания нового отчета
func (h *ReportHandler) New(c echo.Context) error {
	// Получаем CSRF токен
	csrfToken, _ := middleware.GetCSRFToken(c)

	// TODO: Реализовать отображение формы создания отчета
	pageData := &PageData{
		Title: "New Report",
	}

	data := map[string]any{
		"PageData":  pageData,
		"CSRFToken": csrfToken,
	}

	return h.renderPage(c, "pages/reports/new", data)
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

	// Создаем DTO для запроса отчета
	createDTO, err := h.buildReportRequestDTO(*form, sessionData)
	if err != nil {
		return err
	}

	// Генерируем отчет
	reportEntity, err := h.generateReport(c, createDTO)
	if err != nil {
		return err
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
				"Errors": validationErrors,
			})
		}

		return nil, h.renderReportFormWithErrors(c, form, "New Report")
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
		FamilyID:  sessionData.FamilyID,
		UserID:    sessionData.UserID,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

// generateReport генерирует отчет по типу
func (h *ReportHandler) generateReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	switch createDTO.Type {
	case report.TypeExpenses:
		return h.generateExpenseReport(c, createDTO)
	case report.TypeIncome:
		return h.generateIncomeReport(c, createDTO)
	case report.TypeBudget:
		return h.generateBudgetReport(c, createDTO)
	case report.TypeCashFlow:
		return h.generateCashFlowReport(c, createDTO)
	case report.TypeCategoryBreak:
		return h.generateCategoryReport(c, createDTO)
	default:
		return h.handleUnsupportedReportType(c)
	}
}

// generateExpenseReport генерирует отчет по расходам
func (h *ReportHandler) generateExpenseReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	expenseReport, err := h.services.Report.GenerateExpenseReport(c.Request().Context(), createDTO)
	if err != nil {
		return nil, h.handleReportGenerationError(c, err)
	}

	return h.services.Report.SaveReport(c.Request().Context(), expenseReport, createDTO.Type, createDTO)
}

// generateIncomeReport генерирует отчет по доходам
func (h *ReportHandler) generateIncomeReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	incomeReport, err := h.services.Report.GenerateIncomeReport(c.Request().Context(), createDTO)
	if err != nil {
		return nil, h.handleReportGenerationError(c, err)
	}

	return h.services.Report.SaveReport(c.Request().Context(), incomeReport, createDTO.Type, createDTO)
}

// generateBudgetReport генерирует отчет по бюджету
func (h *ReportHandler) generateBudgetReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	budgetReport, err := h.services.Report.GenerateBudgetComparisonReport(
		c.Request().Context(),
		createDTO.FamilyID,
		createDTO.Period,
	)
	if err != nil {
		return nil, h.handleReportGenerationError(c, err)
	}

	return h.services.Report.SaveReport(c.Request().Context(), budgetReport, createDTO.Type, createDTO)
}

// generateCashFlowReport генерирует отчет по денежному потоку
func (h *ReportHandler) generateCashFlowReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	cashFlowReport, err := h.services.Report.GenerateCashFlowReport(
		c.Request().Context(),
		createDTO.FamilyID,
		createDTO.StartDate,
		createDTO.EndDate,
	)
	if err != nil {
		return nil, h.handleReportGenerationError(c, err)
	}

	return h.services.Report.SaveReport(c.Request().Context(), cashFlowReport, createDTO.Type, createDTO)
}

// generateCategoryReport генерирует отчет по категориям
func (h *ReportHandler) generateCategoryReport(c echo.Context, createDTO dto.ReportRequestDTO) (*report.Report, error) {
	categoryReport, err := h.services.Report.GenerateCategoryBreakdownReport(
		c.Request().Context(),
		createDTO.FamilyID,
		createDTO.Period,
	)
	if err != nil {
		return nil, h.handleReportGenerationError(c, err)
	}

	return h.services.Report.SaveReport(c.Request().Context(), categoryReport, createDTO.Type, createDTO)
}

// handleUnsupportedReportType обрабатывает неподдерживаемый тип отчета
func (h *ReportHandler) handleUnsupportedReportType(c echo.Context) (*report.Report, error) {
	errorMsg := "Unsupported report type"
	if h.IsHTMXRequest(c) {
		return nil, h.renderPartial(c, "components/form_errors", map[string]any{
			"Errors": map[string]string{"form": errorMsg},
		})
	}
	return nil, echo.NewHTTPError(http.StatusBadRequest, errorMsg)
}

// handleReportGenerationError обрабатывает ошибки генерации отчетов
func (h *ReportHandler) handleReportGenerationError(c echo.Context, err error) error {
	errorMsg := h.getReportServiceErrorMessage(err)
	if h.IsHTMXRequest(c) {
		return h.renderPartial(c, "components/form_errors", map[string]any{
			"Errors": map[string]string{"form": errorMsg},
		})
	}
	return echo.NewHTTPError(http.StatusInternalServerError, errorMsg)
}

// Show отображает сгенерированный отчет
func (h *ReportHandler) Show(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
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

	// Проверяем, что отчет принадлежит семье пользователя
	if report.FamilyID != sessionData.FamilyID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

	// Конвертируем в view модель
	reportVM := webModels.ReportDataVM{}
	reportVM.FromDomain(report)

	pageData := &PageData{
		Title: "Report: " + report.Name,
	}

	data := map[string]any{
		"PageData": pageData,
		"Report":   reportVM,
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
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
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

	// Проверяем, что отчет принадлежит семье пользователя
	if report.FamilyID != sessionData.FamilyID {
		return echo.NewHTTPError(http.StatusForbidden, "Access denied")
	}

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
			"Errors": validationErrors,
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
		FamilyID:  sessionData.FamilyID,
		UserID:    sessionData.UserID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Генерация отчета через сервис
	var reportData any
	var generateErr error

	switch generateDTO.Type {
	case report.TypeExpenses:
		reportData, generateErr = h.services.Report.GenerateExpenseReport(c.Request().Context(), generateDTO)
	case report.TypeIncome:
		reportData, generateErr = h.services.Report.GenerateIncomeReport(c.Request().Context(), generateDTO)
	case report.TypeBudget:
		reportData, generateErr = h.services.Report.GenerateBudgetComparisonReport(
			c.Request().Context(),
			generateDTO.FamilyID,
			generateDTO.Period,
		)
	case report.TypeCashFlow:
		reportData, generateErr = h.services.Report.GenerateCashFlowReport(
			c.Request().Context(),
			generateDTO.FamilyID,
			generateDTO.StartDate,
			generateDTO.EndDate,
		)
	case report.TypeCategoryBreak:
		reportData, generateErr = h.services.Report.GenerateCategoryBreakdownReport(
			c.Request().Context(),
			generateDTO.FamilyID,
			generateDTO.Period,
		)
	default:
		return h.renderPartial(c, "components/form_errors", map[string]any{
			"Errors": map[string]string{"form": "Unsupported report type"},
		})
	}
	if generateErr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate report")
	}

	// Создаем временный отчет для отображения
	tempReport := &report.Report{
		ID:          uuid.New(),
		Name:        form.Name,
		Type:        generateDTO.Type,
		Period:      generateDTO.Period,
		FamilyID:    generateDTO.FamilyID,
		UserID:      generateDTO.UserID,
		StartDate:   generateDTO.StartDate,
		EndDate:     generateDTO.EndDate,
		GeneratedAt: time.Now(),
	}

	// Конвертируем данные отчета в стандартный формат
	tempReport.Data = h.convertReportDataToStandard(reportData, generateDTO.Type)

	// Конвертируем в view модель
	reportVM := webModels.ReportDataVM{}
	reportVM.FromDomain(tempReport)

	data := map[string]any{
		"Report": reportVM,
	}

	return h.renderPartial(c, "components/report_data", data)
}

// exportReportAsCSV экспортирует отчет в CSV формат
func (h *ReportHandler) exportReportAsCSV(c echo.Context, r *report.Report) error {
	filename := fmt.Sprintf("%s_%s.csv",
		strings.ReplaceAll(r.Name, " ", "_"),
		r.GeneratedAt.Format("2006-01-02"))

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(c.Response().Writer)
	defer writer.Flush()

	// В зависимости от типа отчета экспортируем разные данные
	switch r.Type {
	case report.TypeExpenses, report.TypeIncome, report.TypeCategoryBreak:
		return h.exportCategoryBreakdownCSV(writer, r)
	case report.TypeCashFlow:
		return h.exportDailyBreakdownCSV(writer, r)
	case report.TypeBudget:
		return h.exportBudgetComparisonCSV(writer, r)
	default:
		return h.exportCategoryBreakdownCSV(writer, r)
	}
}

// exportCategoryBreakdownCSV экспортирует разбивку по категориям
func (h *ReportHandler) exportCategoryBreakdownCSV(writer *csv.Writer, r *report.Report) error {
	// Заголовки
	headers := []string{"Category", "Amount", "Percentage", "Transaction Count"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Данные
	for _, item := range r.Data.CategoryBreakdown {
		row := []string{
			item.CategoryName,
			fmt.Sprintf("%.2f", item.Amount),
			fmt.Sprintf("%.1f%%", item.Percentage),
			strconv.Itoa(item.Count),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	// Итого
	totalRow := []string{
		"TOTAL",
		fmt.Sprintf("%.2f", r.Data.TotalExpenses),
		"100.0%",
		"",
	}
	return writer.Write(totalRow)
}

// exportDailyBreakdownCSV экспортирует дневную разбивку
func (h *ReportHandler) exportDailyBreakdownCSV(writer *csv.Writer, r *report.Report) error {
	// Заголовки
	headers := []string{"Date", "Income", "Expenses", "Balance"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Данные
	for _, item := range r.Data.DailyBreakdown {
		row := []string{
			item.Date.Format("2006-01-02"),
			fmt.Sprintf("%.2f", item.Income),
			fmt.Sprintf("%.2f", item.Expenses),
			fmt.Sprintf("%.2f", item.Balance),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportBudgetComparisonCSV экспортирует сравнение с бюджетом
func (h *ReportHandler) exportBudgetComparisonCSV(writer *csv.Writer, r *report.Report) error {
	// Заголовки
	headers := []string{"Budget", "Planned", "Actual", "Difference", "Percentage"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Данные
	for _, item := range r.Data.BudgetComparison {
		row := []string{
			item.BudgetName,
			fmt.Sprintf("%.2f", item.Planned),
			fmt.Sprintf("%.2f", item.Actual),
			fmt.Sprintf("%.2f", item.Difference),
			fmt.Sprintf("%.1f%%", item.Percentage),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// renderReportFormWithErrors отображает форму с ошибками
func (h *ReportHandler) renderReportFormWithErrors(
	c echo.Context,
	form webModels.ReportForm,
	title string,
) error {
	reportTypeOptions := webModels.GetReportTypeOptions()

	pageData := &PageData{
		Title: title,
		Messages: []Message{
			{Type: "error", Text: "Проверьте правильность заполнения формы"},
		},
	}

	data := map[string]any{
		"PageData":          pageData,
		"Form":              form,
		"ReportTypeOptions": reportTypeOptions,
	}

	template := "pages/reports/new"
	if title == "Edit Report" {
		template = "pages/reports/edit"
	}

	return h.renderPage(c, template, data)
}

// convertReportDataToStandard конвертирует специфичные DTO в стандартный report.Data формат
func (h *ReportHandler) convertReportDataToStandard(reportData any, reportType report.Type) report.Data {
	switch reportType {
	case report.TypeExpenses:
		return h.convertExpenseReportData(reportData)
	case report.TypeIncome:
		return h.convertIncomeReportData(reportData)
	case report.TypeBudget:
		return h.convertBudgetReportData(reportData)
	case report.TypeCashFlow:
		return h.convertCashFlowReportData(reportData)
	case report.TypeCategoryBreak:
		return h.convertCategoryReportData(reportData)
	default:
		return report.Data{}
	}
}

// convertExpenseReportData конвертирует данные отчета по расходам
func (h *ReportHandler) convertExpenseReportData(reportData any) report.Data {
	expenseReport, ok := reportData.(*dto.ExpenseReportDTO)
	if !ok {
		return report.Data{}
	}

	categoryBreakdown := h.convertCategoryBreakdownItems(expenseReport.CategoryBreakdown)
	topExpenses := h.convertTopExpensesItems(expenseReport.TopExpenses)

	return report.Data{
		TotalExpenses:     expenseReport.TotalExpenses,
		CategoryBreakdown: categoryBreakdown,
		TopExpenses:       topExpenses,
	}
}

// convertIncomeReportData конвертирует данные отчета по доходам
func (h *ReportHandler) convertIncomeReportData(reportData any) report.Data {
	incomeReport, ok := reportData.(*dto.IncomeReportDTO)
	if !ok {
		return report.Data{}
	}

	categoryBreakdown := h.convertCategoryBreakdownItems(incomeReport.CategoryBreakdown)

	return report.Data{
		TotalIncome:       incomeReport.TotalIncome,
		CategoryBreakdown: categoryBreakdown,
	}
}

// convertBudgetReportData конвертирует данные отчета по бюджету
func (h *ReportHandler) convertBudgetReportData(reportData any) report.Data {
	budgetReport, ok := reportData.(*dto.BudgetComparisonDTO)
	if !ok {
		return report.Data{}
	}

	budgetComparison := make([]report.BudgetComparisonItem, len(budgetReport.Categories))
	for i, item := range budgetReport.Categories {
		budgetComparison[i] = report.BudgetComparisonItem{
			BudgetID:   item.CategoryID, // Используем CategoryID как BudgetID
			BudgetName: item.CategoryName,
			Planned:    item.BudgetAmount,
			Actual:     item.ActualAmount,
			Difference: item.Variance,
			Percentage: item.Utilization,
		}
	}

	return report.Data{
		TotalExpenses:    budgetReport.TotalSpent,
		BudgetComparison: budgetComparison,
	}
}

// convertCashFlowReportData конвертирует данные отчета по денежному потоку
func (h *ReportHandler) convertCashFlowReportData(reportData any) report.Data {
	cashFlowReport, ok := reportData.(*dto.CashFlowReportDTO)
	if !ok {
		return report.Data{}
	}

	dailyBreakdown := make([]report.DailyReportItem, len(cashFlowReport.DailyFlow))
	for i, item := range cashFlowReport.DailyFlow {
		dailyBreakdown[i] = report.DailyReportItem{
			Date:     item.Date,
			Income:   item.Inflow,
			Expenses: item.Outflow,
			Balance:  item.Balance,
		}
	}

	return report.Data{
		TotalIncome:    cashFlowReport.TotalInflows,
		TotalExpenses:  cashFlowReport.TotalOutflows,
		NetIncome:      cashFlowReport.NetCashFlow,
		DailyBreakdown: dailyBreakdown,
	}
}

// convertCategoryReportData конвертирует данные отчета по категориям
func (h *ReportHandler) convertCategoryReportData(reportData any) report.Data {
	categoryReport, ok := reportData.(*dto.CategoryBreakdownDTO)
	if !ok {
		return report.Data{}
	}

	categoryBreakdown := make([]report.CategoryReportItem, len(categoryReport.Categories))
	for i, item := range categoryReport.Categories {
		categoryBreakdown[i] = report.CategoryReportItem{
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			Amount:       item.TotalAmount,
			Percentage:   item.Percentage,
			Count:        item.TransactionCount,
		}
	}

	return report.Data{
		CategoryBreakdown: categoryBreakdown,
	}
}

// convertCategoryBreakdownItems конвертирует элементы разбивки по категориям
func (h *ReportHandler) convertCategoryBreakdownItems(
	items []dto.CategoryBreakdownItemDTO,
) []report.CategoryReportItem {
	categoryBreakdown := make([]report.CategoryReportItem, len(items))
	for i, item := range items {
		categoryBreakdown[i] = report.CategoryReportItem{
			CategoryID:   item.CategoryID,
			CategoryName: item.CategoryName,
			Amount:       item.Amount,
			Percentage:   item.Percentage,
			Count:        item.Count,
		}
	}
	return categoryBreakdown
}

// convertTopExpensesItems конвертирует элементы топ расходов
func (h *ReportHandler) convertTopExpensesItems(items []dto.TransactionSummaryDTO) []report.TransactionReportItem {
	topExpenses := make([]report.TransactionReportItem, len(items))
	for i, item := range items {
		topExpenses[i] = report.TransactionReportItem{
			ID:          item.ID,
			Amount:      item.Amount,
			Description: item.Description,
			Category:    item.Category,
			Date:        item.Date,
		}
	}
	return topExpenses
}

// getReportServiceErrorMessage возвращает пользовательское сообщение об ошибке
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

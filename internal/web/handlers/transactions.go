package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/middleware"
	webModels "family-budget-service/internal/web/models"
)

const (
	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
	DefaultPageSize        = 50
	MaxPageSize            = 100
)

// TransactionHandler обрабатывает HTTP запросы для транзакций
type TransactionHandler struct {
	*BaseHandler

	validator *validator.Validate
}

// NewTransactionHandler создает новый обработчик транзакций
func NewTransactionHandler(
	repositories *handlers.Repositories,
	services *services.Services,
	cookieSecure bool,
) *TransactionHandler {
	return &TransactionHandler{
		BaseHandler: NewBaseHandler(repositories, services, cookieSecure),
		validator:   validator.New(),
	}
}

// TransactionTableData — контракт данных таблицы транзакций.
// Один и тот же тип получают и страница /transactions, и HTMX-фрагмент
// components/transaction_table: шаблон таблицы читает `.Transactions`,
// `.Filters` и `.Pagination` из корня контекста в обоих случаях.
type TransactionTableData struct {
	Transactions []webModels.TransactionViewModel
	Filters      *webModels.TransactionFilters
	Pagination   webModels.TransactionListResponse
}

// transactionIndexData — данные страницы /transactions: к таблице добавляются
// общие данные страницы и опции селекта категорий для формы фильтра.
type transactionIndexData struct {
	*PageData
	TransactionTableData

	CategoryOptions []webModels.CategorySelectOption
}

// Index отображает список транзакций с фильтрами
func (h *TransactionHandler) Index(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим фильтры из query parameters
	var filters webModels.TransactionFilters
	if bindErr := c.Bind(&filters); bindErr != nil {
		// Игнорируем ошибки привязки для фильтров
		filters = webModels.TransactionFilters{}
	}

	// Устанавливаем пагинацию по умолчанию
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 || filters.PageSize > MaxPageSize {
		filters.PageSize = DefaultPageSize
	}

	// Получаем транзакции через сервис
	filterDTO, err := h.buildTransactionFilterDTO(filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid filter parameters")
	}

	transactions, err := h.services.Transaction.GetAllTransactions(
		c.Request().Context(),
		filterDTO,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get transactions")
	}

	// Конвертируем в view модели
	transactionVMs, err := h.convertTransactionsToViewModels(c.Request().Context(), transactions)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to prepare transaction data")
	}

	// Получаем категории для фильтра
	categories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	// Конвертируем категории в опции для селекта
	categoryOptions := buildCategorySelectOptions(categories)

	// Рассчитываем пагинацию по реальному количеству записей под фильтр:
	// len(transactionVMs) — это размер уже урезанной LIMIT'ом страницы, по нему
	// HasNext всегда был бы false и вторая страница оставалась недостижимой.
	total, err := h.services.Transaction.CountTransactions(c.Request().Context(), filterDTO)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get transactions")
	}
	pagination := h.calculatePagination(total, filters.Page, filters.PageSize)

	// Встраиваем *PageData, а не кладём его в map под ключом "PageData":
	// шаблон шапки читает `.CurrentUser` и `.CSRFToken` из корня контекста,
	// а `{{.PageData.X}}` продолжает работать благодаря имени встроенного поля.
	data := transactionIndexData{
		PageData: h.buildPageData(c, titleTransactions),
		TransactionTableData: TransactionTableData{
			Transactions: transactionVMs,
			Filters:      &filters,
			Pagination:   pagination,
		},
		CategoryOptions: categoryOptions,
	}

	return h.renderPage(c, "pages/transactions/index", data)
}

// New отображает форму создания новой транзакции
func (h *TransactionHandler) New(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Получаем категории для селекта
	categories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	categoryOptions := buildCategorySelectOptions(categories)

	// Предзаполняем форму с текущей датой
	form := webModels.TransactionForm{
		Date: time.Now().Format("2006-01-02"),
		Type: TransactionTypeExpense, // По умолчанию расход
	}

	// CSRF-токен приходит из PageData и промотируется в корень контекста.
	data := struct {
		*PageData

		Form            webModels.TransactionForm
		CategoryOptions []webModels.CategorySelectOption
	}{
		PageData:        h.buildPageData(c, titleNewTransaction),
		Form:            form,
		CategoryOptions: categoryOptions,
	}

	return h.renderPage(c, "pages/transactions/new", data)
}

// Create создает новую транзакцию
func (h *TransactionHandler) Create(c echo.Context) error {
	// Получаем данные пользователя из сессии
	sessionData, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим данные формы
	var form webModels.TransactionForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		// Возвращаем форму с ошибками валидации
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			// Для HTMX возвращаем только errors partial
			return h.renderPartial(c, "components/form_errors", map[string]any{
				tplKeyErrors: validationErrors,
			})
		}

		// Для обычных запросов возвращаем форму заново
		return h.renderTransactionFormWithErrors(c, form, validationErrors, nil)
	}

	// Создаем DTO для сервиса
	createDTO, err := h.buildCreateTransactionDTO(form, sessionData.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction data")
	}

	// Создаем транзакцию через сервис
	_, err = h.services.Transaction.CreateTransaction(c.Request().Context(), createDTO)
	if err != nil {
		// Текст ошибки сервиса наружу не отдаём: в нём оказываются имена таблиц
		// и колонок SQLite и текст ограничений. Клиенту — общая формулировка,
		// подробности (включая UserID) — в лог.
		c.Logger().Errorf("create transaction failed (UserID: %s): %q", sessionData.UserID, err.Error())

		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create transaction")
	}

	// Успешное создание - редирект
	if h.IsHTMXRequest(c) {
		// Для HTMX запросов используем Hx-Redirect
		c.Response().Header().Set("Hx-Redirect", "/transactions")
		return c.NoContent(http.StatusOK)
	}

	// Для обычных запросов - стандартный редирект
	return h.redirect(c, "/transactions")
}

// Edit отображает форму редактирования транзакции
func (h *TransactionHandler) Edit(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID транзакции
	id := c.Param("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction ID")
	}

	// Получаем транзакцию
	existing, err := h.services.Transaction.GetTransactionByID(c.Request().Context(), transactionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Transaction not found")
	}

	// In single-family model, all transactions belong to the family
	// No additional access check needed

	// Получаем категории для селекта
	categories, err := h.services.Category.GetCategories(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get categories")
	}

	categoryOptions := buildCategorySelectOptions(categories)

	// Создаем форму из данных транзакции
	form := webModels.TransactionForm{
		Amount:      fmt.Sprintf("%.2f", existing.Amount),
		Type:        string(existing.Type),
		Description: existing.Description,
		CategoryID:  existing.CategoryID.String(),
		Date:        existing.Date.Format("2006-01-02"),
		Tags:        strings.Join(existing.Tags, ", "),
	}

	// CSRF-токен приходит из PageData и промотируется в корень контекста.
	data := struct {
		*PageData

		Form            webModels.TransactionForm
		Transaction     *transaction.Transaction
		CategoryOptions []webModels.CategorySelectOption
	}{
		PageData:        h.buildPageData(c, titleEditTransaction),
		Form:            form,
		Transaction:     existing,
		CategoryOptions: categoryOptions,
	}

	return h.renderPage(c, "pages/transactions/edit", data)
}

// Update обновляет существующую транзакцию
func (h *TransactionHandler) Update(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	// Парсим ID транзакции
	id := c.Param("id")
	transactionID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction ID")
	}

	// Проверяем, что транзакция существует
	existingTransaction, err := h.services.Transaction.GetTransactionByID(c.Request().Context(), transactionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Transaction not found")
	}

	// In single-family model, all transactions belong to the family
	// No additional access check needed

	// Парсим данные формы
	var form webModels.TransactionForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	// Валидируем форму
	if validationErr := h.validator.Struct(form); validationErr != nil {
		// Возвращаем форму с ошибками валидации
		validationErrors := webModels.GetValidationErrors(validationErr)

		if h.IsHTMXRequest(c) {
			// Для HTMX возвращаем только errors partial
			return h.renderPartial(c, "components/form_errors", map[string]any{
				tplKeyErrors: validationErrors,
			})
		}

		// Для обычных запросов возвращаем форму заново
		return h.renderTransactionFormWithErrors(c, form, validationErrors, existingTransaction)
	}

	// Создаем DTO для обновления
	updateDTO, err := h.buildUpdateTransactionDTO(form)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction data")
	}

	// Обновляем транзакцию через сервис
	_, err = h.services.Transaction.UpdateTransaction(c.Request().Context(), transactionID, updateDTO)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	// Успешное обновление - редирект
	if h.IsHTMXRequest(c) {
		// Для HTMX запросов используем Hx-Redirect
		c.Response().Header().Set("Hx-Redirect", "/transactions")
		return c.NoContent(http.StatusOK)
	}

	// Для обычных запросов - стандартный редирект
	return h.redirect(c, "/transactions")
}

// Delete удаляет транзакцию
func (h *TransactionHandler) Delete(c echo.Context) error {
	return h.handleDelete(c, DeleteEntityParams{
		EntityName: "transaction",
		GetEntityFunc: func(ctx echo.Context, entityID uuid.UUID) (any, error) {
			return h.services.Transaction.GetTransactionByID(ctx.Request().Context(), entityID)
		},
		DeleteEntityFunc: func(ctx echo.Context, entityID uuid.UUID) error {
			return h.services.Transaction.DeleteTransaction(ctx.Request().Context(), entityID)
		},
		GetErrorMsgFunc: h.getTransactionServiceErrorMessage,
		RedirectURL:     "/transactions",
	})
}

// BulkDelete удаляет несколько транзакций
func (h *TransactionHandler) BulkDelete(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	var form webModels.BulkOperationForm
	if bindErr := c.Bind(&form); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	if len(form.TransactionIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No transactions selected for deletion")
	}

	// Парсим ID транзакций
	var transactionIDs []uuid.UUID
	for _, idStr := range form.TransactionIDs {
		transactionID, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid transaction ID")
		}
		transactionIDs = append(transactionIDs, transactionID)
	}

	// Проверяем каждую транзакцию и удаляем
	deleted := 0
	errors := []string{}
	for _, transactionID := range transactionIDs {
		// Проверяем существование
		_, getErr := h.services.Transaction.GetTransactionByID(c.Request().Context(), transactionID)
		if getErr != nil {
			errors = append(errors, fmt.Sprintf("Transaction %s not found", transactionID))
			continue
		}

		// In single-family model, all transactions belong to the family
		// No additional access check needed

		// Удаляем
		deleteErr := h.services.Transaction.DeleteTransaction(
			c.Request().Context(),
			transactionID,
		)
		if deleteErr != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete transaction %s", transactionID))
			continue
		}
		deleted++
	}

	if h.IsHTMXRequest(c) {
		// Возвращаем сообщение о результате
		message := fmt.Sprintf("Successfully deleted %d transactions", deleted)
		if len(errors) > 0 {
			message += fmt.Sprintf(". Errors: %s", strings.Join(errors, "; "))
		}

		alertType := "success"
		if len(errors) > 0 {
			alertType = "warning"
		}

		return h.renderPartial(c, "components/alert", map[string]any{
			"Type":    alertType,
			"Message": message,
		})
	}

	return h.redirect(c, "/transactions")
}

// Filter применяет фильтры к списку транзакций (HTMX)
func (h *TransactionHandler) Filter(c echo.Context) error {
	return h.renderTransactionTable(c)
}

// List возвращает страницу списка транзакций для пагинации (HTMX).
//
// Отдаёт тот же фрагмент, что и Filter, по двум причинам:
//   - кнопки пагинации в components/transaction_table.html свапают ответ в
//     <section id="transactions-list">, поэтому голые <tr> (components/transaction_rows)
//     выбрасывались бы парсером HTML и таблица исчезала бы вместе с пагинацией;
//   - ссылки несут активные фильтры, и их нужно применить, иначе переход на
//     соседнюю страницу молча сбрасывал бы фильтрацию.
func (h *TransactionHandler) List(c echo.Context) error {
	return h.renderTransactionTable(c)
}

// renderTransactionTable рендерит components/transaction_table по фильтрам из запроса.
func (h *TransactionHandler) renderTransactionTable(c echo.Context) error {
	// Проверяем авторизацию пользователя
	_, err := middleware.GetUserFromContext(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to get user session")
	}

	var filters webModels.TransactionFilters
	if bindErr := c.Bind(&filters); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid filter data")
	}

	// Устанавливаем пагинацию по умолчанию
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.PageSize <= 0 || filters.PageSize > MaxPageSize {
		filters.PageSize = DefaultPageSize
	}

	// Получаем транзакции через сервис
	filterDTO, err := h.buildTransactionFilterDTO(filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid filter parameters")
	}

	transactions, err := h.services.Transaction.GetAllTransactions(
		c.Request().Context(),
		filterDTO,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get transactions")
	}

	// Конвертируем в view модели
	transactionVMs, err := h.convertTransactionsToViewModels(c.Request().Context(), transactions)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to prepare transaction data")
	}

	// Рассчитываем пагинацию по реальному количеству записей под фильтр
	total, err := h.services.Transaction.CountTransactions(c.Request().Context(), filterDTO)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get transactions")
	}
	pagination := h.calculatePagination(total, filters.Page, filters.PageSize)

	data := TransactionTableData{
		Transactions: transactionVMs,
		Filters:      &filters,
		Pagination:   pagination,
	}

	return h.renderPartial(c, "components/transaction_table", data)
}

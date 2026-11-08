package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
)

const apiVersion = "v1"

const (
	// defaultLimit — размер страницы, если limit не задан.
	defaultLimit = 50
	// maxLimit — потолок страницы на всех списках API (A-08).
	maxLimit = 200
)

// pageParams — разобранные limit/offset одного запроса-списка.
type pageParams struct {
	Limit  int
	Offset int
}

// newAPIValidator — валидатор всех API-хендлеров: имя поля в error.details берётся
// из json-тега, иначе клиент получал бы имя поля Go (StartDate вместо start_date).
// WithRequiredStructEnabled обязателен: без него `required` на вложенной структуре
// (date.Date) молча ничего не проверяет и пропущенная дата доходит до репозитория как 500.
func newAPIValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	auth.RegisterPasswordValidation(v)
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return field.Name
		}

		return name
	})

	return v
}

// familyToday — сегодняшняя дата в часовом поясе семьи (A-06). Семья не прочиталась —
// UTC: в единственном месте вызова это выбор набора активных бюджетов, не отказ запроса.
func familyToday(ctx context.Context, families FamilyRepository) date.Date {
	if families == nil {
		return date.Today(time.UTC)
	}

	family, err := families.Get(ctx)
	if err != nil || family == nil {
		return date.Today(time.UTC)
	}

	return date.Today(family.Location())
}

// isNilClientID — клиентский id из одних нулей: идемпотентный поиск по нему ничего
// не находит, а репозиторий отбивает такой id уже как 500.
func isNilClientID(id *uuid.UUID) bool {
	return id != nil && *id == uuid.Nil
}

// bodyDetail — деталь для ошибки, не привязанной к конкретному полю запроса.
func bodyDetail(code, message string) ErrorDetail {
	return ErrorDetail{Field: fieldBody, Message: message, Code: code}
}

// ignoreWritten гасит sentinel: ответ уже отправлен тем, кто вернул ошибку.
func ignoreWritten(err error) error {
	if errors.Is(err, errResponseAlreadyWritten) {
		return nil
	}

	return err
}

// pageSlice режет окно [offset, offset+limit) для списков, которые репозиторий отдаёт целиком.
func pageSlice[T any](items []T, page pageParams) []T {
	if page.Offset >= len(items) {
		return []T{}
	}

	end := page.Offset + page.Limit
	if end > len(items) {
		end = len(items)
	}

	return items[page.Offset:end]
}

// ErrorCodeForStatus — error.code, соответствующий HTTP-статусу; неизвестный статус
// (503 от таймаута и т. п.) получает код из http.StatusText, чтобы код и статус не расходились.
func ErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusUnprocessableEntity:
		return ErrCodeValidationError
	case http.StatusRequestEntityTooLarge:
		return ErrCodePayloadTooLarge
	case http.StatusTooManyRequests:
		return ErrCodeRateLimited
	case http.StatusInternalServerError:
		return ErrCodeInternal
	}

	text := http.StatusText(status)
	if text == "" {
		return ErrCodeInternal
	}
	return strings.ToUpper(strings.ReplaceAll(text, " ", "_"))
}

// buildValidationErrors раскладывает ошибки validator по полям; на нераспознанной
// ошибке возвращает одну деталь с её текстом, чтобы details никогда не был пустым.
func buildValidationErrors(err error) []ErrorDetail {
	var fieldErrs validator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return []ErrorDetail{bodyDetail(ErrCodeValidationError, err.Error())}
	}

	details := make([]ErrorDetail, 0, len(fieldErrs))
	for _, fieldErr := range fieldErrs {
		details = append(details, ErrorDetail{
			Field:   fieldErr.Field(),
			Message: validationMessage(fieldErr),
			Code:    ErrCodeValidationError,
		})
	}

	return details
}

// validationMessage переводит правило валидатора в текст для клиента.
func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "oneof":
		return "must be one of: " + fieldErr.Param()
	case "email":
		return "must be a valid email"
	case auth.PasswordTag:
		return auth.ErrInvalidPassword.Error()
	}

	if fieldErr.Param() != "" {
		return "must satisfy " + fieldErr.Tag() + "=" + fieldErr.Param()
	}

	return "must satisfy " + fieldErr.Tag()
}

// isNotFoundError отличает «сущности нет» от настоящего сбоя: спека требует 404, а не 500.
func isNotFoundError(err error) bool {
	return errors.Is(err, services.ErrTransactionNotFound) ||
		errors.Is(err, services.ErrBudgetNotFoundService) ||
		errors.Is(err, services.ErrCategoryNotFound)
}

const (
	headerContentType = "Content-Type"
	mimeJSON          = "application/json"
	msgInvalidDate    = "must be a date in YYYY-MM-DD format"
)

var errUnsupportedMediaType = errors.New("content-type must be application/json")

// decodeJSON читает тело: пустое тело (ContentLength == 0) пропускается, иное требует application/json;
// неизвестные поля и хвост после значения разрешены, любая другая ошибка — повод для 400.
func decodeJSON(r *http.Request, v any) error {
	if r.ContentLength == 0 || r.Body == nil {
		return nil
	}
	if base, _, _ := strings.Cut(r.Header.Get(headerContentType), ";"); strings.TrimSpace(base) != mimeJSON {
		return errUnsupportedMediaType
	}

	return json.NewDecoder(r.Body).Decode(v)
}

// writeJSON: ошибка Encode приходит после WriteHeader, ответить на неё уже нечем — её увидит только лог.
func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set(headerContentType, mimeJSON)
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

func apiResponseMeta(w http.ResponseWriter) ResponseMeta {
	return ResponseMeta{
		RequestID: w.Header().Get(observability.RequestIDHeader),
		Timestamp: time.Now(),
		Version:   apiVersion,
	}
}

func respondAPI[T any](w http.ResponseWriter, status int, data T) error {
	return writeJSON(w, status, APIResponse[T]{Data: data, Meta: apiResponseMeta(w)})
}

// RespondError пишет envelope ошибки; пустой code выводится из статуса.
func RespondError(w http.ResponseWriter, status int, code, message string, details ...ErrorDetail) error {
	if code == "" {
		code = ErrorCodeForStatus(status)
	}

	return writeJSON(w, status, ErrorResponse{
		Error: APIError{Code: code, Message: message, Details: details},
		Meta:  apiResponseMeta(w),
	})
}

// respondBindError — дата, не прошедшая разбор, — 422 на поле, остальное — 400.
func respondBindError(w http.ResponseWriter, err error) error {
	if field, ok := date.JSONField(err); ok {
		if field == "" {
			field = fieldDate
		}

		return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: field, Message: msgInvalidDate, Code: ErrCodeValidationError})
	}

	return RespondError(w, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
		bodyDetail(ErrCodeInvalidRequest, err.Error()))
}

// parsePagination — разбор limit/offset; ответ 422 пишется сразу, вернётся errResponseAlreadyWritten.
func parsePagination(w http.ResponseWriter, r *http.Request) (pageParams, error) {
	page := pageParams{Limit: defaultLimit}
	q := r.URL.Query()

	if raw := q.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxLimit {
			return page, writeInvalidQueryParam(w, "limit", raw,
				"must be an integer between 1 and "+strconv.Itoa(maxLimit))
		}
		page.Limit = limit
	}

	if raw := q.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return page, writeInvalidQueryParam(w, "offset", raw, "must be a non-negative integer")
		}
		page.Offset = offset
	}

	return page, nil
}

func writeInvalidQueryParam(w http.ResponseWriter, param, value, reason string) error {
	if err := RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: param, Message: reason + `, got "` + value + `"`, Code: ErrCodeInvalidQueryParam},
	); err != nil {
		return err
	}

	return errResponseAlreadyWritten
}

func respondList[T any](w http.ResponseWriter, items []T, page pageParams, total int) error {
	meta := apiResponseMeta(w)
	meta.Pagination = &PaginationMeta{Limit: page.Limit, Offset: page.Offset, Total: total}

	if items == nil {
		items = []T{}
	}

	return writeJSON(w, http.StatusOK, APIResponse[[]T]{Data: items, Meta: meta})
}

func noContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func respondValidationErrors(w http.ResponseWriter, err error) error {
	return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		buildValidationErrors(err)...)
}

func respondPasswordPolicyError(w http.ResponseWriter) error {
	return RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: fieldNewPassword, Message: auth.ErrInvalidPassword.Error(), Code: ErrCodeValidationError})
}

// handleNotFoundError — 404 с кодом <ENTITY>_NOT_FOUND.
func handleNotFoundError(w http.ResponseWriter, entityType string) error {
	return RespondError(w, http.StatusNotFound, strings.ToUpper(entityType)+"_NOT_FOUND", entityType+" not found")
}

// respondClientID — идемпотентный повтор POST с клиентским id: 200 с найденной записью.
func respondClientID[E, R any](
	w http.ResponseWriter,
	r *http.Request,
	id *uuid.UUID,
	find func(*http.Request, uuid.UUID) (E, bool),
	build func(E) R,
) (bool, error) {
	if isNilClientID(id) {
		return true, RespondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError,
			ErrMessageValidationFailed,
			ErrorDetail{Field: fieldID, Message: "must not be the nil UUID", Code: ErrCodeValidationError})
	}

	if id == nil {
		return false, nil
	}

	existing, found := find(r, *id)
	if !found {
		return false, nil
	}

	return true, respondAPI(w, http.StatusOK, build(existing))
}

// parseArchivedParam — query-параметр archived.
func parseArchivedParam(w http.ResponseWriter, r *http.Request) (bool, error) {
	raw := r.URL.Query().Get("archived")
	if raw == "" {
		return false, nil
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, writeInvalidQueryParam(w, "archived", raw, "must be true or false")
	}

	return parsed, nil
}

// handleIDParseError — 400 на path-параметр id, не разобравшийся как UUID.
func handleIDParseError(w http.ResponseWriter, entityType string) error {
	return RespondError(w, http.StatusBadRequest, ErrCodeInvalidID,
		"Invalid "+strings.ToLower(entityType)+" ID format")
}

// deleteEntity — общий DELETE /:id: 400 на плохой id, 404 на «нет такой», 204 при успехе.
func deleteEntity(w http.ResponseWriter, r *http.Request, deleter func(uuid.UUID) error, entityType string) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return handleIDParseError(w, entityType)
	}

	if err = deleter(id); err != nil {
		if isNotFoundError(err) {
			return handleNotFoundError(w, entityType)
		}

		return RespondError(w, http.StatusInternalServerError, "DELETE_FAILED",
			"Failed to delete "+strings.ToLower(entityType))
	}

	return noContent(w)
}

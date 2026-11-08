package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/observability"
)

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

func respondError(w http.ResponseWriter, status int, code, message string, details ...ErrorDetail) error {
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

		return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
			ErrorDetail{Field: field, Message: msgInvalidDate, Code: ErrCodeValidationError})
	}

	return respondError(w, http.StatusBadRequest, ErrCodeInvalidRequest, ErrMessageInvalidRequest,
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
	if err := respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
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

// RespondAPIError пишет общий envelope для ошибки вне хендлера; пустой code выводится из статуса.
func RespondAPIError(w http.ResponseWriter, status int, code, message string) error {
	if code == "" {
		code = ErrorCodeForStatus(status)
	}

	return respondError(w, status, code, message)
}

func noContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func respondUnauthorized(w http.ResponseWriter) error {
	return respondError(w, http.StatusUnauthorized, ErrCodeUnauthorized, ErrMessageUnauthorized)
}

func respondValidationErrors(w http.ResponseWriter, err error) error {
	return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		buildValidationErrors(err)...)
}

func respondPasswordPolicyError(w http.ResponseWriter) error {
	return respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError, ErrMessageValidationFailed,
		ErrorDetail{Field: fieldNewPassword, Message: auth.ErrInvalidPassword.Error(), Code: ErrCodeValidationError})
}

// handleNotFoundError — 404 с кодом <ENTITY>_NOT_FOUND.
func handleNotFoundError(w http.ResponseWriter, entityType string) error {
	return respondError(w, http.StatusNotFound, strings.ToUpper(entityType)+"_NOT_FOUND", entityType+" not found")
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
		return true, respondError(w, http.StatusUnprocessableEntity, ErrCodeValidationError,
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
	return respondError(w, http.StatusBadRequest, ErrCodeInvalidID,
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

		return respondError(w, http.StatusInternalServerError, "DELETE_FAILED",
			"Failed to delete "+strings.ToLower(entityType))
	}

	return noContent(w)
}

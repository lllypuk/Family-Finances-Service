package handlers

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/date"
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

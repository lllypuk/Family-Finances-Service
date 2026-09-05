# Error Handling - Единая система обработки ошибок

## 🎯 Принципы обработки ошибок

### Философия
- **Fail Fast**: Обнаруживайте ошибки как можно раньше
- **Explicit Errors**: Ошибки должны быть явными, не скрытыми
- **Contextual Information**: Ошибки должны содержать контекст
- **User-Friendly**: Понятные сообщения для пользователей
- **Developer-Friendly**: Подробная информация для разработчиков

### Уровни ошибок
1. **Пользовательские ошибки** - некорректный ввод данных
2. **Бизнес-ошибки** - нарушение бизнес-правил
3. **Системные ошибки** - проблемы инфраструктуры
4. **Критические ошибки** - требующие немедленного вмешательства

## 🏗️ Архитектура обработки ошибок

### Иерархия ошибок
```go
type AppError interface {
    Error() string
    Code() string
    StatusCode() int
    Details() map[string]any
    Cause() error
}

type BaseError struct {
    code       string
    message    string
    statusCode int
    details    map[string]any
    cause      error
}
```

### Типы ошибок
```go
// Валидационные ошибки
type ValidationError struct {
    BaseError
    Fields []FieldError
}

// Бизнес-ошибки
type BusinessError struct {
    BaseError
    BusinessRule string
}

// Системные ошибки
type SystemError struct {
    BaseError
    Component string
}

// Ошибки доступа
type AuthorizationError struct {
    BaseError
    Resource   string
    Permission string
}
```

## 📝 Коды ошибок

### Структура кода ошибки
```
{CATEGORY}_{SPECIFIC_ERROR}
```

### Категории ошибок

#### Валидация (VALIDATION_*)
- **VALIDATION_REQUIRED**: Обязательное поле отсутствует
- **VALIDATION_FORMAT**: Неверный формат данных
- **VALIDATION_RANGE**: Значение вне допустимого диапазона
- **VALIDATION_LENGTH**: Неверная длина строки
- **VALIDATION_UNIQUE**: Нарушение уникальности

#### Аутентификация (AUTH_*)
- **AUTH_REQUIRED**: Требуется аутентификация
- **AUTH_INVALID_TOKEN**: Недействительный токен
- **AUTH_EXPIRED_TOKEN**: Истекший токен
- **AUTH_INVALID_CREDENTIALS**: Неверные учетные данные
- **AUTH_ACCOUNT_LOCKED**: Заблокированная учетная запись

#### Авторизация (AUTHZ_*)
- **AUTHZ_INSUFFICIENT_PERMISSIONS**: Недостаточно прав
- **AUTHZ_RESOURCE_ACCESS_DENIED**: Доступ к ресурсу запрещен
- **AUTHZ_FAMILY_ACCESS_DENIED**: Доступ к семье запрещен
- **AUTHZ_OPERATION_NOT_ALLOWED**: Операция не разрешена

#### Ресурсы (RESOURCE_*)
- **RESOURCE_NOT_FOUND**: Ресурс не найден
- **RESOURCE_ALREADY_EXISTS**: Ресурс уже существует
- **RESOURCE_CONFLICT**: Конфликт ресурсов
- **RESOURCE_LOCKED**: Ресурс заблокирован

#### Бизнес-логика (BUSINESS_*)
- **BUSINESS_INSUFFICIENT_FUNDS**: Недостаточно средств
- **BUSINESS_BUDGET_EXCEEDED**: Превышен бюджет
- **BUSINESS_INVALID_OPERATION**: Недопустимая операция
- **BUSINESS_DEADLINE_PASSED**: Истек срок выполнения

#### Система (SYSTEM_*)
- **SYSTEM_DATABASE_ERROR**: Ошибка базы данных
- **SYSTEM_EXTERNAL_SERVICE_ERROR**: Ошибка внешнего сервиса
- **SYSTEM_RATE_LIMIT_EXCEEDED**: Превышен лимит запросов
- **SYSTEM_MAINTENANCE**: Система на обслуживании

## 🔧 Реализация в Go

### Базовая структура ошибки
```go
package errors

import (
    "fmt"
    "net/http"
)

type AppError struct {
    code       string
    message    string
    statusCode int
    details    map[string]any
    cause      error
}

func (e *AppError) Error() string {
    return e.message
}

func (e *AppError) Code() string {
    return e.code
}

func (e *AppError) StatusCode() int {
    return e.statusCode
}

func (e *AppError) Details() map[string]any {
    return e.details
}

func (e *AppError) Cause() error {
    return e.cause
}

func (e *AppError) WithDetails(details map[string]any) *AppError {
    e.details = details
    return e
}

func (e *AppError) WithCause(cause error) *AppError {
    e.cause = cause
    return e
}
```

### Фабричные методы
```go
// Валидационные ошибки
func ValidationError(message string, field string) *AppError {
    return &AppError{
        code:       "VALIDATION_ERROR",
        message:    message,
        statusCode: http.StatusBadRequest,
        details:    map[string]any{"field": field},
    }
}

// Ошибки "не найдено"
func NotFoundError(resource string, id string) *AppError {
    return &AppError{
        code:       "RESOURCE_NOT_FOUND",
        message:    fmt.Sprintf("%s with id %s not found", resource, id),
        statusCode: http.StatusNotFound,
        details:    map[string]any{"resource": resource, "id": id},
    }
}

// Ошибки авторизации
func UnauthorizedError(message string) *AppError {
    return &AppError{
        code:       "AUTH_REQUIRED",
        message:    message,
        statusCode: http.StatusUnauthorized,
    }
}

// Ошибки доступа
func ForbiddenError(resource string, operation string) *AppError {
    return &AppError{
        code:       "AUTHZ_INSUFFICIENT_PERMISSIONS",
        message:    fmt.Sprintf("Access denied to %s for operation %s", resource, operation),
        statusCode: http.StatusForbidden,
        details:    map[string]any{"resource": resource, "operation": operation},
    }
}

// Бизнес-ошибки
func BusinessError(code string, message string) *AppError {
    return &AppError{
        code:       code,
        message:    message,
        statusCode: http.StatusUnprocessableEntity,
    }
}

// Системные ошибки
func InternalError(message string, cause error) *AppError {
    return &AppError{
        code:       "SYSTEM_INTERNAL_ERROR",
        message:    "Internal server error",
        statusCode: http.StatusInternalServerError,
        cause:      cause,
        details:    map[string]any{"internal_message": message},
    }
}
```

### Middleware для обработки ошибок
```go
package middleware

import (
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "your-project/internal/errors"
)

func ErrorHandler() echo.MiddlewareFunc {
    return echo.Recover()
}

func CustomErrorHandler(err error, c echo.Context) {
    var appErr *errors.AppError

    switch e := err.(type) {
    case *errors.AppError:
        appErr = e
    case *echo.HTTPError:
        appErr = &errors.AppError{
            Code:       "HTTP_ERROR",
            Message:    e.Message.(string),
            StatusCode: e.Code,
        }
    default:
        appErr = errors.InternalError("Unexpected error", err)
    }

    response := buildErrorResponse(appErr, c)
    c.JSON(appErr.StatusCode(), response)
}

func HandleError(c echo.Context, err error) error {
    var appErr *errors.AppError

    switch e := err.(type) {
    case *errors.AppError:
        appErr = e
    default:
        // Конвертируем обычную ошибку в AppError
        appErr = errors.InternalError("Unexpected error", err)
        // Логируем внутренние ошибки
        log.Printf("Internal error: %v", err)
    }

    response := buildErrorResponse(appErr, c)
    return c.JSON(appErr.StatusCode(), response)
}

func buildErrorResponse(err *errors.AppError, c echo.Context) map[string]any {
    errorResponse := map[string]any{
        "code":    err.Code(),
        "message": err.Message(),
    }

    if details := err.Details(); details != nil {
        errorResponse["details"] = details
    }

    // В debug режиме добавляем stack trace
    if c.Echo().Debug && err.Cause() != nil {
        errorResponse["debug"] = map[string]any{
            "cause": err.Cause().Error(),
        }
    }

    return map[string]any{
        "error": errorResponse,
        "meta":  buildMeta(c),
    }
}

func buildMeta(c echo.Context) map[string]any {
    return map[string]any{
        "timestamp":  time.Now().UTC().Format(time.RFC3339),
        "request_id": c.Response().Header().Get(echo.HeaderXRequestID),
        "path":       c.Request().URL.Path,
        "method":     c.Request().Method,
    }
}
```

## 📊 Формат ошибок в API

Единственный способ отдать ошибку из `/api/v1` — `respondError` / `respondValidationErrors`
(`internal/application/handlers/helpers.go`); руками собранный конверт запрещён.
`error.details` — **массив** `{field, message, code}`, `meta` — `{request_id, timestamp, version}`
(плюс `pagination` в списках).

### Структура ответа с ошибкой
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "email",
        "message": "must be a valid email",
        "code": "VALIDATION_ERROR"
      }
    ]
  },
  "meta": {
    "request_id": "req_abc123",
    "timestamp": "2026-09-04T10:30:00Z",
    "version": "v1"
  }
}
```

Имя в `field` — имя поля из json-тега запроса (`start_date`, не `StartDate`): валидатор
хендлеров зарегистрирован через `newAPIValidator`. Для ошибок, не привязанных к полю,
используется `bodyDetail` с `field: "body"`.

## 🔍 Валидация данных

### Валидация входных данных
```go
package validation

import (
    "github.com/go-playground/validator/v10"
    "your-project/internal/errors"
)

type Validator struct {
    validate *validator.Validate
}

func NewValidator() *Validator {
    return &Validator{
        validate: validator.New(),
    }
}

func (v *Validator) ValidateStruct(s any) error {
    if err := v.validate.Struct(s); err != nil {
        return v.convertValidationError(err)
    }
    return nil
}

func (v *Validator) convertValidationError(err error) *errors.AppError {
    var fieldErrors []map[string]any

    for _, err := range err.(validator.ValidationErrors) {
        fieldError := map[string]any{
            "field":   err.Field(),
            "message": v.getErrorMessage(err),
            "code":    v.getErrorCode(err.Tag()),
            "value":   err.Value(),
        }
        fieldErrors = append(fieldErrors, fieldError)
    }

    return &errors.AppError{
        Code:       "VALIDATION_ERROR",
        Message:    "Validation failed",
        StatusCode: http.StatusBadRequest,
        Details:    map[string]any{"fields": fieldErrors},
    }
}

func (v *Validator) getErrorMessage(fe validator.FieldError) string {
    switch fe.Tag() {
    case "required":
        return fmt.Sprintf("%s is required", fe.Field())
    case "email":
        return "Must be a valid email address"
    case "min":
        return fmt.Sprintf("Must be at least %s characters", fe.Param())
    case "max":
        return fmt.Sprintf("Must be no more than %s characters", fe.Param())
    default:
        return "Invalid value"
    }
}
```

## 📝 Логирование ошибок

### Структурированное логирование
```go
package logging

import (
    "github.com/sirupsen/logrus"
    "your-project/internal/errors"
)

type Logger struct {
    logger *logrus.Logger
}

func NewLogger() *Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})
    return &Logger{logger: logger}
}

func (l *Logger) LogError(err *errors.AppError, context map[string]any) {
    entry := l.logger.WithFields(logrus.Fields{
        "error_code":    err.Code(),
        "status_code":   err.StatusCode(),
        "error_message": err.Message(),
    })

    // Добавляем контекст
    for key, value := range context {
        entry = entry.WithField(key, value)
    }

    // Добавляем детали ошибки
    if details := err.Details(); details != nil {
        entry = entry.WithField("error_details", details)
    }

    // Добавляем причину ошибки
    if cause := err.Cause(); cause != nil {
        entry = entry.WithField("error_cause", cause.Error())
    }

    // Определяем уровень логирования
    switch {
    case err.StatusCode() >= 500:
        entry.Error("Server error occurred")
    case err.StatusCode() >= 400:
        entry.Warn("Client error occurred")
    default:
        entry.Info("Request processed with error")
    }
}
```

## 🎯 Лучшие практики

### DO ✅
- **Используйте типизированные ошибки** для разных категорий
- **Предоставляйте контекст** - что пошло не так и почему
- **Логируйте системные ошибки** с полной информацией
- **Возвращайте понятные сообщения** пользователям
- **Включайте request_id** для трассировки
- **Валидируйте данные** на входе в систему
- **Обрабатывайте ошибки** на каждом уровне приложения

### DON'T ❌
- **Не игнорируйте ошибки** - всегда обрабатывайте
- **Не показывайте внутренние детали** пользователям
- **Не логируйте пользовательские ошибки** как критические
- **Не используйте panic** для обычных ошибок
- **Не возвращайте общие сообщения** типа "Something went wrong"
- **Не забывайте проверять nil** при работе с указателями

### Примеры использования
```go
// ✅ Правильно
func (s *FamilyService) GetFamily(ctx context.Context, id string) (*Family, error) {
    family, err := s.repo.GetByID(ctx, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, errors.NotFoundError("family", id)
        }
        return nil, errors.InternalError("Failed to get family", err)
    }
    return family, nil
}

// ❌ Неправильно
func (s *FamilyService) GetFamily(ctx context.Context, id string) (*Family, error) {
    family, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err // Не обрабатываем ошибку
    }
    return family, nil
}
```

## 🔧 Тестирование ошибок

### Unit тесты для ошибок
```go
func TestFamilyService_GetFamily_NotFound(t *testing.T) {
    // Arrange
    mockRepo := &MockFamilyRepository{}
    service := NewFamilyService(mockRepo)

    mockRepo.On("GetByID", mock.Anything, "invalid-id").
        Return(nil, sql.ErrNoRows)

    // Act
    family, err := service.GetFamily(context.Background(), "invalid-id")

    // Assert
    assert.Nil(t, family)
    assert.NotNil(t, err)

    var appErr *errors.AppError
    assert.True(t, errors.As(err, &appErr))
    assert.Equal(t, "RESOURCE_NOT_FOUND", appErr.Code())
    assert.Equal(t, http.StatusNotFound, appErr.StatusCode())
}
```

## 📊 Мониторинг ошибок

### Метрики для мониторинга
- **Количество ошибок по коду** - для выявления проблемных областей
- **Процент ошибок от общего числа запросов**
- **Время ответа при ошибках**
- **Топ ошибок по частоте возникновения**

### Алерты
- **5xx ошибки** > 1% от общего трафика
- **Критические бизнес-ошибки**
- **Недоступность внешних сервисов**
- **Превышение времени ответа**

---

*Документ создан: 2025*
*Владелец: Backend Team*
*Регулярность обновлений: при изменении архитектуры*

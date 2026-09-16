// Package recognize превращает ответ модели по скриншотам банка в кандидатов операций; о транспорте модели не знает.
package recognize

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/transaction"
)

const (
	MaxImages     = 5
	MaxItems      = 50
	MaxImageBytes = 2 << 20
	MaxPixels     = 16_000_000

	maxDescription = 200
)

var (
	// ErrDisabled — плечо модели не настроено.
	ErrDisabled = errors.New("recognize: recognition is disabled")
	// ErrUnavailable — модель не ответила; конкретика и срок повтора в UnavailableError.
	ErrUnavailable = errors.New("recognize: model is unavailable")
	// ErrBadAnswer — модель ответила, но ответ не разобрался.
	ErrBadAnswer = errors.New("recognize: model answer is not the expected JSON")
	// ErrNotImage — не PNG и не JPEG, или заголовок не читается.
	ErrNotImage = errors.New("recognize: not a PNG or JPEG image")
	// ErrImageTooLarge — больше MaxImageBytes или MaxPixels.
	ErrImageTooLarge = errors.New("recognize: image is too large")
)

// UnavailableError несёт срок повтора; errors.Is(err, ErrUnavailable) на нём работает.
type UnavailableError struct {
	RetryAfter time.Duration
	Err        error
}

func (e *UnavailableError) Error() string {
	if e.Err == nil {
		return ErrUnavailable.Error()
	}

	return fmt.Sprintf("%s: %v", ErrUnavailable, e.Err)
}

func (e *UnavailableError) Unwrap() []error {
	return []error{ErrUnavailable, e.Err}
}

// Image — проверенная CheckImage картинка.
type Image struct {
	MIME string
	Data []byte
}

// Category — активная категория семьи; Path — «Родитель / Имя» или «Имя».
type Category struct {
	ID   uuid.UUID
	Type transaction.Type
	Path string
}

// Input — всё, что нужно промпту и нормализации; Today и Currency — в зоне и валюте семьи.
type Input struct {
	Images     []Image
	Categories []Category
	Today      date.Date
	Currency   string
}

// Similar — существующая операция, похожая на кандидата.
type Similar struct {
	ID          uuid.UUID `json:"id"`
	Date        date.Date `json:"date"`
	Description string    `json:"description"`
}

// Item — кандидат операции; Source — индекс картинки в Input.Images.
type Item struct {
	Source      *int             `json:"source"`
	AmountMinor money.Minor      `json:"amount_minor"`
	Currency    *string          `json:"currency"`
	Type        transaction.Type `json:"type"`
	Date        *date.Date       `json:"date"`
	DateAssumed bool             `json:"date_assumed"`
	Description string           `json:"description"`
	CategoryID  *uuid.UUID       `json:"category_id"`
	Similar     []Similar        `json:"similar"`
}

// Result — кандидаты одного вызова модели.
type Result struct {
	Items      []Item `json:"items"`
	Incomplete bool   `json:"incomplete"`
	Model      string `json:"model"`
}

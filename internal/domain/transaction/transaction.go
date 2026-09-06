package transaction

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

type Transaction struct {
	ID          uuid.UUID   `json:"id"`
	AmountMinor money.Minor `json:"amount_minor"`
	Type        Type        `json:"type"`
	Description string      `json:"description"`
	CategoryID  uuid.UUID   `json:"category_id"`
	UserID      uuid.UUID   `json:"user_id"` // Кто создал транзакцию
	Date        date.Date   `json:"date"`    // Календарная дата операции
	Tags        []string    `json:"tags"`    // Теги для поиска
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Type string

const (
	TypeIncome  Type = "income"  // Доход
	TypeExpense Type = "expense" // Расход
)

type Filter struct {
	UserID          *uuid.UUID
	CategoryID      *uuid.UUID
	Type            *Type
	DateFrom        *date.Date
	DateTo          *date.Date
	AmountFromMinor *money.Minor
	AmountToMinor   *money.Minor
	Tags            []string
	Description     string
	Limit           int
	Offset          int
}

func (t *Transaction) AddTag(tag string) {
	if slices.Contains(t.Tags, tag) {
		return // Тег уже существует
	}
	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

func (t *Transaction) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			t.UpdatedAt = time.Now()
			return
		}
	}
}

const (
	// minYear — операция раньше этого года почти наверняка опечатка.
	minYear = 1900
	// maxFutureYears — операция дальше этого срока в будущем тоже опечатка.
	maxFutureYears = 1
)

// ErrDateOutOfRange — дата операции вне допустимого окна.
var ErrDateOutOfRange = errors.New("transaction date is out of the allowed range")

// ValidateDate — единственное место, где живут границы даты операции: сервис
// превращает отказ в 422, репозиторий не даёт записи пройти мимо сервиса.
func ValidateDate(on date.Date) error {
	if on.Before(date.New(minYear, time.January, 1)) ||
		on.After(date.FromTime(time.Now().AddDate(maxFutureYears, 0, 0))) {
		return fmt.Errorf("%w: %s", ErrDateOutOfRange, on)
	}

	return nil
}

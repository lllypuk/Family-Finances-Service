package transaction

import (
	"slices"
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
)

type Transaction struct {
	ID          uuid.UUID   `json:"id"           bson:"_id"`
	AmountMinor money.Minor `json:"amount_minor" bson:"amount_minor"`
	Type        Type        `json:"type"         bson:"type"`
	Description string      `json:"description"  bson:"description"`
	CategoryID  uuid.UUID   `json:"category_id"  bson:"category_id"`
	UserID      uuid.UUID   `json:"user_id"      bson:"user_id"` // Кто создал транзакцию
	Date        date.Date   `json:"date"         bson:"date"`    // Календарная дата операции
	Tags        []string    `json:"tags"         bson:"tags"`    // Теги для поиска
	CreatedAt   time.Time   `json:"created_at"   bson:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"   bson:"updated_at"`
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

func NewTransaction(
	amountMinor money.Minor,
	transactionType Type,
	description string,
	categoryID, userID uuid.UUID,
	on date.Date,
) *Transaction {
	return &Transaction{
		ID:          uuid.New(),
		AmountMinor: amountMinor,
		Type:        transactionType,
		Description: description,
		CategoryID:  categoryID,
		UserID:      userID,
		Date:        on,
		Tags:        []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
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

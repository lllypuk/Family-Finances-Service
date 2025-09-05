package transaction

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID          uuid.UUID `json:"id"          bson:"_id"`
	Amount      float64   `json:"amount"      bson:"amount"`
	Type        Type      `json:"type"        bson:"type"`
	Description string    `json:"description" bson:"description"`
	CategoryID  uuid.UUID `json:"category_id" bson:"category_id"`
	UserID      uuid.UUID `json:"user_id"     bson:"user_id"` // Кто создал транзакцию
	FamilyID    uuid.UUID `json:"family_id"   bson:"family_id"`
	Date        time.Time `json:"date"        bson:"date"` // Дата транзакции
	Tags        []string  `json:"tags"        bson:"tags"` // Теги для поиска
	CreatedAt   time.Time `json:"created_at"  bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  bson:"updated_at"`
}

type Type string

const (
	TypeIncome  Type = "income"  // Доход
	TypeExpense Type = "expense" // Расход
)

type Filter struct {
	FamilyID    uuid.UUID
	UserID      *uuid.UUID
	CategoryID  *uuid.UUID
	Type        *Type
	DateFrom    *time.Time
	DateTo      *time.Time
	AmountFrom  *float64
	AmountTo    *float64
	Tags        []string
	Description string
	Limit       int
	Offset      int
}

func NewTransaction(
	amount float64,
	transactionType Type,
	description string,
	categoryID, userID, familyID uuid.UUID,
	date time.Time,
) *Transaction {
	return &Transaction{
		ID:          uuid.New(),
		Amount:      amount,
		Type:        transactionType,
		Description: description,
		CategoryID:  categoryID,
		UserID:      userID,
		FamilyID:    familyID,
		Date:        date,
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

// GetFamilyID returns the family ID of the transaction
func (t *Transaction) GetFamilyID() uuid.UUID {
	return t.FamilyID
}

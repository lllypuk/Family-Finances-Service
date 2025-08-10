package category

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID     `json:"id" bson:"_id"`
	Name      string        `json:"name" bson:"name"`
	Type      CategoryType  `json:"type" bson:"type"`
	Color     string        `json:"color" bson:"color"`     // Цвет для UI (#FF5733)
	Icon      string        `json:"icon" bson:"icon"`       // Иконка для UI
	ParentID  *uuid.UUID    `json:"parent_id" bson:"parent_id,omitempty"` // Для подкатегорий
	FamilyID  uuid.UUID     `json:"family_id" bson:"family_id"`
	IsActive  bool          `json:"is_active" bson:"is_active"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
}

type CategoryType string

const (
	CategoryTypeIncome  CategoryType = "income"  // Доходы
	CategoryTypeExpense CategoryType = "expense" // Расходы
)

// Предустановленные категории расходов
var DefaultExpenseCategories = []string{
	"Продукты",
	"Транспорт",
	"Жилье и ЖКХ",
	"Здоровье",
	"Образование",
	"Развлечения",
	"Одежда",
	"Ресторан и кафе",
	"Спорт",
	"Подарки",
	"Разное",
}

// Предустановленные категории доходов
var DefaultIncomeCategories = []string{
	"Зарплата",
	"Фриланс",
	"Инвестиции",
	"Подарки",
	"Продажи",
	"Разное",
}

func NewCategory(name string, categoryType CategoryType, familyID uuid.UUID) *Category {
	return &Category{
		ID:        uuid.New(),
		Name:      name,
		Type:      categoryType,
		Color:     "#007BFF", // Дефолтный синий цвет
		Icon:      "default",
		FamilyID:  familyID,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (c *Category) IsSubcategory() bool {
	return c.ParentID != nil
}

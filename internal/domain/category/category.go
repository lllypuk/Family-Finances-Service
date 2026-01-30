package category

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID  `json:"id"         bson:"_id"`
	Name      string     `json:"name"       bson:"name"`
	Type      Type       `json:"type"       bson:"type"`
	Color     string     `json:"color"      bson:"color"`               // Цвет для UI (#FF5733)
	Icon      string     `json:"icon"       bson:"icon"`                // Иконка для UI
	ParentID  *uuid.UUID `json:"parent_id"  bson:"parent_id,omitempty"` // Для подкатегорий
	IsActive  bool       `json:"is_active"  bson:"is_active"`
	CreatedAt time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" bson:"updated_at"`
}

type Type string

const (
	TypeIncome  Type = "income"  // Доходы
	TypeExpense Type = "expense" // Расходы
)

// GetDefaultExpenseCategories возвращает предустановленные категории расходов
func GetDefaultExpenseCategories() []string {
	return []string{
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
}

// GetDefaultIncomeCategories возвращает предустановленные категории доходов
func GetDefaultIncomeCategories() []string {
	return []string{
		"Зарплата",
		"Фриланс",
		"Инвестиции",
		"Подарки",
		"Продажи",
		"Разное",
	}
}

func NewCategory(name string, categoryType Type) *Category {
	return &Category{
		ID:        uuid.New(),
		Name:      name,
		Type:      categoryType,
		Color:     "#007BFF", // Дефолтный синий цвет
		Icon:      "default",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (c *Category) IsSubcategory() bool {
	return c.ParentID != nil
}

package models

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
)

// CategoryForm представляет форму создания/редактирования категории
type CategoryForm struct {
	Name     string `form:"name"      validate:"required,min=1,max=100"        json:"name"`
	Type     string `form:"type"      validate:"required,oneof=income expense" json:"type"`
	Color    string `form:"color"     validate:"required,hexcolor"             json:"color"`
	Icon     string `form:"icon"      validate:"required,min=1,max=50"         json:"icon"`
	ParentID string `form:"parent_id" validate:"omitempty,uuid"                json:"parent_id,omitempty"`
	IsActive bool   `form:"is_active"                                          json:"is_active"`
}

// CategoryFilter представляет фильтры для поиска категорий
type CategoryFilter struct {
	Name       string `form:"name"        json:"name,omitempty"`
	Type       string `form:"type"        json:"type,omitempty"        validate:"omitempty,oneof=income expense"`
	IsActive   *bool  `form:"is_active"   json:"is_active,omitempty"`
	ParentOnly bool   `form:"parent_only" json:"parent_only,omitempty"`
}

// CategorySelectOption представляет опцию для select элементов
type CategorySelectOption struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	Color    string    `json:"color"`
	Icon     string    `json:"icon"`
	IsParent bool      `json:"is_parent"`
	Level    int       `json:"level"` // Для отступов в дереве подкатегорий
}

// CategoryViewModel представляет категорию для отображения в списках
type CategoryViewModel struct {
	ID                 uuid.UUID           `json:"id"`
	Name               string              `json:"name"`
	Type               category.Type       `json:"type"`
	Color              string              `json:"color"`
	Icon               string              `json:"icon"`
	ParentID           *uuid.UUID          `json:"parent_id,omitempty"`
	ParentName         string              `json:"parent_name,omitempty"`
	ParentCategory     *CategoryViewModel  `json:"parent_category,omitempty"`
	IsActive           bool                `json:"is_active"`
	SubCategories      []CategoryViewModel `json:"subcategories,omitempty"`
	TransactionCount   int                 `json:"transaction_count,omitempty"`
	TotalAmount        float64             `json:"total_amount,omitempty"`
	CurrentMonthAmount float64             `json:"current_month_amount,omitempty"`
	BudgetLimit        *float64            `json:"budget_limit,omitempty"`
	LastUsed           *time.Time          `json:"last_used,omitempty"`
	CanDelete          bool                `json:"can_delete"`
}

// FromDomain создает CategoryViewModel из domain модели
func (vm *CategoryViewModel) FromDomain(c *category.Category) {
	vm.ID = c.ID
	vm.Name = c.Name
	vm.Type = c.Type
	vm.Color = c.Color
	vm.Icon = c.Icon
	vm.ParentID = c.ParentID
	vm.IsActive = c.IsActive
	vm.SubCategories = make([]CategoryViewModel, 0)
	vm.CanDelete = true // По умолчанию, может быть изменено в зависимости от связанных транзакций
}

// ToCategoryType конвертирует строку в тип категории
func (f *CategoryForm) ToCategoryType() category.Type {
	switch f.Type {
	case "income":
		return category.TypeIncome
	case "expense":
		return category.TypeExpense
	default:
		return category.TypeExpense
	}
}

// GetParentID возвращает UUID родительской категории или nil
func (f *CategoryForm) GetParentID() *uuid.UUID {
	if f.ParentID == "" {
		return nil
	}

	id, err := uuid.Parse(f.ParentID)
	if err != nil {
		return nil
	}

	return &id
}

// BuildCategoryTree строит дерево категорий из плоского списка
func BuildCategoryTree(categories []CategoryViewModel) []CategoryViewModel {
	// Создаем карту всех категорий
	categoryMap := make(map[uuid.UUID]*CategoryViewModel)
	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}

	// Строим дерево
	var roots []CategoryViewModel
	for i := range categories {
		cat := &categories[i]
		if cat.ParentID == nil {
			// Это корневая категория
			roots = append(roots, *cat)
		} else {
			// Это подкатегория - добавляем к родителю
			if parent, exists := categoryMap[*cat.ParentID]; exists {
				parent.SubCategories = append(parent.SubCategories, *cat)
			}
		}
	}

	return roots
}

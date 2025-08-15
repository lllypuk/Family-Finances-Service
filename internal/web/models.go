package web

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/user"
)

// SessionData содержит данные пользовательской сессии
type SessionData struct {
	UserID    uuid.UUID `json:"user_id"`
	FamilyID  uuid.UUID `json:"family_id"`
	Role      user.Role `json:"role"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DashboardData содержит данные для главной страницы
type DashboardData struct {
	User             *user.User   `json:"user"`
	Family           *user.Family `json:"family"`
	TotalIncome      float64      `json:"total_income"`
	TotalExpenses    float64      `json:"total_expenses"`
	NetIncome        float64      `json:"net_income"`
	TransactionCount int          `json:"transaction_count"`
	BudgetCount      int          `json:"budget_count"`
}

// FormErrors представляет ошибки валидации форм
type FormErrors map[string]string

// PageData содержит общие данные для всех страниц
type PageData struct {
	Title       string       `json:"title"`
	CurrentUser *user.User   `json:"current_user"`
	Family      *user.Family `json:"family"`
	Errors      FormErrors   `json:"errors"`
	Messages    []Message    `json:"messages"`
	CSRFToken   string       `json:"csrf_token"`
}

// Message представляет сообщение для пользователя
type Message struct {
	Type    string `json:"type"` // success, error, warning, info
	Text    string `json:"text"`
	Timeout int    `json:"timeout"` // время отображения в секундах
}

// LoginFormData содержит данные формы входа
type LoginFormData struct {
	Email    string `json:"email"    form:"email"    validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required,min=6"`
}

// RegisterFormData содержит данные формы регистрации семьи
type RegisterFormData struct {
	FamilyName string `json:"family_name" form:"family_name" validate:"required,min=2,max=100"`
	Currency   string `json:"currency"    form:"currency"    validate:"required,len=3"`
	FirstName  string `json:"first_name"  form:"first_name"  validate:"required,min=2,max=50"`
	LastName   string `json:"last_name"   form:"last_name"   validate:"required,min=2,max=50"`
	Email      string `json:"email"       form:"email"       validate:"required,email"`
	Password   string `json:"password"    form:"password"    validate:"required,min=6"`
}

// IsEmpty проверяет, пусты ли ошибки формы
func (fe FormErrors) IsEmpty() bool {
	return len(fe) == 0
}

// Add добавляет ошибку в карту ошибок
func (fe FormErrors) Add(field, message string) {
	fe[field] = message
}

// Get возвращает ошибку для поля
func (fe FormErrors) Get(field string) string {
	return fe[field]
}

// Has проверяет, есть ли ошибка для поля
func (fe FormErrors) Has(field string) bool {
	_, exists := fe[field]
	return exists
}

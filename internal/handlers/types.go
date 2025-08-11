package handlers

import (
	"time"

	"github.com/google/uuid"
)

// Общие типы для HTTP ответов
type APIResponse[T any] struct {
	Data   T                      `json:"data"`
	Meta   ResponseMeta           `json:"meta"`
	Errors []ValidationError     `json:"errors,omitempty"`
}

type ResponseMeta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type ErrorResponse struct {
	Error ErrorDetail  `json:"error"`
	Meta  ResponseMeta `json:"meta"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// User related types
type CreateUserRequest struct {
	Email     string    `json:"email" validate:"required,email"`
	Password  string    `json:"password" validate:"required,min=6"`
	FirstName string    `json:"first_name" validate:"required"`
	LastName  string    `json:"last_name" validate:"required"`
	FamilyID  uuid.UUID `json:"family_id" validate:"required"`
	Role      string    `json:"role" validate:"required,oneof=admin member child"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	FamilyID  uuid.UUID `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Family related types
type CreateFamilyRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Currency string `json:"currency" validate:"required,len=3"`
}

type UpdateFamilyRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Currency *string `json:"currency,omitempty" validate:"omitempty,len=3"`
}

type FamilyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Category related types
type CreateCategoryRequest struct {
	Name     string    `json:"name" validate:"required,min=2,max=50"`
	Type     string    `json:"type" validate:"required,oneof=income expense"`
	Color    string    `json:"color" validate:"required,hexcolor"`
	Icon     string    `json:"icon" validate:"required"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	FamilyID uuid.UUID `json:"family_id" validate:"required"`
}

type UpdateCategoryRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
	Color *string `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Icon  *string `json:"icon,omitempty"`
}

type CategoryResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Color     string     `json:"color"`
	Icon      string     `json:"icon"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	FamilyID  uuid.UUID  `json:"family_id"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Transaction related types
type CreateTransactionRequest struct {
	Amount      float64   `json:"amount" validate:"required,gt=0"`
	Type        string    `json:"type" validate:"required,oneof=income expense"`
	Description string    `json:"description" validate:"required,min=2,max=200"`
	CategoryID  uuid.UUID `json:"category_id" validate:"required"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	FamilyID    uuid.UUID `json:"family_id" validate:"required"`
	Date        time.Time `json:"date" validate:"required"`
	Tags        []string  `json:"tags,omitempty"`
}

type UpdateTransactionRequest struct {
	Amount      *float64   `json:"amount,omitempty" validate:"omitempty,gt=0"`
	Type        *string    `json:"type,omitempty" validate:"omitempty,oneof=income expense"`
	Description *string    `json:"description,omitempty" validate:"omitempty,min=2,max=200"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

type TransactionResponse struct {
	ID          uuid.UUID `json:"id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CategoryID  uuid.UUID `json:"category_id"`
	UserID      uuid.UUID `json:"user_id"`
	FamilyID    uuid.UUID `json:"family_id"`
	Date        time.Time `json:"date"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TransactionFilterParams struct {
	FamilyID    uuid.UUID  `query:"family_id"`
	UserID      *uuid.UUID `query:"user_id"`
	CategoryID  *uuid.UUID `query:"category_id"`
	Type        *string    `query:"type"`
	DateFrom    *time.Time `query:"date_from"`
	DateTo      *time.Time `query:"date_to"`
	AmountFrom  *float64   `query:"amount_from"`
	AmountTo    *float64   `query:"amount_to"`
	Description *string    `query:"description"`
	Limit       int        `query:"limit" validate:"min=1,max=100"`
	Offset      int        `query:"offset" validate:"min=0"`
}

// Budget related types
type CreateBudgetRequest struct {
	Name       string     `json:"name" validate:"required,min=2,max=100"`
	Amount     float64    `json:"amount" validate:"required,gt=0"`
	Period     string     `json:"period" validate:"required,oneof=weekly monthly yearly custom"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	FamilyID   uuid.UUID  `json:"family_id" validate:"required"`
	StartDate  time.Time  `json:"start_date" validate:"required"`
	EndDate    time.Time  `json:"end_date" validate:"required"`
}

type UpdateBudgetRequest struct {
	Name      *string    `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Amount    *float64   `json:"amount,omitempty" validate:"omitempty,gt=0"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	IsActive  *bool      `json:"is_active,omitempty"`
}

type BudgetResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Amount     float64    `json:"amount"`
	Spent      float64    `json:"spent"`
	Period     string     `json:"period"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	FamilyID   uuid.UUID  `json:"family_id"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Report related types
type CreateReportRequest struct {
	Name      string    `json:"name" validate:"required,min=2,max=100"`
	Type      string    `json:"type" validate:"required,oneof=expenses income budget cash_flow category_break"`
	Period    string    `json:"period" validate:"required,oneof=daily weekly monthly yearly custom"`
	FamilyID  uuid.UUID `json:"family_id" validate:"required"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
}

type ReportResponse struct {
	ID          uuid.UUID   `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Period      string      `json:"period"`
	FamilyID    uuid.UUID   `json:"family_id"`
	UserID      uuid.UUID   `json:"user_id"`
	StartDate   time.Time   `json:"start_date"`
	EndDate     time.Time   `json:"end_date"`
	Data        interface{} `json:"data"`
	GeneratedAt time.Time   `json:"generated_at"`
}
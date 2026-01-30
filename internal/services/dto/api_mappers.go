package dto

import (
	"time"

	"github.com/google/uuid"

	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
)

// CreateUserAPIRequest represents API request for user creation
type CreateUserAPIRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

type UpdateUserAPIRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Email     *string `json:"email,omitempty"`
}

type UserAPIResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Role      string    `json:"role"`
	FamilyID  uuid.UUID `json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FromCreateUserAPIRequest converts API CreateUserRequest to CreateUserDTO
func FromCreateUserAPIRequest(req CreateUserAPIRequest) CreateUserDTO {
	return CreateUserDTO{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
		Role:      user.Role(req.Role),
	}
}

// FromUpdateUserAPIRequest converts API UpdateUserRequest to UpdateUserDTO
func FromUpdateUserAPIRequest(req UpdateUserAPIRequest) UpdateUserDTO {
	return UpdateUserDTO(req)
}

// ToUserAPIResponse converts domain User to API UserResponse
func ToUserAPIResponse(u *user.User) UserAPIResponse {
	return UserAPIResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Role:      string(u.Role),
		FamilyID:  uuid.Nil, // Single family model - FamilyID no longer stored in User
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// CreateCategoryAPIRequest represents API request for category creation
type CreateCategoryAPIRequest struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Color    string     `json:"color"`
	Icon     string     `json:"icon"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	FamilyID uuid.UUID  `json:"family_id"`
}

type UpdateCategoryAPIRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}

type CategoryAPIResponse struct {
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

// FromCreateCategoryAPIRequest converts API CreateCategoryRequest to CreateCategoryDTO
func FromCreateCategoryAPIRequest(req CreateCategoryAPIRequest) CreateCategoryDTO {
	return CreateCategoryDTO{
		Name:     req.Name,
		Type:     category.Type(req.Type),
		Color:    req.Color,
		Icon:     req.Icon,
		ParentID: req.ParentID,
		FamilyID: req.FamilyID,
	}
}

// FromUpdateCategoryAPIRequest converts API UpdateCategoryRequest to UpdateCategoryDTO
func FromUpdateCategoryAPIRequest(req UpdateCategoryAPIRequest) UpdateCategoryDTO {
	return UpdateCategoryDTO(req)
}

// ToCategoryAPIResponse converts domain Category to API CategoryResponse
func ToCategoryAPIResponse(c *category.Category) CategoryAPIResponse {
	return CategoryAPIResponse{
		ID:        c.ID,
		Name:      c.Name,
		Type:      string(c.Type),
		Color:     c.Color,
		Icon:      c.Icon,
		ParentID:  c.ParentID,
		FamilyID:  uuid.Nil, // Single family model - FamilyID no longer stored in Category
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

// Transaction API mappers

// CreateTransactionAPIRequest represents API request for transaction creation
type CreateTransactionAPIRequest struct {
	Amount      float64   `json:"amount"         validate:"required,gt=0"`
	Type        string    `json:"type"           validate:"required,oneof=income expense"`
	Description string    `json:"description"    validate:"required,min=2,max=200"`
	CategoryID  uuid.UUID `json:"category_id"    validate:"required"`
	UserID      uuid.UUID `json:"user_id"        validate:"required"`
	FamilyID    uuid.UUID `json:"family_id"      validate:"required"`
	Date        time.Time `json:"date"           validate:"required"`
	Tags        []string  `json:"tags,omitempty"`
}

// UpdateTransactionAPIRequest represents API request for transaction update
type UpdateTransactionAPIRequest struct {
	Amount      *float64   `json:"amount,omitempty"      validate:"omitempty,gt=0"`
	Type        *string    `json:"type,omitempty"        validate:"omitempty,oneof=income expense"`
	Description *string    `json:"description,omitempty" validate:"omitempty,min=2,max=200"`
	CategoryID  *uuid.UUID `json:"category_id,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

// TransactionAPIResponse represents API response for transaction
type TransactionAPIResponse struct {
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

// ToCreateTransactionDTO converts API request to service DTO
func (r CreateTransactionAPIRequest) ToCreateTransactionDTO() CreateTransactionDTO {
	return CreateTransactionDTO{
		Amount:      r.Amount,
		Type:        transaction.Type(r.Type),
		Description: r.Description,
		CategoryID:  r.CategoryID,
		UserID:      r.UserID,
		FamilyID:    r.FamilyID,
		Date:        r.Date,
		Tags:        r.Tags,
	}
}

// ToUpdateTransactionDTO converts API request to service DTO
func (r UpdateTransactionAPIRequest) ToUpdateTransactionDTO() UpdateTransactionDTO {
	dto := UpdateTransactionDTO{
		Amount:      r.Amount,
		Description: r.Description,
		CategoryID:  r.CategoryID,
		Date:        r.Date,
		Tags:        r.Tags,
	}

	if r.Type != nil {
		txType := transaction.Type(*r.Type)
		dto.Type = &txType
	}

	return dto
}

// ToTransactionAPIResponse converts domain transaction to API response
func ToTransactionAPIResponse(tx *transaction.Transaction) TransactionAPIResponse {
	return TransactionAPIResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Type:        string(tx.Type),
		Description: tx.Description,
		CategoryID:  tx.CategoryID,
		UserID:      tx.UserID,
		FamilyID:    uuid.Nil, // Single family model - FamilyID no longer stored in Transaction
		Date:        tx.Date,
		Tags:        tx.Tags,
		CreatedAt:   tx.CreatedAt,
		UpdatedAt:   tx.UpdatedAt,
	}
}

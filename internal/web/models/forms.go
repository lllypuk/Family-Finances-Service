package models

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// LoginForm представляет форму входа в систему
type LoginForm struct {
	Email    string `form:"email"    validate:"required,email,max=254" json:"email"`
	Password string `form:"password" validate:"required,min=6"         json:"password"`
}

// RegisterForm представляет форму регистрации семьи
type RegisterForm struct {
	FamilyName      string `form:"family_name"      validate:"required,min=2,max=100" json:"family_name"`
	Currency        string `form:"currency"         validate:"required,len=3"         json:"currency"`
	Name            string `form:"name"             validate:"required,min=2,max=100" json:"name"`
	FirstName       string `form:"first_name"       validate:"required,min=2,max=50"  json:"first_name"`
	LastName        string `form:"last_name"        validate:"required,min=2,max=50"  json:"last_name"`
	Email           string `form:"email"            validate:"required,email,max=254" json:"email"`
	Password        string `form:"password"         validate:"required,min=6,strong_password" json:"password"`
	ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password" json:"confirm_password"`
}

// CreateUserForm представляет форму создания пользователя в семье
type CreateUserForm struct {
	FirstName string `form:"first_name" validate:"required,min=2,max=50"  json:"first_name"`
	LastName  string `form:"last_name"  validate:"required,min=2,max=50"  json:"last_name"`
	Email     string `form:"email"      validate:"required,email,max=254" json:"email"`
	Password  string `form:"password"   validate:"required,min=6"         json:"password"`
	Role      string `form:"role"       validate:"required"               json:"role"`
}

// GetValidationErrors конвертирует ошибки валидации в карту field->message
func GetValidationErrors(err error) map[string]string {
	validationErrorsMap := make(map[string]string)

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldError := range validationErrors {
			field := getFieldName(fieldError)
			message := getErrorMessage(fieldError)
			validationErrorsMap[field] = message
		}
	}

	return validationErrorsMap
}

// getFieldName получает имя поля из тега form или из имени поля
func getFieldName(fe validator.FieldError) string {
	field := fe.Field()

	// Преобразуем CamelCase в snake_case для соответствия HTML form полям
	var result strings.Builder
	for i, r := range field {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// getErrorMessage создает пользовательское сообщение об ошибке
func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Please enter a valid email address"
	case "min":
		if fe.Kind() == reflect.String {
			return "Must be at least " + fe.Param() + " characters"
		}
		return "Must be at least " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "Must be no more than " + fe.Param() + " characters"
		}
		return "Must be no more than " + fe.Param()
	case "len":
		return "Must be exactly " + fe.Param() + " characters"
	default:
		return "Invalid value"
	}
}

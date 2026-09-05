package services

import (
	"github.com/go-playground/validator/v10"

	"family-budget-service/internal/auth"
)

// newValidator — валидатор сервисов с тегом `password` (auth.ValidatePassword).
func newValidator() *validator.Validate {
	v := validator.New()
	auth.RegisterPasswordValidation(v)
	return v
}

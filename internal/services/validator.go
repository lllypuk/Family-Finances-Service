package services

import (
	"github.com/go-playground/validator/v10"

	"family-budget-service/internal/auth"
)

// newValidator — валидатор сервисов с тегом `password` (auth.ValidatePassword).
// WithRequiredStructEnabled — см. newAPIValidator: иначе `required` на date.Date не работает.
func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	auth.RegisterPasswordValidation(v)
	return v
}

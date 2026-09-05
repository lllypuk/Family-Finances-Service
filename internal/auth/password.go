package auth

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

// Политика пароля: длина в байтах, верхняя граница — предел bcrypt.
const (
	MinPasswordBytes = 10
	MaxPasswordBytes = 72
	bcryptCost       = 12
)

// PasswordTag — тег validator для полей с паролем; регистрируется RegisterPasswordValidation.
const PasswordTag = "password"

// ErrInvalidPassword — пароль не проходит политику.
var ErrInvalidPassword = errors.New("password must be between 10 and 72 bytes")

// ValidatePassword — единственная проверка политики пароля (DTO, CLI, смена пароля).
func ValidatePassword(password string) error {
	if n := len(password); n < MinPasswordBytes || n > MaxPasswordBytes {
		return fmt.Errorf("%w: got %d", ErrInvalidPassword, n)
	}
	return nil
}

// HashPassword — bcrypt cost 12.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// ComparePassword сообщает, соответствует ли пароль хешу.
func ComparePassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// RegisterPasswordValidation подключает тег `password` к валидатору.
func RegisterPasswordValidation(v *validator.Validate) {
	// Ошибка возможна только при пустом теге или nil-функции — оба фиксированы здесь.
	_ = v.RegisterValidation(PasswordTag, func(fl validator.FieldLevel) bool {
		return ValidatePassword(fl.Field().String()) == nil
	})
}

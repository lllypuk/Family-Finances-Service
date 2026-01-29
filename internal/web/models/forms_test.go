package models_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/web/models"
)

func TestLoginForm_StructFields(t *testing.T) {
	form := models.LoginForm{
		Email:    "test@example.com",
		Password: "password123",
	}

	assert.Equal(t, "test@example.com", form.Email)
	assert.Equal(t, "password123", form.Password)

	// Проверяем теги валидации
	formType := reflect.TypeOf(form)

	emailField, found := formType.FieldByName("Email")
	require.True(t, found)
	assert.Equal(t, "email", emailField.Tag.Get("form"))
	assert.Equal(t, "required,email,max=254", emailField.Tag.Get("validate"))
	assert.Equal(t, "email", emailField.Tag.Get("json"))

	passwordField, found := formType.FieldByName("Password")
	require.True(t, found)
	assert.Equal(t, "password", passwordField.Tag.Get("form"))
	assert.Equal(t, "required,min=6", passwordField.Tag.Get("validate"))
	assert.Equal(t, "password", passwordField.Tag.Get("json"))
}

func TestSetupForm_StructFields(t *testing.T) {
	form := models.SetupForm{
		FamilyName: "Test Family",
		Currency:   "USD",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "john@example.com",
		Password:   "securepass123",
	}

	assert.Equal(t, "Test Family", form.FamilyName)
	assert.Equal(t, "USD", form.Currency)
	assert.Equal(t, "John", form.FirstName)
	assert.Equal(t, "Doe", form.LastName)
	assert.Equal(t, "john@example.com", form.Email)
	assert.Equal(t, "securepass123", form.Password)

	// Проверяем теги валидации для всех полей
	formType := reflect.TypeOf(form)

	familyNameField, found := formType.FieldByName("FamilyName")
	require.True(t, found)
	assert.Equal(t, "family_name", familyNameField.Tag.Get("form"))
	assert.Equal(t, "required,min=2,max=100", familyNameField.Tag.Get("validate"))

	currencyField, found := formType.FieldByName("Currency")
	require.True(t, found)
	assert.Equal(t, "currency", currencyField.Tag.Get("form"))
	assert.Equal(t, "required,len=3", currencyField.Tag.Get("validate"))

	firstNameField, found := formType.FieldByName("FirstName")
	require.True(t, found)
	assert.Equal(t, "first_name", firstNameField.Tag.Get("form"))
	assert.Equal(t, "required,min=2,max=50", firstNameField.Tag.Get("validate"))

	lastNameField, found := formType.FieldByName("LastName")
	require.True(t, found)
	assert.Equal(t, "last_name", lastNameField.Tag.Get("form"))
	assert.Equal(t, "required,min=2,max=50", lastNameField.Tag.Get("validate"))

	emailField, found := formType.FieldByName("Email")
	require.True(t, found)
	assert.Equal(t, "email", emailField.Tag.Get("form"))
	assert.Equal(t, "required,email,max=254", emailField.Tag.Get("validate"))

	passwordField, found := formType.FieldByName("Password")
	require.True(t, found)
	assert.Equal(t, "password", passwordField.Tag.Get("form"))
	assert.Equal(t, "required,min=6", passwordField.Tag.Get("validate"))
}

func TestGetValidationErrors_RequiredFields(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name          string
		form          any
		expectedCount int
		expectedField string
		expectedMsg   string
	}{
		{
			name:          "LoginForm - missing email",
			form:          models.LoginForm{Password: "password123"},
			expectedCount: 1,
			expectedField: "email",
			expectedMsg:   "This field is required",
		},
		{
			name:          "LoginForm - missing password",
			form:          models.LoginForm{Email: "test@example.com"},
			expectedCount: 1,
			expectedField: "password",
			expectedMsg:   "This field is required",
		},
		{
			name: "RegisterForm - missing family name",
			form: models.SetupForm{
				Currency:  "USD",
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john@example.com",
				Password:  "password123",
			},
			expectedCount: 1,
			expectedField: "family_name",
			expectedMsg:   "This field is required",
		},
		{
			name:          "RegisterForm - all fields missing",
			form:          models.SetupForm{},
			expectedCount: 6,
			expectedField: "",
			expectedMsg:   "This field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.form)
			require.Error(t, err)

			validationErrors := models.GetValidationErrors(err)
			assert.Len(t, validationErrors, tt.expectedCount)

			if tt.expectedField != "" {
				assert.Contains(t, validationErrors, tt.expectedField)
				assert.Equal(t, tt.expectedMsg, validationErrors[tt.expectedField])
			}

			if tt.expectedCount > 1 {
				// Проверяем, что все ошибки о required полях
				for _, msg := range validationErrors {
					assert.Equal(t, "This field is required", msg)
				}
			}
		})
	}
}

func TestGetValidationErrors_EmailValidation(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name        string
		email       string
		expectError bool
		expectedMsg string
	}{
		{
			name:        "Valid email",
			email:       "user@example.com",
			expectError: false,
		},
		{
			name:        "Valid email with subdomain",
			email:       "user@mail.example.com",
			expectError: false,
		},
		{
			name:        "Invalid email - no @",
			email:       "userexample.com",
			expectError: true,
			expectedMsg: "Please enter a valid email address",
		},
		{
			name:        "Invalid email - no domain",
			email:       "user@",
			expectError: true,
			expectedMsg: "Please enter a valid email address",
		},
		{
			name:        "Invalid email - empty",
			email:       "",
			expectError: true,
			expectedMsg: "This field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.LoginForm{
				Email:    tt.email,
				Password: "password123",
			}

			err := v.Struct(form)
			if tt.expectError {
				require.Error(t, err)
				validationErrors := models.GetValidationErrors(err)
				assert.Contains(t, validationErrors, "email")
				assert.Equal(t, tt.expectedMsg, validationErrors["email"])
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetValidationErrors_LengthValidation(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name          string
		password      string
		familyName    string
		currency      string
		expectError   bool
		expectedField string
		expectedMsg   string
	}{
		{
			name:        "Valid lengths",
			password:    "password123",
			familyName:  "Valid Family Name",
			currency:    "USD",
			expectError: false,
		},
		{
			name:          "Password too short",
			password:      "123",
			familyName:    "Valid Family",
			currency:      "USD",
			expectError:   true,
			expectedField: "password",
			expectedMsg:   "Must be at least 6 characters",
		},
		{
			name:          "Family name too short",
			password:      "password123",
			familyName:    "A",
			currency:      "USD",
			expectError:   true,
			expectedField: "family_name",
			expectedMsg:   "Must be at least 2 characters",
		},
		{
			name:          "Currency wrong length",
			password:      "password123",
			familyName:    "Valid Family",
			currency:      "USDD",
			expectError:   true,
			expectedField: "currency",
			expectedMsg:   "Must be exactly 3 characters",
		},
		{
			name:          "Currency too short",
			password:      "password123",
			familyName:    "Valid Family",
			currency:      "US",
			expectError:   true,
			expectedField: "currency",
			expectedMsg:   "Must be exactly 3 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.SetupForm{
				FamilyName: tt.familyName,
				Currency:   tt.currency,
				FirstName:  "John",
				LastName:   "Doe",
				Email:      "john@example.com",
				Password:   tt.password,
			}

			err := v.Struct(form)
			if tt.expectError {
				require.Error(t, err)
				validationErrors := models.GetValidationErrors(err)
				assert.Contains(t, validationErrors, tt.expectedField)
				assert.Equal(t, tt.expectedMsg, validationErrors[tt.expectedField])
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetValidationErrors_MaxLengthValidation(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name          string
		field         string
		value         string
		expectedField string
		expectedMsg   string
	}{
		{
			name:          "Email too long",
			field:         "email",
			value:         strings.Repeat("a", 250) + "@example.com", // > 254 characters
			expectedField: "email",
			expectedMsg:   "Must be no more than 254 characters",
		},
		{
			name:          "Family name too long",
			field:         "family_name",
			value:         strings.Repeat("A", 101), // > 100 characters
			expectedField: "family_name",
			expectedMsg:   "Must be no more than 100 characters",
		},
		{
			name:          "First name too long",
			field:         "first_name",
			value:         strings.Repeat("A", 51), // > 50 characters
			expectedField: "first_name",
			expectedMsg:   "Must be no more than 50 characters",
		},
		{
			name:          "Last name too long",
			field:         "last_name",
			value:         strings.Repeat("A", 51), // > 50 characters
			expectedField: "last_name",
			expectedMsg:   "Must be no more than 50 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := models.SetupForm{
				FamilyName: "Valid Family",
				Currency:   "USD",
				FirstName:  "John",
				LastName:   "Doe",
				Email:      "john@example.com",
				Password:   "password123",
			}

			// Устанавливаем длинное значение для нужного поля
			switch tt.field {
			case "email":
				form.Email = tt.value
			case "family_name":
				form.FamilyName = tt.value
			case "first_name":
				form.FirstName = tt.value
			case "last_name":
				form.LastName = tt.value
			}

			err := v.Struct(form)
			require.Error(t, err)

			validationErrors := models.GetValidationErrors(err)
			assert.Contains(t, validationErrors, tt.expectedField)
			assert.Equal(t, tt.expectedMsg, validationErrors[tt.expectedField])
		})
	}
}

func TestGetFieldName_CamelCaseConversion(t *testing.T) {
	// Создаем mock FieldError для тестирования
	v := validator.New()

	// Тестируем через реальную валидацию
	form := models.SetupForm{
		// Оставляем FirstName пустым для получения ошибки валидации
		FamilyName: "Test",
		Currency:   "USD",
		LastName:   "Doe",
		Email:      "test@example.com",
		Password:   "password123",
	}

	err := v.Struct(form)
	require.Error(t, err)

	validationErrors := models.GetValidationErrors(err)

	// Проверяем, что поле FirstName конвертируется в first_name
	assert.Contains(t, validationErrors, "first_name")
	assert.Equal(t, "This field is required", validationErrors["first_name"])
}

func TestGetErrorMessage_UnknownTag(t *testing.T) {
	v := validator.New()

	// Регистрируем кастомный валидатор для тестирования неизвестного тега
	v.RegisterValidation("custom", func(_ validator.FieldLevel) bool {
		return false
	})

	type TestForm struct {
		Field string `validate:"custom"`
	}

	form := TestForm{Field: "test"}
	err := v.Struct(form)
	require.Error(t, err)

	validationErrors := models.GetValidationErrors(err)
	assert.Contains(t, validationErrors, "field")
	assert.Equal(t, "Invalid value", validationErrors["field"])
}

func TestGetValidationErrors_EmptyError(t *testing.T) {
	// Тестируем случай, когда ошибка не является ValidationErrors
	nonValidationError := errors.New("some other error")

	validationErrors := models.GetValidationErrors(nonValidationError)
	assert.Empty(t, validationErrors)
}

func TestGetValidationErrors_NilError(t *testing.T) {
	validationErrors := models.GetValidationErrors(nil)
	assert.Empty(t, validationErrors)
}

func TestGetValidationErrors_MultipleErrors(t *testing.T) {
	v := validator.New()

	// Создаем форму с множественными ошибками
	form := models.SetupForm{
		// Все поля пустые или невалидные
		FamilyName: "",
		Currency:   "",
		FirstName:  "",
		LastName:   "",
		Email:      "",
		Password:   "",
	}

	err := v.Struct(form)
	require.Error(t, err)

	validationErrors := models.GetValidationErrors(err)

	// Должно быть 6 ошибок - по одной на каждое обязательное поле
	assert.Len(t, validationErrors, 6)

	expectedFields := []string{"family_name", "currency", "first_name", "last_name", "email", "password"}
	for _, field := range expectedFields {
		assert.Contains(t, validationErrors, field)
		assert.Equal(t, "This field is required", validationErrors[field])
	}
}

// Benchmark тесты для production performance
func BenchmarkGetValidationErrors(b *testing.B) {
	v := validator.New()

	form := models.SetupForm{
		FamilyName: "Test Family",
		Currency:   "USD",
		FirstName:  "John",
		LastName:   "Doe",
		Email:      "invalid-email", // Невалидный email для создания ошибки
		Password:   "12345",         // Слишком короткий пароль
	}

	err := v.Struct(form)
	require.Error(b, err)

	for b.Loop() {
		_ = models.GetValidationErrors(err)
	}
}

func BenchmarkGetFieldName(b *testing.B) {
	v := validator.New()

	form := models.LoginForm{
		Email:    "", // Пустой для создания ошибки
		Password: "password123",
	}

	err := v.Struct(form)
	require.Error(b, err)

	// We don't need fieldError since we test through public API
	_ = err // Use err to validate the test setup

	for b.Loop() {
		// Test private function through public API
		_ = models.GetValidationErrors(err)
	}
}

func BenchmarkGetErrorMessage(b *testing.B) {
	v := validator.New()

	form := models.LoginForm{
		Email:    "", // Пустой для создания ошибки
		Password: "password123",
	}

	err := v.Struct(form)
	require.Error(b, err)

	// We don't need fieldError since we test through public API
	_ = err // Use err to validate the test setup

	for b.Loop() {
		// Test private function through public API
		_ = models.GetValidationErrors(err)
	}
}

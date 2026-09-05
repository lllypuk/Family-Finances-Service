package auth_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "too short", password: "123456789", wantErr: true},
		{name: "min length", password: "1234567890", wantErr: false},
		{name: "max length", password: strings.Repeat("a", 72), wantErr: false},
		{name: "too long", password: strings.Repeat("a", 73), wantErr: true},
		{name: "bytes not runes", password: "ппппп", wantErr: false},
		{name: "empty", password: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidatePassword(tt.password)
			if tt.wantErr {
				require.ErrorIs(t, err, auth.ErrInvalidPassword)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := auth.HashPassword("correct horse battery")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(hash, "$2a$12$"), "cost 12 expected: %s", hash)
	assert.True(t, auth.ComparePassword(hash, "correct horse battery"))
	assert.False(t, auth.ComparePassword(hash, "wrong"))
	assert.False(t, auth.ComparePassword("not a hash", "correct horse battery"))
}

func TestRegisterPasswordValidation(t *testing.T) {
	type form struct {
		Password string `validate:"required,password"`
	}

	v := validator.New()
	auth.RegisterPasswordValidation(v)

	require.NoError(t, v.Struct(form{Password: "long-enough-1"}))

	err := v.Struct(form{Password: "short"})
	require.Error(t, err)
	var fieldErrs validator.ValidationErrors
	require.ErrorAs(t, err, &fieldErrs)
	require.Len(t, fieldErrs, 1)
	assert.Equal(t, auth.PasswordTag, fieldErrs[0].Tag())
}

func TestGenerateToken(t *testing.T) {
	plain, hash := auth.GenerateToken()
	plain2, hash2 := auth.GenerateToken()

	assert.Len(t, plain, 43, "32 bytes in base64url without padding")
	assert.Len(t, hash, 64, "hex(sha256)")
	assert.NotEqual(t, plain, plain2)
	assert.NotEqual(t, hash, hash2)
	assert.Equal(t, hash, auth.HashToken(plain))
	assert.NotContains(t, plain, "=")
}

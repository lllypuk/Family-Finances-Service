package main

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/auth"
)

func fullSetupArgs() []string {
	return []string{
		"--family", "Test Family",
		"--currency", "RUB",
		"--timezone", "Europe/Moscow",
		"--email", "admin@example.com",
		"--first-name", "Admin",
		"--last-name", "User",
		"--password-stdin",
	}
}

func TestParseSetupArgs_Success(t *testing.T) {
	params, err := parseSetupArgs(fullSetupArgs(), strings.NewReader("Admin1234!\nsecond line ignored\n"))
	require.NoError(t, err)

	assert.Equal(t, internal.SetupParams{
		FamilyName: "Test Family",
		Currency:   "RUB",
		Timezone:   "Europe/Moscow",
		Email:      "admin@example.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "Admin1234!",
	}, params)
}

func TestParseSetupArgs_PasswordWithoutTrailingNewline(t *testing.T) {
	params, err := parseSetupArgs(fullSetupArgs(), strings.NewReader("Admin1234!"))
	require.NoError(t, err)
	assert.Equal(t, "Admin1234!", params.Password)
}

func TestParseSetupArgs_StripsCRLF(t *testing.T) {
	params, err := parseSetupArgs(fullSetupArgs(), strings.NewReader("Admin1234!\r\n"))
	require.NoError(t, err)
	assert.Equal(t, "Admin1234!", params.Password)
}

func TestParseSetupArgs_MissingFlags(t *testing.T) {
	_, err := parseSetupArgs([]string{"--family", "X", "--password-stdin"}, strings.NewReader("Admin1234!\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--currency")
	assert.Contains(t, err.Error(), "--email")
	assert.Contains(t, err.Error(), "--first-name")
	assert.Contains(t, err.Error(), "--last-name")
	assert.Contains(t, err.Error(), "--timezone")
	assert.NotContains(t, err.Error(), "--family")
}

func TestParseSetupArgs_PasswordStdinRequired(t *testing.T) {
	args := fullSetupArgs()[:len(fullSetupArgs())-1]
	_, err := parseSetupArgs(args, strings.NewReader("Admin1234!\n"))
	require.ErrorIs(t, err, errPasswordStdinRequired)
}

func TestParseSetupArgs_EmptyStdin(t *testing.T) {
	_, err := parseSetupArgs(fullSetupArgs(), strings.NewReader(""))
	require.ErrorIs(t, err, auth.ErrInvalidPassword)
}

func TestParseSetupArgs_WeakPassword(t *testing.T) {
	_, err := parseSetupArgs(fullSetupArgs(), strings.NewReader("short\n"))
	require.ErrorIs(t, err, auth.ErrInvalidPassword)
}

func TestParseSetupArgs_UnknownFlag(t *testing.T) {
	_, err := parseSetupArgs([]string{"--bogus"}, io.NopCloser(strings.NewReader("")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestParseResetPasswordArgs_Success(t *testing.T) {
	email, password, err := parseResetPasswordArgs(
		[]string{"--email", "admin@example.com", "--password-stdin"},
		strings.NewReader("NewPassw0rd!\n"),
	)
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", email)
	assert.Equal(t, "NewPassw0rd!", password)
}

func TestParseResetPasswordArgs_MissingEmail(t *testing.T) {
	_, _, err := parseResetPasswordArgs([]string{"--password-stdin"}, strings.NewReader("NewPassw0rd!\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email")
}

func TestParseResetPasswordArgs_PasswordStdinRequired(t *testing.T) {
	_, _, err := parseResetPasswordArgs([]string{"--email", "a@b.co"}, strings.NewReader("NewPassw0rd!\n"))
	require.ErrorIs(t, err, errPasswordStdinRequired)
}

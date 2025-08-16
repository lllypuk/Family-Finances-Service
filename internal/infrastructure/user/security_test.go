package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	userRepo "family-budget-service/internal/infrastructure/user"
)

func TestValidateEmail_ValidEmails(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name@domain.co.uk",
		"first+last@subdomain.example.org",
		"TEST@EXAMPLE.COM",     // Should be normalized
		"  test@example.com  ", // Should be trimmed
	}

	for _, email := range validEmails {
		t.Run(email, func(t *testing.T) {
			err := userRepo.ValidateEmail(email)
			assert.NoError(t, err, "Valid email should not return error: %s", email)
		})
	}
}

func TestValidateEmail_InvalidEmails(t *testing.T) {
	invalidEmails := []struct {
		email string
		desc  string
	}{
		{"", "empty email"},
		{"   ", "whitespace only"},
		{"invalid-email", "missing @ symbol"},
		{"@example.com", "missing local part"},
		{"test@", "missing domain"},
		{"test@.com", "invalid domain"},
		{"test..test@example.com", "double dots"},
	}

	for _, tc := range invalidEmails {
		t.Run(tc.desc, func(t *testing.T) {
			err := userRepo.ValidateEmail(tc.email)
			require.Error(t, err, "Invalid email should return error: %s", tc.email)
		})
	}
}

func TestValidateEmail_InjectionAttempts(t *testing.T) {
	injectionAttempts := []struct {
		email string
		desc  string
	}{
		{"test@example.com{$ne:null}", "NoSQL injection with $ne"},
		{"test@example.com{$gt:\"\"}", "NoSQL injection with $gt"},
		{"test@example.com[$regex]", "NoSQL injection with $regex"},
		{"test@example.com{$where:\"this.email\"}", "NoSQL injection with $where"},
		{"test@example.com{}", "empty object injection"},
		{"test@example.com[]", "array injection"},
		{"test@example.com$", "dollar sign injection"},
	}

	for _, tc := range injectionAttempts {
		t.Run(tc.desc, func(t *testing.T) {
			err := userRepo.ValidateEmail(tc.email)
			require.Error(t, err, "Injection attempt should be rejected: %s", tc.email)
			assert.Contains(t, err.Error(), "invalid characters", "Error should mention invalid characters")
		})
	}
}

func TestValidateEmail_LongEmail(t *testing.T) {
	// Create an email longer than 254 characters
	longLocalPart := make([]byte, 250)
	for i := range longLocalPart {
		longLocalPart[i] = 'a'
	}
	longEmail := string(longLocalPart) + "@example.com"

	err := userRepo.ValidateEmail(longEmail)
	require.Error(t, err, "Excessively long email should be rejected")
	assert.Contains(t, err.Error(), "too long", "Error should mention email is too long")
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"Test@Example.COM", "test@example.com", "uppercase normalization"},
		{"  test@example.com  ", "test@example.com", "whitespace trimming"},
		{"TEST@EXAMPLE.COM", "test@example.com", "full uppercase normalization"},
		{"test@example.com", "test@example.com", "already clean email"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			result := userRepo.SanitizeEmail(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRepository_GetByEmail_SecurityValidation(t *testing.T) {
	// This is a unit test focusing on security validation
	// We don't need actual MongoDB connection for this
	repo := &userRepo.Repository{}

	injectionAttempts := []string{
		"test@example.com{$ne:null}",
		"test@example.com{$gt:\"\"}",
		"test@example.com[$regex]",
		"test@example.com{}",
		"test@example.com[]",
		"invalid-email-format",
		"",
	}

	for _, email := range injectionAttempts {
		t.Run(email, func(t *testing.T) {
			ctx := context.Background()
			_, err := repo.GetByEmail(ctx, email)

			// Should return validation error before even attempting database query
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid email parameter")
		})
	}
}

func TestRepository_Create_SecurityValidation(t *testing.T) {
	repo := &userRepo.Repository{}
	ctx := context.Background()

	injectionUser := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com{$ne:null}",
		FirstName: "Test",
		LastName:  "User",
		FamilyID:  uuid.New(),
		Role:      user.RoleMember,
	}

	err := repo.Create(ctx, injectionUser)

	// Should return validation error before attempting database operation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user email")
}

func TestRepository_Update_SecurityValidation(t *testing.T) {
	repo := &userRepo.Repository{}
	ctx := context.Background()

	injectionUser := &user.User{
		ID:        uuid.New(),
		Email:     "test@example.com[$regex]",
		FirstName: "Test",
		LastName:  "User",
		FamilyID:  uuid.New(),
		Role:      user.RoleMember,
	}

	err := repo.Update(ctx, injectionUser)

	// Should return validation error before attempting database operation
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user email")
}

func TestEmailSanitization_ConsistentBehavior(t *testing.T) {
	// Test that email sanitization is consistent across operations
	originalEmail := "  TEST@EXAMPLE.COM  "
	expectedEmail := "test@example.com"

	// Test direct sanitization
	sanitized := userRepo.SanitizeEmail(originalEmail)
	assert.Equal(t, expectedEmail, sanitized)

	// Test that validation accepts the sanitized version
	err := userRepo.ValidateEmail(originalEmail)
	require.NoError(t, err)

	// Test that the final sanitized email is what we expect
	finalSanitized := userRepo.SanitizeEmail(originalEmail)
	assert.Equal(t, expectedEmail, finalSanitized)
}

func TestEmailValidation_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		email     string
		shouldErr bool
		desc      string
	}{
		{"a@b.co", false, "minimal valid email"},
		{"user@sub.domain.tld", false, "subdomain email"},
		{"user+tag@domain.com", false, "email with plus"},
		{"user.name@domain.com", false, "email with dot in local part"},
		{"user@domain-name.com", false, "domain with hyphen"},
		{"user name@domain.com", true, "space in local part"},
		{"user@domain .com", true, "space in domain"},
		{"user@@domain.com", true, "double @ symbols"},
	}

	for _, tc := range edgeCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := userRepo.ValidateEmail(tc.email)
			if tc.shouldErr {
				require.Error(t, err, "Should be invalid: %s", tc.email)
			} else {
				assert.NoError(t, err, "Should be valid: %s", tc.email)
			}
		})
	}
}

// Benchmark tests to ensure validation doesn't significantly impact performance
func BenchmarkValidateEmail(b *testing.B) {
	email := "test@example.com"
	for range b.N {
		userRepo.ValidateEmail(email)
	}
}

func BenchmarkSanitizeEmail(b *testing.B) {
	email := "  TEST@EXAMPLE.COM  "
	for range b.N {
		userRepo.SanitizeEmail(email)
	}
}

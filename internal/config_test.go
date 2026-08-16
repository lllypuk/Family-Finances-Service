package internal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
)

// Тесты на Config.Validate — вторая половина находки S-04
// (docs/specs/002-security-audit.md): проверки только на строку-плейсхолдер
// было мало, `SESSION_SECRET=123` в production её проходил.

const (
	strongSecret       = "8Kx3vQpZ2mNtR7yLwB4cJfHsD6gA1eUo"
	placeholderSession = "your-super-secret-session-key-change-in-production"
	placeholderCSRF    = "your-csrf-secret-key-change-in-production"
)

func productionConfig(sessionSecret, csrfSecret string) *internal.Config {
	return &internal.Config{
		Environment: "production",
		Database:    internal.DatabaseConfig{Path: "./data/budget.db"},
		Web: internal.WebConfig{
			SessionSecret: sessionSecret,
			CSRFSecret:    csrfSecret,
		},
	}
}

func TestConfig_Validate_Production_RejectsPlaceholderSecrets(t *testing.T) {
	t.Run("session secret", func(t *testing.T) {
		err := productionConfig(placeholderSession, strongSecret).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session secret")
	})

	t.Run("csrf secret", func(t *testing.T) {
		err := productionConfig(strongSecret, placeholderCSRF).Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSRF secret")
	})
}

func TestConfig_Validate_Production_RejectsShortSecrets(t *testing.T) {
	cases := map[string]*internal.Config{
		"short session secret": productionConfig("123", strongSecret),
		"empty session secret": productionConfig("", strongSecret),
		"short csrf secret":    productionConfig(strongSecret, "123"),
		"empty csrf secret":    productionConfig(strongSecret, ""),
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			err := cfg.Validate()
			require.Error(t, err, "слабый секрет прошёл валидацию production (S-04)")
			assert.Contains(t, err.Error(), "openssl rand -base64 32",
				"сообщение должно подсказывать, как сгенерировать секрет")
		})
	}
}

func TestConfig_Validate_Production_AcceptsStrongSecrets(t *testing.T) {
	require.NoError(t, productionConfig(strongSecret, strongSecret+"x").Validate())
}

func TestConfig_Validate_Development_AllowsPlaceholders(t *testing.T) {
	cfg := productionConfig(placeholderSession, placeholderCSRF)
	cfg.Environment = "development"

	require.NoError(t, cfg.Validate(), "в dev-режиме плейсхолдеры допустимы")
}

func TestConfig_Validate_RequiresDatabasePath(t *testing.T) {
	cfg := productionConfig(strongSecret, strongSecret)
	cfg.Database.Path = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database path")
}

// COOKIE_SECURE — единственный рычаг, которым развёртывание без TLS
// (deploy/docker-compose.minimal.yml, первый запуск по IP) может отключить
// флаг Secure на session-cookie: браузер выбрасывает такую cookie на любом
// http:// origin, и вход зацикливается на «CSRF token not found in session».
// Раньше значение в production безусловно перетиралось на true.
func TestLoadConfig_CookieSecure(t *testing.T) {
	cases := []struct {
		name        string
		environment string
		cookieValue string
		expected    bool
	}{
		{name: "production по умолчанию", environment: "production", cookieValue: "", expected: true},
		{name: "production с явным false", environment: "production", cookieValue: "false", expected: false},
		{name: "production с явным true", environment: "production", cookieValue: "true", expected: true},
		{name: "development по умолчанию", environment: "development", cookieValue: "", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ENVIRONMENT", tc.environment)
			t.Setenv("SESSION_SECRET", strongSecret)
			t.Setenv("CSRF_SECRET", strongSecret)
			if tc.cookieValue != "" {
				t.Setenv("COOKIE_SECURE", tc.cookieValue)
			}

			assert.Equal(t, tc.expected, internal.LoadConfig().Web.CookieSecure)
		})
	}
}

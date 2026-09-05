package internal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
)

func productionConfig() *internal.Config {
	return &internal.Config{
		Environment: "production",
		Database:    internal.DatabaseConfig{Path: "./data/budget.db"},
	}
}

func TestConfig_Validate_Production_NoSecretsRequired(t *testing.T) {
	require.NoError(t, productionConfig().Validate(), "после плана 03 в конфиге нет секретов")
}

func TestConfig_Validate_RequiresDatabasePath(t *testing.T) {
	cfg := productionConfig()
	cfg.Database.Path = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database path")
}

// GetBackupDir кормит NewBackupService: пустой BACKUP_DIR должен давать
// каталог рядом с БД, иначе бэкапы уезжают мимо смонтированного тома.
func TestConfig_GetBackupDir(t *testing.T) {
	tests := []struct {
		name      string
		dbPath    string
		backupDir string
		want      string
	}{
		{name: "explicit", dbPath: "/data/budget.db", backupDir: "/backups", want: "/backups"},
		{name: "fallback", dbPath: "/data/budget.db", want: "/data/backups"},
		{name: "fallback relative", dbPath: "./data/budget.db", want: "data/backups"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &internal.Config{
				Database: internal.DatabaseConfig{Path: tt.dbPath, BackupDir: tt.backupDir},
			}

			assert.Equal(t, tt.want, cfg.GetBackupDir())
		})
	}
}

func TestLoadConfig_ReadsBackupDir(t *testing.T) {
	t.Setenv("BACKUP_DIR", "/mnt/backups")

	cfg := internal.LoadConfig()

	assert.Equal(t, "/mnt/backups", cfg.GetBackupDir())
}

// После плана 03 production не требует секретов: Validate проходит на голом окружении.
func TestLoadConfig_ProductionNeedsNoSecrets(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("LOG_LEVEL", "debug")

	cfg := internal.LoadConfig()

	require.NoError(t, cfg.Validate())
	assert.Equal(t, "info", cfg.Logging.Level, "debug в production понижается до info")
}

// TRUSTED_PROXIES решает, чей X-Forwarded-For попадает в лимитер логина; опечатка в CIDR
// должна останавливать старт, а не молча превращаться в «доверять никому».
func TestConfig_Validate_TrustedProxies(t *testing.T) {
	cfg := productionConfig()

	cfg.Server.TrustedProxies = "10.0.0.0/8, 172.16.0.0/12"
	require.NoError(t, cfg.Validate())
	assert.Len(t, cfg.TrustedProxyRanges(), 2)

	cfg.Server.TrustedProxies = ""
	require.NoError(t, cfg.Validate())
	assert.Empty(t, cfg.TrustedProxyRanges(), "пустой список — доверять только RemoteAddr")

	cfg.Server.TrustedProxies = "10.0.0.1"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TRUSTED_PROXIES")
}

func TestLoadConfig_TrustedProxies(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.18.0.0/16")

	assert.Equal(t, "172.18.0.0/16", internal.LoadConfig().Server.TrustedProxies)
}

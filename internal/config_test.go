package internal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal"
	"family-budget-service/internal/services"
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

// BACKUP_KEEP: неразбираемое и неположительное значение считается незаданным,
// иначе ретеншен молча удалил бы все файлы или сломался бы на опечатке.
func TestLoadConfig_ReadsBackupKeep(t *testing.T) {
	tests := map[string]int{
		"7":   7,
		"":    services.DefaultBackupKeep,
		"abc": services.DefaultBackupKeep,
		"0":   services.DefaultBackupKeep,
		"-1":  services.DefaultBackupKeep,
	}

	for value, want := range tests {
		t.Run("BACKUP_KEEP="+value, func(t *testing.T) {
			t.Setenv("BACKUP_KEEP", value)

			assert.Equal(t, want, internal.LoadConfig().Database.BackupKeep)
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
	ranges, err := cfg.TrustedProxyRanges()
	require.NoError(t, err)
	assert.Len(t, ranges, 2)

	cfg.Server.TrustedProxies = ""
	require.NoError(t, cfg.Validate())
	ranges, err = cfg.TrustedProxyRanges()
	require.NoError(t, err)
	assert.Empty(t, ranges, "пустой список — доверять только RemoteAddr")

	cfg.Server.TrustedProxies = "10.0.0.1"
	err = cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TRUSTED_PROXIES")
}

func TestLoadConfig_TrustedProxies(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "172.18.0.0/16")

	assert.Equal(t, "172.18.0.0/16", internal.LoadConfig().Server.TrustedProxies)
}

// METRICS_ADDR слушает служебный порт; «9091» без хоста открыло бы его наружу
// не там, где ждёт Alloy, поэтому host:port проверяется на старте.
func TestConfig_Validate_MetricsAddr(t *testing.T) {
	cfg := productionConfig()

	cfg.Server.MetricsAddr = ""
	require.NoError(t, cfg.Validate(), "пусто — метрики выключены")

	cfg.Server.MetricsAddr = "0.0.0.0:9091"
	require.NoError(t, cfg.Validate())

	cfg.Server.MetricsAddr = "9091"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "METRICS_ADDR")

	// SplitHostPort принимает и то, и другое; отказ на старте слушателя снял бы
	// вместе со служебным портом основной, поэтому порт проверяется здесь.
	for _, addr := range []string{"0.0.0.0:abc", "0.0.0.0:99999", "0.0.0.0:-1"} {
		cfg.Server.MetricsAddr = addr
		portErr := cfg.Validate()
		require.Error(t, portErr, addr)
		assert.Contains(t, portErr.Error(), "METRICS_ADDR", addr)
	}
}

func TestLoadConfig_MetricsAddr(t *testing.T) {
	t.Setenv("METRICS_ADDR", "127.0.0.1:9091")

	assert.Equal(t, "127.0.0.1:9091", internal.LoadConfig().Server.MetricsAddr)
}

func TestConfig_Validate_LLM(t *testing.T) {
	cfg := productionConfig()
	require.NoError(t, cfg.Validate(), "пустой хост — плечо выключено, срок и модель не проверяются")

	cfg.LLM = internal.LLMConfig{
		OllamaHost: "http://192.168.1.10:11434",
		Model:      "gemma4:31b-cloud",
		Timeout:    time.Minute,
	}
	require.NoError(t, cfg.Validate())

	tests := []struct {
		name   string
		mutate func(*internal.LLMConfig)
		want   string
	}{
		{
			name:   "host without scheme",
			mutate: func(c *internal.LLMConfig) { c.OllamaHost = "192.168.1.10:11434" },
			want:   "LLM_OLLAMA_HOST",
		},
		{
			name:   "unsupported scheme",
			mutate: func(c *internal.LLMConfig) { c.OllamaHost = "ftp://mini:11434" },
			want:   "LLM_OLLAMA_HOST",
		},
		{name: "empty model", mutate: func(c *internal.LLMConfig) { c.Model = "" }, want: "LLM_MODEL"},
		{name: "zero timeout", mutate: func(c *internal.LLMConfig) { c.Timeout = 0 }, want: "LLM_TIMEOUT"},
		{
			name:   "timeout above a minute",
			mutate: func(c *internal.LLMConfig) { c.Timeout = 61 * time.Second },
			want:   "LLM_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broken := *cfg
			tt.mutate(&broken.LLM)

			err := broken.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestLoadConfig_LLM(t *testing.T) {
	cfg := internal.LoadConfig()
	assert.Empty(t, cfg.LLM.OllamaHost)
	assert.Equal(t, "gemma4:31b-cloud", cfg.LLM.Model)
	assert.Equal(t, time.Minute, cfg.LLM.Timeout)

	t.Setenv("LLM_OLLAMA_HOST", "http://192.168.1.10:11434")
	t.Setenv("LLM_MODEL", "qwen3-vl:235b-cloud")
	t.Setenv("LLM_TIMEOUT", "45s")

	cfg = internal.LoadConfig()
	assert.Equal(t, internal.LLMConfig{
		OllamaHost: "http://192.168.1.10:11434",
		Model:      "qwen3-vl:235b-cloud",
		Timeout:    45 * time.Second,
	}, cfg.LLM)
}

package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmizerany/assert"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupWatcherFiles(t *testing.T) (configPath, secretPath string) {
	t.Helper()
	dir := t.TempDir()
	configPath = filepath.Join(dir, "config.yaml")
	secretPath = filepath.Join(dir, "secret.yaml")

	writeFile(t, configPath, `
strategy:
  name: test-strategy
params:
  key1: "100"
  key2: "hello"
`)
	writeFile(t, secretPath, `
exchanges:
  coinex:
    api_key: "initial_key"
    secret: "initial_secret"
influxdb:
  url: "http://localhost:8086"
  token: "initial_token"
discord:
  info_url: "https://discord.com/info"
  warn_url: "https://discord.com/warn"
  emergency_url: "https://discord.com/emergency"
`)
	return
}

func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name     string
		newConfig string
		wantKey1 string
	}{
		{
			name: "applies update",
			newConfig: `
strategy:
  name: test-strategy
params:
  key1: "200"
  key2: "world"
`,
			wantKey1: "200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, secretPath := setupWatcherFiles(t)

			noopCancel := func(error) {}
			w, err := NewWatcher(noopCancel, configPath, secretPath)
			assert.Equal(t, nil, err)

			assert.Equal(t, "100", w.GetConfig().Params.Get("key1"))

			writeFile(t, configPath, tt.newConfig)
			err = w.updateConfig()
			assert.Equal(t, nil, err)

			assert.Equal(t, tt.wantKey1, w.GetConfig().Params.Get("key1"))
		})
	}
}

func TestUpdateSecretFromFile(t *testing.T) {
	tests := []struct {
		name         string
		newSecret    string
		wantApiKey   string
		wantUrl      string
	}{
		{
			name: "updates coinex credential",
			newSecret: `
exchanges:
  coinex:
    api_key: "new_key"
    secret: "new_secret"
influxdb:
  url: "http://localhost:8086"
  token: "initial_token"
discord:
  info_url: "https://discord.com/info"
  warn_url: "https://discord.com/warn"
  emergency_url: "https://discord.com/emergency"
`,
			wantApiKey: "new_key",
			wantUrl:    "http://localhost:8086",
		},
		{
			name: "updates influxdb config",
			newSecret: `
exchanges:
  coinex:
    api_key: "initial_key"
    secret: "initial_secret"
influxdb:
  url: "http://new-host:8086"
  token: "new_token"
discord:
  info_url: "https://discord.com/info"
  warn_url: "https://discord.com/warn"
  emergency_url: "https://discord.com/emergency"
`,
			wantApiKey: "initial_key",
			wantUrl:    "http://new-host:8086",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, secretPath := setupWatcherFiles(t)
			noopCancel := func(error) {}
			w, err := NewWatcher(noopCancel, configPath, secretPath)
			assert.Equal(t, nil, err)

			assert.Equal(t, "initial_key", w.GetSecret().CoinEx.GetApiKey())
			assert.Equal(t, "http://localhost:8086", w.GetSecret().InfluxDB.GetUrl())

			writeFile(t, secretPath, tt.newSecret)
			err = w.updateSecretFromFile()
			assert.Equal(t, nil, err)

			assert.Equal(t, tt.wantApiKey, w.GetSecret().CoinEx.GetApiKey())
			assert.Equal(t, tt.wantUrl, w.GetSecret().InfluxDB.GetUrl())
		})
	}
}

func TestNewWatcher(t *testing.T) {
	t.Run("initializes config and secret from files", func(t *testing.T) {
		configPath, secretPath := setupWatcherFiles(t)
		noopCancel := func(error) {}
		w, err := NewWatcher(noopCancel, configPath, secretPath)
		assert.Equal(t, nil, err)

		assert.Equal(t, "test-strategy", w.GetConfig().Strategy.Name)
		assert.Equal(t, "100", w.GetConfig().Params.Get("key1"))
		assert.Equal(t, "hello", w.GetConfig().Params.Get("key2"))
		assert.Equal(t, "initial_key", w.GetSecret().CoinEx.GetApiKey())
		assert.Equal(t, "initial_secret", w.GetSecret().CoinEx.GetSecret())
		assert.Equal(t, "http://localhost:8086", w.GetSecret().InfluxDB.GetUrl())
		assert.Equal(t, "initial_token", w.GetSecret().InfluxDB.GetToken())
		assert.Equal(t, "https://discord.com/info", w.GetSecret().Discord.GetInfoUrl())
		assert.Equal(t, "https://discord.com/warn", w.GetSecret().Discord.GetWarnUrl())
		assert.Equal(t, "https://discord.com/emergency", w.GetSecret().Discord.GetEmergencyUrl())
	})
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

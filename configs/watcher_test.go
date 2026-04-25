package configs

import (
	"os"
	"path/filepath"
	"testing"
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
		name      string
		newConfig string
		wantKey1  string
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got, want := w.GetConfig().Params.Get("key1"), "100"; got != want {
				t.Errorf("got %v, want %v", got, want)
			}

			writeFile(t, configPath, tt.newConfig)
			err = w.updateConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got, want := w.GetConfig().Params.Get("key1"), tt.wantKey1; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestUpdateSecretFromFile(t *testing.T) {
	tests := []struct {
		name       string
		newSecret  string
		wantApiKey string
		wantUrl    string
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got, want := w.GetSecret().CoinEx.GetApiKey(), "initial_key"; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := w.GetSecret().InfluxDB.GetUrl(), "http://localhost:8086"; got != want {
				t.Errorf("got %v, want %v", got, want)
			}

			writeFile(t, secretPath, tt.newSecret)
			err = w.updateSecretFromFile()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got, want := w.GetSecret().CoinEx.GetApiKey(), tt.wantApiKey; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := w.GetSecret().InfluxDB.GetUrl(), tt.wantUrl; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestNewWatcher(t *testing.T) {
	t.Run("initializes config and secret from files", func(t *testing.T) {
		configPath, secretPath := setupWatcherFiles(t)
		noopCancel := func(error) {}
		w, err := NewWatcher(noopCancel, configPath, secretPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got, want := w.GetConfig().Strategy.Name, "test-strategy"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetConfig().Params.Get("key1"), "100"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetConfig().Params.Get("key2"), "hello"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().CoinEx.GetApiKey(), "initial_key"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().CoinEx.GetSecret(), "initial_secret"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().InfluxDB.GetUrl(), "http://localhost:8086"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().InfluxDB.GetToken(), "initial_token"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().Discord.GetInfoUrl(), "https://discord.com/info"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().Discord.GetWarnUrl(), "https://discord.com/warn"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := w.GetSecret().Discord.GetEmergencyUrl(), "https://discord.com/emergency"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
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

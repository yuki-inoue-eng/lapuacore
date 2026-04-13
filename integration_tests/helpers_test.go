//go:build integration

package integration_tests

import (
	"os"
	"testing"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

type testCredential struct {
	apiKey string
	secret string
}

func (c *testCredential) GetApiKey() string { return c.apiKey }
func (c *testCredential) GetSecret() string { return c.secret }

// requireCoinExCredential reads CoinEx credentials from environment variables.
// Skips the test if not set.
func requireCoinExCredential(t *testing.T) gateways.Credential {
	t.Helper()
	apiKey := os.Getenv("COINEX_API_KEY")
	secret := os.Getenv("COINEX_SECRET")
	if apiKey == "" || secret == "" {
		t.Skip("COINEX_API_KEY and COINEX_SECRET must be set")
	}
	return &testCredential{apiKey: apiKey, secret: secret}
}

// requireBybitCredential reads Bybit credentials from environment variables.
// Skips the test if not set.
func requireBybitCredential(t *testing.T) gateways.Credential {
	t.Helper()
	apiKey := os.Getenv("BYBIT_API_KEY")
	secret := os.Getenv("BYBIT_SECRET")
	if apiKey == "" || secret == "" {
		t.Skip("BYBIT_API_KEY and BYBIT_SECRET must be set")
	}
	return &testCredential{apiKey: apiKey, secret: secret}
}

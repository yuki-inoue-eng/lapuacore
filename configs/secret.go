package configs

import (
	"log/slog"
	"sync"
)

// Credential holds API credentials with thread-safe access.
// It satisfies the gateways.Credential interface.
type Credential struct {
	mu     sync.RWMutex
	apiKey string
	secret string
}

func (c *Credential) setApiKey(apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKey = apiKey
}
func (c *Credential) setSecret(secret string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.secret = secret
}

func (c *Credential) GetApiKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKey
}

func (c *Credential) GetSecret() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.secret
}

// AuthConfig holds authentication configuration for services like InfluxDB.
type AuthConfig struct {
	url   string
	token string
}

func (a *AuthConfig) setUrl(url string) {
	a.url = url
}

func (a *AuthConfig) setToken(token string) {
	a.token = token
}

func (a *AuthConfig) GetUrl() string {
	return a.url
}

func (a *AuthConfig) GetToken() string {
	return a.token
}

// Secret holds processed credentials for exchanges and InfluxDB.
type Secret struct {
	CoinEx   *Credential
	InfluxDB *AuthConfig
}

func newSecret(raw *RawSecret) *Secret {
	secret := &Secret{
		CoinEx: &Credential{
			apiKey: raw.Exchange.CoinEx.APIKey,
			secret: raw.Exchange.CoinEx.Secret,
		},
		InfluxDB: &AuthConfig{
			url:   raw.InfluxDB.Url,
			token: raw.InfluxDB.Token,
		},
	}
	return secret
}

// update conditionally updates credentials from a new raw secret,
// logging which exchanges were updated.
func (s *Secret) update(raw *RawSecret) {
	shouldUpdateCoinEx := s.CoinEx.GetApiKey() != raw.Exchange.CoinEx.APIKey ||
		s.CoinEx.GetSecret() != raw.Exchange.CoinEx.Secret

	shouldUpdateInfluxDB := s.InfluxDB.GetUrl() != raw.InfluxDB.Url ||
		s.InfluxDB.GetToken() != raw.InfluxDB.Token

	if shouldUpdateCoinEx {
		s.CoinEx.setApiKey(raw.Exchange.CoinEx.APIKey)
		s.CoinEx.setSecret(raw.Exchange.CoinEx.Secret)
		slog.Info("Updated CoinEx Credential")
	}

	if shouldUpdateInfluxDB {
		s.InfluxDB.setUrl(raw.InfluxDB.Url)
		s.InfluxDB.setToken(raw.InfluxDB.Token)
		slog.Info("Updated InfluxDB Credential")
	}
}

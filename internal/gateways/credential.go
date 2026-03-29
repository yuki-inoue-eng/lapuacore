package gateways

// Credential provides exchange API authentication credentials.
type Credential interface {
	GetApiKey() string
	GetSecret() string
}

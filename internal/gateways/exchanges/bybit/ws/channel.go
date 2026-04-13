package ws

import (
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

// NewPublicChannel creates a public WebSocket channel for Bybit market data.
func NewPublicChannel(product Product, measurer *gateways.LatencyMeasurer) *gateways.Channel {
	endpoint := "wss://" + HostName + APIv5 + ScopePathPublic + product.Path()
	return gateways.NewChannel(
		"bybit",
		endpoint,
		gateways.ScopeTypePublic,
		nil,
		measurer,
		nil,
		newHealthChecker,
	)
}

// NewPrivateChannel creates a private WebSocket channel for Bybit account data.
func NewPrivateChannel(credential gateways.Credential, measurer *gateways.LatencyMeasurer) *gateways.Channel {
	endpoint := "wss://" + HostName + APIv5 + ScopePathPrivate
	return gateways.NewChannel(
		"bybit",
		endpoint,
		gateways.ScopeTypePrivate,
		credential,
		measurer,
		newAuthor,
		newHealthChecker,
	)
}

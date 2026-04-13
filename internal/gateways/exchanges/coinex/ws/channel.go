package ws

import (
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

const wsEndpoint = "wss://socket.coinex.com/v2/futures"

// NewPublicChannel creates a public WebSocket channel for CoinEx market data.
func NewPublicChannel(measurer *gateways.LatencyMeasurer) *gateways.Channel {
	return gateways.NewChannel(
		"coinex",
		wsEndpoint,
		gateways.ScopeTypePublic,
		nil,
		measurer,
		nil,
		newHealthChecker,
	)
}

// NewPrivateChannel creates a private WebSocket channel for CoinEx account data.
func NewPrivateChannel(credential gateways.Credential, measurer *gateways.LatencyMeasurer) *gateways.Channel {
	return gateways.NewChannel(
		"coinex",
		wsEndpoint,
		gateways.ScopeTypePrivate,
		credential,
		measurer,
		newAuthor,
		newHealthChecker,
	)
}

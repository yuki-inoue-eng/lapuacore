package ws

import (
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

// NewPublicChannelGroup creates a group of N redundant public WebSocket channels.
func NewPublicChannelGroup(product Product, measurer *gateways.LatencyMeasurer, n int, dedupTTL time.Duration) *gateways.ChannelGroup {
	return gateways.NewChannelGroup(
		n,
		func() *gateways.Channel { return NewPublicChannel(product, measurer) },
		func() gateways.TopicManager { return topics.NewManager() },
		dedupTTL,
	)
}

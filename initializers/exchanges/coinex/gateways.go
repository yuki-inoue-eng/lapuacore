package coinex

import (
	"context"
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/agent"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

const defaultDedupTTL = 10 * time.Second

var gatewayManager *GatewayManager

// GatewayManager orchestrates CoinEx exchange connectivity.
type GatewayManager struct {
	cred            gateways.Credential
	latencyMeasurer *gateways.LatencyMeasurer

	insights *coinexInsights
	deals    *coinexDeals

	privateTopicMg *topics.Manager

	publicChGroup  *gateways.ChannelGroup
	privateChannel *gateways.Channel

	privateAPIAgent *agent.PrivateAPIAgent
}

// InitGatewayManager initializes the CoinEx gateway manager.
// publicChannelCount controls the number of redundant public WebSocket connections.
// Must be called after lapua.InitAndStart (requires lapua.Exporter).
func InitGatewayManager(cred gateways.Credential, publicChannelCount int) {
	const aggInterval = 5 * time.Second
	if cred == nil {
		panic(fmt.Errorf("credential must not be nil (simulator mode is not yet supported)"))
	}
	if lapua.Exporter == nil {
		panic(fmt.Errorf("lapua.Exporter must be initialized before calling InitGatewayManager"))
	}
	exporter := lapua.Exporter

	// initialize and set up ws latency measurer
	latencyMeasurer := gateways.NewLatencyMeasurer(aggInterval)
	exporter.SetLatencyMeasurer(latencyMeasurer)

	// initialize private channel
	privateAPIAgent := agent.NewPrivateAPIAgent(cred)
	privateTopicMg := topics.NewManager()
	privateChannel := ws.NewPrivateChannel(cred, latencyMeasurer)
	privateChannel.SetTopicMg(privateTopicMg)

	// initialize public channel group
	publicChGroup := ws.NewPublicChannelGroup(
		latencyMeasurer,
		publicChannelCount,
		defaultDedupTTL,
	)

	gatewayManager = &GatewayManager{
		cred:            cred,
		latencyMeasurer: latencyMeasurer,

		privateTopicMg: privateTopicMg,

		privateChannel:  privateChannel,
		publicChGroup:   publicChGroup,
		privateAPIAgent: privateAPIAgent,
	}
}

func (m *GatewayManager) setDeals(deals *coinexDeals) {
	m.deals = deals
}

func (m *GatewayManager) setInsights(insights *coinexInsights) {
	m.insights = insights
}

// StartGateway launches all gateway goroutines.
func StartGateway(ctx context.Context) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	go gatewayManager.latencyMeasurer.Start(ctx)

	gatewayManager.privateAPIAgent.Start(ctx)

	go func() {
		if err := gatewayManager.privateChannel.Start(ctx); err != nil {
			panic(err)
		}
	}()

	go gatewayManager.publicChGroup.Start(ctx)
}

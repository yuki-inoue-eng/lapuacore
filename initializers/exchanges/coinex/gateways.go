package coinex

import (
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
// If cred is nil, only public channels are initialized (no private channel or API agent).
func InitGatewayManager(cred gateways.Credential, publicChannelCount int) {
	const aggInterval = 5 * time.Second
	if lapua.Exporter == nil {
		panic(fmt.Errorf("lapua.Exporter must be initialized before calling InitGatewayManager"))
	}
	exporter := lapua.Exporter

	// initialize and set up ws latency measurer
	latencyMeasurer := gateways.NewLatencyMeasurer(aggInterval)
	exporter.SetLatencyMeasurer(latencyMeasurer)

	gm := &GatewayManager{
		cred:            cred,
		latencyMeasurer: latencyMeasurer,
	}

	// initialize private channel (only when credentials are provided)
	if cred != nil {
		gm.privateAPIAgent = agent.NewPrivateAPIAgent(cred)
		gm.privateTopicMg = topics.NewManager()
		gm.privateChannel = ws.NewPrivateChannel(cred, latencyMeasurer)
		gm.privateChannel.SetTopicMg(gm.privateTopicMg)
	}

	// initialize public channel group
	gm.publicChGroup = ws.NewPublicChannelGroup(
		latencyMeasurer,
		publicChannelCount,
		defaultDedupTTL,
	)

	gatewayManager = gm
}

func (m *GatewayManager) setDeals(deals *coinexDeals) {
	m.deals = deals
}

func (m *GatewayManager) setInsights(insights *coinexInsights) {
	m.insights = insights
}

// StartGateway launches all gateway goroutines using lapua.Ctx.
func StartGateway() {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	go gatewayManager.latencyMeasurer.Start(lapua.Ctx)

	if gatewayManager.privateAPIAgent != nil {
		gatewayManager.privateAPIAgent.Start(lapua.Ctx)
	}

	if gatewayManager.privateChannel != nil {
		go func() {
			if err := gatewayManager.privateChannel.Start(lapua.Ctx); err != nil {
				panic(err)
			}
		}()
	}

	if gatewayManager.publicChGroup != nil {
		go func() {
			if err := gatewayManager.publicChGroup.Start(lapua.Ctx); err != nil {
				panic(err)
			}
		}()
	}
}

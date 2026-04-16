package bybit

import (
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/agent"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

const defaultDedupTTL = 10 * time.Second

var gatewayManager *GatewayManager

// GatewayManager orchestrates Bybit exchange connectivity.
type GatewayManager struct {
	cred            gateways.Credential
	latencyMeasurer *gateways.LatencyMeasurer

	insights *bybitInsights
	deals    *bybitDeals

	// topics
	orderTopic        *topics.OrderTopic
	posTopic          *topics.PositionTopic
	orderBookTopicMap map[*OrderBookDesignator]*topics.OrderBookTopic
	tradeTopicMap     map[*domains.Symbol]*topics.TradeTopic

	privateTopicMg *topics.Manager

	privateChannel      *gateways.Channel
	publicLinearChGroup *gateways.ChannelGroup

	privateAPIAgent *agent.PrivateAPIAgent
}

// InitGatewayManager initializes the Bybit gateway manager.
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

		orderTopic:        topics.NewOrderTopic(),
		posTopic:          topics.NewPositionTopic(),
		orderBookTopicMap: map[*OrderBookDesignator]*topics.OrderBookTopic{},
		tradeTopicMap:     map[*domains.Symbol]*topics.TradeTopic{},
	}

	// initialize private channel (only when credentials are provided)
	if cred != nil {
		gm.privateAPIAgent = agent.NewPrivateAPIAgent(cred, latencyMeasurer)
		gm.privateTopicMg = topics.NewManager()
		gm.privateChannel = ws.NewPrivateChannel(cred, latencyMeasurer)
		gm.privateChannel.SetTopicMg(gm.privateTopicMg)
	}

	// initialize public channel group
	gm.publicLinearChGroup = ws.NewPublicChannelGroup(
		latencyMeasurer,
		publicChannelCount,
		defaultDedupTTL,
	)

	gatewayManager = gm
}

func (m *GatewayManager) setDeals(deals *bybitDeals) {
	m.deals = deals
}

func (m *GatewayManager) setInsights(insights *bybitInsights) {
	m.insights = insights
}

func (m *GatewayManager) getTradeTopic(symbol *domains.Symbol) *topics.TradeTopic {
	if _, ok := m.tradeTopicMap[symbol]; !ok {
		m.tradeTopicMap[symbol] = topics.NewTradeTopic(symbol)
	}
	return m.tradeTopicMap[symbol]
}

func (m *GatewayManager) getOrderBookTopic(designator *OrderBookDesignator) *topics.OrderBookTopic {
	if _, ok := m.orderBookTopicMap[designator]; !ok {
		m.orderBookTopicMap[designator] = topics.NewOrderBookTopic(designator.Symbol, designator.Depth)
	}
	return m.orderBookTopicMap[designator]
}

// StartGateway launches all gateway goroutines using lapua.Ctx.
func StartGateway() {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	go gatewayManager.latencyMeasurer.Start(lapua.Ctx)

	if gatewayManager.privateChannel != nil {
		go func() {
			if err := gatewayManager.privateChannel.Start(lapua.Ctx); err != nil {
				panic(err)
			}
		}()
	}

	if gatewayManager.publicLinearChGroup != nil {
		go func() {
			if err := gatewayManager.publicLinearChGroup.Start(lapua.Ctx); err != nil {
				panic(err)
			}
		}()
	}
}

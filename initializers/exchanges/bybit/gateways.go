package bybit

import (
	"context"
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/agent"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
	"github.com/yuki-inoue-eng/lapuacore/metrics"
)

const defaultDedupTTL = 10 * time.Second

var gatewayManager *GatewayManager

// GatewayManager orchestrates Bybit exchange connectivity.
type GatewayManager struct {
	cred            gateways.Credential
	exporter        *metrics.Exporter
	latencyMeasurer *gateways.LatencyMeasurer

	insights *bybitInsights
	deals    *bybitDeals

	// topics
	orderTopic        *topics.OrderTopic
	posTopic          *topics.PositionTopic
	orderBookTopicMap map[*OrderBookDesignator]*topics.OrderBookTopic
	tradeTopicMap     map[*domains.Symbol]*topics.TradeTopic

	privateTopicMg *topics.Manager

	privateChannel        *gateways.Channel
	publicLinearChGroup   *gateways.ChannelGroup

	privateAPIAgent *agent.PrivateAPIAgent
}

// InitGatewayManager initializes the Bybit gateway manager.
// publicChannelCount controls the number of redundant public WebSocket connections.
func InitGatewayManager(cred gateways.Credential, exporter *metrics.Exporter, publicChannelCount int) {
	const aggInterval = 5 * time.Second

	if cred == nil {
		panic(fmt.Errorf("credential must not be nil (simulator mode is not yet supported)"))
	}
	if exporter == nil {
		panic(fmt.Errorf("exporter must not be nil"))
	}

	latencyMeasurer := gateways.NewLatencyMeasurer(aggInterval)
	exporter.SetLatencyMeasurer(latencyMeasurer)

	privateAPIAgent := agent.NewPrivateAPIAgent(cred, latencyMeasurer)

	privateTopicMg := topics.NewManager()

	privateChannel := ws.NewPrivateChannel(cred, latencyMeasurer)
	privateChannel.SetTopicMg(privateTopicMg)

	publicLinearChGroup := ws.NewPublicChannelGroup(ws.ProductLinear, latencyMeasurer, publicChannelCount, defaultDedupTTL)

	gatewayManager = &GatewayManager{
		cred:            cred,
		exporter:        exporter,
		latencyMeasurer: latencyMeasurer,

		orderTopic:        topics.NewOrderTopic(),
		posTopic:          topics.NewPositionTopic(),
		orderBookTopicMap: map[*OrderBookDesignator]*topics.OrderBookTopic{},
		tradeTopicMap:     map[*domains.Symbol]*topics.TradeTopic{},

		privateTopicMg: privateTopicMg,

		privateChannel:        privateChannel,
		publicLinearChGroup:   publicLinearChGroup,
		privateAPIAgent:       privateAPIAgent,
	}
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

// StartGateway launches all gateway goroutines.
func StartGateway(ctx context.Context) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	go gatewayManager.latencyMeasurer.Start(ctx)

	if gatewayManager.privateChannel != nil {
		go func() {
			if err := gatewayManager.privateChannel.Start(ctx); err != nil {
				panic(err)
			}
		}()
	}

	if gatewayManager.publicLinearChGroup != nil {
		go gatewayManager.publicLinearChGroup.Start(ctx)
	}
}

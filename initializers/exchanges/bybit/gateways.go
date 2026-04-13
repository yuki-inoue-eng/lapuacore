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

	privateChannel     *ws.Channel
	publicLinearChannel *ws.Channel

	privateAPIAgent *agent.PrivateAPIAgent
}

// InitGatewayManager initializes the Bybit gateway manager.
func InitGatewayManager(cred gateways.Credential, exporter *metrics.Exporter) {
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
	privateChannel := ws.NewPrivateChannel(cred, latencyMeasurer)
	publicLinearChannel := ws.NewPublicChannel(ws.ProductLinear, latencyMeasurer)

	gatewayManager = &GatewayManager{
		cred:            cred,
		exporter:        exporter,
		latencyMeasurer: latencyMeasurer,

		orderTopic:        topics.NewOrderTopic(),
		posTopic:          topics.NewPositionTopic(),
		orderBookTopicMap: map[*OrderBookDesignator]*topics.OrderBookTopic{},
		tradeTopicMap:     map[*domains.Symbol]*topics.TradeTopic{},

		privateChannel:     privateChannel,
		publicLinearChannel: publicLinearChannel,
		privateAPIAgent:    privateAPIAgent,
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

	if gatewayManager.publicLinearChannel != nil {
		go func() {
			if err := gatewayManager.publicLinearChannel.Start(ctx); err != nil {
				panic(err)
			}
		}()
	}
}

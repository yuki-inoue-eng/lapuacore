package bybit

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	ex "github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

// Insights holds Bybit market data sources.
// Initialized by InitInsights().
var Insights *bybitInsights

type bybitInsights struct {
	trades     map[*domains.Symbol]*insights.Trade
	orderBooks map[*domains.Symbol]map[topics.OBDepth]*insights.OrderBook
}

func (i *bybitInsights) EXName() string {
	return "bybit"
}

func (i *bybitInsights) GetOrderBook(designator *OrderBookDesignator) *insights.OrderBook {
	obs, ok := i.orderBooks[designator.Symbol]
	if !ok {
		return nil
	}
	ob, ok := obs[designator.Depth]
	if !ok {
		return nil
	}
	return ob
}

func (i *bybitInsights) GetTrade(symbol *domains.Symbol) *insights.Trade {
	tr, ok := i.trades[symbol]
	if !ok {
		return nil
	}
	return tr
}

func (i *bybitInsights) IsEverythingReady() bool {
	for _, o := range i.orderBooks {
		for _, ob := range o {
			if !ob.IsReady() {
				return false
			}
		}
	}
	return true
}

// InitInsights initializes Bybit market data for the given symbols.
func InitInsights(
	tradeSymbols []*domains.Symbol,
	obDesignators []*OrderBookDesignator,
) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup trades
	trades := map[*domains.Symbol]*insights.Trade{}
	for _, symbol := range tradeSymbols {
		trades[symbol] = insights.NewTrade(symbol)
	}

	// setup trade topics
	var linearTradeTopics []topics.Topic
	for symbol, trade := range trades {
		tradeTopic := gatewayManager.getTradeTopic(symbol)
		tradeTopic.SetHandler(trade.Update)
		linearTradeTopics = append(linearTradeTopics, tradeTopic)
	}

	// setup orderBooks
	orderBooks := map[*domains.Symbol]map[topics.OBDepth]*insights.OrderBook{}
	for _, designator := range obDesignators {
		orderBook := insights.NewOrderBook(designator.Symbol)
		if orderBooks[designator.Symbol] == nil {
			orderBooks[designator.Symbol] = map[topics.OBDepth]*insights.OrderBook{designator.Depth: orderBook}
		} else {
			orderBooks[designator.Symbol][designator.Depth] = orderBook
		}
	}

	// setup orderBook topics
	var linearOBTopics []topics.Topic
	for _, designator := range obDesignators {
		orderBook := orderBooks[designator.Symbol][designator.Depth]
		obTopic := gatewayManager.getOrderBookTopic(designator)
		obTopic.SetHandler(orderBook.UpdateByOBData)
		linearOBTopics = append(linearOBTopics, obTopic)
	}

	// set topics on public channel
	if gatewayManager.publicLinearChannel != nil {
		gatewayManager.publicLinearChannel.SetTopics(linearTradeTopics)
		gatewayManager.publicLinearChannel.SetTopics(linearOBTopics)
	}

	ins := &bybitInsights{
		trades:     trades,
		orderBooks: orderBooks,
	}
	gatewayManager.setInsights(ins)
	Insights = ins
	ex.AppendInsight(ins)
}

// OrderBookDesignator identifies an order book by symbol and depth.
type OrderBookDesignator struct {
	Symbol *domains.Symbol
	Depth  topics.OBDepth
}

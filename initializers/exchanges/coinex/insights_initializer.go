package coinex

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	ex "github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

// Insights holds CoinEx market data sources.
// Initialized by InitInsights().
var Insights *coinexInsights

type coinexInsights struct {
	orderBooks map[*domains.Symbol]*insights.OrderBook
}

func (i *coinexInsights) EXName() string {
	return "coinex"
}

func (i *coinexInsights) GetOrderBook(symbol *domains.Symbol) *insights.OrderBook {
	ob, ok := i.orderBooks[symbol]
	if !ok {
		return nil
	}
	return ob
}

func (i *coinexInsights) IsEverythingReady() bool {
	for _, ob := range i.orderBooks {
		if !ob.IsReady() {
			return false
		}
	}
	return true
}

// InitInsights initializes CoinEx market data for the given symbols.
func InitInsights(obSymbols []*domains.Symbol) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup orderBooks
	orderBooks := map[*domains.Symbol]*insights.OrderBook{}
	for _, symbol := range obSymbols {
		orderBooks[symbol] = insights.NewOrderBook(symbol)
	}

	// setup orderBook topics
	var obTopics []topics.Topic
	for symbol, ob := range orderBooks {
		obTopic := topics.NewOrderBookTopic(symbol.Name())
		obTopic.SetHandler(ob.UpdateByOBData)
		obTopics = append(obTopics, obTopic)
		_ = symbol // used as map key
	}

	// set topics on public channel
	if gatewayManager.publicChannel != nil {
		gatewayManager.publicChannel.SetTopics(obTopics)
	}

	ins := &coinexInsights{
		orderBooks: orderBooks,
	}
	gatewayManager.setInsights(ins)
	Insights = ins
	ex.AppendInsight(ins)
}

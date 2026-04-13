package bybit

import (
	"log/slog"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	ex "github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/translators"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

// Insights holds Bybit market data sources.
// Initialized by InitInsights().
var Insights *bybitInsights

type bybitInsights struct {
	trades      map[*domains.Symbol]*insights.TradeImpl
	orderBooks  map[*domains.Symbol]map[topics.OBDepth]*insights.OrderBookImpl
	bookTickers map[*domains.Symbol]*insights.BookTickerImpl
}

func (i *bybitInsights) EXName() string {
	return "bybit"
}

func (i *bybitInsights) GetOrderBook(designator *OrderBookDesignator) insights.OrderBook {
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

func (i *bybitInsights) GetTrade(symbol *domains.Symbol) insights.Trade {
	tr, ok := i.trades[symbol]
	if !ok {
		return nil
	}
	return tr
}

func (i *bybitInsights) GetBookTicker(symbol *domains.Symbol) insights.BookTicker {
	bt, ok := i.bookTickers[symbol]
	if !ok {
		return nil
	}
	return bt
}

func (i *bybitInsights) IsEverythingReady() bool {
	for _, o := range i.orderBooks {
		for _, ob := range o {
			if !ob.IsReady() {
				return false
			}
		}
	}
	for _, bt := range i.bookTickers {
		if !bt.IsReady() {
			return false
		}
	}
	return true
}

// InitInsights initializes Bybit market data for the given symbols.
func InitInsights(
	tradeSymbols []*domains.Symbol,
	obDesignators []*OrderBookDesignator,
	btSymbols []*domains.Symbol,
) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup trades
	trades := map[*domains.Symbol]*insights.TradeImpl{}
	for _, symbol := range tradeSymbols {
		trades[symbol] = insights.NewTrade(symbol)
	}

	// setup trade topics
	var linearTradeTopics []gateways.Topic
	for symbol, trade := range trades {
		tradeTopic := gatewayManager.getTradeTopic(symbol)
		tradeTopic.SetHandler(trade.Update)
		linearTradeTopics = append(linearTradeTopics, tradeTopic)
	}

	// setup orderBooks
	orderBooks := map[*domains.Symbol]map[topics.OBDepth]*insights.OrderBookImpl{}
	for _, designator := range obDesignators {
		orderBook := insights.NewOrderBook(designator.Symbol)
		if orderBooks[designator.Symbol] == nil {
			orderBooks[designator.Symbol] = map[topics.OBDepth]*insights.OrderBookImpl{designator.Depth: orderBook}
		} else {
			orderBooks[designator.Symbol][designator.Depth] = orderBook
		}
	}

	// setup orderBook topics
	var linearOBTopics []gateways.Topic
	for _, designator := range obDesignators {
		orderBook := orderBooks[designator.Symbol][designator.Depth]
		obTopic := gatewayManager.getOrderBookTopic(designator)
		obTopic.SetHandler(orderBook.UpdateByOBData)
		linearOBTopics = append(linearOBTopics, obTopic)
	}

	// setup bookTickers (via orderbook depth=1 adapter)
	bookTickers := map[*domains.Symbol]*insights.BookTickerImpl{}
	adapter := translators.NewBookTickerAdapter()
	for _, symbol := range btSymbols {
		bt := insights.NewBookTicker(symbol)
		bookTickers[symbol] = bt

		designator := &OrderBookDesignator{Symbol: symbol, Depth: topics.LinearOBDepth1}
		obTopic := gatewayManager.getOrderBookTopic(designator)
		obTopic.SetHandler(func(data *insights.OrderBookData) {
			btData, err := adapter.Convert(data)
			if err != nil {
				slog.Error("failed to convert orderbook to bookticker", "error", err)
				return
			}
			bt.Update(btData)
		})
		linearOBTopics = append(linearOBTopics, obTopic)
	}

	// set topics on public channel group
	if gatewayManager.publicLinearChGroup != nil {
		gatewayManager.publicLinearChGroup.SetTopics(linearTradeTopics)
		gatewayManager.publicLinearChGroup.SetTopics(linearOBTopics)
	}

	ins := &bybitInsights{
		trades:      trades,
		orderBooks:  orderBooks,
		bookTickers: bookTickers,
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

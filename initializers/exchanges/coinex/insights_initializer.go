package coinex

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	ex "github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

// Insights holds CoinEx market data sources.
// Initialized by InitInsights().
var Insights *coinexInsights

type coinexInsights struct {
	trades      map[*domains.Symbol]*insights.Trade
	orderBooks  map[*domains.Symbol]*insights.OrderBook
	bookTickers map[*domains.Symbol]*insights.BookTicker
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

func (i *coinexInsights) GetTrade(symbol *domains.Symbol) *insights.Trade {
	tr, ok := i.trades[symbol]
	if !ok {
		return nil
	}
	return tr
}

func (i *coinexInsights) GetBookTicker(symbol *domains.Symbol) *insights.BookTicker {
	bt, ok := i.bookTickers[symbol]
	if !ok {
		return nil
	}
	return bt
}

func (i *coinexInsights) IsEverythingReady() bool {
	for _, tr := range i.trades {
		if !tr.IsReady() {
			return false
		}
	}
	for _, ob := range i.orderBooks {
		if !ob.IsReady() {
			return false
		}
	}
	for _, bt := range i.bookTickers {
		if !bt.IsReady() {
			return false
		}
	}
	return true
}

// InitInsights initializes CoinEx market data for the given symbols.
func InitInsights(tradeSymbols []*domains.Symbol, obSymbols []*domains.Symbol, btSymbols []*domains.Symbol) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup trades
	trades := map[*domains.Symbol]*insights.Trade{}
	for _, symbol := range tradeSymbols {
		trades[symbol] = insights.NewTrade(symbol)
	}

	// setup trade topics
	var tradeTopics []gateways.Topic
	for symbol, trade := range trades {
		tradeTopic := topics.NewTradeTopic(symbol)
		tradeTopic.SetHandler(trade.Update)
		tradeTopics = append(tradeTopics, tradeTopic)
	}

	// setup orderBooks
	orderBooks := map[*domains.Symbol]*insights.OrderBook{}
	for _, symbol := range obSymbols {
		orderBooks[symbol] = insights.NewOrderBook(symbol)
	}

	// setup orderBook topics
	var obTopics []gateways.Topic
	for symbol, ob := range orderBooks {
		obTopic := topics.NewOrderBookTopic(symbol)
		obTopic.SetHandler(ob.UpdateByOBData)
		obTopics = append(obTopics, obTopic)
		_ = symbol // used as map key
	}

	// setup bookTickers
	bookTickers := map[*domains.Symbol]*insights.BookTicker{}
	for _, symbol := range btSymbols {
		bookTickers[symbol] = insights.NewBookTicker(symbol)
	}

	// setup bookTicker topics
	var btTopics []gateways.Topic
	for symbol, bt := range bookTickers {
		btTopic := topics.NewBookTickerTopic(symbol)
		btTopic.SetHandler(bt.Update)
		btTopics = append(btTopics, btTopic)
		_ = symbol // used as map key
	}

	// set topics on public channel
	if gatewayManager.publicTopicMg != nil {
		gatewayManager.publicTopicMg.SetTopics(tradeTopics)
		gatewayManager.publicTopicMg.SetTopics(obTopics)
		gatewayManager.publicTopicMg.SetTopics(btTopics)
	}

	ins := &coinexInsights{
		trades:      trades,
		orderBooks:  orderBooks,
		bookTickers: bookTickers,
	}
	gatewayManager.setInsights(ins)
	Insights = ins
	ex.AppendInsight(ins)
}

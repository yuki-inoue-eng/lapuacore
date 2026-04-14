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
	trades      map[*domains.Symbol]*insights.TradeImpl
	orderBooks  map[*domains.Symbol]*insights.OrderBookImpl
	quotes map[*domains.Symbol]*insights.QuoteImpl
}

func (i *coinexInsights) EXName() string {
	return "coinex"
}

func (i *coinexInsights) GetOrderBook(symbol *domains.Symbol) insights.OrderBook {
	ob, ok := i.orderBooks[symbol]
	if !ok {
		return nil
	}
	return ob
}

func (i *coinexInsights) GetTrade(symbol *domains.Symbol) insights.Trade {
	tr, ok := i.trades[symbol]
	if !ok {
		return nil
	}
	return tr
}

func (i *coinexInsights) GetQuote(symbol *domains.Symbol) insights.Quote {
	bt, ok := i.quotes[symbol]
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
	for _, q := range i.quotes {
		if !q.IsReady() {
			return false
		}
	}
	return true
}

// InitInsights initializes CoinEx market data for the given symbols.
func InitInsights(tradeSymbols []*domains.Symbol, obSymbols []*domains.Symbol, quoteSymbols []*domains.Symbol) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup trades
	trades := map[*domains.Symbol]*insights.TradeImpl{}
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
	orderBooks := map[*domains.Symbol]*insights.OrderBookImpl{}
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

	// setup quotes
	quotes := map[*domains.Symbol]*insights.QuoteImpl{}
	for _, symbol := range quoteSymbols {
		quotes[symbol] = insights.NewQuote(symbol)
	}

	// setup quote topics
	var quoteTopics []gateways.Topic
	for symbol, quote := range quotes {
		quoteTopic := topics.NewBookTickerTopic(symbol)
		quoteTopic.SetHandler(quote.Update)
		quoteTopics = append(quoteTopics, quoteTopic)
		_ = symbol // used as map key
	}

	// set topics on public channel group
	if gatewayManager.publicChGroup != nil {
		gatewayManager.publicChGroup.SetTopics(tradeTopics)
		gatewayManager.publicChGroup.SetTopics(obTopics)
		gatewayManager.publicChGroup.SetTopics(quoteTopics)
	}

	ins := &coinexInsights{
		trades:      trades,
		orderBooks:  orderBooks,
		quotes: quotes,
	}
	gatewayManager.setInsights(ins)
	Insights = ins
	ex.AppendInsight(ins)
}

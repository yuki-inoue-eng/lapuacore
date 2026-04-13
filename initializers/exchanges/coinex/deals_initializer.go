package coinex

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

// Deals holds CoinEx trading infrastructure.
// Initialized by InitDeals().
var Deals *coinexDeals

type coinexDeals struct {
	dealers map[*domains.Symbol]*deals.Dealer
}

func (d *coinexDeals) GetDealer(symbol *domains.Symbol) *deals.Dealer {
	dealer, ok := d.dealers[symbol]
	if !ok {
		return nil
	}
	return dealer
}

// InitDeals initializes CoinEx dealing (order management) for the given symbols.
func InitDeals(symbols []*domains.Symbol, onError func(err error)) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup dealers
	dealers := map[*domains.Symbol]*deals.Dealer{}
	for _, symbol := range symbols {
		dealers[symbol] = deals.NewDealer(symbol, gatewayManager.privateAPIAgent, onError)
	}

	// setup order topics
	var orderTopics []topics.Topic
	for symbol, dealer := range dealers {
		orderTopic := topics.NewOrderTopic(symbol)
		orderTopic.SetHandler(dealer.HandleOrderData)
		orderTopics = append(orderTopics, orderTopic)
	}

	// setup position topics
	var posTopics []topics.Topic
	for symbol, dealer := range dealers {
		posTopic := topics.NewPositionTopic(symbol)
		posTopic.SetHandler(dealer.HandlePositionData)
		posTopics = append(posTopics, posTopic)
	}

	// set topics on private channel
	if gatewayManager.privateChannel != nil {
		gatewayManager.privateChannel.SetTopics(orderTopics)
		gatewayManager.privateChannel.SetTopics(posTopics)
	}

	dls := &coinexDeals{
		dealers: dealers,
	}
	gatewayManager.setDeals(dls)
	Deals = dls
}

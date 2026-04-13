package bybit

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

// Deals holds Bybit trading infrastructure.
// Initialized by InitDeals().
var Deals *bybitDeals

type bybitDeals struct {
	dealers map[*domains.Symbol]*deals.Dealer
}

func (d *bybitDeals) GetDealer(symbol *domains.Symbol) *deals.Dealer {
	dealer, ok := d.dealers[symbol]
	if !ok {
		return nil
	}
	return dealer
}

// InitDeals initializes Bybit dealing (order management) for the given symbols.
func InitDeals(symbols []*domains.Symbol, onError func(err error)) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	// setup dealers
	dealers := map[*domains.Symbol]*deals.Dealer{}
	for _, symbol := range symbols {
		dealers[symbol] = deals.NewDealer(symbol, gatewayManager.privateAPIAgent, onError)
	}

	// setup order topic
	for symbol, dealer := range dealers {
		gatewayManager.orderTopic.SetHandlers(symbol, dealer.HandleOrderData)
	}

	// setup position topic
	for symbol, dealer := range dealers {
		gatewayManager.posTopic.SetHandlers(symbol, dealer.HandlePositionData)
	}

	// set topics on private channel
	if gatewayManager.privateChannel != nil {
		gatewayManager.privateChannel.SetTopics([]topics.Topic{
			gatewayManager.orderTopic,
			gatewayManager.posTopic,
		})
	}

	dls := &bybitDeals{
		dealers: dealers,
	}
	gatewayManager.setDeals(dls)
	Deals = dls
}

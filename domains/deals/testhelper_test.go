package deals

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/mutex"
)

// mockAgent is a test double for the Agent interface.
type mockAgent struct {
	sendOrderFunc    func(*domains.Symbol, *Order, CreateOrderRespHandler) error
	sendOrdersFunc   func(*domains.Symbol, []*Order, CreateOrdersRespHandler) error
	cancelOrderFunc  func(*domains.Symbol, *Order, CancelOrderRespHandler) error
	cancelOrdersFunc func(*domains.Symbol, []*Order, CancelOrdersRespHandler) error
	amendOrderFunc   func(*domains.Symbol, *Order, AmendDetail, AmendOrderRespHandler) error
	amendOrdersFunc  func(*domains.Symbol, map[*Order]AmendDetail, AmendOrdersRespHandler) error
}

func (m *mockAgent) SendOrder(s *domains.Symbol, o *Order, h CreateOrderRespHandler) error {
	if m.sendOrderFunc != nil {
		return m.sendOrderFunc(s, o, h)
	}
	return nil
}

func (m *mockAgent) SendOrders(s *domains.Symbol, os []*Order, h CreateOrdersRespHandler) error {
	if m.sendOrdersFunc != nil {
		return m.sendOrdersFunc(s, os, h)
	}
	return nil
}

func (m *mockAgent) CancelOrder(s *domains.Symbol, o *Order, h CancelOrderRespHandler) error {
	if m.cancelOrderFunc != nil {
		return m.cancelOrderFunc(s, o, h)
	}
	return nil
}

func (m *mockAgent) CancelOrders(s *domains.Symbol, os []*Order, h CancelOrdersRespHandler) error {
	if m.cancelOrdersFunc != nil {
		return m.cancelOrdersFunc(s, os, h)
	}
	return nil
}

func (m *mockAgent) AmendOrder(s *domains.Symbol, o *Order, d AmendDetail, h AmendOrderRespHandler) error {
	if m.amendOrderFunc != nil {
		return m.amendOrderFunc(s, o, d, h)
	}
	return nil
}

func (m *mockAgent) AmendOrders(s *domains.Symbol, ds map[*Order]AmendDetail, h AmendOrdersRespHandler) error {
	if m.amendOrdersFunc != nil {
		return m.amendOrdersFunc(s, ds, h)
	}
	return nil
}

// newTestDealer creates a Dealer without using the singleton registry.
func newTestDealer(agent Agent) *Dealer {
	return &Dealer{
		acceptOrder:       mutex.NewFlag(true),
		onError:           func(err error) {},
		agent:             agent,
		Symbol:            domains.SymbolCoinExFuturesBTCUSDT,
		LivingOrders:      NewOrdersMap(nil),
		UnrelatedOrders:   NewOrdersMap(nil),
		amendingDetailMap: mutex.NewMap[string, AmendDetail](nil),
		doneOrders:        NewOrderMuArray(nil),
		CurrentPosition:   NewPosition(),
	}
}
package deals

import "github.com/yuki-inoue-eng/lapuacore/domains"

// Agent is the interface for exchange-specific HTTP API operations.
type Agent interface {
	SendOrders(symbol *domains.Symbol, orders []*Order, handler CreateOrdersRespHandler) error
	CancelOrders(symbol *domains.Symbol, orders []*Order, handler CancelOrdersRespHandler) error
	AmendOrders(symbol *domains.Symbol, details map[*Order]AmendDetail, handler AmendOrdersRespHandler) error

	SendOrder(symbol *domains.Symbol, order *Order, handler CreateOrderRespHandler) error
	CancelOrder(symbol *domains.Symbol, order *Order, handler CancelOrderRespHandler) error
	AmendOrder(symbol *domains.Symbol, order *Order, amendDetail AmendDetail, handler AmendOrderRespHandler) error
}
package deals

import "github.com/yuki-inoue-eng/lapuacore/domains"

// Dealer is the consumer-facing interface for order management.
// It excludes HandleOrderData and HandlePositionData which are only used by internal gateways.
type Dealer interface {
	GetSymbol() *domains.Symbol
	GetLivingOrders() *OrdersMutexMap
	GetUnrelatedOrders() *OrdersMutexMap
	GetCurrentPosition() *Position
	SendOrder(order *Order) error
	SendOrders(orders []*Order) error
	AmendOrder(order *Order, detail AmendDetail) error
	AmendOrders(details map[*Order]AmendDetail) error
	CancelOrder(order *Order) error
	CancelOrders(orders Orders) error
	ExportDoneOrders() *OrderMutexSlice
	SetPosUpdatedHandler(handler PositionDataHandler)
}

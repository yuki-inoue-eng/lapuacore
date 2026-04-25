package deals

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type OrderDataStatus int

const (
	OrderDataStatusUnknown         OrderDataStatus = iota
	OrderDataStatusOpened                          // Order was accepted successfully
	OrderDataStatusCanceled                        // Order was cancelled successfully
	OrderDataStatusFilled                          // Order was fully filled
	OrderDataStatusPartiallyFilled                 // Order was partially filled
	OrderDataStatusRejected                        // Order was rejected
)

// OrderData is order information received via WebSocket.
type OrderData struct {
	ID       string
	PublicID string
	Side     domains.Side    // Used only for order amend
	Price    decimal.Decimal // Used only for order amend
	Qty      decimal.Decimal // Used only for order amend

	AvgExecPrice decimal.Decimal // Average execution price (used for fill and partial fill)
	CumFee       decimal.Decimal // Cumulative fee (used for fill and partial fill)
	CumExecQty   decimal.Decimal // Cumulative executed quantity (used for fill and partial fill)

	Status      OrderDataStatus
	ArrivedAt   *time.Time // Time the event arrived at or occurred on the exchange
	ConfirmedAt *time.Time // Time the operation was confirmed as accepted
}

func (o *OrderData) isDone() bool {
	return o.Status == OrderDataStatusFilled ||
		o.Status == OrderDataStatusCanceled ||
		o.Status == OrderDataStatusRejected
}

func (o *OrderData) isUnrelated() bool {
	return o.ID == ""
}

func (o *OrderData) DoneReason() OrderDoneReason {
	switch o.Status {
	case OrderDataStatusFilled:
		return OrderDoneReasonFilled
	case OrderDataStatusCanceled:
		return OrderDoneReasonCanceled
	case OrderDataStatusRejected:
		return OrderDoneReasonRejected
	default:
		return OrderDoneReasonUnknown
	}
}

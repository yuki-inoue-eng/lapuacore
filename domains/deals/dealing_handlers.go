package deals

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// Handler types for HTTP response callbacks.

type CreateOrdersRespHandler func(resps CreateOrdersRespMap, err error)
type CancelOrdersRespHandler func(resps CancelOrdersRespMap, err error)
type AmendOrdersRespHandler func(resps AmendOrdersRespMap, err error)
type CreateOrderRespHandler func(resp CreateOrderResp, err error)
type AmendOrderRespHandler func(resp AmendOrderResp, err error)
type CancelOrderRespHandler func(resp CancelOrderResp, err error)

// Handler types for WebSocket callbacks.

type OrderDataHandler func(msg []*OrderData)

func (d *Dealer) handleSendOrdersResp(resps CreateOrdersRespMap, err error) {
	orders := d.LivingOrders.getSendingOrders(resps.IDs())
	if err != nil {
		d.rejectCreates(orders)
		slog.Error(fmt.Sprintf("send orders resp: %s", err.Error()))
		d.noticeError(err)
		return
	}
	for _, order := range orders {
		resp := resps[order.GetID()]
		d.handleSendOrderResp(*resp, nil)
	}
}

// handleSendOrderResp processes the send order response.
// If the order was already confirmed via WebSocket, it is ignored.
func (d *Dealer) handleSendOrderResp(resp CreateOrderResp, err error) {
	order := d.LivingOrders.getSendingOrder(resp.OrderID)
	if order == nil {
		return
	}
	order.WithOpLock(func() {
		if err != nil {
			d.rejectCreate(order)
			slog.Error(fmt.Sprintf("send order resp: %s", err.Error()))
			d.noticeError(err)
			return
		}
		if err := resp.Err; err != nil {
			d.rejectCreate(order)
			slog.Error(fmt.Sprintf("send order error: %s", err.Error()))
			d.noticeError(err)
			return
		}
		if order.GetStatus() == OrderStatusSending {
			orderType := order.GetOrderType()
			switch orderType {
			default:
				return
			case domains.OrderTypeMarket:
				// Handled via private channel notification; nothing to do here.
				return
			case domains.OrderTypeLimit, domains.OrderTypeLimitFOK, domains.OrderTypeLimitIOC, domains.OrderTypeLimitMaker:
				d.acceptCreate(order, resp.PublicID, resp.ArrivedAt, resp.ConfirmedAt)
				return
			}
		}
	})
}

func (d *Dealer) handleAmendOrdersResp(resps AmendOrdersRespMap, err error) {
	orders := d.LivingOrders.getAmendingOrders(resps.IDs())
	if err != nil {
		d.rejectAmends(orders)
		slog.Error(fmt.Sprintf("amend orders resp: %s", err.Error()))
		d.noticeError(err)
		return
	}
	for _, order := range orders {
		resp := resps[order.GetID()]
		d.handleAmendOrderResp(*resp, nil)
	}
}

func (d *Dealer) handleAmendOrderResp(resp AmendOrderResp, err error) {
	order := d.LivingOrders.getAmendingOrder(resp.OrderID)
	if order == nil {
		return
	}
	order.WithOpLock(func() {
		if err != nil {
			if errors.Is(err, OrderIsNotExists) {
				d.abandonOrder(order)
				return
			}
			d.rejectAmend(order)
			slog.Error(fmt.Sprintf("amend order resp: %s", err.Error()))
			d.noticeError(err)
			return
		}
		if err := resp.Err; err != nil {
			if errors.Is(err, OrderIsNotExists) {
				d.abandonOrder(order)
				return
			}
			d.rejectAmend(order)
			slog.Error(fmt.Sprintf("amend order resp: %s", err.Error()))
			d.noticeError(err)
			return
		}
		d.acceptAmend(order, resp.Detail, resp.ArrivedAt, resp.ConfirmedAt)
	})
}

func (d *Dealer) handleCancelOrdersResp(resps CancelOrdersRespMap, err error) {
	orders := d.LivingOrders.getCancelingOrders(resps.IDs())
	if err != nil {
		d.rejectCancels(orders)
		slog.Error(fmt.Sprintf("cancel orders resp: %s", err.Error()))
		d.noticeError(err)
		return
	}
	for _, order := range orders {
		resp := resps[order.GetID()]
		d.handleCancelOrderResp(*resp, nil)
	}
}

// handleCancelOrderResp processes the cancel order response.
// If the order was already confirmed via WebSocket, it is ignored.
func (d *Dealer) handleCancelOrderResp(resp CancelOrderResp, err error) {
	order := d.LivingOrders.getCancelingOrder(resp.OrderID)
	if order == nil {
		return
	}
	order.WithOpLock(func() {
		if err != nil {
			if errors.Is(err, OrderIsNotExists) {
				d.abandonOrder(order)
				return
			}
			d.rejectCancel(order)
			slog.Error(fmt.Sprintf("cancel order resp: %s", err.Error()))
			d.noticeError(err)
			return
		}
		if err := resp.Err; err != nil {
			if errors.Is(err, OrderIsNotExists) {
				d.abandonOrder(order)
				return
			}
			d.rejectCancel(order)
			slog.Error(fmt.Sprintf("cancel order resp: %s", err.Error()))
			d.noticeError(err)
			return
		}
		d.acceptCancel(order)
	})
}

// noticeError notifies errors via the onError callback.
// For critical errors it also stops accepting new orders.
func (d *Dealer) noticeError(err error) {
	if errors.Is(err, InfoError) {
		return
	}
	if errors.Is(err, WarnError) || errors.Is(err, HttpRequestError) {
		d.onError(err)
		return
	}

	// Critical errors: stop accepting orders before notifying.
	d.acceptOrder.Set(false)
	d.onError(err)
}

// HandleOrderData processes order update events received via WebSocket.
func (d *Dealer) HandleOrderData(datas []*OrderData) {
	for _, data := range datas {
		if data.Status == OrderDataStatusUnknown {
			slog.Info(fmt.Sprintf("unknown order status: %d", data.Status))
		}

		// Track unrelated orders (manual trades, other bots, etc.).
		if data.isUnrelated() {
			if data.Status == OrderDataStatusOpened {
				o := Order{publicID: data.PublicID}
				o.setConfirmTimestamps(data.ArrivedAt, data.ConfirmedAt)
				d.UnrelatedOrders.Set(data.PublicID, &o)
			}
			if data.isDone() {
				if order := d.UnrelatedOrders.getOrder(data.PublicID); order != nil {
					order.WithOpLock(func() {
						d.doneOrders.Append(order)
						d.UnrelatedOrders.Delete(data.PublicID)
					})
				}
			}
		}

		{ // Order fully filled
			order := d.LivingOrders.getOrder(data.ID)
			isFilled := data.Status == OrderDataStatusFilled && order != nil
			if isFilled {
				order.WithOpLock(func() {
					d.setOrderFillInfo(order, data)
					d.acceptFilled(order)
				})
				continue
			}
		}

		{ // Order partially filled
			order := d.LivingOrders.getOrder(data.ID)
			isPartiallyFilled := data.Status == OrderDataStatusPartiallyFilled && order != nil
			if isPartiallyFilled {
				order.WithOpLock(func() {
					d.setOrderPartiallyFillInfo(order, data)
					d.acceptPartiallyFilled(order)
				})
				continue
			}
		}
	}
}

func (d *Dealer) setOrderFillInfo(order *Order, data *OrderData) {
	order.setFilledAt(data.ArrivedAt)
	order.setAvgPrice(data.AvgExecPrice)
	order.setExecQty(data.CumExecQty)
	order.setFee(data.CumFee)
	if order.GetOrderType() == domains.OrderTypeMarket {
		order.setSentTimestamp(order.getLastOperatedTimestamp())
		order.setConfirmTimestamps(data.ArrivedAt, data.ConfirmedAt)
	}
}

func (d *Dealer) setOrderPartiallyFillInfo(order *Order, data *OrderData) {
	order.setFilledAt(data.ArrivedAt)
	order.setAvgPrice(data.AvgExecPrice)
	order.setExecQty(data.CumExecQty)
	order.setFee(data.CumFee)
}

func (d *Dealer) acceptFilled(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonFilled)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execFillCallbacks()
}

func (d *Dealer) acceptPartiallyFilled(order *Order) {
	if order.orderType == domains.OrderTypeLimitIOC {
		order.setStatus(OrderStatusDone)
		order.setOrderDoneReason(OrderDoneReasonPartiallyFilledAndCanceled)
		d.LivingOrders.Delete(order.GetID())
		d.doneOrders.Append(order)
		order.execPartiallyFillCallbacks()
	}
}

func (d *Dealer) acceptAmend(order *Order, detail *AmendDetail, arrivedAt, confirmedAt *time.Time) {
	order.setStatus(OrderStatusPending)
	order.setAmendingDetail(nil)
	order.amend(detail)
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	order.setConfirmTimestamps(arrivedAt, confirmedAt)
	order.execAmendCallbacks()
}

func (d *Dealer) acceptCreate(order *Order, publicID string, arrivedAt, confirmedAt *time.Time) {
	order.setStatus(OrderStatusPending)
	if publicID != "" {
		order.setPublicID(publicID)
	}
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	order.setConfirmTimestamps(arrivedAt, confirmedAt)
	order.execCreateCallbacks()
}

func (d *Dealer) acceptCancel(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonCanceled)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execCancelCallbacks()
}

func (d *Dealer) rejectAmends(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectAmend(order)
		})
	}
}

func (d *Dealer) rejectAmend(order *Order) {
	order.setStatus(OrderStatusPending)
	order.setAmendingDetail(nil)
	order.execAmendRejectCallbacks()
}

func (d *Dealer) rejectCreates(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectCreate(orders[i])
		})
	}
}

func (d *Dealer) rejectCreate(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonRejected)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execCreateRejectCallbacks()
}

func (d *Dealer) rejectCancels(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectCancel(order)
		})
	}
}

func (d *Dealer) rejectCancel(order *Order) {
	order.setStatus(OrderStatusPending)
	order.execCancelRejectCallbacks()
}

func (d *Dealer) abandonOrder(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonAbandoned)
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
}

package deals

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
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
type PositionDataHandler func(msg []*PositionData)

func (d *DealerImpl) handleSendOrdersResp(resps CreateOrdersRespMap, err error) {
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
func (d *DealerImpl) handleSendOrderResp(resp CreateOrderResp, err error) {
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

func (d *DealerImpl) handleAmendOrdersResp(resps AmendOrdersRespMap, err error) {
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

func (d *DealerImpl) handleAmendOrderResp(resp AmendOrderResp, err error) {
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

func (d *DealerImpl) handleCancelOrdersResp(resps CancelOrdersRespMap, err error) {
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
func (d *DealerImpl) handleCancelOrderResp(resp CancelOrderResp, err error) {
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
func (d *DealerImpl) noticeError(err error) {
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
func (d *DealerImpl) HandleOrderData(datas []*OrderData) {
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

		// Status transitions for order placement (Sending→Pending), cancellation (Canceling→Done),
		// and amendment (Amending→Pending) are driven by HTTP response callbacks
		// (handleSendOrderResp, handleCancelOrderResp, handleAmendOrderResp), not by this handler.
	}
}

func (d *DealerImpl) setOrderFillInfo(order *Order, data *OrderData) {
	order.setFilledAt(data.ArrivedAt)
	order.setAvgPrice(data.AvgExecPrice)
	order.setExecQty(data.CumExecQty)
	order.setFee(data.CumFee)
	if order.GetOrderType() == domains.OrderTypeMarket {
		order.setSentTimestamp(order.getLastOperatedTimestamp())
		order.setConfirmTimestamps(data.ArrivedAt, data.ConfirmedAt)
	}
}

func (d *DealerImpl) setOrderPartiallyFillInfo(order *Order, data *OrderData) {
	order.setFilledAt(data.ArrivedAt)
	order.setAvgPrice(data.AvgExecPrice)
	order.setExecQty(data.CumExecQty)
	order.setFee(data.CumFee)
}

func (d *DealerImpl) acceptFilled(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonFilled)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execFillCallbacks()
}

func (d *DealerImpl) acceptPartiallyFilled(order *Order) {
	if order.orderType == domains.OrderTypeLimitIOC {
		order.setStatus(OrderStatusDone)
		order.setOrderDoneReason(OrderDoneReasonPartiallyFilledAndCanceled)
		d.LivingOrders.Delete(order.GetID())
		d.doneOrders.Append(order)
		order.execPartiallyFillCallbacks()
	}
}

func (d *DealerImpl) acceptAmend(order *Order, detail *AmendDetail, arrivedAt, confirmedAt *time.Time) {
	order.setStatus(OrderStatusPending)
	order.setAmendingDetail(nil)
	order.amend(detail)
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	order.setConfirmTimestamps(arrivedAt, confirmedAt)
	order.execAmendCallbacks()
}

func (d *DealerImpl) acceptCreate(order *Order, publicID string, arrivedAt, confirmedAt *time.Time) {
	order.setStatus(OrderStatusPending)
	if publicID != "" {
		order.setPublicID(publicID)
	}
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	order.setConfirmTimestamps(arrivedAt, confirmedAt)
	order.execCreateCallbacks()
}

func (d *DealerImpl) acceptCancel(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonCanceled)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execCancelCallbacks()
}

func (d *DealerImpl) rejectAmends(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectAmend(order)
		})
	}
}

func (d *DealerImpl) rejectAmend(order *Order) {
	order.setStatus(OrderStatusPending)
	order.setAmendingDetail(nil)
	order.execAmendRejectCallbacks()
}

func (d *DealerImpl) rejectCreates(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectCreate(orders[i])
		})
	}
}

func (d *DealerImpl) rejectCreate(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonRejected)
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
	order.execCreateRejectCallbacks()
}

func (d *DealerImpl) rejectCancels(orders []*Order) {
	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.rejectCancel(order)
		})
	}
}

func (d *DealerImpl) rejectCancel(order *Order) {
	order.setStatus(OrderStatusPending)
	order.execCancelRejectCallbacks()
}

// HandlePositionData processes position updates received via WebSocket.
func (d *DealerImpl) HandlePositionData(datas []*PositionData) {
	data, err := d.extractPositionData(datas)
	if err != nil {
		slog.Error(err.Error())
		slog.Info(fmt.Sprintf("invalid position data: %v", datas))
		return
	}

	dataTs := data.Timestamp
	lastUpdateAt := d.CurrentPosition.getLastUpdateAt()
	if dataTs.After(lastUpdateAt) || dataTs.Equal(lastUpdateAt) {
		size := decimal.Zero
		if data.Side == domains.SideBuy {
			size = data.Qty
		}
		if data.Side == domains.SideSell {
			size = data.Qty.Neg()
		}
		d.CurrentPosition.update(data.Timestamp, size)

		for _, h := range d.posUpdatedHandlers {
			h(datas)
		}
	}
}

// SetPosUpdatedHandler registers a callback invoked when position updates.
func (d *DealerImpl) SetPosUpdatedHandler(handler PositionDataHandler) {
	d.posUpdatedHandlers = append(d.posUpdatedHandlers, handler)
}

func (d *DealerImpl) extractPositionData(datas []*PositionData) (*PositionData, error) {
	var oneWayDatas []*PositionData
	var invalidDatas []*PositionData
	for _, data := range datas {
		if data.PositionMode == PositionModeOneWay {
			oneWayDatas = append(oneWayDatas, data)
		} else {
			invalidDatas = append(invalidDatas, data)
		}
	}
	if len(invalidDatas) > 0 {
		return nil, errors.New("invalid position data: include invalid position data")
	}
	if len(oneWayDatas) >= 2 {
		return nil, errors.New("invalid position data: more than one one-way position data")
	}
	if len(oneWayDatas) == 0 {
		return nil, errors.New("invalid position data: no one-way position data")
	}
	return oneWayDatas[0], nil
}

func (d *DealerImpl) abandonOrder(order *Order) {
	order.setStatus(OrderStatusDone)
	order.setOrderDoneReason(OrderDoneReasonAbandoned)
	order.setSentTimestamp(order.getLastOperatedTimestamp())
	d.LivingOrders.Delete(order.GetID())
	d.doneOrders.Append(order)
}

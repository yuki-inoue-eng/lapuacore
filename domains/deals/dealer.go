package deals

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/mutex"
)

// dealerInstances is a per-symbol singleton registry.
var dealerInstances = map[*domains.Symbol]*DealerImpl{}

type DealerImpl struct {
	sync.RWMutex
	acceptOrder     *mutex.Flag // When false, no new orders are accepted (e.g., on critical error).
	onError         func(err error)
	agent           Agent
	Symbol          *domains.Symbol
	LivingOrders    *OrdersMutexMap // Active orders (including market orders awaiting fill notification).
	UnrelatedOrders *OrdersMutexMap // Unrelated orders (manual trades, other bots, etc.).

	// amendingDetailMap temporarily stores the pending amend detail for an order.
	amendingDetailMap *mutex.Map[string, AmendDetail]

	// doneOrders retains completed orders until exported.
	doneOrders *OrderMutexSlice

	CurrentPosition    *Position
	posUpdatedHandlers []PositionDataHandler
}

func (d *DealerImpl) GetSymbol() *domains.Symbol          { return d.Symbol }
func (d *DealerImpl) GetLivingOrders() *OrdersMutexMap    { return d.LivingOrders }
func (d *DealerImpl) GetUnrelatedOrders() *OrdersMutexMap { return d.UnrelatedOrders }
func (d *DealerImpl) GetCurrentPosition() *Position       { return d.CurrentPosition }

// NewDealer returns the DealerImpl for the given symbol, creating it if it does not exist.
func NewDealer(symbol *domains.Symbol, agent Agent, onError func(err error)) *DealerImpl {
	if ins, exists := dealerInstances[symbol]; exists {
		return ins
	}

	ins := &DealerImpl{
		acceptOrder:       mutex.NewFlag(true),
		onError:           onError,
		agent:             agent,
		Symbol:            symbol,
		LivingOrders:      NewOrdersMap(nil),
		UnrelatedOrders:   NewOrdersMap(nil),
		amendingDetailMap: mutex.NewMap[string, AmendDetail](nil),
		doneOrders:        NewOrderMuArray(nil),
		CurrentPosition:   NewPosition(),
	}
	dealerInstances[symbol] = ins
	return ins
}

func (d *DealerImpl) SendOrder(order *Order) error {
	if !d.acceptOrder.Get() {
		return errors.New("dealer does not accept order")
	}
	if o := d.LivingOrders.getOrder(order.GetID()); o != nil {
		return DealingErrorOrderIsAlreadyExists
	}

	var err error
	order.WithOpLock(func() {
		d.LivingOrders.Set(order.GetID(), order)
		order.setStatus(OrderStatusSending)
		order.recordLastOperatedTimestamp()
		err = d.agent.SendOrder(d.Symbol, order, d.handleSendOrderResp)
	})
	if err != nil {
		order.WithOpLock(func() { d.rejectCreate(order) })
		return err
	}
	return nil
}

func (d *DealerImpl) SendOrders(orders []*Order) error {
	if !d.acceptOrder.Get() {
		return errors.New("dealer does not accept order")
	}

	for i := range orders {
		if o := d.LivingOrders.getOrder(orders[i].GetID()); o != nil {
			return DealingErrorOrderIsAlreadyExists
		}
	}

	for i := range orders {
		order := orders[i]
		order.WithOpLock(func() {
			d.LivingOrders.Set(order.GetID(), order)
			order.setStatus(OrderStatusSending)
			order.recordLastOperatedTimestamp()
		})
	}

	if err := d.agent.SendOrders(d.Symbol, orders, d.handleSendOrdersResp); err != nil {
		d.rejectCreates(orders)
		return err
	}
	return nil
}

func SendOrder(dealer Dealer, order *Order) {
	if err := dealer.SendOrder(order); err != nil {
		slog.Error("failed to send order", err.Error(), dealer.GetSymbol().ID())
	}
}

func SendOrders(dealer Dealer, orders []*Order) {
	if err := dealer.SendOrders(orders); err != nil {
		slog.Error("failed to send orders", err.Error(), dealer.GetSymbol().ID())
	}
}

// AmendOrders is reserved for future use when a supported exchange becomes available.
func (d *DealerImpl) AmendOrders(_ map[*Order]AmendDetail) error {
	return nil
}

// AmendOrder amends the order with the given detail.
// If the order is currently being sent or amended, the amend is deferred to the operation's completion callback.
func (d *DealerImpl) AmendOrder(order *Order, detail AmendDetail) error {
	if !d.acceptOrder.Get() {
		return errors.New("dealer does not accept order")
	}

	cb := func(o *Order) {
		oid := o.GetID()
		amendDetail, ok := d.amendingDetailMap.Get(oid)
		if !ok {
			amendDetail = detail
		}
		d.amendingDetailMap.Delete(oid)
		if err := d.amendOrder(order, amendDetail); err != nil {
			slog.Error("failed to amend order", err.Error(), d.Symbol.ID())
		}
	}

	var err error
	var pending bool
	order.WithOpLock(func() {
		oid := order.GetID()
		switch order.GetStatus() {
		default:
			err = DealingErrorOrderNotReadyForOperation
		case OrderStatusDone:
			order.execAmendRejectOrderNotExistCallbacks()
		case OrderStatusPending:
			pending = true
		case OrderStatusSending:
			if _, ok := d.amendingDetailMap.Get(oid); !ok {
				order.SetCreateCallback(cb)
			}
			d.amendingDetailMap.Set(oid, detail)
		case OrderStatusAmending:
			if _, ok := d.amendingDetailMap.Get(oid); !ok {
				order.SetAmendCallback(cb)
			}
			d.amendingDetailMap.Set(oid, detail)
		}
	})
	if pending {
		return d.amendOrder(order, detail)
	}
	return err
}

func (d *DealerImpl) amendOrder(order *Order, detail AmendDetail) error {
	if o := d.LivingOrders.getOrder(order.GetID()); o == nil {
		order.execAmendRejectOrderNotExistCallbacks()
		return nil
	}
	var err error
	order.WithOpLock(func() {
		if !order.isAmendAble() {
			err = DealingErrorOrderIsNotAmendable
			return
		}
		if !order.isNeedToAmend(detail) {
			return
		}
		if order.GetStatus() != OrderStatusPending {
			err = Error(DealingErrorOrderNotReadyForOperation, fmt.Sprintf("(order status: %s)", order.GetStatus().String()))
			return
		}
		order.setStatus(OrderStatusAmending)
		order.setAmendingDetail(&detail)
		order.recordLastOperatedTimestamp()
		err = d.agent.AmendOrder(d.Symbol, order, detail, d.handleAmendOrderResp)
	})
	if err != nil {
		order.WithOpLock(func() { d.rejectAmend(order) })
		return err
	}
	return nil
}

func AmendOrder(dealer Dealer, order *Order, detail AmendDetail) {
	if order == nil {
		slog.Error("failed to amend order", "order is nil", dealer.GetSymbol().ID())
		return
	}
	if err := dealer.AmendOrder(order, detail); err != nil {
		slog.Error("failed to amend order:", err.Error(), dealer.GetSymbol().ID())
	}
}

// CancelOrder cancels an order. If the order is being sent or amended, the cancel is deferred to the callback.
// Replaces any previously deferred amend/cancel callbacks so that only one cancel executes.
func (d *DealerImpl) CancelOrder(order *Order) error {
	cb := func(o *Order) {
		if err := d.cancelOrder(o); err != nil {
			slog.Error("failed to cancel order", err.Error(), d.Symbol.ID())
		}
	}
	order.WithOpLock(func() {
		switch order.GetStatus() {
		default:
		case OrderStatusPending:
			cb(order)
		case OrderStatusSending:
			d.amendingDetailMap.Delete(order.GetID())
			order.ReplaceCreateCallbacks(cb)
		case OrderStatusAmending:
			d.amendingDetailMap.Delete(order.GetID())
			order.ReplaceAmendCallbacks(cb)
		}
	})
	return nil
}

func (d *DealerImpl) cancelOrder(order *Order) error {
	if order.GetStatus() == OrderStatusCanceling {
		return nil
	}
	if order.GetStatus() != OrderStatusPending {
		return DealingErrorOrderNotReadyForOperation
	}
	order.setStatus(OrderStatusCanceling)
	order.recordLastOperatedTimestamp()
	if err := d.agent.CancelOrder(d.Symbol, order, d.handleCancelOrderResp); err != nil {
		d.rejectCancel(order)
		return err
	}
	return nil
}

// CancelOrders cancels multiple orders. Defers cancellation for orders that are being sent or amended.
func (d *DealerImpl) CancelOrders(orders Orders) error {
	orders = orders.Unique()

	cb := func(o *Order) {
		if err := d.cancelOrder(o); err != nil {
			slog.Error("failed to cancel order", err.Error(), d.Symbol.ID())
		}
	}

	var err error
	WithOpeLocks(orders, func() {
		var cancelTargets []*Order
		for i := range orders {
			order := orders[i]
			switch order.GetStatus() {
			case OrderStatusPending:
				cancelTargets = append(cancelTargets, order)
			case OrderStatusSending:
				d.amendingDetailMap.Delete(order.GetID())
				order.ReplaceCreateCallbacks(cb)
			case OrderStatusAmending:
				d.amendingDetailMap.Delete(order.GetID())
				order.ReplaceAmendCallbacks(cb)
			}
		}
		if len(cancelTargets) > 0 {
			for i := range cancelTargets {
				order := cancelTargets[i]
				order.setStatus(OrderStatusCanceling)
				order.recordLastOperatedTimestamp()
			}
			err = d.agent.CancelOrders(d.Symbol, cancelTargets, d.handleCancelOrdersResp)
		}
	})

	if err != nil {
		d.rejectCancels(orders)
	}
	return nil
}

func CancelOrder(dealer Dealer, order *Order) {
	if order == nil {
		return
	}
	if err := dealer.CancelOrder(order); err != nil {
		slog.Error("failed to cancel order", err.Error(), dealer.GetSymbol().ID())
	}
}

func CancelOrders(dealer Dealer, orders []*Order) {
	if len(orders) == 0 {
		return
	}
	if err := dealer.CancelOrders(orders); err != nil {
		slog.Error("failed to cancel orders", err.Error(), dealer.GetSymbol().ID())
	}
}

// ExportDoneOrders exports completed orders and resets the internal slice.
func (d *DealerImpl) ExportDoneOrders() *OrderMutexSlice {
	d.Lock()
	defer d.Unlock()
	doneOrders := d.doneOrders
	d.doneOrders = NewOrderMuArray(nil)
	return doneOrders
}

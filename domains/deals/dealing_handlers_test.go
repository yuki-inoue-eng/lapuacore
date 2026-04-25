package deals

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func handlersNow() *time.Time {
	t := time.Now()
	return &t
}

func TestHandleSendOrderResp(t *testing.T) {
	tests := []struct {
		name           string
		orderType      domains.OrderType
		requestErr     error
		orderErr       error
		wantStatus     OrderStatus
		wantDoneReason *OrderDoneReason
		wantPublicID   string
		wantInLiving   bool
	}{
		{
			name:           "request error rejects order",
			orderType:      domains.OrderTypeLimit,
			requestErr:     errors.New("timeout"),
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonRejected),
			wantInLiving:   false,
		},
		{
			name:           "per-order error rejects order",
			orderType:      domains.OrderTypeLimit,
			orderErr:       errors.New("insufficient balance"),
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonRejected),
			wantInLiving:   false,
		},
		{
			name:         "Limit order accepted with PublicID",
			orderType:    domains.OrderTypeLimit,
			wantStatus:   OrderStatusPending,
			wantPublicID: "pub-001",
			wantInLiving: true,
		},
		{
			name:         "LimitMaker order accepted",
			orderType:    domains.OrderTypeLimitMaker,
			wantStatus:   OrderStatusPending,
			wantPublicID: "pub-002",
			wantInLiving: true,
		},
		{
			name:         "Market order leaves status unchanged (waits for WebSocket)",
			orderType:    domains.OrderTypeMarket,
			wantStatus:   OrderStatusSending,
			wantInLiving: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDealer(&mockAgent{})

			var order *Order
			switch tt.orderType {
			case domains.OrderTypeMarket:
				order = NewMarketOrder(decimal.RequireFromString("0.01"), domains.SideBuy, "")
			case domains.OrderTypeLimitMaker:
				order = NewLimitMakerOrder(
					decimal.RequireFromString("100"), decimal.RequireFromString("0.01"),
					domains.SideBuy, false, "",
				)
			default:
				order = dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
			}
			dealerAddLivingOrder(d, order, OrderStatusSending)

			publicID := tt.wantPublicID
			resp := CreateOrderResp{OrderID: order.GetID(), PublicID: publicID, Err: tt.orderErr, ArrivedAt: handlersNow(), ConfirmedAt: handlersNow()}
			d.handleSendOrderResp(resp, tt.requestErr)

			if got, want := order.GetStatus(), tt.wantStatus; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if tt.wantDoneReason != nil {
				if got, want := *order.GetOrderDoneReason(), *tt.wantDoneReason; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
			if tt.wantPublicID != "" {
				if got, want := order.GetPublicID(), tt.wantPublicID; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
			if got, want := d.LivingOrders.getOrder(order.GetID()) != nil, tt.wantInLiving; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}

	t.Run("order not in LivingOrders is silently ignored", func(t *testing.T) {
		d := newTestDealer(&mockAgent{})
		d.handleSendOrderResp(CreateOrderResp{OrderID: "nonexistent"}, nil)
	})
}

func TestHandleCancelOrderResp(t *testing.T) {
	tests := []struct {
		name           string
		requestErr     error
		orderErr       error
		wantStatus     OrderStatus
		wantDoneReason *OrderDoneReason
		wantInLiving   bool
		wantDoneCount  int
	}{
		{
			name:           "OrderIsNotExists error abandons order",
			orderErr:       OrderIsNotExists,
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonAbandoned),
			wantInLiving:   false,
			wantDoneCount:  1,
		},
		{
			name:          "request error rejects cancel (order returns to Pending)",
			requestErr:    errors.New("timeout"),
			wantStatus:    OrderStatusPending,
			wantInLiving:  true,
			wantDoneCount: 0,
		},
		{
			name:          "per-order error rejects cancel (order returns to Pending)",
			orderErr:      errors.New("unknown error"),
			wantStatus:    OrderStatusPending,
			wantInLiving:  true,
			wantDoneCount: 0,
		},
		{
			name:           "success accepts cancel",
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonCanceled),
			wantInLiving:   false,
			wantDoneCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDealer(&mockAgent{})
			order := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
			dealerAddLivingOrder(d, order, OrderStatusCanceling)

			d.handleCancelOrderResp(CancelOrderResp{OrderID: order.GetID(), Err: tt.orderErr}, tt.requestErr)

			if got, want := order.GetStatus(), tt.wantStatus; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if tt.wantDoneReason != nil {
				if got, want := *order.GetOrderDoneReason(), *tt.wantDoneReason; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
			if got, want := d.LivingOrders.getOrder(order.GetID()) != nil, tt.wantInLiving; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := d.doneOrders.Len(), tt.wantDoneCount; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestHandleAmendOrderResp(t *testing.T) {
	tests := []struct {
		name           string
		requestErr     error
		orderErr       error
		newDetail      *AmendDetail
		wantStatus     OrderStatus
		wantDoneReason *OrderDoneReason
		wantPrice      string
	}{
		{
			name:           "OrderIsNotExists error abandons order",
			orderErr:       OrderIsNotExists,
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonAbandoned),
		},
		{
			name:       "request error rejects amend (order returns to Pending)",
			requestErr: errors.New("timeout"),
			wantStatus: OrderStatusPending,
		},
		{
			name:       "per-order error rejects amend (order returns to Pending)",
			orderErr:   errors.New("unknown error"),
			wantStatus: OrderStatusPending,
		},
		{
			name:       "success accepts amend and updates price/qty",
			newDetail:  &AmendDetail{Price: decimal.RequireFromString("105"), Qty: decimal.RequireFromString("0.02")},
			wantStatus: OrderStatusPending,
			wantPrice:  "105",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDealer(&mockAgent{})
			order := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
			dealerAddLivingOrder(d, order, OrderStatusAmending)

			resp := AmendOrderResp{
				OrderID:     order.GetID(),
				Detail:      tt.newDetail,
				Err:         tt.orderErr,
				ArrivedAt:   handlersNow(),
				ConfirmedAt: handlersNow(),
			}
			d.handleAmendOrderResp(resp, tt.requestErr)

			if got, want := order.GetStatus(), tt.wantStatus; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if tt.wantDoneReason != nil {
				if got, want := *order.GetOrderDoneReason(), *tt.wantDoneReason; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
			if tt.wantPrice != "" {
				if !order.GetPrice().Equal(decimal.RequireFromString(tt.wantPrice)) {
					t.Errorf("got %v, want true", false)
				}
			}
		})
	}
}

func TestHandleOrderData(t *testing.T) {
	tests := []struct {
		name  string
		setup func(d *DealerImpl) []*Order
		data  func(orders []*Order) []*OrderData
		check func(t *testing.T, d *DealerImpl, orders []*Order)
	}{
		{
			name: "Filled: moves order to doneOrders with DoneReasonFilled",
			setup: func(d *DealerImpl) []*Order {
				o := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
				dealerAddLivingOrder(d, o, OrderStatusPending)
				return []*Order{o}
			},
			data: func(orders []*Order) []*OrderData {
				return []*OrderData{{ID: orders[0].GetID(), Status: OrderDataStatusFilled, ArrivedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, orders []*Order) {
				if got, want := orders[0].GetStatus(), OrderStatusDone; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := *orders[0].GetOrderDoneReason(), OrderDoneReasonFilled; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if d.LivingOrders.getOrder(orders[0].GetID()) != nil {
					t.Errorf("got non-nil, want nil")
				}
				if got, want := d.doneOrders.Len(), 1; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "Filled: executes fill callback",
			setup: func(d *DealerImpl) []*Order {
				o := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
				dealerAddLivingOrder(d, o, OrderStatusPending)
				return []*Order{o}
			},
			data: func(orders []*Order) []*OrderData {
				return []*OrderData{{ID: orders[0].GetID(), Status: OrderDataStatusFilled, ArrivedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, orders []*Order) {
				called := make(chan struct{}, 1)
				orders[0].fillCallbacks = []OrderCallback{func(_ *Order) { called <- struct{}{} }}
				d.HandleOrderData([]*OrderData{{ID: orders[0].GetID(), Status: OrderDataStatusFilled, ArrivedAt: handlersNow()}})
				select {
				case <-called:
				case <-time.After(time.Second):
					t.Error("fill callback was not called")
				}
			},
		},
		{
			name: "PartiallyFilled LimitIOC: moves order to doneOrders",
			setup: func(d *DealerImpl) []*Order {
				o := NewLimitIOCOrder(
					decimal.RequireFromString("100"), decimal.Zero,
					decimal.RequireFromString("0.01"), domains.SideBuy, "",
				)
				dealerAddLivingOrder(d, o, OrderStatusPending)
				return []*Order{o}
			},
			data: func(orders []*Order) []*OrderData {
				return []*OrderData{{ID: orders[0].GetID(), Status: OrderDataStatusPartiallyFilled, CumExecQty: decimal.RequireFromString("0.005"), ArrivedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, orders []*Order) {
				if got, want := orders[0].GetStatus(), OrderStatusDone; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := *orders[0].GetOrderDoneReason(), OrderDoneReasonPartiallyFilledAndCanceled; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := d.doneOrders.Len(), 1; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "PartiallyFilled Limit: does not change status",
			setup: func(d *DealerImpl) []*Order {
				o := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
				dealerAddLivingOrder(d, o, OrderStatusPending)
				return []*Order{o}
			},
			data: func(orders []*Order) []*OrderData {
				return []*OrderData{{ID: orders[0].GetID(), Status: OrderDataStatusPartiallyFilled, CumExecQty: decimal.RequireFromString("0.005"), ArrivedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, orders []*Order) {
				if got, want := orders[0].GetStatus(), OrderStatusPending; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := d.doneOrders.Len(), 0; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			// Regression test for the return->continue bug fix.
			name: "two Filled events in one batch: both are processed",
			setup: func(d *DealerImpl) []*Order {
				o1 := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
				o2 := dealerMakeLimitOrder("101", "0.01", domains.SideSell)
				dealerAddLivingOrder(d, o1, OrderStatusPending)
				dealerAddLivingOrder(d, o2, OrderStatusPending)
				return []*Order{o1, o2}
			},
			data: func(orders []*Order) []*OrderData {
				return []*OrderData{
					{ID: orders[0].GetID(), Status: OrderDataStatusFilled, ArrivedAt: handlersNow()},
					{ID: orders[1].GetID(), Status: OrderDataStatusFilled, ArrivedAt: handlersNow()},
				}
			},
			check: func(t *testing.T, d *DealerImpl, orders []*Order) {
				if got, want := orders[0].GetStatus(), OrderStatusDone; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := orders[1].GetStatus(), OrderStatusDone; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := d.doneOrders.Len(), 2; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "Unrelated Opened: added to UnrelatedOrders",
			setup: func(d *DealerImpl) []*Order {
				return nil
			},
			data: func(_ []*Order) []*OrderData {
				return []*OrderData{{ID: "", PublicID: "ext-001", Status: OrderDataStatusOpened, ArrivedAt: handlersNow(), ConfirmedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, _ []*Order) {
				if d.UnrelatedOrders.getOrder("ext-001") == nil {
					t.Errorf("got nil, want non-nil")
				}
			},
		},
		{
			name: "Unrelated Done: moved from UnrelatedOrders to doneOrders",
			setup: func(d *DealerImpl) []*Order {
				existing := &Order{publicID: "ext-001"}
				d.UnrelatedOrders.Set("ext-001", existing)
				return nil
			},
			data: func(_ []*Order) []*OrderData {
				return []*OrderData{{ID: "", PublicID: "ext-001", Status: OrderDataStatusFilled, ArrivedAt: handlersNow()}}
			},
			check: func(t *testing.T, d *DealerImpl, _ []*Order) {
				if d.UnrelatedOrders.getOrder("ext-001") != nil {
					t.Errorf("got non-nil, want nil")
				}
				if got, want := d.doneOrders.Len(), 1; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDealer(&mockAgent{})
			orders := tt.setup(d)

			// Skip HandleOrderData for the callback test which calls it internally.
			if tt.name != "Filled: executes fill callback" {
				d.HandleOrderData(tt.data(orders))
			}

			tt.check(t, d, orders)
		})
	}
}

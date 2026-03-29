package deals

import (
	"errors"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func dealerMakeLimitOrder(price, qty string, side domains.Side) *Order {
	return NewLimitOrder(
		decimal.RequireFromString(price),
		decimal.RequireFromString(qty),
		side, false, "",
	)
}

func dealerAddLivingOrder(d *Dealer, order *Order, status OrderStatus) {
	order.setStatus(status)
	d.LivingOrders.Set(order.GetID(), order)
}

func TestSendOrder(t *testing.T) {
	tests := []struct {
		name          string
		acceptOrder   bool
		duplicate     bool
		agentErr      error
		wantErr       bool
		wantStatus    OrderStatus
		wantDoneReason *OrderDoneReason
		wantInLiving  bool
	}{
		{
			name:         "rejects when acceptOrder is false",
			acceptOrder:  false,
			wantErr:      true,
			wantInLiving: false,
		},
		{
			name:        "rejects when order already exists",
			acceptOrder: true,
			duplicate:   true,
			wantErr:     true,
		},
		{
			name:           "rejects order when agent returns error",
			acceptOrder:    true,
			agentErr:       errors.New("network error"),
			wantErr:        true,
			wantStatus:     OrderStatusDone,
			wantDoneReason: ptr(OrderDoneReasonRejected),
			wantInLiving:   false,
		},
		{
			name:         "adds order to LivingOrders with StatusSending on success",
			acceptOrder:  true,
			wantErr:      false,
			wantStatus:   OrderStatusSending,
			wantInLiving: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &mockAgent{
				sendOrderFunc: func(_ *domains.Symbol, _ *Order, _ CreateOrderRespHandler) error {
					return tt.agentErr
				},
			}
			d := newTestDealer(agent)
			d.acceptOrder.Set(tt.acceptOrder)

			order := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)

			if tt.duplicate {
				d.LivingOrders.Set(order.GetID(), order)
			}

			err := d.SendOrder(order)

			assert.Equal(t, tt.wantErr, err != nil)
			if tt.wantStatus != 0 {
				assert.Equal(t, tt.wantStatus, order.GetStatus())
			}
			if tt.wantDoneReason != nil {
				assert.Equal(t, *tt.wantDoneReason, *order.GetOrderDoneReason())
			}
			if !tt.duplicate {
				assert.Equal(t, tt.wantInLiving, d.LivingOrders.getOrder(order.GetID()) != nil)
			}
		})
	}
}

func TestAmendOrder(t *testing.T) {
	newDetail := func(price string) AmendDetail {
		return AmendDetail{Price: decimal.RequireFromString(price), Qty: decimal.RequireFromString("0.01")}
	}

	tests := []struct {
		name              string
		acceptOrder       bool
		orderStatus       OrderStatus
		secondCall        *AmendDetail // non-nil triggers a second AmendOrder call
		wantErr           bool
		wantAgentCalled   bool
		wantOrderStatus   OrderStatus
		wantCallbackCount int    // expected len(createCallbacks)
		wantDetailPrice   string // expected price in amendingDetailMap after the call
		wantRejectCB      bool   // expect amendRejectOrderNotExistCallback to fire
	}{
		{
			name:        "rejects when acceptOrder is false",
			acceptOrder: false,
			orderStatus: OrderStatusPending,
			wantErr:     true,
		},
		{
			name:            "Pending: amends immediately via agent",
			acceptOrder:     true,
			orderStatus:     OrderStatusPending,
			wantAgentCalled: true,
			wantOrderStatus: OrderStatusAmending,
		},
		{
			name:              "Sending: defers via createCallback with detail stored",
			acceptOrder:       true,
			orderStatus:       OrderStatusSending,
			wantCallbackCount: 1,
			wantDetailPrice:   "101",
		},
		{
			name:              "Sending: second call overwrites detail but does not add another callback",
			acceptOrder:       true,
			orderStatus:       OrderStatusSending,
			secondCall:        ptr(newDetail("102")),
			wantCallbackCount: 1,
			wantDetailPrice:   "102",
		},
		{
			name:         "Done: executes amendRejectOrderNotExistCallback",
			acceptOrder:  true,
			orderStatus:  OrderStatusDone,
			wantRejectCB: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentCalled := false
			agent := &mockAgent{
				amendOrderFunc: func(_ *domains.Symbol, _ *Order, _ AmendDetail, _ AmendOrderRespHandler) error {
					agentCalled = true
					return nil
				},
			}
			d := newTestDealer(agent)
			d.acceptOrder.Set(tt.acceptOrder)

			order := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
			order.setStatus(tt.orderStatus)
			if tt.orderStatus == OrderStatusPending || tt.orderStatus == OrderStatusSending {
				d.LivingOrders.Set(order.GetID(), order)
			}

			rejectCBCalled := make(chan struct{}, 1)
			if tt.wantRejectCB {
				order.SetAmendRejectOrderNotExistCallback(func(_ *Order) { rejectCBCalled <- struct{}{} })
			}

			detail := newDetail("101")
			err := d.AmendOrder(order, detail)

			if tt.secondCall != nil {
				_ = d.AmendOrder(order, *tt.secondCall)
			}

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantAgentCalled, agentCalled)

			if tt.wantOrderStatus != 0 {
				assert.Equal(t, tt.wantOrderStatus, order.GetStatus())
			}
			if tt.wantCallbackCount > 0 {
				assert.Equal(t, tt.wantCallbackCount, len(order.createCallbacks))
			}
			if tt.wantDetailPrice != "" {
				stored, ok := d.amendingDetailMap.Get(order.GetID())
				assert.Equal(t, true, ok)
				assert.Equal(t, true, stored.Price.Equal(decimal.RequireFromString(tt.wantDetailPrice)))
			}
			if tt.wantRejectCB {
				select {
				case <-rejectCBCalled:
				case <-time.After(time.Second):
					t.Error("amendRejectOrderNotExistCallback was not called")
				}
			}
		})
	}
}

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name              string
		orderStatus       OrderStatus
		wantAgentCalled   bool
		wantOrderStatus   OrderStatus
		wantCreateCBCount int
		wantAmendCBCount  int
	}{
		{
			name:            "Pending: cancels immediately via agent",
			orderStatus:     OrderStatusPending,
			wantAgentCalled: true,
			wantOrderStatus: OrderStatusCanceling,
		},
		{
			name:              "Sending: registers createCallback for deferred cancel",
			orderStatus:       OrderStatusSending,
			wantAgentCalled:   false,
			wantCreateCBCount: 1,
		},
		{
			name:             "Amending: registers amendCallback for deferred cancel",
			orderStatus:      OrderStatusAmending,
			wantAgentCalled:  false,
			wantAmendCBCount: 1,
		},
		{
			name:            "Done: does nothing",
			orderStatus:     OrderStatusDone,
			wantAgentCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentCalled := false
			agent := &mockAgent{
				cancelOrderFunc: func(_ *domains.Symbol, _ *Order, _ CancelOrderRespHandler) error {
					agentCalled = true
					return nil
				},
			}
			d := newTestDealer(agent)
			order := dealerMakeLimitOrder("100", "0.01", domains.SideBuy)
			order.setStatus(tt.orderStatus)
			if tt.orderStatus == OrderStatusPending {
				d.LivingOrders.Set(order.GetID(), order)
			}

			err := d.CancelOrder(order)

			assert.Equal(t, nil, err)
			assert.Equal(t, tt.wantAgentCalled, agentCalled)
			if tt.wantOrderStatus != 0 {
				assert.Equal(t, tt.wantOrderStatus, order.GetStatus())
			}
			if tt.wantCreateCBCount > 0 {
				assert.Equal(t, tt.wantCreateCBCount, len(order.createCallbacks))
			}
			if tt.wantAmendCBCount > 0 {
				assert.Equal(t, tt.wantAmendCBCount, len(order.amendCallbacks))
			}
		})
	}
}

// ptr is a generic helper to take a pointer to a value.
func ptr[T any](v T) *T { return &v }

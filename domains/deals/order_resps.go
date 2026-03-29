package deals

import "time"

// CreateOrdersRespMap is a map of order creation responses keyed by order ID (used for batch operations).
type CreateOrdersRespMap map[string]*CreateOrderResp

func (m *CreateOrdersRespMap) IDs() []string {
	var ids []string
	for k := range *m {
		ids = append(ids, k)
	}
	return ids
}

// CancelOrdersRespMap is a map of order cancellation responses keyed by order ID (used for batch operations).
type CancelOrdersRespMap map[string]*CancelOrderResp

func (m *CancelOrdersRespMap) IDs() []string {
	var ids []string
	for k := range *m {
		ids = append(ids, k)
	}
	return ids
}

// AmendOrdersRespMap is a map of order amendment responses keyed by order ID (used for batch operations).
type AmendOrdersRespMap map[string]*AmendOrderResp

func (m *AmendOrdersRespMap) IDs() []string {
	var ids []string
	for k := range *m {
		ids = append(ids, k)
	}
	return ids
}

// CreateOrderResp is the response for a single order placement via REST API.
type CreateOrderResp struct {
	OrderID     string
	PublicID    string     // Exchange-assigned ID (populated only when needed)
	ArrivedAt   *time.Time // Time the order arrived at the exchange
	ConfirmedAt *time.Time // Time the order was confirmed as accepted

	Err error // Per-order error
}

// AmendOrderResp is the response for a single order amendment via REST API.
type AmendOrderResp struct {
	OrderID     string
	Detail      *AmendDetail
	ArrivedAt   *time.Time
	ConfirmedAt *time.Time

	Err error
}

// CancelOrderResp is the response for a single order cancellation via REST API.
type CancelOrderResp struct {
	OrderID string

	Err error
}
package deals

import (
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/mutex"
)

type OrderCallback func(o *Order)

type OrderDoneReason int

const (
	OrderDoneReasonUnknown OrderDoneReason = iota
	OrderDoneReasonFilled
	OrderDoneReasonPartiallyFilledAndCanceled
	OrderDoneReasonCanceled
	OrderDoneReasonAmended
	OrderDoneReasonRejected
	OrderDoneReasonAbandoned
)

func (r OrderDoneReason) String() string {
	switch r {
	case OrderDoneReasonFilled:
		return "Filled"
	case OrderDoneReasonCanceled:
		return "Canceled"
	case OrderDoneReasonAmended:
		return "Amended"
	default:
		return "Unknown"
	}
}

type OrderStatus int

const (
	OrderStatusUnknown OrderStatus = iota
	OrderStatusBorn                // Created but not yet sent
	OrderStatusSending
	OrderStatusPending
	OrderStatusCanceling
	OrderStatusAmending
	OrderStatusDone
)

func (s OrderStatus) String() string {
	switch s {
	case OrderStatusBorn:
		return "Born"
	case OrderStatusSending:
		return "Sending"
	case OrderStatusPending:
		return "Pending"
	case OrderStatusCanceling:
		return "Canceling"
	case OrderStatusAmending:
		return "Amending"
	case OrderStatusDone:
		return "Done"
	default:
		return "Unknown"
	}
}

type AmendDetail struct {
	Price decimal.Decimal
	Qty   decimal.Decimal
}

type Orders []*Order

func (os Orders) Unique() Orders {
	unique := make(map[string]*Order)
	for _, o := range os {
		unique[o.GetID()] = o
	}
	var result Orders
	for _, o := range unique {
		result = append(result, o)
	}
	return result
}

type Order struct {
	mu             sync.RWMutex
	muOpe          sync.RWMutex
	id             string
	publicID       string
	hide           bool
	memo           string
	orderType      domains.OrderType
	price          decimal.Decimal
	qty            decimal.Decimal
	side           domains.Side
	sentAt         *time.Time
	arrivedAt      *time.Time
	confirmedAt    *time.Time
	lastOperatedAt *time.Time
	status         OrderStatus

	filledAt *time.Time
	avgPrice decimal.Decimal
	execQty  decimal.Decimal
	fee      decimal.Decimal

	orderDoneReason *OrderDoneReason

	createCallbacks        []OrderCallback
	amendCallbacks         []OrderCallback
	fillCallbacks          []OrderCallback
	partiallyFillCallBacks []OrderCallback
	cancelCallbacks        []OrderCallback

	createRejectCallbacks             []OrderCallback
	amendRejectCallbacks              []OrderCallback
	amendRejectOrderNotExistCallbacks []OrderCallback
	cancelRejectCallbacks             []OrderCallback

	amendingDetail *AmendDetail
	amendCount     int
}

func (o *Order) WithOpLock(f func()) {
	o.muOpe.Lock()
	defer o.muOpe.Unlock()
	f()
}

func WithOpeLocks(orders Orders, f func()) {
	orders = orders.Unique()
	for i := range orders {
		orders[i].muOpe.Lock()
	}
	f()
	for i := range orders {
		orders[i].muOpe.Unlock()
	}
}

func (o *Order) GetID() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.id
}

func (o *Order) setPublicID(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.publicID = id
}

func (o *Order) GetPublicID() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.publicID
}

func (o *Order) IsHide() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.hide
}

func (o *Order) GetMemo() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.memo
}

func (o *Order) SetMemo(memo string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.memo = memo
}

func (o *Order) GetOrderType() domains.OrderType {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.orderType
}

func (o *Order) GetPrice() decimal.Decimal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.price
}

func (o *Order) GetQty() decimal.Decimal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.qty
}

func (o *Order) GetSize() decimal.Decimal {
	return ToSize(o.side, o.GetQty())
}

func (o *Order) GetSide() domains.Side {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.side
}

func (o *Order) GetSentAt() *time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.sentAt
}

func (o *Order) GetArrivedAt() *time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.arrivedAt
}

func (o *Order) GetConfirmedAt() *time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.confirmedAt
}

func (o *Order) GetFilledAt() *time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.filledAt
}

func (o *Order) setFilledAt(filledAt *time.Time) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.filledAt = filledAt
}

func (o *Order) GetAvgPrice() decimal.Decimal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.avgPrice
}

func (o *Order) setAvgPrice(avgPrice decimal.Decimal) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.avgPrice = avgPrice
}

func (o *Order) GetExecQty() decimal.Decimal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.execQty
}

func (o *Order) setExecQty(execQty decimal.Decimal) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.execQty = execQty
}

func (o *Order) GetFee() decimal.Decimal {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.fee
}

func (o *Order) setFee(fee decimal.Decimal) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fee = fee
}

func (o *Order) GetOrderDoneReason() *OrderDoneReason {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.orderDoneReason
}

func (o *Order) IsOneOfStatus(statuses ...OrderStatus) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, s := range statuses {
		if o.status == s {
			return true
		}
	}
	return false
}

func (o *Order) isConfirmed() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.confirmedAt != nil
}

func (o *Order) recordLastOperatedTimestamp() {
	o.mu.Lock()
	defer o.mu.Unlock()
	now := time.Now()
	o.lastOperatedAt = &now
}

func (o *Order) setSentTimestamp(orderingAt *time.Time) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if orderingAt != nil {
		o.sentAt = orderingAt
	}
}

func (o *Order) setConfirmTimestamps(arrivedAt, confirmedAt *time.Time) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if arrivedAt != nil {
		o.arrivedAt = arrivedAt
	}
	if confirmedAt != nil {
		o.confirmedAt = confirmedAt
	}
}

func (o *Order) amend(detail *AmendDetail) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.price = detail.Price
	o.qty = detail.Qty
}

func (o *Order) setOrderDoneReason(reason OrderDoneReason) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.orderDoneReason = &reason
}

func (o *Order) GetStatus() OrderStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

func (o *Order) setStatus(status OrderStatus) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status = status
}

func (o *Order) getLastOperatedTimestamp() *time.Time {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.lastOperatedAt
}

func (o *Order) setAmendingDetail(detail *AmendDetail) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.amendingDetail = detail
}

func (o *Order) GetAmendingDetail() *AmendDetail {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.amendingDetail
}

func (o *Order) incAmendCount() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.amendCount++
}

// SetCreateCallback must be called inside WithOpLock (use SetFillCallback instead for taker orders).
func (o *Order) SetCreateCallback(callback OrderCallback) {
	if o.GetOrderType().IsTaker() {
		return
	}
	o.createCallbacks = append(o.createCallbacks, callback)
}

// SetCreateRejectCallback must be called inside WithOpLock.
func (o *Order) SetCreateRejectCallback(callback OrderCallback) {
	o.createRejectCallbacks = append(o.createRejectCallbacks, callback)
}

// SetCancelCallback must be called inside WithOpLock.
func (o *Order) SetCancelCallback(callback OrderCallback) {
	o.cancelCallbacks = append(o.cancelCallbacks, callback)
}

// SetCancelRejectCallback must be called inside WithOpLock.
func (o *Order) SetCancelRejectCallback(callback OrderCallback) {
	o.cancelRejectCallbacks = append(o.cancelRejectCallbacks, callback)
}

// SetAmendCallback must be called inside WithOpLock.
func (o *Order) SetAmendCallback(callback OrderCallback) {
	if !o.GetOrderType().IsAmenable() {
		return
	}
	o.amendCallbacks = append(o.amendCallbacks, callback)
}

// SetAmendRejectCallback must be called inside WithOpLock.
func (o *Order) SetAmendRejectCallback(callback OrderCallback) {
	o.amendRejectCallbacks = append(o.amendRejectCallbacks, callback)
}

// SetAmendRejectOrderNotExistCallback must be called inside WithOpLock.
func (o *Order) SetAmendRejectOrderNotExistCallback(callback OrderCallback) {
	o.amendRejectOrderNotExistCallbacks = append(o.amendRejectOrderNotExistCallbacks, callback)
}

// SetFillCallback must be called inside WithOpLock.
func (o *Order) SetFillCallback(callback OrderCallback) {
	o.fillCallbacks = append(o.fillCallbacks, callback)
}

// SetPartiallyFillCallback must be called inside WithOpLock.
func (o *Order) SetPartiallyFillCallback(callback OrderCallback) {
	o.partiallyFillCallBacks = append(o.partiallyFillCallBacks, callback)
}

// ReplaceCreateCallbacks replaces all pending create callbacks with a single callback.
func (o *Order) ReplaceCreateCallbacks(callback OrderCallback) {
	o.createCallbacks = []OrderCallback{callback}
}

// ReplaceAmendCallbacks replaces all pending amend callbacks with a single callback.
func (o *Order) ReplaceAmendCallbacks(callback OrderCallback) {
	o.amendCallbacks = []OrderCallback{callback}
}

func (o *Order) execCreateCallbacks() {
	for _, cb := range o.createCallbacks {
		go cb(o)
	}
}

func (o *Order) execAmendCallbacks() {
	for _, cb := range o.amendCallbacks {
		go cb(o)
	}
}

func (o *Order) execFillCallbacks() {
	for _, cb := range o.fillCallbacks {
		go cb(o)
	}
}

func (o *Order) execPartiallyFillCallbacks() {
	for _, cb := range o.partiallyFillCallBacks {
		go cb(o)
	}
}

func (o *Order) execCancelCallbacks() {
	for _, cb := range o.cancelCallbacks {
		go cb(o)
	}
}

func (o *Order) execCreateRejectCallbacks() {
	for _, cb := range o.createRejectCallbacks {
		go cb(o)
	}
}

func (o *Order) execAmendRejectCallbacks() {
	for _, cb := range o.amendRejectCallbacks {
		go cb(o)
	}
}

func (o *Order) execAmendRejectOrderNotExistCallbacks() {
	for _, cb := range o.amendRejectOrderNotExistCallbacks {
		go cb(o)
	}
}

func (o *Order) execCancelRejectCallbacks() {
	for _, cb := range o.cancelRejectCallbacks {
		go cb(o)
	}
}

func (o *Order) isNeedToAmend(detail AmendDetail) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	switch o.status {
	default:
		return false
	case OrderStatusPending:
		return !o.price.Equal(detail.Price) || !o.qty.Equal(detail.Qty)
	case OrderStatusAmending:
		ad := o.amendingDetail
		oDiff := !o.price.Equal(detail.Price) || !o.qty.Equal(detail.Qty)
		aDiff := !ad.Price.Equal(detail.Price) || !ad.Qty.Equal(detail.Qty)
		return oDiff && aDiff
	}
}

func (o *Order) isAmendAble() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	const limitNOfAmend = 9500
	return o.amendCount < limitNOfAmend
}

func (o *Order) OutboundDuration() *time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.arrivedAt == nil || o.sentAt == nil {
		return nil
	}
	d := o.arrivedAt.Sub(*o.sentAt)
	return &d
}

func (o *Order) ReturnDuration() *time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.arrivedAt == nil || o.confirmedAt == nil {
		return nil
	}
	d := o.confirmedAt.Sub(*o.arrivedAt)
	return &d
}

func (o *Order) RoundTripDuration() *time.Duration {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.sentAt == nil || o.confirmedAt == nil {
		return nil
	}
	d := o.confirmedAt.Sub(*o.sentAt)
	return &d
}

func NewLimitOrderBySize(price, size decimal.Decimal, hide bool, memo string) *Order {
	qty, side := ToQty(size)
	return NewLimitOrder(price, qty, side, hide, memo)
}

func NewLimitOrder(price, qty decimal.Decimal, side domains.Side, hide bool, memo string) *Order {
	return &Order{
		id:        xid.New().String(),
		orderType: domains.OrderTypeLimit,
		price:     price,
		qty:       qty,
		side:      side,
		status:    OrderStatusBorn,
		hide:      hide,
		memo:      memo,
	}
}

func NewLimitFOKOrderBySize(price, size decimal.Decimal, memo string) *Order {
	qty, side := ToQty(size)
	return NewLimitFOKOrder(price, qty, side, memo)
}

func NewLimitFOKOrder(price, qty decimal.Decimal, side domains.Side, memo string) *Order {
	return &Order{
		id:        xid.New().String(),
		orderType: domains.OrderTypeLimitFOK,
		price:     price,
		qty:       qty,
		side:      side,
		status:    OrderStatusBorn,
		hide:      false,
		memo:      memo,
	}
}

func NewLimitIOCOrderBySize(price, arrowSlippage, size decimal.Decimal, memo string) *Order {
	qty, side := ToQty(size)
	return NewLimitIOCOrder(price, arrowSlippage, qty, side, memo)
}

func NewLimitIOCOrder(price, arrowSlippage, qty decimal.Decimal, side domains.Side, memo string) *Order {
	adjustedPrice := price
	switch side {
	case domains.SideSell:
		adjustedPrice = price.Sub(arrowSlippage)
	case domains.SideBuy:
		adjustedPrice = price.Add(arrowSlippage)
	}
	return &Order{
		id:        xid.New().String(),
		orderType: domains.OrderTypeLimitIOC,
		price:     adjustedPrice,
		qty:       qty,
		side:      side,
		status:    OrderStatusBorn,
		hide:      false,
		memo:      memo,
	}
}

func NewLimitMakerOrderBySize(price, size decimal.Decimal, hide bool, memo string) *Order {
	qty, side := ToQty(size)
	return NewLimitMakerOrder(price, qty, side, hide, memo)
}

func NewLimitMakerOrder(price, qty decimal.Decimal, side domains.Side, hide bool, memo string) *Order {
	return &Order{
		id:        xid.New().String(),
		orderType: domains.OrderTypeLimitMaker,
		price:     price,
		qty:       qty,
		side:      side,
		status:    OrderStatusBorn,
		hide:      hide,
		memo:      memo,
	}
}

func NewMarketOrderBySize(size decimal.Decimal, memo string) *Order {
	qty, side := ToQty(size)
	return NewMarketOrder(qty, side, memo)
}

func NewMarketOrder(qty decimal.Decimal, side domains.Side, memo string) *Order {
	return &Order{
		id:        xid.New().String(),
		orderType: domains.OrderTypeMarket,
		price:     decimal.Zero,
		qty:       qty,
		side:      side,
		status:    OrderStatusBorn,
		hide:      false,
		memo:      memo,
	}
}

// ToSize converts side + qty into a signed position-style size value.
func ToSize(side domains.Side, qty decimal.Decimal) decimal.Decimal {
	switch side {
	case domains.SideBuy:
		return qty
	case domains.SideSell:
		return qty.Neg()
	default:
		return decimal.Zero
	}
}

// ToQty converts a signed size value into qty and side.
func ToQty(size decimal.Decimal) (decimal.Decimal, domains.Side) {
	var side = domains.SideNone
	if size.IsNegative() {
		side = domains.SideSell
	} else if size.IsPositive() {
		side = domains.SideBuy
	}
	return size.Abs(), side
}

type OrderMutexSlice struct {
	*mutex.Slice[*Order]
}

func NewOrderMuArray(s []*Order) *OrderMutexSlice {
	return &OrderMutexSlice{mutex.NewSlice[*Order](s, 0)}
}

type OrdersMutexMap struct {
	mutex.Map[string, *Order]
}

func NewOrdersMap(m map[string]*Order) *OrdersMutexMap {
	return &OrdersMutexMap{
		Map: *mutex.NewMap[string, *Order](m),
	}
}

func (m *OrdersMutexMap) getOrder(orderID string) *Order {
	if order, ok := m.Get(orderID); ok {
		return order
	}
	return nil
}

func (m *OrdersMutexMap) getAmendingOrders(orderIDs []string) []*Order {
	var os []*Order
	for _, order := range m.Gets(orderIDs) {
		if order != nil && order.GetStatus() == OrderStatusAmending {
			os = append(os, order)
		}
	}
	return os
}

func (m *OrdersMutexMap) getAmendingOrder(orderID string) *Order {
	if order, ok := m.Get(orderID); ok && order.GetStatus() == OrderStatusAmending {
		return order
	}
	return nil
}

func (m *OrdersMutexMap) getSendingOrders(orderIDs []string) []*Order {
	var os []*Order
	for _, order := range m.Gets(orderIDs) {
		if order != nil && order.GetStatus() == OrderStatusSending {
			os = append(os, order)
		}
	}
	return os
}

func (m *OrdersMutexMap) getSendingOrder(orderID string) *Order {
	if order, ok := m.Get(orderID); ok && order.GetStatus() == OrderStatusSending {
		return order
	}
	return nil
}

func (m *OrdersMutexMap) getCancelingOrders(orderIDs []string) []*Order {
	var os []*Order
	for _, order := range m.Gets(orderIDs) {
		if order != nil && order.GetStatus() == OrderStatusCanceling {
			os = append(os, order)
		}
	}
	return os
}

func (m *OrdersMutexMap) getCancelingOrder(orderID string) *Order {
	if order, ok := m.Get(orderID); ok && order.GetStatus() == OrderStatusCanceling {
		return order
	}
	return nil
}

func (m *OrdersMutexMap) getPendingOrder(orderID string) *Order {
	if order, ok := m.Get(orderID); ok && order.GetStatus() == OrderStatusPending {
		return order
	}
	return nil
}

func (m *OrdersMutexMap) sumSize() decimal.Decimal {
	var sum decimal.Decimal
	m.Range(func(_ string, order *Order) bool {
		sum = sum.Add(order.GetSize())
		return true
	})
	return sum
}

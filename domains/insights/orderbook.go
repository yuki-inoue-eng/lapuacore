package insights

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type OrderBookDataHandler func(data *OrderBookData)

type OrderBookData struct {
	Type      DataType
	ExecAt    time.Time // Time when the execution occurred on the exchange
	ArrivedAt time.Time // Time when the data arrived at lapua
	Bids      []OBRecord
	Asks      []OBRecord
	SeqID     int64
}

type OrderBook struct {
	mu                    sync.RWMutex
	muLockToUpdateHandler sync.Mutex
	symbol                *domains.Symbol
	lastExecAt            *time.Time
	lastArrivedAt         *time.Time
	BidsMap               *OBRecordMap
	AsksMap               *OBRecordMap

	// These are updated by the update handler.
	// Recomputing them from RecordMap on every access would be wasteful,
	// so they are cached whenever the order book is updated.
	currentBestAsk *OBRecord
	currentBestBid *OBRecord
	beforeBestAsk  *OBRecord
	beforeBestBid  *OBRecord

	updateCallback []func()
	deferCallback  func()
}

func NewOrderBook(symbol *domains.Symbol) *OrderBook {
	return &OrderBook{
		symbol:  symbol,
		BidsMap: newOBRecordMap(domains.QuoteBid, symbol.TickSize()),
		AsksMap: newOBRecordMap(domains.QuoteAsk, symbol.TickSize()),
	}
}

// CalcBestPrice calculates the Ask and Bid closest to the specified midPrice.
func (p *OrderBook) CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tickSize := p.symbol.TickSize()
	bestAsk := p.RoundToTickSize(midPrice).Add(tickSize)
	bestBid := p.RoundToTickSize(midPrice)
	return bestAsk, bestBid
}

func (p *OrderBook) GetBestAsk() *OBRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentBestAsk
}

func (p *OrderBook) GetBestBid() *OBRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentBestBid
}

func (p *OrderBook) GetDiffBestAsk() *OBRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &OBRecord{
		// The SeqID difference is not very meaningful, but keep it for completeness.
		SeqID:  p.currentBestAsk.SeqID - p.beforeBestAsk.SeqID,
		Price:  p.currentBestAsk.Price.Sub(p.beforeBestAsk.Price),
		Volume: p.currentBestAsk.Volume.Sub(p.beforeBestAsk.Volume),
	}
}

func (p *OrderBook) GetDiffBestBid() *OBRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &OBRecord{
		// The SeqID difference is not very meaningful, but keep it for completeness.
		SeqID:  p.currentBestBid.SeqID - p.beforeBestBid.SeqID,
		Price:  p.currentBestBid.Price.Sub(p.beforeBestBid.Price),
		Volume: p.currentBestBid.Volume.Sub(p.beforeBestBid.Volume),
	}
}

func (p *OrderBook) setCurrentBestAsk(r *OBRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBestAsk = r
}

func (p *OrderBook) setCurrentBestBid(r *OBRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBestBid = r
}

func (p *OrderBook) setBeforeBestAsk(r *OBRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beforeBestAsk = r
}

func (p *OrderBook) setBeforeBestBid(r *OBRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beforeBestBid = r
}

func (p *OrderBook) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentBestBid != nil && p.currentBestAsk != nil &&
		p.beforeBestBid != nil && p.beforeBestAsk != nil &&
		p.lastExecAt != nil
}

func (p *OrderBook) GetTickSize() decimal.Decimal {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.symbol.TickSize()
}

func (p *OrderBook) setLastExecAt(t *time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastExecAt = t
}

func (p *OrderBook) setLastArrivedAt(t *time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastArrivedAt = t
}

func (p *OrderBook) GetLastExecAt() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastExecAt
}

func (p *OrderBook) GetLastArrivedAt() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastArrivedAt
}

// RoundToTickSize rounds the given price down to the order book tick size
// (i.e. truncates it to a multiple of tickSize).
// This is used when retrieving order book data at a specific price.
func (p *OrderBook) RoundToTickSize(price decimal.Decimal) decimal.Decimal {
	quotient := price.Div(p.symbol.TickSize()).Truncate(0)
	return quotient.Mul(p.symbol.TickSize())
}

func (p *OrderBook) GetMinOrderQty() decimal.Decimal {
	return p.symbol.MinOrderQty()
}

// SumVolume returns the market order quantity required to fully execute
// the limit order resting at the specified price.
// If the specified price is already marketable and would have been executed,
// it returns 0.
func (p *OrderBook) SumVolume(quote domains.Quote, price decimal.Decimal) decimal.Decimal {
	switch quote {
	case domains.QuoteAsk:
		return p.AsksMap.SumVolume(price)
	case domains.QuoteBid:
		return p.BidsMap.SumVolume(price)
	default:
		return decimal.Zero
	}
}

// AvgExecPrice returns the average execution price for a market order
// with the specified quantity.
// The result is rounded down to the order book tick size.
func (p *OrderBook) AvgExecPrice(quote domains.Quote, qty decimal.Decimal) decimal.Decimal {
	switch quote {
	case domains.QuoteAsk:
		return p.RoundToTickSize(p.AsksMap.AvgExecPrice(qty))
	case domains.QuoteBid:
		return p.RoundToTickSize(p.BidsMap.AvgExecPrice(qty))
	default:
		return decimal.Zero
	}
}

func (p *OrderBook) AvgExecPriceBySide(side domains.Side, qty decimal.Decimal) decimal.Decimal {
	switch side {
	case domains.SideBuy:
		return p.RoundToTickSize(p.AsksMap.AvgExecPrice(qty))
	case domains.SideSell:
		return p.RoundToTickSize(p.BidsMap.AvgExecPrice(qty))
	default:
		return decimal.Zero
	}
}

func (p *OrderBook) CalculateBidsVolSumMap() *OBRecordMap {
	bidsVolSumMap := newOBRecordMap(domains.QuoteBid, p.symbol.TickSize())
	sumVol := decimal.Zero
	p.BidsMap.SortedRange(func(price decimal.Decimal, record OBRecord) bool {
		sumVol = sumVol.Add(record.Volume)
		bidsVolSumMap.set(OBRecord{Price: price, Volume: sumVol})
		return true
	})
	return bidsVolSumMap
}

func (p *OrderBook) CalculateAsksVolSumMap() *OBRecordMap {
	asksVolSumMap := newOBRecordMap(domains.QuoteAsk, p.symbol.TickSize())
	sumVol := decimal.Zero
	p.AsksMap.SortedRange(func(price decimal.Decimal, record OBRecord) bool {
		sumVol = sumVol.Add(record.Volume)
		asksVolSumMap.set(OBRecord{Price: price, Volume: sumVol})
		return true
	})
	return asksVolSumMap
}

func (p *OrderBook) SetUpdateCallback(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateCallback = append(p.updateCallback, callback)
}

func (p *OrderBook) getUpdateCallback() []func() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.updateCallback
}

func (p *OrderBook) SetDeferUpdateCallBack(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deferCallback = callback
}

func (p *OrderBook) getDeferUpdateCallback() func() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deferCallback
}

func (p *OrderBook) DropDeferUpdateCallBack() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deferCallback = func() {}
}

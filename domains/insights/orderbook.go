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
	Bids      []PriceLevel
	Asks      []PriceLevel
	SeqID     int64
}

type OrderBookImpl struct {
	mu                    sync.RWMutex
	muLockToUpdateHandler sync.Mutex
	symbol                *domains.Symbol
	lastExecAt            *time.Time
	lastArrivedAt         *time.Time
	BidsMap               *PriceLevelMap
	AsksMap               *PriceLevelMap

	// These are updated by the update handler.
	// Recomputing them from RecordMap on every access would be wasteful,
	// so they are cached whenever the order book is updated.
	currentBestAsk *PriceLevel
	currentBestBid *PriceLevel
	beforeBestAsk  *PriceLevel
	beforeBestBid  *PriceLevel

	updateCallback []func()
	deferCallback  func()
}

func NewOrderBook(symbol *domains.Symbol) *OrderBookImpl {
	return &OrderBookImpl{
		symbol:  symbol,
		BidsMap: newPriceLevelMap(domains.BookSideBid, symbol.TickSize()),
		AsksMap: newPriceLevelMap(domains.BookSideAsk, symbol.TickSize()),
	}
}

// CalcBestPrice calculates the Ask and Bid closest to the specified midPrice.
func (p *OrderBookImpl) CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tickSize := p.symbol.TickSize()
	bestAsk := p.RoundToTickSize(midPrice).Add(tickSize)
	bestBid := p.RoundToTickSize(midPrice)
	return bestAsk, bestBid
}

func (p *OrderBookImpl) GetBestAsk() *PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentBestAsk
}

func (p *OrderBookImpl) GetBestBid() *PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentBestBid
}

func (p *OrderBookImpl) GetDiffBestAsk() *PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &PriceLevel{
		// The SeqID difference is not very meaningful, but keep it for completeness.
		SeqID:  p.currentBestAsk.SeqID - p.beforeBestAsk.SeqID,
		Price:  p.currentBestAsk.Price.Sub(p.beforeBestAsk.Price),
		Volume: p.currentBestAsk.Volume.Sub(p.beforeBestAsk.Volume),
	}
}

func (p *OrderBookImpl) GetDiffBestBid() *PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &PriceLevel{
		// The SeqID difference is not very meaningful, but keep it for completeness.
		SeqID:  p.currentBestBid.SeqID - p.beforeBestBid.SeqID,
		Price:  p.currentBestBid.Price.Sub(p.beforeBestBid.Price),
		Volume: p.currentBestBid.Volume.Sub(p.beforeBestBid.Volume),
	}
}

func (p *OrderBookImpl) setCurrentBestAsk(r *PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBestAsk = r
}

func (p *OrderBookImpl) setCurrentBestBid(r *PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentBestBid = r
}

func (p *OrderBookImpl) setBeforeBestAsk(r *PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beforeBestAsk = r
}

func (p *OrderBookImpl) setBeforeBestBid(r *PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beforeBestBid = r
}

func (p *OrderBookImpl) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentBestBid != nil && p.currentBestAsk != nil &&
		p.beforeBestBid != nil && p.beforeBestAsk != nil &&
		p.lastExecAt != nil
}

func (p *OrderBookImpl) GetTickSize() decimal.Decimal {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.symbol.TickSize()
}

func (p *OrderBookImpl) setLastExecAt(t *time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastExecAt = t
}

func (p *OrderBookImpl) setLastArrivedAt(t *time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastArrivedAt = t
}

func (p *OrderBookImpl) GetLastExecAt() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastExecAt
}

func (p *OrderBookImpl) GetLastArrivedAt() *time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastArrivedAt
}

// RoundToTickSize rounds the given price down to the order book tick size
// (i.e. truncates it to a multiple of tickSize).
// This is used when retrieving order book data at a specific price.
func (p *OrderBookImpl) RoundToTickSize(price decimal.Decimal) decimal.Decimal {
	quotient := price.Div(p.symbol.TickSize()).Truncate(0)
	return quotient.Mul(p.symbol.TickSize())
}

func (p *OrderBookImpl) GetMinOrderQty() decimal.Decimal {
	return p.symbol.MinOrderQty()
}

// SumVolume returns the market order quantity required to fully execute
// the limit order resting at the specified price.
// If the specified price is already marketable and would have been executed,
// it returns 0.
func (p *OrderBookImpl) SumVolume(bookSide domains.BookSide, price decimal.Decimal) decimal.Decimal {
	switch bookSide {
	case domains.BookSideAsk:
		return p.AsksMap.SumVolume(price)
	case domains.BookSideBid:
		return p.BidsMap.SumVolume(price)
	default:
		return decimal.Zero
	}
}

// AvgExecPrice returns the average execution price for a market order
// with the specified quantity.
// The result is rounded down to the order book tick size.
func (p *OrderBookImpl) AvgExecPrice(bookSide domains.BookSide, qty decimal.Decimal) decimal.Decimal {
	switch bookSide {
	case domains.BookSideAsk:
		return p.RoundToTickSize(p.AsksMap.AvgExecPrice(qty))
	case domains.BookSideBid:
		return p.RoundToTickSize(p.BidsMap.AvgExecPrice(qty))
	default:
		return decimal.Zero
	}
}

func (p *OrderBookImpl) AvgExecPriceBySide(side domains.Side, qty decimal.Decimal) decimal.Decimal {
	switch side {
	case domains.SideBuy:
		return p.RoundToTickSize(p.AsksMap.AvgExecPrice(qty))
	case domains.SideSell:
		return p.RoundToTickSize(p.BidsMap.AvgExecPrice(qty))
	default:
		return decimal.Zero
	}
}

func (p *OrderBookImpl) CalculateBidsVolSumMap() *PriceLevelMap {
	bidsVolSumMap := newPriceLevelMap(domains.BookSideBid, p.symbol.TickSize())
	sumVol := decimal.Zero
	p.BidsMap.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		sumVol = sumVol.Add(record.Volume)
		bidsVolSumMap.set(PriceLevel{Price: price, Volume: sumVol})
		return true
	})
	return bidsVolSumMap
}

func (p *OrderBookImpl) CalculateAsksVolSumMap() *PriceLevelMap {
	asksVolSumMap := newPriceLevelMap(domains.BookSideAsk, p.symbol.TickSize())
	sumVol := decimal.Zero
	p.AsksMap.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		sumVol = sumVol.Add(record.Volume)
		asksVolSumMap.set(PriceLevel{Price: price, Volume: sumVol})
		return true
	})
	return asksVolSumMap
}

func (p *OrderBookImpl) GetAsks(depth int) []PriceLevel {
	var levels []PriceLevel
	p.AsksMap.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		levels = append(levels, record)
		return len(levels) < depth
	})
	return levels
}

func (p *OrderBookImpl) GetBids(depth int) []PriceLevel {
	var levels []PriceLevel
	p.BidsMap.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		levels = append(levels, record)
		return len(levels) < depth
	})
	return levels
}

func (p *OrderBookImpl) SetUpdateCallback(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateCallback = append(p.updateCallback, callback)
}

func (p *OrderBookImpl) getUpdateCallback() []func() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.updateCallback
}

func (p *OrderBookImpl) SetDeferUpdateCallBack(callback func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deferCallback = callback
}

func (p *OrderBookImpl) getDeferUpdateCallback() func() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deferCallback
}

func (p *OrderBookImpl) DropDeferUpdateCallBack() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deferCallback = func() {}
}

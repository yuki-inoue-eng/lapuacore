package insights

import (
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type BookTickerDataHandler func(msg *BookTickerData)
type BookTickerData struct {
	SeqID     int64
	ExecAt    time.Time // Time when the price change occurred on the exchange
	EventAt   time.Time
	ArrivedAt time.Time // Time when the data arrived locally
	BestAsk   *OBRecord
	BestBid   *OBRecord
}

type BookTicker struct {
	mu                    sync.RWMutex
	muLockToUpdateHandler sync.Mutex
	seqID                 int64
	symbol                *domains.Symbol
	lastExecAt            *time.Time
	lastEventAt           *time.Time
	lastArrivedAt         *time.Time
	bestAsk               *OBRecord
	bestBid               *OBRecord

	beforeBestAsk *OBRecord
	beforeBestBid *OBRecord

	updateCallback []func()
}

func NewBookTicker(symbol *domains.Symbol) *BookTicker {
	return &BookTicker{
		symbol: symbol,
	}
}

func (bt *BookTicker) IsReady() bool {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.bestBid != nil && bt.bestAsk != nil &&
		bt.beforeBestBid != nil && bt.beforeBestAsk != nil &&
		bt.lastArrivedAt != nil
}

func (bt *BookTicker) GetTickSize() decimal.Decimal {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.symbol.TickSize()
}

func (bt *BookTicker) setBestAsk(bestAsk *OBRecord) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.bestAsk = bestAsk
}

// CalcBestPrice calculates the Ask and Bid closest to the specified midPrice.
func (bt *BookTicker) CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	tickSize := bt.symbol.TickSize()
	bestAsk := bt.RoundToTickSize(midPrice).Add(tickSize)
	bestBid := bt.RoundToTickSize(midPrice)
	return bestAsk, bestBid
}

func (bt *BookTicker) GetBestAsk() *OBRecord {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.bestAsk
}

func (bt *BookTicker) setBestBid(bestBid *OBRecord) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.bestBid = bestBid
}

func (bt *BookTicker) GetBestBid() *OBRecord {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.bestBid
}

func (bt *BookTicker) GetDiffBestAsk() *OBRecord {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return &OBRecord{
		SeqID:  bt.bestAsk.SeqID - bt.beforeBestAsk.SeqID,
		Price:  bt.bestAsk.Price.Sub(bt.beforeBestAsk.Price),
		Volume: bt.bestAsk.Volume.Sub(bt.beforeBestAsk.Volume),
	}
}

func (bt *BookTicker) GetDiffBestBid() *OBRecord {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return &OBRecord{
		SeqID:  bt.bestBid.SeqID - bt.beforeBestBid.SeqID,
		Price:  bt.bestBid.Price.Sub(bt.beforeBestBid.Price),
		Volume: bt.bestBid.Volume.Sub(bt.beforeBestBid.Volume),
	}
}

func (bt *BookTicker) setBeforeBestAsk(r *OBRecord) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.beforeBestAsk = r
}

func (bt *BookTicker) setBeforeBestBid(r *OBRecord) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.beforeBestBid = r
}

func (bt *BookTicker) setLastExecAt(t *time.Time) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.lastExecAt = t
}

func (bt *BookTicker) setLastEventAt(t *time.Time) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.lastEventAt = t
}

func (bt *BookTicker) setLastArrivedAt(t *time.Time) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.lastArrivedAt = t
}

func (bt *BookTicker) GetLastExecAt() *time.Time {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.lastExecAt
}

func (bt *BookTicker) GetLastEventAt() *time.Time {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.lastEventAt
}

func (bt *BookTicker) GetLastArrivedAt() *time.Time {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.lastArrivedAt
}

func (bt *BookTicker) setSeqID(id int64) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.seqID = id
}

func (bt *BookTicker) GetSeqID() int64 {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.seqID
}

func (bt *BookTicker) SetUpdateCallback(callback func()) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.updateCallback = append(bt.updateCallback, callback)
}

func (bt *BookTicker) getUpdateCallback() []func() {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.updateCallback
}

// RoundToTickSize rounds the given price down to the tick size
// (i.e. truncates it to a multiple of tickSize).
func (bt *BookTicker) RoundToTickSize(price decimal.Decimal) decimal.Decimal {
	quotient := price.Div(bt.symbol.TickSize()).Truncate(0)
	return quotient.Mul(bt.symbol.TickSize())
}

func (bt *BookTicker) GetMinOrderQty() decimal.Decimal {
	return bt.symbol.MinOrderQty()
}

func (bt *BookTicker) Update(msg *BookTickerData) {
	bt.muLockToUpdateHandler.Lock()
	defer bt.muLockToUpdateHandler.Unlock()

	// Record previous best prices
	bestAsk := bt.GetBestAsk()
	bestBid := bt.GetBestBid()
	if bestAsk != nil && bestBid != nil {
		bt.setBeforeBestAsk(bestAsk.Copy())
		bt.setBeforeBestBid(bestBid.Copy())
	}
	if bt.GetLastExecAt() != nil && msg.ExecAt.Before(*bt.GetLastExecAt()) {
		slog.Info("book ticker old data received")
		return
	}
	if bt.GetSeqID() > msg.SeqID {
		slog.Info("book ticker old seq_id data received")
		return
	}

	bt.setSeqID(msg.SeqID)
	bt.setLastExecAt(&msg.ExecAt)
	bt.setLastEventAt(&msg.EventAt)
	bt.setLastArrivedAt(&msg.ArrivedAt)
	bt.setBestAsk(msg.BestAsk)
	bt.setBestBid(msg.BestBid)

	// Execute registered callbacks
	callbacks := bt.getUpdateCallback()
	for _, c := range callbacks {
		c()
	}
}

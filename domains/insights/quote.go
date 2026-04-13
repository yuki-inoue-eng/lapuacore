package insights

import (
	"log/slog"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type QuoteDataHandler func(msg *QuoteData)
type QuoteData struct {
	SeqID     int64
	ExecAt    time.Time // Time when the price change occurred on the exchange
	EventAt   time.Time
	ArrivedAt time.Time // Time when the data arrived locally
	BestAsk   *OBRecord
	BestBid   *OBRecord
}

type QuoteImpl struct {
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

func NewQuote(symbol *domains.Symbol) *QuoteImpl {
	return &QuoteImpl{
		symbol: symbol,
	}
}

func (q *QuoteImpl) IsReady() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.bestBid != nil && q.bestAsk != nil &&
		q.beforeBestBid != nil && q.beforeBestAsk != nil &&
		q.lastArrivedAt != nil
}

func (q *QuoteImpl) GetTickSize() decimal.Decimal {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.symbol.TickSize()
}

func (q *QuoteImpl) setBestAsk(bestAsk *OBRecord) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.bestAsk = bestAsk
}

// CalcBestPrice calculates the Ask and Bid closest to the specified midPrice.
func (q *QuoteImpl) CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	tickSize := q.symbol.TickSize()
	bestAsk := q.RoundToTickSize(midPrice).Add(tickSize)
	bestBid := q.RoundToTickSize(midPrice)
	return bestAsk, bestBid
}

func (q *QuoteImpl) GetBestAsk() *OBRecord {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.bestAsk
}

func (q *QuoteImpl) setBestBid(bestBid *OBRecord) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.bestBid = bestBid
}

func (q *QuoteImpl) GetBestBid() *OBRecord {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.bestBid
}

func (q *QuoteImpl) GetDiffBestAsk() *OBRecord {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return &OBRecord{
		SeqID:  q.bestAsk.SeqID - q.beforeBestAsk.SeqID,
		Price:  q.bestAsk.Price.Sub(q.beforeBestAsk.Price),
		Volume: q.bestAsk.Volume.Sub(q.beforeBestAsk.Volume),
	}
}

func (q *QuoteImpl) GetDiffBestBid() *OBRecord {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return &OBRecord{
		SeqID:  q.bestBid.SeqID - q.beforeBestBid.SeqID,
		Price:  q.bestBid.Price.Sub(q.beforeBestBid.Price),
		Volume: q.bestBid.Volume.Sub(q.beforeBestBid.Volume),
	}
}

func (q *QuoteImpl) setBeforeBestAsk(r *OBRecord) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.beforeBestAsk = r
}

func (q *QuoteImpl) setBeforeBestBid(r *OBRecord) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.beforeBestBid = r
}

func (q *QuoteImpl) setLastExecAt(t *time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.lastExecAt = t
}

func (q *QuoteImpl) setLastEventAt(t *time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.lastEventAt = t
}

func (q *QuoteImpl) setLastArrivedAt(t *time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.lastArrivedAt = t
}

func (q *QuoteImpl) GetLastExecAt() *time.Time {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.lastExecAt
}

func (q *QuoteImpl) GetLastEventAt() *time.Time {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.lastEventAt
}

func (q *QuoteImpl) GetLastArrivedAt() *time.Time {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.lastArrivedAt
}

func (q *QuoteImpl) setSeqID(id int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.seqID = id
}

func (q *QuoteImpl) GetSeqID() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.seqID
}

func (q *QuoteImpl) SetUpdateCallback(callback func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.updateCallback = append(q.updateCallback, callback)
}

func (q *QuoteImpl) getUpdateCallback() []func() {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.updateCallback
}

// RoundToTickSize rounds the given price down to the tick size
// (i.e. truncates it to a multiple of tickSize).
func (q *QuoteImpl) RoundToTickSize(price decimal.Decimal) decimal.Decimal {
	quotient := price.Div(q.symbol.TickSize()).Truncate(0)
	return quotient.Mul(q.symbol.TickSize())
}

func (q *QuoteImpl) GetMinOrderQty() decimal.Decimal {
	return q.symbol.MinOrderQty()
}

func (q *QuoteImpl) Update(msg *QuoteData) {
	q.muLockToUpdateHandler.Lock()
	defer q.muLockToUpdateHandler.Unlock()

	// Record previous best prices
	bestAsk := q.GetBestAsk()
	bestBid := q.GetBestBid()
	if bestAsk != nil && bestBid != nil {
		q.setBeforeBestAsk(bestAsk.Copy())
		q.setBeforeBestBid(bestBid.Copy())
	}
	if q.GetLastExecAt() != nil && msg.ExecAt.Before(*q.GetLastExecAt()) {
		slog.Info("quote old data received")
		return
	}
	if q.GetSeqID() > msg.SeqID {
		slog.Info("quote old seq_id data received")
		return
	}

	q.setSeqID(msg.SeqID)
	q.setLastExecAt(&msg.ExecAt)
	q.setLastEventAt(&msg.EventAt)
	q.setLastArrivedAt(&msg.ArrivedAt)
	q.setBestAsk(msg.BestAsk)
	q.setBestBid(msg.BestBid)

	// Execute registered callbacks
	callbacks := q.getUpdateCallback()
	for _, c := range callbacks {
		c()
	}
}

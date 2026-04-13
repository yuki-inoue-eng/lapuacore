package insights

import (
	"log/slog"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type DataType int

const (
	DataTypeUnknown DataType = iota
	DataTypeSnapshot
	DataTypeDelta
)

func (p *OrderBookImpl) UpdateByOBData(data *OrderBookData) {
	// Acquire the lock at the update-handler level so that
	// the best price and the order book do not get out of sync.
	p.muLockToUpdateHandler.Lock()
	defer p.muLockToUpdateHandler.Unlock()

	// Record the past best price
	bestAsk := p.GetBestAsk()
	bestBid := p.GetBestBid()
	if bestAsk != nil && bestBid != nil {
		p.setBeforeBestAsk(bestAsk.Copy())
		p.setBeforeBestBid(bestBid.Copy())
	}

	p.setLastExecAt(&data.ExecAt)
	p.setLastArrivedAt(&data.ArrivedAt)
	switch data.Type {
	case DataTypeSnapshot:
		p.resetBySnapshot(data)
	case DataTypeDelta:
		p.updateByDelta(data)
	default:
	}

	// Record the current best price
	bestAsk_ := p.AsksMap.BestLevel()
	bestBid_ := p.BidsMap.BestLevel()
	p.setCurrentBestAsk(bestAsk_.Copy())
	p.setCurrentBestBid(bestBid_.Copy())

	// Execute the callbacks configured for this order book
	callbacks := p.getUpdateCallback()
	for _, c := range callbacks {
		c()
	}
	if deferCb := p.getDeferUpdateCallback(); deferCb != nil {
		deferCb()
	}
}

func (p *OrderBookImpl) updateByDelta(delta *OrderBookData) {
	hasOldRecord := false

	// Update asks
	if len(delta.Asks) > 0 {
		for _, a := range delta.Asks {
			// Ignore the record if the existing order book row is newer.
			if r, ok := p.AsksMap.Get(a.Price); ok && !r.isOld(a.SeqID) {
				hasOldRecord = true
				continue
			}
			if a.Volume.IsZero() {
				p.AsksMap.drop(a.Price)
			} else {
				p.AsksMap.set(a)
			}
		}
	}

	// Update bids
	if len(delta.Bids) > 0 {
		for _, b := range delta.Bids {
			// Ignore the record if the existing order book row is newer.
			if r, ok := p.BidsMap.Get(b.Price); ok && !r.isOld(b.SeqID) {
				hasOldRecord = true
				continue
			}
			if b.Volume.IsZero() {
				p.BidsMap.drop(b.Price)
			} else {
				p.BidsMap.set(b)
			}
		}
	}

	if hasOldRecord {
		slog.Info("order book old delta received")
	}
}

func (p *OrderBookImpl) resetBySnapshot(snapshot *OrderBookData) {

	hasOldRecord := false

	// Reset asks
	if len(snapshot.Asks) > 0 {
		askMap := newPriceLevelMap(domains.BookSideAsk, p.symbol.TickSize())
		for _, a := range snapshot.Asks {
			askMap.set(a)
		}
		// Overwrite with existing rows if they are newer than the snapshot.
		p.AsksMap.Range(func(price decimal.Decimal, record PriceLevel) bool {
			if sr, ok := askMap.Get(price); ok && sr.isOld(record.SeqID) {
				hasOldRecord = true
				askMap.set(record)
			}
			return true
		})
		p.AsksMap.replace(askMap)
	}

	// Reset bids
	if len(snapshot.Bids) > 0 {
		bidMap := newPriceLevelMap(domains.BookSideBid, p.symbol.TickSize())
		for _, a := range snapshot.Bids {
			bidMap.set(a)
		}
		// Overwrite with existing rows if they are newer than the snapshot.
		p.BidsMap.Range(func(price decimal.Decimal, record PriceLevel) bool {
			if sr, ok := bidMap.Get(price); ok && sr.isOld(record.SeqID) {
				hasOldRecord = true
				bidMap.set(record)
			}
			return true
		})
		p.BidsMap.replace(bidMap)
	}

	// Log when an older snapshot is received.
	if hasOldRecord {
		slog.Info("order book snapshot received (this snapshot is old)")
	}
}

package insights

import (
	"sync"

	"github.com/google/btree"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// PriceLevelMap stores price levels in a B-Tree ordered by price.
// Ask maps iterate in ascending price order (lowest = best).
// Bid maps iterate in descending price order (highest = best).
type PriceLevelMap struct {
	mu   sync.RWMutex
	ts   decimal.Decimal // tickSize
	q    domains.Quote
	tree *btree.BTreeG[PriceLevel]
}

func newPriceLevelMap(quote domains.Quote, tickSize decimal.Decimal) *PriceLevelMap {
	less := func(a, b PriceLevel) bool {
		if quote == domains.QuoteBid {
			return a.Price.GreaterThan(b.Price)
		}
		return a.Price.LessThan(b.Price)
	}
	return &PriceLevelMap{
		ts:   tickSize,
		q:    quote,
		tree: btree.NewG(16, less),
	}
}

func (p *PriceLevelMap) set(record PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if record.Volume.IsZero() {
		p.tree.Delete(PriceLevel{Price: record.Price})
		return
	}
	p.tree.ReplaceOrInsert(record)
}

func (p *PriceLevelMap) Get(price decimal.Decimal) (PriceLevel, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tree.Get(PriceLevel{Price: price})
}

func (p *PriceLevelMap) drop(price decimal.Decimal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tree.Delete(PriceLevel{Price: price})
}

func (p *PriceLevelMap) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tree.Len()
}

func (p *PriceLevelMap) replace(r *PriceLevelMap) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tree = r.tree
}

func (p *PriceLevelMap) Range(f func(price decimal.Decimal, record PriceLevel) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.tree.Ascend(func(item PriceLevel) bool {
		return f(item.Price, item)
	})
}

// BestLevel returns the best price level (lowest ask or highest bid).
// O(1) — B-Tree minimum access.
func (p *PriceLevelMap) BestLevel() PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	best, ok := p.tree.Min()
	if !ok {
		return PriceLevel{}
	}
	return *best.Copy()
}

// SumVolume returns the total market order quantity required to fully execute
// the limit order resting at the specified price.
// If the specified price is already marketable and would have been executed,
// it returns 0.
// O(k) where k is the number of levels up to the target price.
func (p *PriceLevelMap) SumVolume(price decimal.Decimal) decimal.Decimal {
	sumVol := decimal.Zero
	p.SortedRange(func(itaPrice decimal.Decimal, itaRecord PriceLevel) bool {
		if p.q == domains.QuoteAsk && itaPrice.GreaterThan(price) {
			return false
		}
		if p.q == domains.QuoteBid && itaPrice.LessThan(price) {
			return false
		}
		sumVol = sumVol.Add(itaRecord.Volume)
		return true
	})
	return sumVol
}

// AvgExecPrice returns the average execution price for a market order
// with the specified quantity.
// O(k) where k is the number of levels consumed to fill the quantity.
func (p *PriceLevelMap) AvgExecPrice(qty decimal.Decimal) decimal.Decimal {
	remainingQty := qty
	weightedSum := decimal.Zero

	p.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		nextRemainingQty := remainingQty.Sub(record.Volume)
		if nextRemainingQty.LessThanOrEqual(decimal.Zero) {
			weightedSum = weightedSum.Add(price.Mul(remainingQty))
			return false
		}

		weightedSum = weightedSum.Add(price.Mul(record.Volume))
		remainingQty = nextRemainingQty
		return true
	})

	return weightedSum.Div(qty)
}

// SortedRange iterates over price levels in best-to-worst order.
// O(n) — in-order B-Tree traversal without sorting.
func (p *PriceLevelMap) SortedRange(f func(price decimal.Decimal, record PriceLevel) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.tree.Ascend(func(item PriceLevel) bool {
		return f(item.Price, item)
	})
}

type PriceLevel struct {
	SeqID  int64
	Price  decimal.Decimal
	Volume decimal.Decimal
}

func (r *PriceLevel) isOld(nowID int64) bool {
	return r.SeqID < nowID
}

func (r *PriceLevel) Copy() *PriceLevel {
	return &PriceLevel{
		SeqID:  r.SeqID,
		Price:  r.Price,
		Volume: r.Volume,
	}
}

package insights

import (
	"sort"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type PriceLevelMap struct {
	mu         sync.RWMutex
	ts         decimal.Decimal // tickSize
	q          domains.Quote
	data       map[string]PriceLevel
	bestLevel *PriceLevel
}

func newPriceLevelMap(quote domains.Quote, tickSize decimal.Decimal) *PriceLevelMap {
	return &PriceLevelMap{
		ts:   tickSize,
		q:    quote,
		data: map[string]PriceLevel{},
	}
}

func (p *PriceLevelMap) set(record PriceLevel) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[record.Price.String()] = record
	p.updateBestLevelOnSetLocked(record)
}

func (p *PriceLevelMap) Get(price decimal.Decimal) (PriceLevel, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.data[price.String()]
	return val, ok
}

func (p *PriceLevelMap) drop(price decimal.Decimal) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.data, price.String())

	if p.bestLevel != nil && p.bestLevel.Price.Equal(price) {
		p.recalculateBestLevelLocked()
	}
}

func (p *PriceLevelMap) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.data)
}

func (p *PriceLevelMap) replace(r *PriceLevelMap) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data = r.data
	p.recalculateBestLevelLocked()
}

func (p *PriceLevelMap) Range(f func(price decimal.Decimal, record PriceLevel) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, record := range p.data {
		if !f(record.Price, record) {
			break
		}
	}
}

func (p *PriceLevelMap) BestLevel() PriceLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.bestLevel == nil {
		return PriceLevel{}
	}
	return *p.bestLevel.Copy()
}

// SumVolume returns the total market order quantity required to fully execute
// the limit order resting at the specified price.
// If the specified price is already marketable and would have been executed,
// it returns 0.
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

func (p *PriceLevelMap) SortedRange(f func(price decimal.Decimal, record PriceLevel) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, key := range p.sortedKeys() {
		record := p.data[key.String()]
		if !f(key, record) {
			break
		}
	}
}

func (p *PriceLevelMap) sortedKeys() []decimal.Decimal {
	var keys []decimal.Decimal
	for k := range p.data {
		price, err := decimal.NewFromString(k)
		if err != nil {
			continue
		}
		keys = append(keys, price)
	}

	sortFunc := func(i, j int) bool {
		if p.q == domains.QuoteBid {
			return keys[i].GreaterThan(keys[j])
		}
		return keys[i].LessThan(keys[j])
	}
	sort.Slice(keys, sortFunc)
	return keys
}

func (p *PriceLevelMap) updateBestLevelOnSetLocked(record PriceLevel) {
	if record.Volume.IsZero() {
		if p.bestLevel != nil && p.bestLevel.Price.Equal(record.Price) {
			p.recalculateBestLevelLocked()
		}
		return
	}

	if p.bestLevel == nil {
		p.bestLevel = record.Copy()
		return
	}

	if p.bestLevel.Price.Equal(record.Price) {
		p.bestLevel = record.Copy()
		return
	}

	if p.isBetterPrice(record.Price, p.bestLevel.Price) {
		p.bestLevel = record.Copy()
	}
}

func (p *PriceLevelMap) recalculateBestLevelLocked() {
	p.bestLevel = nil

	for _, record := range p.data {
		if record.Volume.IsZero() {
			continue
		}

		if p.bestLevel == nil || p.isBetterPrice(record.Price, p.bestLevel.Price) {
			p.bestLevel = record.Copy()
		}
	}
}

func (p *PriceLevelMap) isBetterPrice(left, right decimal.Decimal) bool {
	if p.q == domains.QuoteBid {
		return left.GreaterThan(right)
	}
	return left.LessThan(right)
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

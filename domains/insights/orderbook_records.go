package insights

import (
	"sort"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type OBRecordMap struct {
	mu         sync.RWMutex
	ts         decimal.Decimal // tickSize
	q          domains.Quote
	data       map[string]OBRecord
	bestRecord *OBRecord
}

func newOBRecordMap(quote domains.Quote, tickSize decimal.Decimal) *OBRecordMap {
	return &OBRecordMap{
		ts:   tickSize,
		q:    quote,
		data: map[string]OBRecord{},
	}
}

func (p *OBRecordMap) set(record OBRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[record.Price.String()] = record
	p.updateBestRecordOnSetLocked(record)
}

func (p *OBRecordMap) Get(price decimal.Decimal) (OBRecord, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	val, ok := p.data[price.String()]
	return val, ok
}

func (p *OBRecordMap) drop(price decimal.Decimal) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.data, price.String())

	if p.bestRecord != nil && p.bestRecord.Price.Equal(price) {
		p.recalculateBestRecordLocked()
	}
}

func (p *OBRecordMap) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.data)
}

func (p *OBRecordMap) replace(r *OBRecordMap) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data = r.data
	p.recalculateBestRecordLocked()
}

func (p *OBRecordMap) Range(f func(price decimal.Decimal, record OBRecord) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, record := range p.data {
		if !f(record.Price, record) {
			break
		}
	}
}

func (p *OBRecordMap) BestRecord() OBRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.bestRecord == nil {
		return OBRecord{}
	}
	return *p.bestRecord.Copy()
}

// SumVolume returns the total market order quantity required to fully execute
// the limit order resting at the specified price.
// If the specified price is already marketable and would have been executed,
// it returns 0.
func (p *OBRecordMap) SumVolume(price decimal.Decimal) decimal.Decimal {
	sumVol := decimal.Zero
	p.SortedRange(func(itaPrice decimal.Decimal, itaRecord OBRecord) bool {
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
func (p *OBRecordMap) AvgExecPrice(qty decimal.Decimal) decimal.Decimal {
	remainingQty := qty
	weightedSum := decimal.Zero

	p.SortedRange(func(price decimal.Decimal, record OBRecord) bool {
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

func (p *OBRecordMap) SortedRange(f func(price decimal.Decimal, record OBRecord) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, key := range p.sortedKeys() {
		record := p.data[key.String()]
		if !f(key, record) {
			break
		}
	}
}

func (p *OBRecordMap) sortedKeys() []decimal.Decimal {
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

func (p *OBRecordMap) updateBestRecordOnSetLocked(record OBRecord) {
	if record.Volume.IsZero() {
		if p.bestRecord != nil && p.bestRecord.Price.Equal(record.Price) {
			p.recalculateBestRecordLocked()
		}
		return
	}

	if p.bestRecord == nil {
		p.bestRecord = record.Copy()
		return
	}

	if p.bestRecord.Price.Equal(record.Price) {
		p.bestRecord = record.Copy()
		return
	}

	if p.isBetterPrice(record.Price, p.bestRecord.Price) {
		p.bestRecord = record.Copy()
	}
}

func (p *OBRecordMap) recalculateBestRecordLocked() {
	p.bestRecord = nil

	for _, record := range p.data {
		if record.Volume.IsZero() {
			continue
		}

		if p.bestRecord == nil || p.isBetterPrice(record.Price, p.bestRecord.Price) {
			p.bestRecord = record.Copy()
		}
	}
}

func (p *OBRecordMap) isBetterPrice(left, right decimal.Decimal) bool {
	if p.q == domains.QuoteBid {
		return left.GreaterThan(right)
	}
	return left.LessThan(right)
}

type OBRecord struct {
	SeqID  int64
	Price  decimal.Decimal
	Volume decimal.Decimal
}

func (r *OBRecord) isOld(nowID int64) bool {
	return r.SeqID < nowID
}

func (r *OBRecord) Copy() *OBRecord {
	return &OBRecord{
		SeqID:  r.SeqID,
		Price:  r.Price,
		Volume: r.Volume,
	}
}

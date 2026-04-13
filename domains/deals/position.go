package deals

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type Position struct {
	sync.RWMutex
	value        decimal.Decimal
	lastUpdateAt time.Time
}

func PosSide(pos decimal.Decimal) domains.Side {
	if pos.IsPositive() {
		return domains.SideBuy
	}
	if pos.IsNegative() {
		return domains.SideSell
	}
	return domains.SideNone
}

func NewPosition() *Position {
	return &Position{
		value:        decimal.Zero,
		lastUpdateAt: time.Now(),
	}
}

func (p *Position) Get() decimal.Decimal {
	p.RLock()
	defer p.RUnlock()
	return p.value
}

func (p *Position) GetQty() (domains.Side, decimal.Decimal) {
	pos := p.Get()
	return PosSide(pos), pos.Abs()
}

func (p *Position) GetSide() domains.Side {
	return PosSide(p.Get())
}

func (p *Position) getLastUpdateAt() time.Time {
	p.RLock()
	defer p.RUnlock()
	return p.lastUpdateAt
}

func (p *Position) update(timestamp time.Time, value decimal.Decimal) {
	p.Lock()
	defer p.Unlock()
	p.value = value
	p.lastUpdateAt = timestamp
}

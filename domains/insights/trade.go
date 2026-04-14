package insights

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
)

type TradeDataHandler func(msg TradeDataList)
type TradeDataList []*TradeData
type TradeData struct {
	ExecAt    time.Time // Time when the order was executed on the exchange
	ArrivedAt time.Time // Time when the data arrived locally
	Side      domains.Side
	Volume    decimal.Decimal
	Price     decimal.Decimal
}

func (d *TradeData) GetSize() decimal.Decimal {
	return deals.ToSize(d.Side, d.Volume)
}

func (l TradeDataList) GetSumSize() decimal.Decimal {
	var sum decimal.Decimal
	for _, m := range l {
		sum = sum.Add(m.GetSize())
	}
	return sum
}

func (l TradeDataList) GetSumSellVolume() decimal.Decimal {
	var sum decimal.Decimal
	for _, m := range l {
		if m.Side == domains.SideSell {
			sum = sum.Add(m.Volume)
		}
	}
	return sum
}

func (l TradeDataList) GetSumBuyVolume() decimal.Decimal {
	var sum decimal.Decimal
	for _, m := range l {
		if m.Side == domains.SideBuy {
			sum = sum.Add(m.Volume)
		}
	}
	return sum
}

func (l TradeDataList) GetExecAt() time.Time {
	return l[0].ExecAt
}

func (l TradeDataList) GetArrivedAt() time.Time {
	return l[0].ArrivedAt
}

// TradeImpl manages trade data updates for a symbol.
type TradeImpl struct {
	symbol       *domains.Symbol
	handlers     []TradeDataHandler
	lastUpdateAt *time.Time
}

func NewTrade(symbol *domains.Symbol) *TradeImpl {
	return &TradeImpl{
		symbol: symbol,
	}
}

func (t *TradeImpl) SetUpdateCallback(callback func(TradeDataList)) {
	t.handlers = append(t.handlers, callback)
}

func (t *TradeImpl) Update(msg TradeDataList) {
	for _, handler := range t.handlers {
		handler(msg)
	}
	ts := time.Now()
	t.lastUpdateAt = &ts
}

func (t *TradeImpl) IsReady() bool {
	return t.lastUpdateAt != nil
}

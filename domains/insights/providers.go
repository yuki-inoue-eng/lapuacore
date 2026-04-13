package insights

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// BestPriceProvider provides best price information.
type BestPriceProvider interface {
	CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal)
	GetBestAsk() *OBRecord
	GetBestBid() *OBRecord
	GetDiffBestAsk() *OBRecord
	GetDiffBestBid() *OBRecord
	GetLastExecAt() *time.Time
	GetLastArrivedAt() *time.Time
	GetTickSize() decimal.Decimal
	GetMinOrderQty() decimal.Decimal
	RoundToTickSize(price decimal.Decimal) decimal.Decimal
	SetUpdateCallback(callback func())
}

// OrderBook is the consumer-facing interface for order book data.
// It excludes UpdateByOBData which is only used by internal gateways.
type OrderBook interface {
	BestPriceProvider
	IsReady() bool
	SumVolume(quote domains.Quote, price decimal.Decimal) decimal.Decimal
	AvgExecPrice(quote domains.Quote, qty decimal.Decimal) decimal.Decimal
	AvgExecPriceBySide(side domains.Side, qty decimal.Decimal) decimal.Decimal
	CalculateBidsVolSumMap() *OBRecordMap
	CalculateAsksVolSumMap() *OBRecordMap
	SetDeferUpdateCallBack(callback func())
	DropDeferUpdateCallBack()
}

// BookTicker is the consumer-facing interface for book ticker data.
// It excludes Update which is only used by internal gateways.
type BookTicker interface {
	BestPriceProvider
	IsReady() bool
	GetLastEventAt() *time.Time
	GetSeqID() int64
}

// Trade is the consumer-facing interface for trade data.
// It excludes Update which is only used by internal gateways.
type Trade interface {
	IsReady() bool
	SetHandler(handler TradeDataHandler)
}

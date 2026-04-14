package insights

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// Quote provides best price information.
// Both OrderBookImpl and QuoteImpl implement this interface.
type Quote interface {
	IsReady() bool
	CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal)
	GetBestAsk() *PriceLevel
	GetBestBid() *PriceLevel
	GetDiffBestAsk() *PriceLevel
	GetDiffBestBid() *PriceLevel
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
	Quote
	SumVolume(bookSide domains.BookSide, price decimal.Decimal) decimal.Decimal
	AvgExecPrice(bookSide domains.BookSide, qty decimal.Decimal) decimal.Decimal
	AvgExecPriceBySide(side domains.Side, qty decimal.Decimal) decimal.Decimal
	CalculateBidsVolSumMap() *PriceLevelMap
	CalculateAsksVolSumMap() *PriceLevelMap
	SetDeferUpdateCallBack(callback func())
	DropDeferUpdateCallBack()
}

// Trade is the consumer-facing interface for trade data.
// It excludes Update which is only used by internal gateways.
type Trade interface {
	IsReady() bool
	SetUpdateCallback(callback func(msg TradeDataList))
}

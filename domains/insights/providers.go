package insights

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

// Quote provides best price information.
// Both OrderBookImpl and QuoteImpl implement this interface.
type Quote interface {
	// IsReady returns true when the initial data has been received.
	IsReady() bool
	// CalcBestPrice returns (bestAsk, bestBid) calculated from midPrice.
	// bestAsk is midPrice rounded up by one tick; bestBid is midPrice rounded down to the nearest tick.
	CalcBestPrice(midPrice decimal.Decimal) (decimal.Decimal, decimal.Decimal)
	// GetBestAsk returns the current best ask (lowest sell) price level.
	GetBestAsk() *PriceLevel
	// GetBestBid returns the current best bid (highest buy) price level.
	GetBestBid() *PriceLevel
	// GetDiffBestAsk returns the delta from the previous best ask (SeqID, Price, Volume differences).
	GetDiffBestAsk() *PriceLevel
	// GetDiffBestBid returns the delta from the previous best bid (SeqID, Price, Volume differences).
	GetDiffBestBid() *PriceLevel
	// GetLastExecAt returns the timestamp when the latest update occurred on the exchange.
	GetLastExecAt() *time.Time
	// GetLastArrivedAt returns the timestamp when the latest update arrived locally.
	GetLastArrivedAt() *time.Time
	// GetTickSize returns the minimum price increment for the symbol.
	GetTickSize() decimal.Decimal
	// GetMinOrderQty returns the minimum order quantity for the symbol.
	GetMinOrderQty() decimal.Decimal
	// RoundToTickSize rounds the given price down to the nearest valid tick.
	RoundToTickSize(price decimal.Decimal) decimal.Decimal
	// SetUpdateCallback registers a callback invoked on each quote update.
	SetUpdateCallback(callback func())
}

// OrderBook is the consumer-facing interface for order book data.
// It excludes UpdateByOBData which is only used by internal gateways.
type OrderBook interface {
	Quote
	// SumVolume returns the cumulative volume from the best price up to the given price on the specified book side.
	SumVolume(bookSide domains.BookSide, price decimal.Decimal) decimal.Decimal
	// AvgExecPrice returns the average execution price to fill the given quantity on the specified book side.
	AvgExecPrice(bookSide domains.BookSide, qty decimal.Decimal) decimal.Decimal
	// AvgExecPriceBySide returns the average execution price to fill the given quantity for the specified order side.
	AvgExecPriceBySide(side domains.Side, qty decimal.Decimal) decimal.Decimal
	// CalculateBidsVolSumMap returns a PriceLevelMap of cumulative bid volumes.
	CalculateBidsVolSumMap() *PriceLevelMap
	// CalculateAsksVolSumMap returns a PriceLevelMap of cumulative ask volumes.
	CalculateAsksVolSumMap() *PriceLevelMap
	// GetAsks returns the top N ask price levels in ascending price order (best first).
	GetAsks(depth int) []PriceLevel
	// GetBids returns the top N bid price levels in descending price order (best first).
	GetBids(depth int) []PriceLevel
	// SetDeferUpdateCallBack registers a deferred callback invoked on order book updates, separate from the regular update callbacks.
	SetDeferUpdateCallBack(callback func())
	// DropDeferUpdateCallBack removes the deferred update callback.
	DropDeferUpdateCallBack()
}

// Trade is the consumer-facing interface for trade data.
// It excludes Update which is only used by internal gateways.
type Trade interface {
	// IsReady returns true when the initial trade data has been received.
	IsReady() bool
	// SetUpdateCallback registers a callback invoked on each trade update.
	// msg contains the latest execution data received from the exchange.
	SetUpdateCallback(callback func(msg TradeDataList))
}

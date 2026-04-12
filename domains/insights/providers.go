package insights

import (
	"time"

	"github.com/shopspring/decimal"
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

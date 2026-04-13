package domains

import "github.com/shopspring/decimal"

// Side is a type for side of order. (Sell or Buy)
type Side int

const (
	SideNone Side = iota
	SideBuy
	SideSell
)

func (s Side) String() string {
	switch s {
	case SideBuy:
		return "BUY"
	case SideSell:
		return "SELL"
	default:
		return "NONE"
	}
}

func (s Side) Opposite() Side {
	switch s {
	case SideBuy:
		return SideSell
	case SideSell:
		return SideBuy
	default:
		return SideNone
	}
}

func (s Side) Sign() int {
	switch s {
	default:
		return 0
	case SideBuy:
		return 1
	case SideSell:
		return -1
	}
}
func (s Side) DecimalSign() decimal.Decimal {
	return decimal.NewFromInt(int64(s.Sign()))
}

type BookSide int

const (
	BookSideNone BookSide = iota
	BookSideBid
	BookSideAsk
)

type OrderType int

const (
	OrderTypeUnknown OrderType = iota
	OrderTypeMarket
	OrderTypeLimit
	OrderTypeLimitFOK
	OrderTypeLimitIOC
	OrderTypeLimitMaker
)

func (o OrderType) IsTaker() bool {
	return o == OrderTypeMarket || o == OrderTypeLimitFOK || o == OrderTypeLimitIOC
}

func (o OrderType) IsMaker() bool {
	return o == OrderTypeLimitMaker || o == OrderTypeLimit
}

func (o OrderType) IsAmenable() bool {
	return o == OrderTypeLimitMaker || o == OrderTypeLimit
}

func (o OrderType) IsPricable() bool {
	return o == OrderTypeLimit ||
		o == OrderTypeLimitFOK ||
		o == OrderTypeLimitIOC ||
		o == OrderTypeLimitMaker
}

func (o OrderType) String() string {
	switch o {
	case OrderTypeMarket:
		return "MARKET"
	case OrderTypeLimit:
		return "LIMIT"
	case OrderTypeLimitFOK: // 今の所 mexc のみ対応
		return "LIMIT_FOK"
	case OrderTypeLimitIOC: // 今の所 mexc のみ対応
		return "LIMIT_IOC"
	case OrderTypeLimitMaker:
		return "LIMIT_MAKER"
	default:
		return "UNKNOWN"
	}
}

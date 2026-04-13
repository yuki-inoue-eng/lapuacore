package deals

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type PositionMode int

const (
	PositionModeUnknown PositionMode = iota
	PositionModeOneWay
	PositionModeHedgeOfBuySide
	PositionModeHedgeOfSellSide
)

type PositionData struct {
	Timestamp    time.Time
	PositionMode PositionMode
	Side         domains.Side
	Qty          decimal.Decimal
}

package translators

import (
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
)

type PositionMsgTranslator struct {
	sideTranslator         *SideTranslator
	positionModeTranslator *PositionModeTranslator
}

func NewPositionMsgTranslator() *PositionMsgTranslator {
	return &PositionMsgTranslator{
		sideTranslator:         NewSideTranslator(),
		positionModeTranslator: NewPositionModeTranslator(),
	}
}

func (t *PositionMsgTranslator) TranslateToData(msg *dtos.PositionDataDto) (*deals.PositionData, error) {
	qty, err := decimal.NewFromString(msg.Qty)
	if err != nil {
		return nil, err
	}

	unixUpdatedAt, err := strconv.ParseInt(msg.UpdatedTimeMs, 10, 64)
	if err != nil {
		return nil, err
	}

	return &deals.PositionData{
		Timestamp:    time.UnixMilli(unixUpdatedAt),
		PositionMode: t.positionModeTranslator.Translate(msg.PositionIdx),
		Side:         t.sideTranslator.Translate(msg.Side),
		Qty:          qty,
	}, nil
}

type PositionModeTranslator struct{}

func NewPositionModeTranslator() *PositionModeTranslator {
	return &PositionModeTranslator{}
}

// Translate converts Bybit positionIdx to domain PositionMode.
// see: https://bybit-exchange.github.io/docs/v5/enum#positionidx
func (t *PositionModeTranslator) Translate(mode int) deals.PositionMode {
	switch mode {
	case 0:
		return deals.PositionModeOneWay
	case 1:
		return deals.PositionModeHedgeOfBuySide
	case 2:
		return deals.PositionModeHedgeOfSellSide
	default:
		return deals.PositionModeUnknown
	}
}

package translators

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type PositionMsgTranslator struct {
	sideTranslator *sideTranslator
}

func NewPositionMsgTranslator() *PositionMsgTranslator {
	return &PositionMsgTranslator{
		sideTranslator: newSideTranslator(),
	}
}

func (t *PositionMsgTranslator) TranslateToData(msg *dtos.PositionDataDto) (*deals.PositionData, error) {
	qty, err := decimal.NewFromString(msg.Position.OpenInterest)
	if err != nil {
		return nil, err
	}

	return &deals.PositionData{
		Timestamp:    time.UnixMilli(msg.Position.UpdatedAt),
		PositionMode: deals.PositionModeOneWay,
		Qty:          qty,
		Side:         t.sideTranslator.TranslateFromLongShort(msg.Position.Side),
	}, nil
}

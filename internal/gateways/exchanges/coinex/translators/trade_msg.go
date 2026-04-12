package translators

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type TradeMsgTranslator struct {
	sideTranslator *sideTranslator
}

func NewTradeMsgTranslator() *TradeMsgTranslator {
	return &TradeMsgTranslator{
		sideTranslator: newSideTranslator(),
	}
}

func (t *TradeMsgTranslator) TranslateToDatas(
	timestamp time.Time,
	msg *dtos.TradeMsgDto,
) ([]*insights.TradeData, error) {
	var tradeDatas []*insights.TradeData
	for _, d := range msg.Data.Dtos {
		tradeData, err := t.translateToData(timestamp, d)
		if err != nil {
			return nil, err
		}
		tradeDatas = append(tradeDatas, tradeData)
	}
	return tradeDatas, nil
}

func (t *TradeMsgTranslator) translateToData(
	timestamp time.Time,
	msg *dtos.TradeDataDto,
) (*insights.TradeData, error) {
	volume, err := decimal.NewFromString(msg.Volume)
	if err != nil {
		return nil, err
	}

	price, err := decimal.NewFromString(msg.Price)
	if err != nil {
		return nil, err
	}

	return &insights.TradeData{
		ExecAt:    time.UnixMilli(msg.TransactionTime),
		ArrivedAt: timestamp,
		Side:      t.sideTranslator.Translate(msg.Side),
		Volume:    volume,
		Price:     price,
	}, nil
}

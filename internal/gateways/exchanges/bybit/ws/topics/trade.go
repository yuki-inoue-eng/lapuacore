package topics

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/translators"
)

type TradeTopic struct {
	topicBase
	name               string
	symbol             *domains.Symbol
	tradeMsgTranslator *translators.TradeMsgTranslator
	dataHandlers       []insights.TradeDataHandler
}

func NewTradeTopic(symbol *domains.Symbol) *TradeTopic {
	return &TradeTopic{
		name:               fmt.Sprintf("publicTrade.%s", symbol.Name()),
		symbol:             symbol,
		tradeMsgTranslator: translators.NewTradeMsgTranslator(),
	}
}

func (t *TradeTopic) SetHandler(handler insights.TradeDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *TradeTopic) TopicName() string {
	return t.name
}

func (t *TradeTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	dto := &dtos.TradeMsgDto{}
	if err := json.Unmarshal(rawMsg, &dto); err != nil {
		return err
	}

	datas, err := t.tradeMsgTranslator.TranslateDatas(*ts, dto)
	if err != nil {
		return nil
	}

	for _, h := range t.dataHandlers {
		h(datas)
	}

	return nil
}

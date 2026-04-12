package topics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/translators"
)

type TradeTopic struct {
	name               string
	symbolName         string
	tradeMsgTranslator *translators.TradeMsgTranslator
	dataHandlers       []insights.TradeDataHandler
	msgID              string
}

func NewTradeTopic(symbolName string) *TradeTopic {
	return &TradeTopic{
		name:               fmt.Sprintf("trade@%s", symbolName),
		symbolName:         symbolName,
		tradeMsgTranslator: translators.NewTradeMsgTranslator(),
	}
}

func (t *TradeTopic) SetHandler(handler insights.TradeDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *TradeTopic) TopicName() string {
	return t.name
}

func (t *TradeTopic) SubscribeMsgID() string {
	return t.msgID
}

func (t *TradeTopic) SubscribeMsg() []byte {
	id := genMsgID()
	t.msgID = strconv.FormatInt(id, 10)
	return []byte(fmt.Sprintf(`{"method":"deals.subscribe","params":{"market_list":["%s"]},"id":%d}`, t.symbolName, id))
}

func (t *TradeTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	dto := &dtos.TradeMsgDto{}
	if err := json.Unmarshal(rawMsg, dto); err != nil {
		return err
	}

	datas, err := t.tradeMsgTranslator.TranslateToDatas(*ts, dto)
	if err != nil {
		return err
	}

	for _, h := range t.dataHandlers {
		h(datas)
	}

	return nil
}

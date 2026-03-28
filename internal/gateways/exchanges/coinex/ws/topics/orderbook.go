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

type OrderBookTopic struct {
	name            string
	symbolName      string
	obMsgTranslator *translators.OrderBookMsgTranslator
	dataHandlers    []insights.OrderBookDataHandler
	msgID           string
}

func NewOrderBookTopic(symbolName string) *OrderBookTopic {
	return &OrderBookTopic{
		name:            fmt.Sprintf("orderBook@%s", symbolName),
		symbolName:      symbolName,
		obMsgTranslator: translators.NewOrderBookMsgTranslator(),
	}
}

func (t *OrderBookTopic) SetHandler(handler insights.OrderBookDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *OrderBookTopic) TopicName() string {
	return t.name
}

func (t *OrderBookTopic) SubscribeMsgID() string {
	return t.msgID
}

func (t *OrderBookTopic) SubscribeMsg() []byte {
	id := genMsgID()
	t.msgID = strconv.FormatInt(id, 10)
	return []byte(fmt.Sprintf(`{"method":"depth.subscribe","params":{"market_list":[["%s",50,"0",true]]},"id":%d}`, t.symbolName, id))
}

func (t *OrderBookTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	dto := &dtos.OrderBookMsgDto{}
	if err := json.Unmarshal(rawMsg, dto); err != nil {
		return err
	}
	data, err := t.obMsgTranslator.Translate(*ts, dto)
	if err != nil {
		return err
	}
	for _, h := range t.dataHandlers {
		h(data)
	}
	return nil
}
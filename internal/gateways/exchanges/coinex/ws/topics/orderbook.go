package topics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/translators"
)

type OrderBookTopic struct {
	name            string
	symbol          *domains.Symbol
	obMsgTranslator *translators.OrderBookMsgTranslator
	dataHandlers    []insights.OrderBookDataHandler
	msgID           string
}

func NewOrderBookTopic(symbol *domains.Symbol) *OrderBookTopic {
	return &OrderBookTopic{
		name:            fmt.Sprintf("orderBook@%s", symbol.Name()),
		symbol:          symbol,
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
	return []byte(fmt.Sprintf(`{"method":"depth.subscribe","params":{"market_list":[["%s",50,"0",true]]},"id":%d}`, t.symbol.Name(), id))
}

// MessageID returns depth.updated_at as a unique identifier.
func (t *OrderBookTopic) MessageID(rawMsg []byte) string {
	msg := struct {
		Data struct {
			Depth struct {
				UpdatedAt int64 `json:"updated_at"`
			} `json:"depth"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil || msg.Data.Depth.UpdatedAt == 0 {
		return ""
	}
	return fmt.Sprintf("ob:%d", msg.Data.Depth.UpdatedAt)
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
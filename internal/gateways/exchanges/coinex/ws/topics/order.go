package topics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/translators"
)

type OrderTopic struct {
	name               string
	symbol             *domains.Symbol
	orderMsgTranslator *translators.OrderMsgTranslator
	dataHandlers       []deals.OrderDataHandler

	msgID string
}

func NewOrderTopic(symbol *domains.Symbol) *OrderTopic {
	return &OrderTopic{
		name:               fmt.Sprintf("personal.order@%s", symbol.Name()),
		symbol:             symbol,
		orderMsgTranslator: translators.NewOrderMsgTranslator(),
	}
}

// SetHandler registers a callback to be called when an order update is received.
func (t *OrderTopic) SetHandler(handler deals.OrderDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *OrderTopic) TopicName() string {
	return t.name
}

func (t *OrderTopic) SubscribeMsgID() string {
	return t.msgID
}

func (t *OrderTopic) SubscribeMsg() []byte {
	msgID := genMsgID()
	t.msgID = strconv.FormatInt(msgID, 10)

	return []byte(fmt.Sprintf(`{
		"method": "order.subscribe",
		"params": {
			"market_list": ["%s"]
		},
		"id": %d
	}`, t.symbol.Name(), msgID))
}

// MessageID returns order_id:updated_at as a unique identifier.
func (t *OrderTopic) MessageID(rawMsg []byte) string {
	msg := struct {
		Data struct {
			Order struct {
				OrderID   int64 `json:"order_id"`
				UpdatedAt int64 `json:"updated_at"`
			} `json:"order"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil || msg.Data.Order.OrderID == 0 {
		return ""
	}
	return fmt.Sprintf("order:%d:%d", msg.Data.Order.OrderID, msg.Data.Order.UpdatedAt)
}

func (t *OrderTopic) MsgHandler(timestamp *time.Time, rawMsg []byte) error {
	msgDto := &dtos.OrderMsgDto{}
	if err := json.Unmarshal(rawMsg, msgDto); err != nil {
		return err
	}
	data, err := t.orderMsgTranslator.TranslateToData(timestamp, msgDto.Data)
	if err != nil {
		return err
	}
	for _, h := range t.dataHandlers {
		h([]*deals.OrderData{data})
	}
	return nil
}

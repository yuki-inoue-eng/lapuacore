package topics

import (
	"encoding/json"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/translators"
)

type OrderTopic struct {
	handlersMap        map[*domains.Symbol][]deals.OrderDataHandler
	orderMsgTranslator *translators.OrderMsgTranslator
}

func NewOrderTopic() *OrderTopic {
	return &OrderTopic{
		handlersMap:        map[*domains.Symbol][]deals.OrderDataHandler{},
		orderMsgTranslator: translators.NewOrderMsgTranslator(),
	}
}

// SetHandlers registers a handler for order updates of the given symbol.
func (t *OrderTopic) SetHandlers(symbol *domains.Symbol, handler deals.OrderDataHandler) {
	if handlers, ok := t.handlersMap[symbol]; ok {
		t.handlersMap[symbol] = append(handlers, handler)
	} else {
		t.handlersMap[symbol] = []deals.OrderDataHandler{handler}
	}
}

func (t *OrderTopic) TopicName() string {
	const tName = "order"
	return tName
}

func (t *OrderTopic) MsgHandler(timestamp *time.Time, rawMsg []byte) error {
	msgDto := &dtos.OrderMsgDto{}
	if err := json.Unmarshal(rawMsg, &msgDto); err != nil {
		return err
	}

	datasBySymbol := map[*domains.Symbol][]*deals.OrderData{}
	for _, dataDto := range msgDto.DataDto {
		data, err := t.orderMsgTranslator.TranslateToData(timestamp, dataDto)
		if err != nil {
			return err
		}

		symbol := domains.GetSymbol("Bybit_" + dataDto.Category + "_" + dataDto.Symbol)
		if datas, ok := datasBySymbol[symbol]; ok {
			datasBySymbol[symbol] = append(datas, data)
		} else {
			datasBySymbol[symbol] = []*deals.OrderData{data}
		}
	}

	for symbol, datas := range datasBySymbol {
		if handlers, ok := t.handlersMap[symbol]; ok {
			for _, h := range handlers {
				h(datas)
			}
		}
	}
	return nil
}

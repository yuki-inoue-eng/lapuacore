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

type OBDepth int

const (
	LinearOBDepth1   OBDepth = 1
	LinearOBDepth50  OBDepth = 50
	LinearOBDepth200 OBDepth = 200
	LinearOBDepth500 OBDepth = 500

	InverseOBDepth1   OBDepth = 1
	InverseOBDepth50  OBDepth = 50
	InverseOBDepth200 OBDepth = 200
	InverseOBDepth500 OBDepth = 500

	SpotOBDepth1   OBDepth = 1
	SpotOBDepth50  OBDepth = 50
	SpotOBDepth200 OBDepth = 200

	OptionOBDepth50  OBDepth = 50
	OptionOBDepth100 OBDepth = 100
)

type OrderBookTopic struct {
	topicBase
	name            string
	depth           OBDepth
	symbol          *domains.Symbol
	obMsgTranslator *translators.OrderBookMsgTranslator
	dataHandlers    []insights.OrderBookDataHandler
}

func NewOrderBookTopic(symbol *domains.Symbol, depth OBDepth) *OrderBookTopic {
	return &OrderBookTopic{
		name:            fmt.Sprintf("orderbook.%d.%s", depth, symbol.Name()),
		symbol:          symbol,
		depth:           depth,
		obMsgTranslator: translators.NewOrderBookMsgTranslator(),
	}
}

func (t *OrderBookTopic) SetHandler(handler insights.OrderBookDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *OrderBookTopic) TopicName() string {
	return t.name
}

func (t *OrderBookTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	dto := &dtos.OrderBookMsgDto{}
	if err := json.Unmarshal(rawMsg, &dto); err != nil {
		return nil
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

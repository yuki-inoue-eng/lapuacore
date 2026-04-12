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

type BookTickerTopic struct {
	name            string
	symbolName      string
	btMsgTranslator *translators.BookTickerMsgTranslator
	dataHandlers    []insights.BookTickerDataHandler
	msgID           string
}

func NewBookTickerTopic(symbolName string) *BookTickerTopic {
	return &BookTickerTopic{
		name:            fmt.Sprintf("bookTicker@%s", symbolName),
		symbolName:      symbolName,
		btMsgTranslator: translators.NewBookTickerMsgTranslator(),
	}
}

func (t *BookTickerTopic) SetHandler(handler insights.BookTickerDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *BookTickerTopic) TopicName() string {
	return t.name
}

func (t *BookTickerTopic) SubscribeMsgID() string {
	return t.msgID
}

func (t *BookTickerTopic) SubscribeMsg() []byte {
	id := genMsgID()
	t.msgID = strconv.FormatInt(id, 10)
	return []byte(fmt.Sprintf(`{"method":"bbo.subscribe","params":{"market_list":["%s"]},"id":%d}`, t.symbolName, id))
}

func (t *BookTickerTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	dto := &dtos.BookTickerMsgDto{}
	if err := json.Unmarshal(rawMsg, dto); err != nil {
		return err
	}
	data, err := t.btMsgTranslator.Translate(*ts, dto)
	if err != nil {
		return err
	}
	for _, h := range t.dataHandlers {
		h(data)
	}
	return nil
}

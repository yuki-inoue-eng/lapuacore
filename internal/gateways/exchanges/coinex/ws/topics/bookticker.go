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

type BookTickerTopic struct {
	name            string
	symbol          *domains.Symbol
	btMsgTranslator *translators.BookTickerMsgTranslator
	dataHandlers    []insights.BookTickerDataHandler
	msgID           string
}

func NewBookTickerTopic(symbol *domains.Symbol) *BookTickerTopic {
	return &BookTickerTopic{
		name:            fmt.Sprintf("bookTicker@%s", symbol.Name()),
		symbol:          symbol,
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
	return []byte(fmt.Sprintf(`{"method":"bbo.subscribe","params":{"market_list":["%s"]},"id":%d}`, t.symbol.Name(), id))
}

// MessageID returns updated_at as a unique identifier.
func (t *BookTickerTopic) MessageID(rawMsg []byte) string {
	msg := struct {
		Data struct {
			UpdatedAt int64 `json:"updated_at"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil || msg.Data.UpdatedAt == 0 {
		return ""
	}
	return fmt.Sprintf("bbo:%d", msg.Data.UpdatedAt)
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

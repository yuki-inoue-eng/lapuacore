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

type PositionTopic struct {
	name                  string
	symbol                *domains.Symbol
	positionMsgTranslator *translators.PositionMsgTranslator
	dataHandlers          []deals.PositionDataHandler
	msgID                 string
}

func NewPositionTopic(symbol *domains.Symbol) *PositionTopic {
	return &PositionTopic{
		name:                  fmt.Sprintf("personal.position@%s", symbol.Name()),
		symbol:                symbol,
		positionMsgTranslator: translators.NewPositionMsgTranslator(),
	}
}

func (t *PositionTopic) SetHandler(handler deals.PositionDataHandler) {
	t.dataHandlers = append(t.dataHandlers, handler)
}

func (t *PositionTopic) TopicName() string {
	return t.name
}

func (t *PositionTopic) SubscribeMsgID() string {
	return t.msgID
}

func (t *PositionTopic) SubscribeMsg() []byte {
	id := genMsgID()
	t.msgID = strconv.FormatInt(id, 10)
	return []byte(fmt.Sprintf(`{"method":"position.subscribe","params":{"market_list":["%s"]},"id":%d}`, t.symbol.Name(), id))
}

// MessageID returns market:side:updated_at as a unique identifier.
func (t *PositionTopic) MessageID(rawMsg []byte) string {
	msg := struct {
		Data struct {
			Position struct {
				Market    string `json:"market"`
				Side      string `json:"side"`
				UpdatedAt int64  `json:"updated_at"`
			} `json:"position"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil || msg.Data.Position.Market == "" {
		return ""
	}
	return fmt.Sprintf("pos:%s:%s:%d", msg.Data.Position.Market, msg.Data.Position.Side, msg.Data.Position.UpdatedAt)
}

func (t *PositionTopic) MsgHandler(ts *time.Time, rawMsg []byte) error {
	msgDto := &dtos.PositionMsgDto{}
	if err := json.Unmarshal(rawMsg, msgDto); err != nil {
		return err
	}

	data, err := t.positionMsgTranslator.TranslateToData(msgDto.Data)
	if err != nil {
		return err
	}

	for _, h := range t.dataHandlers {
		h([]*deals.PositionData{data})
	}

	return nil
}

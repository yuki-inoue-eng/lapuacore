package topics

import (
	"encoding/json"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/translators"
)

type PositionTopic struct {
	topicBase
	handlersMap      map[*domains.Symbol][]deals.PositionDataHandler
	posMsgTranslator *translators.PositionMsgTranslator
}

func NewPositionTopic() *PositionTopic {
	return &PositionTopic{
		handlersMap:      map[*domains.Symbol][]deals.PositionDataHandler{},
		posMsgTranslator: translators.NewPositionMsgTranslator(),
	}
}

// SetHandlers registers a handler for position updates of the given symbol.
func (t *PositionTopic) SetHandlers(symbol *domains.Symbol, handler deals.PositionDataHandler) {
	if handlers, ok := t.handlersMap[symbol]; ok {
		t.handlersMap[symbol] = append(handlers, handler)
	} else {
		t.handlersMap[symbol] = []deals.PositionDataHandler{handler}
	}
}

func (t *PositionTopic) TopicName() string {
	const tName = "position"
	return tName
}

func (t *PositionTopic) MsgHandler(_ *time.Time, rawMsg []byte) error {
	msgDto := &dtos.PositionMsgDto{}
	if err := json.Unmarshal(rawMsg, &msgDto); err != nil {
		return err
	}

	datasBySymbol := map[*domains.Symbol][]*deals.PositionData{}
	for _, dataDto := range msgDto.DataDto {
		data, err := t.posMsgTranslator.TranslateToData(dataDto)
		if err != nil {
			return err
		}

		symbol := domains.GetSymbol("Bybit_" + dataDto.Category + "_" + dataDto.Symbol)
		if datas, ok := datasBySymbol[symbol]; ok {
			datasBySymbol[symbol] = append(datas, data)
		} else {
			datasBySymbol[symbol] = []*deals.PositionData{data}
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

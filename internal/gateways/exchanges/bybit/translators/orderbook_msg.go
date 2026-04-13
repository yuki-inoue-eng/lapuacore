package translators

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
)

type OrderBookMsgTranslator struct{}

func NewOrderBookMsgTranslator() *OrderBookMsgTranslator {
	return &OrderBookMsgTranslator{}
}

func (t *OrderBookMsgTranslator) Translate(arrivedAt time.Time, msg *dtos.OrderBookMsgDto) (*insights.OrderBookData, error) {
	seqID := msg.Data.UpdateID
	asks, err := t.translateRecords(seqID, msg.Data.Asks)
	if err != nil {
		return nil, err
	}
	bids, err := t.translateRecords(seqID, msg.Data.Bids)
	if err != nil {
		return nil, err
	}

	model := &insights.OrderBookData{
		Type:      toMessageType(msg.Type).translate(),
		ExecAt:    time.UnixMilli(msg.Ts),
		ArrivedAt: arrivedAt,
		Asks:      asks,
		Bids:      bids,
		SeqID:     seqID,
	}
	return model, nil
}

func (t *OrderBookMsgTranslator) translateRecords(seqID int64, records [][]string) ([]insights.OBRecord, error) {
	var result []insights.OBRecord
	for _, r := range records {
		p, err := decimal.NewFromString(r[0])
		if err != nil {
			return nil, err
		}
		v, err := decimal.NewFromString(r[1])
		if err != nil {
			return nil, err
		}

		result = append(result, insights.OBRecord{
			SeqID:  seqID,
			Price:  p,
			Volume: v,
		})
	}
	return result, nil
}

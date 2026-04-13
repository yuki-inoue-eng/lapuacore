package translators

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type OrderBookMsgTranslator struct{}

func NewOrderBookMsgTranslator() *OrderBookMsgTranslator {
	return &OrderBookMsgTranslator{}
}

func (t *OrderBookMsgTranslator) Translate(
	arrivedAt time.Time,
	msg *dtos.OrderBookMsgDto,
) (*insights.OrderBookData, error) {
	seqID := msg.Data.Depth.UpdatedAt
	dataType := insights.DataTypeDelta
	if msg.Data.IsSnapshot {
		dataType = insights.DataTypeSnapshot
	}

	asks, err := t.translateRecords(seqID, msg.Data.Depth.Asks)
	if err != nil {
		return nil, err
	}
	bids, err := t.translateRecords(seqID, msg.Data.Depth.Bids)
	if err != nil {
		return nil, err
	}

	return &insights.OrderBookData{
		Type:      dataType,
		ExecAt:    time.UnixMilli(msg.Data.Depth.UpdatedAt),
		ArrivedAt: arrivedAt,
		Asks:      asks,
		Bids:      bids,
		SeqID:     seqID,
	}, nil
}

func (t *OrderBookMsgTranslator) translateRecords(
	seqID int64,
	records [][]string,
) ([]insights.PriceLevel, error) {
	result := make([]insights.PriceLevel, 0, len(records))
	for _, r := range records {
		price, err := decimal.NewFromString(r[0])
		if err != nil {
			return nil, err
		}
		volume, err := decimal.NewFromString(r[1])
		if err != nil {
			return nil, err
		}
		result = append(result, insights.PriceLevel{
			SeqID:  seqID,
			Price:  price,
			Volume: volume,
		})
	}
	return result, nil
}

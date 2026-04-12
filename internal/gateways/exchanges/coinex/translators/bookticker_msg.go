package translators

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type BookTickerMsgTranslator struct{}

func NewBookTickerMsgTranslator() *BookTickerMsgTranslator {
	return &BookTickerMsgTranslator{}
}

func (t *BookTickerMsgTranslator) Translate(
	arrivedAt time.Time,
	msg *dtos.BookTickerMsgDto,
) (*insights.BookTickerData, error) {
	seqID := msg.Data.Ts
	ts := time.UnixMilli(msg.Data.Ts)
	asks, err := t.translateRecord(seqID, msg.Data.AskPrice, msg.Data.AskSize)
	if err != nil {
		return nil, err
	}
	bids, err := t.translateRecord(seqID, msg.Data.BidPrice, msg.Data.BidSize)
	if err != nil {
		return nil, err
	}
	return &insights.BookTickerData{
		SeqID:     seqID,
		ExecAt:    ts,
		EventAt:   ts,
		ArrivedAt: arrivedAt,
		BestAsk:   asks,
		BestBid:   bids,
	}, nil
}

func (t *BookTickerMsgTranslator) translateRecord(
	seqID int64,
	price, size string,
) (*insights.OBRecord, error) {
	p, err := decimal.NewFromString(price)
	if err != nil {
		return nil, err
	}
	s, err := decimal.NewFromString(size)
	if err != nil {
		return nil, err
	}
	return &insights.OBRecord{
		SeqID:  seqID,
		Price:  p,
		Volume: s,
	}, nil
}

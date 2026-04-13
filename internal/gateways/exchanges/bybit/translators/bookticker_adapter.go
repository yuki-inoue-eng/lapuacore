package translators

import (
	"fmt"

	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
)

// BookTickerAdapter converts OrderBookData (depth=1) into BookTickerData.
type BookTickerAdapter struct{}

func NewBookTickerAdapter() *BookTickerAdapter {
	return &BookTickerAdapter{}
}

func (a *BookTickerAdapter) Convert(ob *insights.OrderBookData) (*insights.BookTickerData, error) {
	if len(ob.Asks) == 0 || len(ob.Bids) == 0 {
		return nil, fmt.Errorf("orderbook depth=1 has no asks or bids")
	}
	bestAsk := &ob.Asks[0]
	bestBid := &ob.Bids[0]
	return &insights.BookTickerData{
		SeqID:     ob.SeqID,
		ExecAt:    ob.ExecAt,
		EventAt:   ob.ExecAt,
		ArrivedAt: ob.ArrivedAt,
		BestAsk:   bestAsk,
		BestBid:   bestBid,
	}, nil
}

//go:build integration

package ws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

func TestTradeStream(t *testing.T) {
	ch := ws.NewPublicChannel(nil)

	tradeTopic := topics.NewTradeTopic(domains.SymbolCoinExFuturesBTCUSDT.Name())
	tradeTopic.SetHandler(func(msg insights.TradeDataList) {
		for _, d := range msg {
			fmt.Printf("[%s] side=%s price=%s volume=%s execAt=%s\n",
				d.ArrivedAt.Format("15:04:05.000"), d.Side, d.Price.String(), d.Volume.String(),
				d.ExecAt.Format("15:04:05.000"))
		}
	})
	ch.SetTopics([]topics.Topic{tradeTopic})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("channel error: %v", err)
	}
}

func TestOrderBookStream(t *testing.T) {
	ch := ws.NewPublicChannel(nil)

	obTopic := topics.NewOrderBookTopic(domains.SymbolCoinExFuturesBTCUSDT.Name())
	obTopic.SetHandler(func(data *insights.OrderBookData) {
		bestAsk := "N/A"
		bestBid := "N/A"
		if len(data.Asks) > 0 {
			bestAsk = data.Asks[0].Price.String()
		}
		if len(data.Bids) > 0 {
			bestBid = data.Bids[0].Price.String()
		}
		fmt.Printf("[%s] type=%v seqID=%d bestAsk=%s bestBid=%s\n",
			data.ArrivedAt.Format("15:04:05.000"), data.Type, data.SeqID, bestAsk, bestBid)
	})
	ch.SetTopics([]topics.Topic{obTopic})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("channel error: %v", err)
	}
}

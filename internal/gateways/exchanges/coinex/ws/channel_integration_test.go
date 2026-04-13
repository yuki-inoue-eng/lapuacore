//go:build integration

package ws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

func TestTradeStream(t *testing.T) {
	ch := ws.NewPublicChannel(nil)

	mg := topics.NewManager()
	tradeTopic := topics.NewTradeTopic(domains.SymbolCoinExFuturesBTCUSDT)
	tradeTopic.SetHandler(func(msg insights.TradeDataList) {
		for _, d := range msg {
			fmt.Printf("[%s] side=%s price=%s volume=%s execAt=%s\n",
				d.ArrivedAt.Format("15:04:05.000"), d.Side, d.Price.String(), d.Volume.String(),
				d.ExecAt.Format("15:04:05.000"))
		}
	})
	mg.SetTopics([]gateways.Topic{tradeTopic})
	ch.SetTopicMg(mg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("channel error: %v", err)
	}
}

func TestOrderBookStream(t *testing.T) {
	ch := ws.NewPublicChannel(nil)

	mg := topics.NewManager()
	obTopic := topics.NewOrderBookTopic(domains.SymbolCoinExFuturesBTCUSDT)
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
	mg.SetTopics([]gateways.Topic{obTopic})
	ch.SetTopicMg(mg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ch.Start(ctx); err != nil {
		t.Fatalf("channel error: %v", err)
	}
}

func TestChannelGroupAllTopics(t *testing.T) {
	var tradeCount, obCount, btCount int64
	group := gateways.NewChannelGroup(
		3,
		func() *gateways.Channel { return ws.NewPublicChannel(nil) },
		func() gateways.TopicManager { return topics.NewManager() },
		10*time.Second,
	)

	tradeTopic := topics.NewTradeTopic(domains.SymbolCoinExFuturesBTCUSDT)
	tradeTopic.SetHandler(func(msg insights.TradeDataList) {
		for _, d := range msg {
			tradeCount++
			fmt.Printf("[trade] #%d [%s] side=%s price=%s volume=%s execAt=%s\n",
				tradeCount, d.ArrivedAt.Format("15:04:05.000"), d.Side, d.Price.String(), d.Volume.String(),
				d.ExecAt.Format("15:04:05.000"))
		}
	})

	obTopic := topics.NewOrderBookTopic(domains.SymbolCoinExFuturesBTCUSDT)
	obTopic.SetHandler(func(data *insights.OrderBookData) {
		obCount++
		bestAsk := "N/A"
		bestBid := "N/A"
		if len(data.Asks) > 0 {
			bestAsk = data.Asks[0].Price.String()
		}
		if len(data.Bids) > 0 {
			bestBid = data.Bids[0].Price.String()
		}
		fmt.Printf("[ob]    #%d [%s] type=%v seqID=%d bestAsk=%s bestBid=%s\n",
			obCount, data.ArrivedAt.Format("15:04:05.000"), data.Type, data.SeqID, bestAsk, bestBid)
	})

	btTopic := topics.NewBookTickerTopic(domains.SymbolCoinExFuturesBTCUSDT)
	btTopic.SetHandler(func(data *insights.BookTickerData) {
		btCount++
		bestAsk := "N/A"
		bestBid := "N/A"
		if data.BestAsk != nil {
			bestAsk = data.BestAsk.Price.String()
		}
		if data.BestBid != nil {
			bestBid = data.BestBid.Price.String()
		}
		fmt.Printf("[bt]    #%d [%s] bestAsk=%s bestBid=%s\n",
			btCount, data.ArrivedAt.Format("15:04:05.000"), bestAsk, bestBid)
	})

	group.SetTopics([]gateways.Topic{tradeTopic, obTopic, btTopic})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	group.Start(ctx)
	fmt.Printf("\ntotal: trade=%d orderbook=%d bookticker=%d\n", tradeCount, obCount, btCount)
}

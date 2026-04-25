//go:build integration

package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/bybit"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

// TestBybitInsightsFlow verifies the full Bybit initialization flow:
// InitAndStartNoopMode -> InitGatewayManager -> InitInsights -> StartGateway
// then checks that insights become ready with valid market data.
func TestBybitInsightsFlow(t *testing.T) {
	lapua.InitAndStartNoopMode()
	defer lapua.Cancel(nil)

	symbol := domains.SymbolBybitLinearBTCUSDT

	bybit.InitGatewayManager(nil, 1)
	bybit.InitInsights(
		[]*domains.Symbol{symbol},
		[]*bybit.OrderBookDesignator{
			{Symbol: symbol, Depth: topics.LinearOBDepth50},
		},
		[]*domains.Symbol{symbol},
	)

	bybit.StartGateway()

	ctx, cancel := context.WithTimeout(lapua.Ctx, 30*time.Second)
	defer cancel()

	// Poll until insights are ready or timeout
	ready := waitForReady(ctx, func() bool {
		return bybit.Insights.IsEverythingReady()
	})
	if !ready {
		t.Fatal("Bybit insights did not become ready within timeout")
	}

	// Verify order book data
	ob := bybit.Insights.GetOrderBook(&bybit.OrderBookDesignator{
		Symbol: symbol,
		Depth:  topics.LinearOBDepth50,
	})
	if ob == nil {
		t.Fatal("expected non-nil OrderBook")
	}
	if ob.GetBestAsk() == nil {
		t.Fatal("expected non-nil BestAsk")
	}
	if ob.GetBestBid() == nil {
		t.Fatal("expected non-nil BestBid")
	}
	if !ob.GetBestAsk().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}
	if !ob.GetBestBid().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}
	if !ob.GetBestAsk().Price.GreaterThan(ob.GetBestBid().Price) {
		t.Errorf("got %v, want true", false)
	}

	// Verify quote data
	quote := bybit.Insights.GetQuote(symbol)
	if quote == nil {
		t.Fatal("expected non-nil Quote")
	}
	if !quote.GetBestAsk().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}
	if !quote.GetBestBid().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}

	// Verify trade data
	trade := bybit.Insights.GetTrade(symbol)
	if trade == nil {
		t.Fatal("expected non-nil Trade")
	}
	if !trade.IsReady() {
		t.Errorf("got %v, want true", false)
	}

	t.Logf("OrderBook bestAsk=%s bestBid=%s", ob.GetBestAsk().Price, ob.GetBestBid().Price)
	t.Logf("Quote bestAsk=%s bestBid=%s", quote.GetBestAsk().Price, quote.GetBestBid().Price)
}

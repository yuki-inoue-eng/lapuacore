//go:build integration

package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/bmizerany/assert"
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
	defer lapua.Cancel()

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
	assert.Equal(t, true, ready, "Bybit insights did not become ready within timeout")

	// Verify order book data
	ob := bybit.Insights.GetOrderBook(&bybit.OrderBookDesignator{
		Symbol: symbol,
		Depth:  topics.LinearOBDepth50,
	})
	assert.NotEqual(t, nil, ob)
	assert.NotEqual(t, nil, ob.GetBestAsk())
	assert.NotEqual(t, nil, ob.GetBestBid())
	assert.Equal(t, true, ob.GetBestAsk().Price.IsPositive())
	assert.Equal(t, true, ob.GetBestBid().Price.IsPositive())
	assert.Equal(t, true, ob.GetBestAsk().Price.GreaterThan(ob.GetBestBid().Price))

	// Verify quote data
	quote := bybit.Insights.GetQuote(symbol)
	assert.NotEqual(t, nil, quote)
	assert.Equal(t, true, quote.GetBestAsk().Price.IsPositive())
	assert.Equal(t, true, quote.GetBestBid().Price.IsPositive())

	// Verify trade data
	trade := bybit.Insights.GetTrade(symbol)
	assert.NotEqual(t, nil, trade)
	assert.Equal(t, true, trade.IsReady())

	t.Logf("OrderBook bestAsk=%s bestBid=%s", ob.GetBestAsk().Price, ob.GetBestBid().Price)
	t.Logf("Quote bestAsk=%s bestBid=%s", quote.GetBestAsk().Price, quote.GetBestBid().Price)
}

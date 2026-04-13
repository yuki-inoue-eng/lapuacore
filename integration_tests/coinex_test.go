//go:build integration

package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/coinex"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
)

// TestCoinExInsightsFlow verifies the full initialization flow:
// InitAndStartNoopMode -> InitGatewayManager -> InitInsights -> StartGateway
// then checks that insights become ready with valid market data.
func TestCoinExInsightsFlow(t *testing.T) {
	lapua.InitAndStartNoopMode()
	defer lapua.Cancel()

	symbol := domains.SymbolCoinExFuturesBTCUSDT

	coinex.InitGatewayManager(nil, 1)
	coinex.InitInsights(
		[]*domains.Symbol{symbol},
		[]*domains.Symbol{symbol},
		[]*domains.Symbol{symbol},
	)

	ctx, cancel := context.WithTimeout(lapua.Ctx, 30*time.Second)
	defer cancel()
	coinex.StartGateway(ctx)

	// Poll until insights are ready or timeout
	ready := waitForReady(ctx, func() bool {
		return coinex.Insights.IsEverythingReady()
	})
	assert.Equal(t, true, ready, "CoinEx insights did not become ready within timeout")

	// Verify market data
	ob := coinex.Insights.GetOrderBook(symbol)
	assert.NotEqual(t, nil, ob)
	assert.NotEqual(t, nil, ob.GetBestAsk())
	assert.NotEqual(t, nil, ob.GetBestBid())
	assert.Equal(t, true, ob.GetBestAsk().Price.IsPositive())
	assert.Equal(t, true, ob.GetBestBid().Price.IsPositive())
	assert.Equal(t, true, ob.GetBestAsk().Price.GreaterThan(ob.GetBestBid().Price))

	quote := coinex.Insights.GetQuote(symbol)
	assert.NotEqual(t, nil, quote)
	assert.Equal(t, true, quote.GetBestAsk().Price.IsPositive())
	assert.Equal(t, true, quote.GetBestBid().Price.IsPositive())

	trade := coinex.Insights.GetTrade(symbol)
	assert.NotEqual(t, nil, trade)
	assert.Equal(t, true, trade.IsReady())

	t.Logf("OrderBook bestAsk=%s bestBid=%s", ob.GetBestAsk().Price, ob.GetBestBid().Price)
	t.Logf("Quote bestAsk=%s bestBid=%s", quote.GetBestAsk().Price, quote.GetBestBid().Price)
}

func waitForReady(ctx context.Context, check func() bool) bool {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if check() {
				return true
			}
		}
	}
}

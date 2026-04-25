//go:build integration

package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/coinex"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
)

// TestCoinExInsightsFlow verifies the full initialization flow:
// InitAndStartNoopMode -> InitGatewayManager -> InitInsights -> StartGateway
// then checks that insights become ready with valid market data.
func TestCoinExInsightsFlow(t *testing.T) {
	lapua.InitAndStartNoopMode()
	defer lapua.Cancel(nil)

	symbol := domains.SymbolCoinExFuturesBTCUSDT

	coinex.InitGatewayManager(nil, 1)
	coinex.InitInsights(
		[]*domains.Symbol{symbol},
		[]*domains.Symbol{symbol},
		[]*domains.Symbol{symbol},
	)

	coinex.StartGateway()

	ctx, cancel := context.WithTimeout(lapua.Ctx, 30*time.Second)
	defer cancel()

	// Poll until insights are ready or timeout
	ready := waitForReady(ctx, func() bool {
		return coinex.Insights.IsEverythingReady()
	})
	if !ready {
		t.Fatal("CoinEx insights did not become ready within timeout")
	}

	// Verify market data
	ob := coinex.Insights.GetOrderBook(symbol)
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

	quote := coinex.Insights.GetQuote(symbol)
	if quote == nil {
		t.Fatal("expected non-nil Quote")
	}
	if !quote.GetBestAsk().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}
	if !quote.GetBestBid().Price.IsPositive() {
		t.Errorf("got %v, want true", false)
	}

	trade := coinex.Insights.GetTrade(symbol)
	if trade == nil {
		t.Fatal("expected non-nil Trade")
	}
	if !trade.IsReady() {
		t.Errorf("got %v, want true", false)
	}

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

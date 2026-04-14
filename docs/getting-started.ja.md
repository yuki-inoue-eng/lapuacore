# Getting Started

## Installation

```bash
go get github.com/yuki-inoue-eng/lapuacore
```

Go 1.26 以上が必要です。

## 初期化フロー

lapuacore の起動は以下の 4 ステップで行います。

1. **InitAndStartNoopMode** — コアコンテキストとメトリクス（noop）を初期化
2. **InitGatewayManager** — 取引所ゲートウェイの初期化（認証情報・接続数を指定）
3. **InitInsights** — マーケットデータの購読設定（Trade / OrderBook / Quote）
4. **StartGateway** — WebSocket 接続を開始し、データ受信を開始

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/coinex"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
)

func main() {
	// 1. Core initialization (noop mode: no real metrics export)
	lapua.InitAndStartNoopMode()
	defer lapua.Cancel()

	symbol := domains.SymbolCoinExFuturesBTCUSDT

	// 2. Initialize gateway (nil credentials = public data only, 1 connection)
	coinex.InitGatewayManager(nil, 1)

	// 3. Configure market data subscriptions
	coinex.InitInsights(
		[]*domains.Symbol{symbol}, // Trade
		[]*domains.Symbol{symbol}, // OrderBook
		[]*domains.Symbol{symbol}, // Quote
	)

	// 4. Start WebSocket connections
	ctx, cancel := context.WithTimeout(lapua.Ctx, 30*time.Second)
	defer cancel()
	coinex.StartGateway(ctx)

	// Wait until all insights are ready
	lapua.WaitForInsightsToBeReady()

	ob := coinex.Insights.GetOrderBook(symbol)
	fmt.Printf("BestAsk: %s  BestBid: %s\n",
		ob.GetBestAsk().Price, ob.GetBestBid().Price)
}
```

> `InitAndStartNoopMode` はメトリクスエクスポートを無効化したテスト・開発用モードです。本番環境では `InitAndStart` を使用し、設定ファイルを指定します。

## Insights の利用

### OrderBook

板情報を取得し、最良気配や累積出来高を算出できます。

```go
ob := coinex.Insights.GetOrderBook(symbol)

// Best bid/ask
bestAsk := ob.GetBestAsk() // *PriceLevel{Price, Qty}
bestBid := ob.GetBestBid()

// Cumulative volume up to a price
volume := ob.SumVolume(insights.BookSideAsk, targetPrice)

// Average execution price for a given quantity
avgPrice := ob.AvgExecPrice(insights.BookSideAsk, qty)
```

### Quote

最良気配に特化した軽量インターフェースです。OrderBook と同じ `Quote` interface を実装しています。

```go
quote := coinex.Insights.GetQuote(symbol)

bestAsk := quote.GetBestAsk()
bestBid := quote.GetBestBid()
tickSize := quote.GetTickSize()
rounded := quote.RoundToTickSize(price)
```

### Trade

取引所内の約定データをリアルタイムで受信します。

```go
trade := coinex.Insights.GetTrade(symbol)
trade.SetHandler(func(data []*insights.TradeData) {
	for _, d := range data {
		fmt.Printf("Trade: %s %s @ %s\n", d.Side, d.Qty, d.Price)
	}
})
```

### Callback registration

OrderBook と Quote は更新時のコールバックを登録できます。

```go
ob.SetUpdateCallback(func() {
	// Triggered on every order book update
})

quote.SetUpdateCallback(func() {
	// Triggered on every quote update
})
```

## Deals の概要

注文の送信・変更・キャンセルには認証付きの初期化が必要です。

```go
// Initialize with credentials
coinex.InitGatewayManager(credential, 1)
coinex.InitInsights(...)
coinex.InitDeals(dealSymbols)
coinex.StartGateway(ctx)
```

```go
dealer := coinex.Deals.GetDealer(symbol)

// Create and send a limit order
order := deals.NewLimitOrder(price, qty, domains.SideBuy, false, "")
dealer.SendOrder(order)

// Amend
dealer.AmendOrder(order, deals.AmendDetail{Price: newPrice, Qty: newQty})

// Cancel
dealer.CancelOrder(order)
```

Dealer はシンボルごとのシングルトンで、注文のステートマシン管理・遅延オペレーション・コールバック実行を担います。詳細は [README](../README.ja.md) の「非同期による注文実行」を参照してください。

## 対応シンボル

### CoinEx Futures

| Symbol | Price Tick | Min Qty |
|---|---|---|
| `SymbolCoinExFuturesBTCUSDT` | 0.01 | 0.0001 |
| `SymbolCoinExFuturesETHUSDT` | 0.01 | 0.005 |
| `SymbolCoinExFuturesSOLUSDT` | 0.01 | 0.05 |
| `SymbolCoinExFuturesXRPUSDT` | 0.0001 | 5 |

### Bybit Linear

| Symbol | Price Tick | Min Qty |
|---|---|---|
| `SymbolBybitLinearBTCUSDT` | 0.1 | 0.001 |
| `SymbolBybitLinearETHUSDT` | 0.01 | 0.01 |
| `SymbolBybitLinearSOLUSDT` | 0.001 | 0.1 |
| `SymbolBybitLinearXRPUSDT` | 0.0001 | 10 |

## Bybit の差分

Bybit では OrderBook の初期化時に depth（板の深さ）を明示的に指定する必要があります。

```go
import (
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/bybit"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

symbol := domains.SymbolBybitLinearBTCUSDT

bybit.InitGatewayManager(nil, 1)
bybit.InitInsights(
	[]*domains.Symbol{symbol},
	[]*bybit.OrderBookDesignator{
		{Symbol: symbol, Depth: topics.LinearOBDepth50},
	},
	[]*domains.Symbol{symbol},
)
```

OrderBook の取得にも `OrderBookDesignator` を使用します。

```go
ob := bybit.Insights.GetOrderBook(&bybit.OrderBookDesignator{
	Symbol: symbol,
	Depth:  topics.LinearOBDepth50,
})
```

Quote と Trade の API は CoinEx と同一です。

# Getting Started

lapuacore を利用してリアルタイムのマーケットデータを取得する手順を示します。

## Installation

```bash
go get github.com/yuki-inoue-eng/lapuacore
```

Go 1.26+ が必要です。

## 初期化フロー

lapuacore は4ステップで初期化します。

```
InitAndStartNoopMode → InitGatewayManager → InitInsights → StartGateway
```

### 完全なコード例（CoinEx）

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
	// Step 1: Core initialization (noop mode for testing — no InfluxDB/Discord)
	lapua.InitAndStartNoopMode()
	defer lapua.Cancel()

	// Step 2: Initialize gateway manager (nil = no credentials, public channels only)
	symbol := domains.SymbolCoinExFuturesBTCUSDT
	coinex.InitGatewayManager(nil, 1) // 1 = number of redundant public WebSocket connections

	// Step 3: Initialize market data subscriptions
	coinex.InitInsights(
		[]*domains.Symbol{symbol}, // Trade
		[]*domains.Symbol{symbol}, // OrderBook
		[]*domains.Symbol{symbol}, // Quote (BookTicker)
	)

	// Step 4: Start WebSocket connections
	ctx, cancel := context.WithTimeout(lapua.Ctx, 30*time.Second)
	defer cancel()
	coinex.StartGateway(ctx)

	// Wait for data to be ready
	for !coinex.Insights.IsEverythingReady() {
		time.Sleep(500 * time.Millisecond)
	}

	// Use insights
	ob := coinex.Insights.GetOrderBook(symbol)
	fmt.Printf("BestAsk: %s  BestBid: %s\n",
		ob.GetBestAsk().Price, ob.GetBestBid().Price)
}
```

### 初期化モード

| モード | 関数 | 用途 |
|---|---|---|
| Noop | `lapua.InitAndStartNoopMode()` | テスト・開発用。InfluxDB / Discord に接続しない |
| Full | `lapua.InitAndStart()` | 本番用。設定ファイル・メトリクス・通知すべて有効 |
| DC | `lapua.InitAndStartDCMode()` | データ収集専用モード |

## Insights の利用

`InitInsights` で登録したシンボルについて、3種類のマーケットデータにアクセスできます。

### OrderBook

板情報の全体像にアクセスします。B-Tree によりソート済みデータへの効率的なアクセスが可能です。

```go
ob := coinex.Insights.GetOrderBook(symbol)

// Best bid/ask — O(1)
bestAsk := ob.GetBestAsk() // *PriceLevel{Price, Volume}
bestBid := ob.GetBestBid()

// Cumulative volume at a price level
vol := ob.SumVolume(domains.BookSideAsk, targetPrice)

// Average execution price for a market order of given quantity
avgPrice := ob.AvgExecPrice(domains.BookSideAsk, qty)
```

### Quote

OrderBook より軽量な最良気配データです。BookTicker（取引所が配信する best bid/ask）に基づきます。

```go
quote := coinex.Insights.GetQuote(symbol)
fmt.Printf("Ask: %s  Bid: %s\n",
	quote.GetBestAsk().Price, quote.GetBestBid().Price)

// Price update callback
quote.SetUpdateCallback(func() {
	// called on every price update
})
```

### Trade

リアルタイムの約定データをコールバックで受信します。

```go
trade := coinex.Insights.GetTrade(symbol)
trade.SetHandler(func(trades insights.TradeDataList) {
	for _, t := range trades {
		fmt.Printf("%s %s @ %s\n", t.Side, t.Volume, t.Price)
	}
})
```

## Deals の概要

注文管理には認証が必要です。`InitGatewayManager` に credential を渡して初期化します。

```go
// Initialize with credentials
coinex.InitGatewayManager(cred, 1)
coinex.InitInsights(tradeSymbols, obSymbols, quoteSymbols)
coinex.InitDeals([]*domains.Symbol{symbol})
coinex.StartGateway(ctx)
```

Dealer interface を通じて注文の送信・変更・キャンセルを行います。

```go
dealer := coinex.Deals.GetDealer(symbol)

// Send an order
dealer.SendOrder(order)

// Amend price/quantity
dealer.AmendOrder(order, deals.AmendDetail{Price: newPrice, Qty: newQty})

// Cancel
dealer.CancelOrder(order)

// Current position
pos := dealer.GetCurrentPosition()
fmt.Printf("Position: %s %s\n", pos.GetSide(), pos.Get())
```

### Order lifecycle

注文は内部ステートマシンで管理されます。

```
Born → Sending → Pending ⇄ Amending → Done
                    ↓                    ↑
                Canceling ───────────────┘
```

| ステータス | 説明 |
|---|---|
| Born | 作成済み、未送信 |
| Sending | 取引所へ送信中、応答待ち |
| Pending | 取引所で受理済み、板に載っている |
| Amending | 変更リクエスト送信中 |
| Canceling | キャンセルリクエスト送信中 |
| Done | 終端状態（約定 / キャンセル / リジェクト） |

## Bybit の差分

Bybit も同じ初期化パターンですが、OrderBook の初期化に depth 指定が必要です。

```go
import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
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
bybit.StartGateway(ctx)

// OrderBook access requires designator
ob := bybit.Insights.GetOrderBook(&bybit.OrderBookDesignator{
	Symbol: symbol,
	Depth:  topics.LinearOBDepth50,
})
```

## 対応シンボル

| 取引所 | プロダクト | シンボル | TickSize | MinOrderQty |
|---|---|---|---|---|
| CoinEx | Futures | BTCUSDT | 0.01 | 0.0001 |
| CoinEx | Futures | ETHUSDT | 0.01 | 0.005 |
| CoinEx | Futures | XRPUSDT | 0.0001 | 5 |
| CoinEx | Futures | SOLUSDT | 0.01 | 0.05 |
| Bybit | Linear | BTCUSDT | 0.1 | 0.001 |
| Bybit | Linear | ETHUSDT | 0.01 | 0.01 |
| Bybit | Linear | SOLUSDT | 0.001 | 0.1 |
| Bybit | Linear | XRPUSDT | 0.0001 | 10 |

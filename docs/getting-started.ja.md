# Getting Started

## Installation

```bash
go get github.com/yuki-inoue-eng/lapuacore
```

Go 1.25 以上が必要です。

## 初期化フロー

lapuacore の起動は以下の 6 ステップで行います。

1. **InitAndStart / InitAndStartNoopMode** — コアコンテキスト・ロガー等を初期化。グローバルな `lapua.Ctx` / `lapua.Cancel` が生成され、戦略レイヤーではこのコンテキストを基準にライフサイクルを管理します
2. **InitGatewayManager** — 取引所ゲートウェイの初期化（認証情報・接続数を指定）
3. **InitInsights** — マーケットデータの購読設定（Trade / OrderBook / Quote）
4. **InitDeals** — 注文管理の初期化（対象シンボル・エラーハンドラを指定）
5. **StartGateway** — WebSocket 接続を開始し、データ受信を開始
6. **WaitForInsightsToBeReady** — 全 Insights の初期データ受信完了までブロック

`InitAndStart` を使用する場合、以下の環境変数でファイルパスを指定してください。

| Environment Variable | Description |
|---|---|
| `LAPUA_CONFIG_PATH` | config.yaml のパス |
| `LAPUA_SECRET_PATH` | secret.yaml のパス（API キー等） |
| `LAPUA_LOG_PATH` | ログファイルの出力先パス |


## 設定ファイル

`InitAndStart` は config.yaml と secret.yaml を読み込みます。どちらも fsnotify によるホットリロードに対応しており、ファイルを書き換えると自動的に再読み込みされます。

### config.yaml

戦略名とパラメータを定義します。`params` は任意の key-value ペアで、`lapua.Params.Get(key)` で取得できます。

```yaml
strategy:
  name: my-strategy

params:
  symbol: CoinExFuturesBTCUSDT
  threshold: "0.5"
```

### secret.yaml

取引所の API 認証情報を定義します。`lapua.Secrets.CoinEx` 等で取得できますが、lapua 起動時に自動的に gateway に渡されるので、取得する機会はないと思います。

```yaml
exchanges:
  coinex:
    api_key: "your-api-key"
    secret: "your-secret"

influxdb:
  url: ""
  token: ""

discord:
  info_url: ""
  warn_url: ""
  emergency_url: ""
```

## 実装例: 注文の送信・変更・キャンセル
```go
package main

import (
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/coinex"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
)

var symbol = domains.SymbolCoinExFuturesBTCUSDT

func lapuaInit() {
	// 1. Core initialization with config/secret files
	//    Reads file paths from environment variables.
	//    For development/testing, use lapua.InitAndStartNoopMode() instead
	//    (no config files needed).
	lapua.InitAndStart()

	// 2. Initialize gateway with credentials from secret.yaml
	coinex.InitGatewayManager(lapua.Secrets.CoinEx, 3)

	// 3. Configure market data subscriptions
	coinex.InitInsights(nil, []*domains.Symbol{symbol}, nil)

	// 4. Initialize deals (order management) for the symbol
	coinex.InitDeals([]*domains.Symbol{symbol}, func(err error) {
		slog.Error("deal error", "error", err)
	})

	// 5. Start WebSocket connections (public + private)
	coinex.StartGateway()

	lapua.WaitForInsightsToBeReady()
}

func strategy() {
	dealer := coinex.Deals.GetDealer(symbol)
	ob := coinex.Insights.GetOrderBook(symbol)
	bestBid := ob.GetBestBid().Price
	qty := decimal.NewFromFloat(0.001)

	// Place a limit buy at 10% below the best bid, rounded to tick size
	price := ob.RoundToTickSize(bestBid.Mul(decimal.NewFromFloat(0.9)))
	order := deals.NewLimitOrder(price, qty, domains.SideBuy, false, "")
	deals.SendOrder(dealer, order)
	time.Sleep(5 * time.Second)

	// Amend to 20% below the best bid
	newPrice := ob.RoundToTickSize(bestBid.Mul(decimal.NewFromFloat(0.8)))
	deals.AmendOrder(dealer, order, deals.AmendDetail{
		Price: newPrice,
		Qty:   qty,
	})
	time.Sleep(5 * time.Second)

	// Cancel the order
	deals.CancelOrder(dealer, order)
}

func main() {
	// Initialize lapua core components (logger, gateway, insights, deals)
	lapuaInit()

	go strategy()

	// Block until SIGTERM is received
	lapua.WaitForCtxDone()
}
```

## Insights の利用

### OrderBook

板情報を取得し、最良気配や累積出来高を算出できます。

```go
ob := coinex.Insights.GetOrderBook(symbol)

// Best bid/ask
bestAsk := ob.GetBestAsk() // *PriceLevel{Price, Volume}
bestBid := ob.GetBestBid()

// Top N price levels
asks := ob.GetAsks(5) // []PriceLevel, ascending price (best first)
bids := ob.GetBids(5) // []PriceLevel, descending price (best first)

// Cumulative volume up to a price
volume := ob.SumVolume(domains.BookSideAsk, targetPrice)

// Average execution price for a given quantity
avgPrice := ob.AvgExecPrice(domains.BookSideAsk, qty)
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
trade.SetUpdateCallback(func(msg insights.TradeDataList) {
	for _, d := range msg {
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

注文の送信・変更・キャンセルには認証付きの初期化（`InitGatewayManager` に credential を渡す）と `InitDeals` が必要です。初期化の全体像は上記の初期化フローを参照してください。

```go
dealer := coinex.Deals.GetDealer(symbol)

// Create and send a limit order
order := deals.NewLimitOrder(price, qty, domains.SideBuy, false, "")
deals.SendOrder(dealer, order)

// Amend
deals.AmendOrder(dealer, order, deals.AmendDetail{Price: newPrice, Qty: newQty})

// Cancel
deals.CancelOrder(dealer, order)
```

Dealer はシンボルごとのシングルトンで、注文のステートマシン管理・遅延オペレーション・コールバック実行を担います。詳細は [README](../README.ja.md) の「非同期による注文実行」を参照してください。

## サンプル戦略

`examples/` ディレクトリにすぐに実行できるサンプル戦略を用意しています。今後さらに追加していく予定です。

### 実行方法

```bash
cd examples
make run STRATEGY=<strategy-name>
```

環境変数 `LAPUA_CONFIG_PATH`, `LAPUA_SECRET_PATH`, `LAPUA_LOG_PATH` が自動的に設定されます。

### book-monitor

CoinEx Futures の BTC/USDT と SOL/USDT の板情報をリアルタイムで横並び表示するサンプルです。認証不要（public data only）で動作します。
- ソース: [`examples/book-monitor/main.go`](../examples/book-monitor/main.go)

```bash
make run STRATEGY=book-monitor
```

**表示例:**

```
  ====== BTCUSDT =======       ====== SOLUSDT =======
  Ask 74369.00: 0.2082         Ask 85.88: 395.58
  Ask 74368.00: 0.1499         Ask 85.87: 348.88
  Ask 74367.00: 0.1344         Ask 85.86: 336.89
  ---- spread(1.00) -----      ---- spread(0.02) ----
  Bid 74364.00: 0.3531         Bid 85.82: 34.87
  Bid 74363.00: 0.0675         Bid 85.81: 232.42
```

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

## 監視

メトリクス（InfluxDB）や Discord 通知による監視機能を提供していますが、現在ドキュメントを整備中です。

# Getting Started

## Installation

```bash
go get github.com/yuki-inoue-eng/lapuacore
```

Requires Go 1.26 or later.

## Initialization Flow

Startup consists of the following 6 steps:

1. **InitAndStart / InitAndStartNoopMode** — Initialize core context, logger, etc. Creates global `lapua.Ctx` / `lapua.Cancel` used by the strategy layer to manage its lifecycle
2. **InitGatewayManager** — Initialize exchange gateway (credentials, connection count)
3. **InitInsights** — Configure market data subscriptions (Trade / OrderBook / Quote)
4. **InitDeals** — Initialize order management (target symbols, error handler)
5. **StartGateway** — Open WebSocket connections and begin receiving data
6. **WaitForInsightsToBeReady** — Block until all Insights have received their initial data

When using `InitAndStart`, specify file paths via the following environment variables:

| Environment Variable | Description |
|---|---|
| `LAPUA_CONFIG_PATH` | Path to config.yaml |
| `LAPUA_SECRET_PATH` | Path to secret.yaml (API keys, etc.) |
| `LAPUA_LOG_PATH` | Log file output path |


## Configuration Files

`InitAndStart` loads config.yaml and secret.yaml. Both support hot-reload via fsnotify — changes to these files are automatically picked up at runtime.

### config.yaml

Defines the strategy name and parameters. `params` accepts arbitrary key-value pairs, accessible via `lapua.Params.Get(key)`.

```yaml
strategy:
  name: my-strategy

params:
  symbol: CoinExFuturesBTCUSDT
  threshold: "0.5"
```

### secret.yaml

Defines exchange API credentials. Accessible via `lapua.Secrets.CoinEx` etc., though they are automatically passed to the gateway during initialization.

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

## Example: Sending, Amending, and Canceling Orders
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

## Using Insights

### OrderBook

Access order book data including best prices, depth levels, and cumulative volumes.

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

A lightweight interface focused on best prices. Implements the same `Quote` interface as OrderBook.

```go
quote := coinex.Insights.GetQuote(symbol)

bestAsk := quote.GetBestAsk()
bestBid := quote.GetBestBid()
tickSize := quote.GetTickSize()
rounded := quote.RoundToTickSize(price)
```

### Trade

Receive real-time execution data from the exchange.

```go
trade := coinex.Insights.GetTrade(symbol)
trade.SetUpdateCallback(func(msg insights.TradeDataList) {
	for _, d := range msg {
		fmt.Printf("Trade: %s %s @ %s\n", d.Side, d.Qty, d.Price)
	}
})
```

### Callback Registration

OrderBook and Quote support update callbacks.

```go
ob.SetUpdateCallback(func() {
	// Triggered on every order book update
})

quote.SetUpdateCallback(func() {
	// Triggered on every quote update
})
```

## Deals Overview

Sending, amending, and canceling orders requires authenticated initialization (pass credentials to `InitGatewayManager`) and `InitDeals`. Refer to the initialization flow above for the full picture.

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

Dealer is a per-symbol singleton that manages the order state machine, deferred operations, and callback dispatch. See the "Async Order Execution" section in [README](../README.md) for details.

## Sample Strategies

The `examples/` directory contains ready-to-run sample strategies. More will be added over time.

### Running

```bash
cd examples
make run STRATEGY=<strategy-name>
```

Environment variables `LAPUA_CONFIG_PATH`, `LAPUA_SECRET_PATH`, and `LAPUA_LOG_PATH` are set automatically.

### book-monitor

Displays real-time order books for CoinEx Futures BTC/USDT and SOL/USDT side by side in the terminal. No authentication required (public data only).
- Source: [`examples/book-monitor/main.go`](../examples/book-monitor/main.go)

```bash
make run STRATEGY=book-monitor
```

**Sample output:**

```
  ====== BTCUSDT =======       ====== SOLUSDT =======
  Ask 74369.00: 0.2082         Ask 85.88: 395.58
  Ask 74368.00: 0.1499         Ask 85.87: 348.88
  Ask 74367.00: 0.1344         Ask 85.86: 336.89
  ---- spread(1.00) -----      ---- spread(0.02) ----
  Bid 74364.00: 0.3531         Bid 85.82: 34.87
  Bid 74363.00: 0.0675         Bid 85.81: 232.42
```

## Supported Symbols

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

## Bybit Differences

Bybit requires an explicit depth when initializing OrderBook subscriptions.

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

OrderBook retrieval also uses `OrderBookDesignator`.

```go
ob := bybit.Insights.GetOrderBook(&bybit.OrderBookDesignator{
	Symbol: symbol,
	Depth:  topics.LinearOBDepth50,
})
```

Quote and Trade APIs are identical to CoinEx.

## Monitoring

Monitoring via metrics (InfluxDB) and Discord notifications is available, but documentation is currently being prepared.

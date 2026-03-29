# lapuacore

Exchange-agnostic core for low-latency trading systems.

lapuacore is a reduced-feature edition of **lapua**, a private HFT library used to run a market-making operation on centralized exchanges. lapua provides exchange-independent domain models, adapters for multiple exchanges, and shared infrastructure for order management and market-data processing — all decoupled from any specific trading strategy. lapuacore inherits that design philosophy and exposes it as a minimal, readable codebase.

## Background

lapua was built to solve a practical problem: trading across multiple exchanges without duplicating logic for each venue. The author operated as an official market maker on CoinEx using lapua as the underlying trading engine, sustaining approximately $10 M/month in liquidity provision at its peak.

lapuacore extracts the architectural foundations of lapua — the domain models, gateway abstractions, and concurrency primitives — into a standalone package. Adapters for two exchanges and sample code demonstrating end-to-end order execution are currently under development.

> **Note:** This project is a design reference, not an actively maintained OSS library.

## Design Principles

**Exchange as a replaceable dependency.** Each exchange has its own WebSocket frame format, order lifecycle semantics, and rate-limit rules. lapuacore defines a Gateway interface that normalises these differences. Exchange adapters implement the interface; the rest of the system operates against the abstraction.

**Internal order-state authority.** Relying on the exchange as the source of truth for order state introduces round-trip latency that matters at high frequency. lapuacore maintains its own order state machine and reconciles asynchronous events — fills, cancels, expiries — internally, so consumers always have a consistent view.

**Strategy-independent infrastructure.** The library provides building blocks — normalised market-data streams, an order manager, a synchronised order book — without prescribing what to trade or when. Strategy logic is the consumer's responsibility.

## Architecture

```
Strategy Layer  (user-provided)
        │
        ▼
┌──────────────────────────────────────┐
│            lapuacore                 │
│                                     │
│  domains/                           │
│    ├── Order      state machine     │
│    ├── OrderBook  L2 book           │
│    └── Market Data                  │
│                                     │
│  internal/gateways/                 │
│    └── Gateway interface            │
│                                     │
│  mutex/                             │
│    └── sync utilities               │
├──────────────────────────────────────┤
│  Exchange Adapters  (in progress)   │
│    ├── CoinEx                       │
│    └── (+ one additional exchange)  │
└──────────────────────────────────────┘
        │
        ▼
   Exchange APIs  (WebSocket / REST)
```

## Package Overview

| Package | Role |
|---|---|
| `domains/` | Core domain models. **Order** manages state transitions across the order lifecycle. **OrderBook** maintains an L2 book representation. **Market Data** handles price and tick normalisation. |
| `internal/gateways/` | Gateway interface that exchange adapters implement. Covers order submission, cancellation, market-data subscription, and connection lifecycle. |
| `mutex/` | Synchronisation utilities for concurrent order and book updates. |

## Getting Started

```bash
go get github.com/yuki-inoue-eng/lapuacore
```

Requires Go 1.26+.

### Dependencies

| Dependency | Purpose |
|---|---|
| `gorilla/websocket` | WebSocket stream handling |
| `shopspring/decimal` | Arbitrary-precision decimal arithmetic for price/quantity |
| `google/uuid` | Unique identifiers |
| `rs/xid` | Globally unique, sortable IDs |

## Examples

*Coming soon.* The `examples/` directory will contain sample code that connects to a live exchange, subscribes to market data, and places/cancels orders.

## License

[Apache License 2.0](LICENSE)
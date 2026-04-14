# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-14

Initial release.

### Added

- **Domain layer** (`domains/`)
  - Order state machine with lifecycle management (Born, Sending, Pending, Amending, Canceling, Done)
  - Deferred operations: amend/cancel requests queued during in-flight operations
  - Dealer: per-symbol singleton order manager with callback dispatch
  - OrderBook: B-Tree backed order book with O(1) best bid/ask and O(n) sorted iteration
  - Quote: lightweight best-price interface (implemented by both OrderBook and BookTicker)
  - Trade: real-time execution data stream with update callbacks
- **Exchange adapters** (`internal/gateways/`)
  - CoinEx Futures: REST API client (HMAC signing, rate limiter) + WebSocket (redundant connections, TTL dedup)
  - Bybit Linear: REST API client (HMAC signing, rate limiter) + WebSocket (redundant connections, TTL dedup)
- **Initializers** (`initializers/`)
  - `lapua/`: top-level orchestrator (InitAndStart, InitAndStartDCMode, InitAndStartNoopMode)
  - `exchanges/coinex/`, `exchanges/bybit/`: per-exchange gateway, insights, and deals initialization
  - Discord notification client, structured logger setup
- **Configuration** (`configs/`)
  - YAML config/secret loading with fsnotify hot reload
  - ParamMap for strategy parameters, Credential interface for API keys
- **Metrics** (`metrics/`)
  - InfluxDB exporter with latency and custom metric measurements
- **Concurrency primitives** (`mutex/`)
  - Generic thread-safe Flag, Map, Slice
- **Sample strategies** (`examples/`)
  - `book-monitor`: real-time terminal order book display for CoinEx BTC/USDT and SOL/USDT
- **Documentation** (`docs/`)
  - Getting Started guides in English and Japanese
- **CI**
  - `go test -race ./...` on every push

[1.0.0]: https://github.com/yuki-inoue-eng/lapuacore/releases/tag/v1.0.0

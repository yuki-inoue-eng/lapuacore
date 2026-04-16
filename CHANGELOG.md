# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-04-16

### Changed

- **Minimum Go version**: bumped to Go 1.25.9 (from 1.21)
- **Panic removal**: all `panic()` calls removed from library code (`gateways/`, `configs/`, `metrics/`) and consolidated into `initializers/`
- **`initializers/context`**: `NewCancellableContext` now returns `context.CancelCauseFunc` instead of `context.CancelFunc`
- **`configs/watcher`**: `NewWatcher` takes `context.CancelCauseFunc` as first argument; `Option`/`WithParamValidator` pattern removed
- **`metrics/exporter`**: `NewExporter` returns `(*Exporter, error)` instead of panicking
- **`internal/gateways`**: `ChannelGroup.Start()` returns `error` instead of panicking

### Added

- **ParamMap cache fallback**: getter methods (`GetBool`, `GetInt`, etc.) return previously cached values on parse failure instead of panicking, with periodic logging of failed keys
- **Graceful shutdown on file watch failure**: Watcher calls `CancelCauseFunc` when fsnotify breaks, triggering orderly shutdown
- **CI enhancements**: coverage profiling, `go vet` lint job, `govulncheck` security scanning (3 parallel jobs)

### Fixed

- Unkeyed struct fields in `examples/book-monitor/main.go`

### Security

- Updated `golang.org/x/net` to v0.53.0 (resolves HTTP/2 vulnerabilities)
- Updated `hashicorp/go-retryablehttp` to v0.7.8

[1.1.0]: https://github.com/yuki-inoue-eng/lapuacore/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/yuki-inoue-eng/lapuacore/releases/tag/v1.0.0

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

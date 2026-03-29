# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go test ./...                        # Run all tests (also used in CI)
go test ./domains/deals/...          # Run tests for a specific package
go test -run TestSendOrder ./domains/deals/...  # Run a single test
go mod tidy                          # Tidy dependencies (run before commits)
```

No linter is configured. CI runs `go test ./...` on every push.

## Architecture

lapuacore is an exchange-agnostic core library for low-latency trading systems. It provides order management, market data handling, and exchange adapters as a reusable Go module. A separate repository (`lapua`) serves as the reference implementation that consumes this library.

### Layer Structure

```
Strategy Layer (user-provided)
    ↓
domains/          ← Exchange-agnostic domain logic
  deals/          ← Order state machine, Dealer (per-symbol order manager), Agent interface
  insights/       ← L2 OrderBook with tick-aware price records
    ↓
internal/gateways/exchanges/   ← Exchange-specific adapters
  coinex/
    agent/        ← REST API client (HTTP + HMAC signing)
    ws/           ← WebSocket client (channels, topics, auth, health)
    dtos/         ← Wire-format structs
    translators/  ← DTO ↔ domain model conversion
    ↓
mutex/            ← Generic thread-safe Flag, Map, Slice
```

### Key Design Decisions

- **Internal order authority**: Order state machine lives in lapuacore, not delegated to exchange. The `Order` struct tracks its own lifecycle (Born → Sending → Pending → Amending/Canceling → Done) using RWMutex-protected fields.
- **Callback-driven async**: REST and WebSocket responses invoke handler callbacks. No explicit goroutine management in domain code — callbacks are dispatched with `go callback(o)`.
- **Deferred operations**: Amend/cancel requests during Sending or Amending states are queued as callbacks, executed when the in-flight operation completes.
- **Singleton Dealers**: One `Dealer` instance per `Symbol`, retrieved via global registry. Prevents order fragmentation.
- **Agent interface** (`deals/agent.go`): Exchange adapters implement this to plug into the Dealer. Supports single and batch send/cancel/amend with typed response handlers.
- **Tick-aware OrderBook**: Records stored by price string at symbol tick granularity. Caches best bid/ask for O(1) access.

### Test Patterns

Tests use table-driven subtests with a `mockAgent` (defined in `testhelper_test.go`) that exposes function fields for injection. Assertions use `github.com/bmizerany/assert`.

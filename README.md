# lapuacore

A reference implementation of domain and infrastructure design for low-latency trading systems, written in Go.

## Overview

lapuacore is a portfolio project demonstrating the architectural patterns used in a personal high-frequency trading system. It is not intended for production use or active development as an open-source library.

The codebase illustrates how to structure an HFT system with a strict boundary between exchange-agnostic domain logic and exchange-specific infrastructure

## Background

I previously operated as an official CoinEx market maker using a personal HFT engine named lapua, contributing over $10M/month in liquidity at peak. lapuacore is the domain and infrastructure layer of lapua, extracted as a self-contained codebase for portfolio purposes.

## Architecture

```
lapuacore/
├── domains/
│   ├── deals/      # Active order management (placement, amendment, cancellation)
│   └── insights/   # Market data (order book)
├── internal/
│   └── gateways/
│       └── exchanges/
│           └── coinex/        # CoinEx-specific implementations
│               ├── agent/     # REST API client (order operations)
│               ├── ws/        # WebSocket client (market data + private channel)
│               ├── dtos/      # Exchange API data transfer objects
│               └── translators/  # DTO ↔ domain model conversion
└── mutex/          # Thread-safe generic primitives
```

### Design Principles

- `domains/` contains pure domain logic with no exchange dependencies. It defines the `Agent` interface that exchange implementations must satisfy, but never imports gateway code.
- `internal/gateways/` contains exchange-specific implementations, inaccessible to external consumers.
- Infrastructure concerns (notifications, metrics, shutdown signals) are injected by the application layer via callbacks and interfaces, keeping the domain layer free of operational dependencies.

## Key Design Decisions

### Order Lifecycle State Machine

Orders transition through a well-defined set of states: `Born → Sending → Pending → (Canceling | Amending) → Done`. Each transition is explicit and guarded. Operations that arrive while an order is mid-flight — for example, an amend request while the placement HTTP call is still in progress — are deferred and applied once the in-flight operation completes. Multiple deferred amend calls collapse into a single execution with the latest parameters, avoiding redundant round-trips.

### Exchange-Agnostic Agent Interface

The `domains/deals` package defines an `Agent` interface that abstracts all exchange HTTP operations (order placement, cancellation, amendment). Each exchange provides its own implementation under `internal/gateways/exchanges/<exchange>/agent/`. This inversion of dependency allows domain logic and tests to operate independently of any real exchange.

### Callback-Driven Async Model

All REST API calls are non-blocking. Each operation dispatches the HTTP request in a goroutine and invokes a typed response handler on completion. Concurrently, the private WebSocket channel delivers order update events. HTTP responses and WebSocket events reconcile order state independently, with each path handling only the transitions it owns — fills and partial fills are driven by WebSocket events; placement, cancellation, and amendment confirmations are driven by HTTP responses.

### Extensible Gateway Design

The `gateways.Credential` interface (`GetApiKey`, `GetSecret`) and the `deals.Agent` interface are defined without reference to any specific exchange. The design allows a new exchange to be integrated by implementing these interfaces under a new `internal/gateways/exchanges/<exchange>/` subtree, with no changes to domain logic.

### Thread-Safe Primitives

The `mutex` package provides generic thread-safe types (`Map[K,V]`, `Slice[T]`, `Flag`) used throughout the order management layer to safely share state across the goroutines handling HTTP responses and WebSocket events.

## Package Reference

| Package | Description |
|---|---|
| `domains/deals` | Order lifecycle management: placement, amendment, cancellation, and fill handling via state machine |
| `domains/insights` | Read-only market data: order book state maintained from WebSocket delta updates |
| `internal/gateways/exchanges/coinex/agent` | CoinEx REST API client implementing `deals.Agent` |
| `internal/gateways/exchanges/coinex/ws` | CoinEx WebSocket client: public market data channel and authenticated private order channel |
| `internal/gateways/exchanges/coinex/dtos` | CoinEx API request/response data transfer objects |
| `internal/gateways/exchanges/coinex/translators` | Conversion between CoinEx DTOs and domain models |
| `mutex` | Generic thread-safe primitives: `Map[K,V]`, `Slice[T]`, `Flag` |

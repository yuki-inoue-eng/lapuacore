# lapuacore

取引所非依存の低レイテンシ取引システム基盤ライブラリ (Go)

[English](README.md)

## Background

lapuacore は、著者が開発・運用するプライベートHFTライブラリ **lapua** のアーキテクチャ基盤を抽出したスタンドアロンパッケージです。

lapua は複数の暗号資産取引所にまたがる取引を、取引所ごとのロジック重複なしに実現するために構築されました。著者は lapua を基盤として CoinEx の公式マーケットメイカーを務め、ピーク時に月間約 $10M の流動性を供給していました。

lapuacore はそのドメインモデル、ゲートウェイ抽象化、並行処理プリミティブを、可読性の高いコードベースとして公開しています。CoinEx と Bybit の2取引所アダプタを実装済みです。

> **Note:** このプロジェクトは設計リファレンスです。OSS ライブラリとしてのメンテナンスは対象外です。

## Design Highlights

### 取引所非依存のドメインレイヤー

取引所ごとに WebSocket フレーム形式、注文ライフサイクル、レート制限ルールが異なります。lapuacore は Gateway interface でこれらの差異を吸収し、ドメインロジックは抽象化に対してのみ動作します。取引所アダプタの追加・差し替えがドメインコードに一切影響しません。

### 内部オーダーステートマシン

取引所を state of truth とすると、状態確認のたびに往復レイテンシが発生します。lapuacore は注文の状態遷移を内部で管理し、非同期イベント（約定・キャンセル・失効）を内部で整合させます。

```
Born → Sending → Pending ⇄ Amending → Done
                    ↓                    ↑
                Canceling ───────────────┘
```

Sending / Amending 中に到着した Amend / Cancel は、先行オペレーションの完了を待って自動実行されます。複数回の Amend は最後のリクエストのみ保持し、Cancel は既存の Amend を上書きします。

### B-Tree 板情報

板情報（OrderBook）の価格帯管理に `google/btree` を採用しています。

- **最良気配の取得**: O(1) — キャッシュ済み
- **価格順イテレーション**: O(n) — B-Tree の構造的順序を利用し、ソート不要
- **累積出来高 / 平均約定価格**: B-Tree を順方向に走査して算出

### コールバック駆動の非同期処理

REST / WebSocket レスポンスはハンドラコールバックを呼び出す設計です。ドメインコードに明示的な goroutine 管理を持ち込まず、コールバックの dispatch で `go callback(o)` を使用します。Trade データのリアルタイム配信もコールバック登録で実現しています。

### 冗長 WebSocket + 重複排除

単一の WebSocket 接続が切断されると板情報が途切れるリスクがあります。lapuacore は ChannelGroup で N 本の冗長接続を管理し、同一トピックを並行購読します。TTL キャッシュにより重複メッセージを排除し、最初に到着したデータのみを処理します。

## Architecture

```
Strategy Layer (user-provided)
        │
        ▼
┌─────────────────────────────────────────┐
│              lapuacore                  │
│                                         │
│  domains/                               │
│    ├── deals/     Order state machine,  │
│    │              Dealer, Agent          │
│    └── insights/  OrderBook, Quote,     │
│                   Trade, PriceLevel     │
│                                         │
│  initializers/                          │
│    ├── lapua/     Startup orchestration │
│    └── exchanges/ Per-exchange init     │
│       ├── coinex/                       │
│       └── bybit/                        │
│                                         │
│  configs/    YAML config + hot reload   │
│  metrics/    InfluxDB + latency         │
│  mutex/      Thread-safe primitives     │
├─────────────────────────────────────────┤
│  internal/gateways/exchanges/           │
│    ├── coinex/  REST + WebSocket        │
│    └── bybit/   REST + WebSocket        │
└─────────────────────────────────────────┘
        │
        ▼
   Exchange APIs (WebSocket / REST)
```

## Project Structure

| パッケージ | 役割 |
|---|---|
| `domains/deals` | 注文ステートマシン、Dealer（シンボルごとのシングルトン注文マネージャー）、Agent interface |
| `domains/insights` | OrderBook（B-Tree 板情報）、Quote（最良気配）、Trade（約定データストリーム） |
| `initializers/` | 起動オーケストレーション。`lapua/` で全体初期化、`exchanges/` で取引所別の初期化 |
| `configs/` | YAML 設定・シークレットの読み込み、fsnotify によるホットリロード |
| `metrics/` | InfluxDB エクスポーター、WebSocket レイテンシ・カスタムメトリクス計測 |
| `internal/gateways/` | 取引所アダプタ実装。REST API（HMAC 署名、レートリミッター）、WebSocket（チャネル、トピック、認証、ヘルスチェック） |
| `mutex/` | 汎用スレッドセーフ型（Flag, Map, Slice） |

## Supported Exchanges

| 取引所 | プロダクト | シンボル |
|---|---|---|
| CoinEx | Futures | BTCUSDT, ETHUSDT, XRPUSDT, SOLUSDT |
| Bybit | Linear | BTCUSDT, ETHUSDT, SOLUSDT, XRPUSDT |

## Dependencies

| ライブラリ | 用途 |
|---|---|
| `google/btree` | 板情報の価格帯管理（B-Tree） |
| `gorilla/websocket` | WebSocket ストリーム処理 |
| `shopspring/decimal` | 価格・数量の任意精度十進演算 |
| `fsnotify/fsnotify` | 設定ファイルのホットリロード |
| `InfluxCommunity/influxdb3-go` | メトリクスエクスポート |
| `hashicorp/go-retryablehttp` | リトライ付き HTTP クライアント |

## Documentation

- [Getting Started](docs/getting-started.ja.md) — 初期化フローとサンプルコード

## License

[Apache License 2.0](LICENSE)

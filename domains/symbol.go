package domains

import (
	"log/slog"

	"github.com/shopspring/decimal"
)

var (
	symbolMapByID   map[string]*Symbol
	symbolMapByName map[string]*Symbol
)

func GetSymbol(id string) *Symbol {
	if s, ok := symbolMapByID[id]; ok {
		return s
	} else {
		slog.Error("use Unknown Symbol", "id", id)
		return SymbolUnknown
	}
}

// Symbol 各種取引所のシンボルを表します。 Dealer はシンボルによって生成されます。
type Symbol struct {
	id          string
	exchange    string
	product     string
	name        string
	tickSize    decimal.Decimal
	minOrderQty decimal.Decimal
	baseAsset   Asset // 損益計算に使用されるコイン
}

func newSymbol(exchange, product, name string, tickSize, minOrderQty float64, baseAsset Asset) *Symbol {

	if symbolMapByName == nil {
		symbolMapByName = map[string]*Symbol{}
	}
	if symbolMapByID == nil {
		symbolMapByID = map[string]*Symbol{}
	}

	s := &Symbol{
		id:          exchange + "_" + product + "_" + name, // ex: CoinEx_futures_BTCUSDT
		exchange:    exchange,
		product:     product,
		name:        name,
		tickSize:    decimal.NewFromFloat(tickSize),
		minOrderQty: decimal.NewFromFloat(minOrderQty),
		baseAsset:   baseAsset,
	}
	symbolMapByID[s.id] = s
	symbolMapByName[s.name] = s
	return s
}

func (s *Symbol) ID() string {
	return s.id
}
func (s *Symbol) Exchange() string {
	return s.exchange
}
func (s *Symbol) Product() string {
	return s.product
}
func (s *Symbol) Name() string {
	return s.name
}
func (s *Symbol) TickSize() decimal.Decimal {
	return s.tickSize
}
func (s *Symbol) MinOrderQty() decimal.Decimal {
	return s.minOrderQty
}
func (s *Symbol) BaseAsset() Asset {
	return s.baseAsset
}

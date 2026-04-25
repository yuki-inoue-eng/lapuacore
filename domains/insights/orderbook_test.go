package insights

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type orderBookTestLevel struct {
	seqID  int64
	price  string
	volume string
}

func orderBookMustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()

	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("invalid decimal %q: %v", s, err)
	}
	return d
}

func orderBookMakeRecords(t *testing.T, levels []orderBookTestLevel) []PriceLevel {
	t.Helper()

	out := make([]PriceLevel, 0, len(levels))
	for _, lv := range levels {
		out = append(out, PriceLevel{
			SeqID:  lv.seqID,
			Price:  orderBookMustDecimal(t, lv.price),
			Volume: orderBookMustDecimal(t, lv.volume),
		})
	}
	return out
}

func orderBookSeedMaps(t *testing.T, book *OrderBookImpl, asks []orderBookTestLevel, bids []orderBookTestLevel) {
	t.Helper()

	for _, r := range orderBookMakeRecords(t, asks) {
		book.AsksMap.set(r)
	}
	for _, r := range orderBookMakeRecords(t, bids) {
		book.BidsMap.set(r)
	}
}

func orderBookCollectLevels(m *PriceLevelMap) []string {
	var out []string
	m.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
		out = append(out, fmt.Sprintf("%s:%s:%d", record.Price.String(), record.Volume.String(), record.SeqID))
		return true
	})
	return out
}

func TestOrderBook_RoundToTickSize(t *testing.T) {
	book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)

	tests := []struct {
		name  string
		price string
		want  string
	}{
		{
			name:  "rounds down to the nearest tick",
			price: "100.129",
			want:  "100.12",
		},
		{
			name:  "keeps the price unchanged when it is already on a tick",
			price: "100.12",
			want:  "100.12",
		},
		{
			name:  "rounds small remainder down",
			price: "100.001",
			want:  "100",
		},
		{
			name:  "returns zero when the price is smaller than one tick",
			price: "0.009",
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := book.RoundToTickSize(orderBookMustDecimal(t, tt.price))
			want := orderBookMustDecimal(t, tt.want)

			if !got.Equal(want) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestOrderBook_CalcBestPrice(t *testing.T) {
	book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)

	tests := []struct {
		name        string
		midPrice    string
		wantBestAsk string
		wantBestBid string
	}{
		{
			name:        "calculates best prices around a non-tick mid price",
			midPrice:    "100.129",
			wantBestAsk: "100.13",
			wantBestBid: "100.12",
		},
		{
			name:        "calculates best prices around an exact tick mid price",
			midPrice:    "100.12",
			wantBestAsk: "100.13",
			wantBestBid: "100.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAsk, gotBid := book.CalcBestPrice(orderBookMustDecimal(t, tt.midPrice))

			if !gotAsk.Equal(orderBookMustDecimal(t, tt.wantBestAsk)) {
				t.Errorf("got %v, want true", false)
			}
			if !gotBid.Equal(orderBookMustDecimal(t, tt.wantBestBid)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestOrderBook_AvgExecPrice(t *testing.T) {
	tests := []struct {
		name  string
		bookSide domains.BookSide
		asks  []orderBookTestLevel
		bids  []orderBookTestLevel
		qty   string
		want  string
	}{
		{
			name:  "ask side average execution price is rounded down to tick size",
			bookSide: domains.BookSideAsk,
			asks: []orderBookTestLevel{
				{seqID: 1, price: "100", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			qty:  "3",
			want: "100.66",
		},
		{
			name:  "bid side average execution price is rounded down to tick size",
			bookSide: domains.BookSideBid,
			bids: []orderBookTestLevel{
				{seqID: 1, price: "102", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			qty:  "3",
			want: "101.33",
		},
		{
			name:  "returns zero for an unknown book side",
			bookSide: domains.BookSideNone,
			qty:   "3",
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookSeedMaps(t, book, tt.asks, tt.bids)

			got := book.AvgExecPrice(tt.bookSide, orderBookMustDecimal(t, tt.qty))

			if !got.Equal(orderBookMustDecimal(t, tt.want)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestOrderBook_AvgExecPriceBySide(t *testing.T) {
	tests := []struct {
		name string
		side domains.Side
		asks []orderBookTestLevel
		bids []orderBookTestLevel
		qty  string
		want string
	}{
		{
			name: "buy side consumes asks and rounds down to tick size",
			side: domains.SideBuy,
			asks: []orderBookTestLevel{
				{seqID: 1, price: "100", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			qty:  "3",
			want: "100.66",
		},
		{
			name: "sell side consumes bids and rounds down to tick size",
			side: domains.SideSell,
			bids: []orderBookTestLevel{
				{seqID: 1, price: "102", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			qty:  "3",
			want: "101.33",
		},
		{
			name: "returns zero for an unknown side",
			side: domains.SideNone,
			qty:  "3",
			want: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookSeedMaps(t, book, tt.asks, tt.bids)

			got := book.AvgExecPriceBySide(tt.side, orderBookMustDecimal(t, tt.qty))

			if !got.Equal(orderBookMustDecimal(t, tt.want)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestOrderBook_CalculateBidsVolSumMap(t *testing.T) {
	tests := []struct {
		name string
		bids []orderBookTestLevel
		want []string
	}{
		{
			name: "builds cumulative bid volume in bid order",
			bids: []orderBookTestLevel{
				{seqID: 1, price: "100", volume: "3"},
				{seqID: 1, price: "102", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			want: []string{
				"102:1:0",
				"101:3:0",
				"100:6:0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookSeedMaps(t, book, nil, tt.bids)

			got := orderBookCollectLevels(book.CalculateBidsVolSumMap())

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderBook_CalculateAsksVolSumMap(t *testing.T) {
	tests := []struct {
		name string
		asks []orderBookTestLevel
		want []string
	}{
		{
			name: "builds cumulative ask volume in ask order",
			asks: []orderBookTestLevel{
				{seqID: 1, price: "102", volume: "3"},
				{seqID: 1, price: "100", volume: "1"},
				{seqID: 1, price: "101", volume: "2"},
			},
			want: []string{
				"100:1:0",
				"101:3:0",
				"102:6:0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookSeedMaps(t, book, tt.asks, nil)

			got := orderBookCollectLevels(book.CalculateAsksVolSumMap())

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderBook_Getters(t *testing.T) {
	book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)

	if !book.GetTickSize().Equal(orderBookMustDecimal(t, "0.01")) {
		t.Errorf("got %v, want true", false)
	}
	if !book.GetMinOrderQty().Equal(orderBookMustDecimal(t, "0.0001")) {
		t.Errorf("got %v, want true", false)
	}
}

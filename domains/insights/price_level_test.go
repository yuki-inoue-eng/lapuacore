package insights

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type testLevel struct {
	price  string
	volume string
}

func mustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()

	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("invalid decimal %q: %v", s, err)
	}
	return d
}

func newTestPriceLevelMap(t *testing.T, bookSide domains.BookSide, levels []testLevel) *PriceLevelMap {
	t.Helper()

	m := newPriceLevelMap(bookSide, mustDecimal(t, "0.1"))
	for i, lv := range levels {
		m.set(PriceLevel{
			SeqID:  int64(i + 1),
			Price:  mustDecimal(t, lv.price),
			Volume: mustDecimal(t, lv.volume),
		})
	}
	return m
}

func TestPriceLevelMap_ordering(t *testing.T) {
	tests := []struct {
		name   string
		bookSide  domains.BookSide
		levels []testLevel
		want   []string
	}{
		{
			name:  "ask sorts prices in ascending order",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			want: []string{"99", "100", "101"},
		},
		{
			name:  "bid sorts prices in descending order",
			bookSide: domains.BookSideBid,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			want: []string{"101", "100", "99"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestPriceLevelMap(t, tt.bookSide, tt.levels)

			var got []string
			m.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
				got = append(got, price.String())
				return true
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceLevelMap_SortedRange(t *testing.T) {
	tests := []struct {
		name       string
		bookSide      domains.BookSide
		levels     []testLevel
		breakAfter int
		want       []string
	}{
		{
			name:  "ask iterates in ascending order",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			breakAfter: 99,
			want:       []string{"99", "100", "101"},
		},
		{
			name:  "bid iterates in descending order",
			bookSide: domains.BookSideBid,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			breakAfter: 99,
			want:       []string{"101", "100", "99"},
		},
		{
			name:  "iteration stops when callback returns false",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			breakAfter: 2,
			want:       []string{"99", "100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestPriceLevelMap(t, tt.bookSide, tt.levels)

			var got []string
			count := 0
			m.SortedRange(func(price decimal.Decimal, record PriceLevel) bool {
				got = append(got, price.String())
				count++
				return count < tt.breakAfter
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriceLevelMap_SumVolume(t *testing.T) {
	tests := []struct {
		name   string
		bookSide  domains.BookSide
		levels []testLevel
		price  string
		want   string
	}{
		{
			name:  "ask returns cumulative volume up to the specified price",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "100", volume: "1"},
				{price: "101", volume: "2"},
				{price: "102", volume: "3"},
			},
			price: "101",
			want:  "3",
		},
		{
			name:  "ask returns zero when the specified price is already fully marketable",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "100", volume: "1"},
				{price: "101", volume: "2"},
				{price: "102", volume: "3"},
			},
			price: "99",
			want:  "0",
		},
		{
			name:  "bid returns cumulative volume up to the specified price",
			bookSide: domains.BookSideBid,
			levels: []testLevel{
				{price: "102", volume: "1"},
				{price: "101", volume: "2"},
				{price: "100", volume: "3"},
			},
			price: "101",
			want:  "3",
		},
		{
			name:  "bid returns zero when the specified price is already fully marketable",
			bookSide: domains.BookSideBid,
			levels: []testLevel{
				{price: "102", volume: "1"},
				{price: "101", volume: "2"},
				{price: "100", volume: "3"},
			},
			price: "103",
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestPriceLevelMap(t, tt.bookSide, tt.levels)

			got := m.SumVolume(mustDecimal(t, tt.price))

			if got, want := got.String(), tt.want; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestPriceLevelMap_AvgExecPrice(t *testing.T) {
	tests := []struct {
		name   string
		bookSide  domains.BookSide
		levels []testLevel
		qty    string
		want   string
	}{
		{
			name:  "ask returns average execution price",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "100", volume: "1"},
				{price: "101", volume: "2"},
				{price: "102", volume: "3"},
			},
			qty:  "2.5",
			want: "100.6", // (100*1 + 101*1.5) / 2.5
		},
		{
			name:  "bid returns average execution price",
			bookSide: domains.BookSideBid,
			levels: []testLevel{
				{price: "102", volume: "1"},
				{price: "101", volume: "2"},
				{price: "100", volume: "3"},
			},
			qty:  "2.5",
			want: "101.4", // (102*1 + 101*1.5) / 2.5
		},
		{
			name:  "returns average execution price when qty exactly matches total available volume",
			bookSide: domains.BookSideAsk,
			levels: []testLevel{
				{price: "100", volume: "1"},
				{price: "101", volume: "2"},
			},
			qty:  "3",
			want: "100.6666666666666667",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestPriceLevelMap(t, tt.bookSide, tt.levels)

			got := m.AvgExecPrice(mustDecimal(t, tt.qty))

			if got, want := got.String(), tt.want; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestPriceLevelMap_BestLevel(t *testing.T) {
	tests := []struct {
		name  string
		bookSide domains.BookSide
		setup []testLevel
		drop  string
		want  string
	}{
		{
			name:  "ask returns the lowest price as best",
			bookSide: domains.BookSideAsk,
			setup: []testLevel{
				{price: "101", volume: "1"},
				{price: "100", volume: "2"},
				{price: "102", volume: "3"},
			},
			want: "100:2",
		},
		{
			name:  "bid returns the highest price as best",
			bookSide: domains.BookSideBid,
			setup: []testLevel{
				{price: "101", volume: "1"},
				{price: "100", volume: "2"},
				{price: "102", volume: "3"},
			},
			want: "102:3",
		},
		{
			name:  "recalculates best record after dropping current best",
			bookSide: domains.BookSideAsk,
			setup: []testLevel{
				{price: "100", volume: "1"},
				{price: "101", volume: "2"},
			},
			drop: "100",
			want: "101:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestPriceLevelMap(t, tt.bookSide, tt.setup)

			if tt.drop != "" {
				m.drop(mustDecimal(t, tt.drop))
			}

			got := m.BestLevel()
			if got, want := got.Price.String()+":"+got.Volume.String(), tt.want; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

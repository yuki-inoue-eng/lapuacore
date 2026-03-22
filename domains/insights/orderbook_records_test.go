package insights

import (
	"testing"

	"github.com/bmizerany/assert"
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

func newTestOBRecordMap(t *testing.T, quote domains.Quote, levels []testLevel) *OBRecordMap {
	t.Helper()

	m := newOBRecordMap(quote, mustDecimal(t, "0.1"))
	for i, lv := range levels {
		m.set(OBRecord{
			SeqID:  int64(i + 1),
			Price:  mustDecimal(t, lv.price),
			Volume: mustDecimal(t, lv.volume),
		})
	}
	return m
}

func decimalStrings(values []decimal.Decimal) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.String())
	}
	return out
}

func TestOBRecordMap_sortedKeys(t *testing.T) {
	tests := []struct {
		name   string
		quote  domains.Quote
		levels []testLevel
		want   []string
	}{
		{
			name:  "ask sorts prices in ascending order",
			quote: domains.QuoteAsk,
			levels: []testLevel{
				{price: "101", volume: "1"},
				{price: "99", volume: "2"},
				{price: "100", volume: "3"},
			},
			want: []string{"99", "100", "101"},
		},
		{
			name:  "bid sorts prices in descending order",
			quote: domains.QuoteBid,
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
			m := newTestOBRecordMap(t, tt.quote, tt.levels)

			got := decimalStrings(m.sortedKeys())

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOBRecordMap_SortedRange(t *testing.T) {
	tests := []struct {
		name       string
		quote      domains.Quote
		levels     []testLevel
		breakAfter int
		want       []string
	}{
		{
			name:  "ask iterates in ascending order",
			quote: domains.QuoteAsk,
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
			quote: domains.QuoteBid,
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
			quote: domains.QuoteAsk,
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
			m := newTestOBRecordMap(t, tt.quote, tt.levels)

			var got []string
			count := 0
			m.SortedRange(func(price decimal.Decimal, record OBRecord) bool {
				got = append(got, price.String())
				count++
				return count < tt.breakAfter
			})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOBRecordMap_SumVolume(t *testing.T) {
	tests := []struct {
		name   string
		quote  domains.Quote
		levels []testLevel
		price  string
		want   string
	}{
		{
			name:  "ask returns cumulative volume up to the specified price",
			quote: domains.QuoteAsk,
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
			quote: domains.QuoteAsk,
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
			quote: domains.QuoteBid,
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
			quote: domains.QuoteBid,
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
			m := newTestOBRecordMap(t, tt.quote, tt.levels)

			got := m.SumVolume(mustDecimal(t, tt.price))

			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestOBRecordMap_AvgExecPrice(t *testing.T) {
	tests := []struct {
		name   string
		quote  domains.Quote
		levels []testLevel
		qty    string
		want   string
	}{
		{
			name:  "ask returns average execution price",
			quote: domains.QuoteAsk,
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
			quote: domains.QuoteBid,
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
			quote: domains.QuoteAsk,
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
			m := newTestOBRecordMap(t, tt.quote, tt.levels)

			got := m.AvgExecPrice(mustDecimal(t, tt.qty))

			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestOBRecordMap_BestRecord(t *testing.T) {
	tests := []struct {
		name  string
		quote domains.Quote
		setup []testLevel
		drop  string
		want  string
	}{
		{
			name:  "ask returns the lowest price as best",
			quote: domains.QuoteAsk,
			setup: []testLevel{
				{price: "101", volume: "1"},
				{price: "100", volume: "2"},
				{price: "102", volume: "3"},
			},
			want: "100:2",
		},
		{
			name:  "bid returns the highest price as best",
			quote: domains.QuoteBid,
			setup: []testLevel{
				{price: "101", volume: "1"},
				{price: "100", volume: "2"},
				{price: "102", volume: "3"},
			},
			want: "102:3",
		},
		{
			name:  "recalculates best record after dropping current best",
			quote: domains.QuoteAsk,
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
			m := newTestOBRecordMap(t, tt.quote, tt.setup)

			if tt.drop != "" {
				m.drop(mustDecimal(t, tt.drop))
			}

			got := m.BestRecord()
			assert.Equal(t, tt.want, got.Price.String()+":"+got.Volume.String())
		})
	}
}

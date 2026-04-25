package display

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
)

func setupOrderBook(t *testing.T, asks, bids []insights.PriceLevel) insights.OrderBook {
	t.Helper()
	symbol := domains.SymbolCoinExFuturesBTCUSDT
	ob := insights.NewOrderBook(symbol)
	now := time.Now()
	ob.UpdateByOBData(&insights.OrderBookData{
		Type:      insights.DataTypeSnapshot,
		ExecAt:    now,
		ArrivedAt: now,
		Asks:      asks,
		Bids:      bids,
		SeqID:     1,
	})
	return ob
}

func mustDecimal(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func TestVisibleLen(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"plain text", "hello", 5},
		{"empty string", "", 0},
		{"with ANSI red", "\033[31mhello\033[0m", 5},
		{"with ANSI green", "\033[32mBid 100.00: 0.5\033[0m", 15},
		{"no ANSI", "  Ask 100.00: 0.5", 17},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := visibleLen(tt.s), tt.want; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestBuildBookLines_WaitingForData(t *testing.T) {
	symbol := domains.SymbolCoinExFuturesBTCUSDT
	ob := insights.NewOrderBook(symbol)

	d := NewBookDisplay(3, []BookEntry{{Title: "BTC/USDT", OB: ob}})
	lines, width := d.buildBookLines("BTC/USDT", ob)

	if got, want := len(lines), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := lines[0], "BTC/USDT: waiting for data..."; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := width, len("BTC/USDT: waiting for data..."); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildBookLines_Structure(t *testing.T) {
	asks := []insights.PriceLevel{
		{SeqID: 1, Price: mustDecimal("100.00"), Volume: mustDecimal("0.5")},
		{SeqID: 1, Price: mustDecimal("101.00"), Volume: mustDecimal("0.3")},
		{SeqID: 1, Price: mustDecimal("102.00"), Volume: mustDecimal("0.2")},
	}
	bids := []insights.PriceLevel{
		{SeqID: 1, Price: mustDecimal("99.00"), Volume: mustDecimal("0.4")},
		{SeqID: 1, Price: mustDecimal("98.00"), Volume: mustDecimal("0.6")},
	}
	ob := setupOrderBook(t, asks, bids)

	d := NewBookDisplay(3, []BookEntry{{Title: "TEST", OB: ob}})
	lines, _ := d.buildBookLines("TEST", ob)

	// Structure: header, 3 asks (reversed), spread, 2 bids, empty line
	// Total: 1 + 3 + 1 + 2 + 1 = 8
	if got, want := len(lines), 8; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Header contains title
	if !(visibleLen(lines[0]) > 0) {
		t.Error("header should not be empty")
	}

	// Asks are in reverse order (highest first)
	if !containsVisible(lines[1], "Ask 102.00") {
		t.Error("expected true")
	}
	if !containsVisible(lines[2], "Ask 101.00") {
		t.Error("expected true")
	}
	if !containsVisible(lines[3], "Ask 100.00") {
		t.Error("expected true")
	}

	// Spread line
	if !containsVisible(lines[4], "spread(1.00)") {
		t.Error("expected true")
	}

	// Bids in best-first order
	if !containsVisible(lines[5], "Bid 99.00") {
		t.Error("expected true")
	}
	if !containsVisible(lines[6], "Bid 98.00") {
		t.Error("expected true")
	}

	// Last line is empty
	if got, want := lines[7], ""; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildBookLines_WidthConsistency(t *testing.T) {
	asks := []insights.PriceLevel{
		{SeqID: 1, Price: mustDecimal("74365.00"), Volume: mustDecimal("0.0672")},
		{SeqID: 1, Price: mustDecimal("74366.00"), Volume: mustDecimal("0.0699")},
		{SeqID: 1, Price: mustDecimal("74367.00"), Volume: mustDecimal("1.1344")},
	}
	bids := []insights.PriceLevel{
		{SeqID: 1, Price: mustDecimal("74364.00"), Volume: mustDecimal("0.3531")},
		{SeqID: 1, Price: mustDecimal("74363.00"), Volume: mustDecimal("0.0675")},
	}
	ob := setupOrderBook(t, asks, bids)

	d := NewBookDisplay(3, []BookEntry{{Title: "BTC/USDT", OB: ob}})
	lines, maxWidth := d.buildBookLines("BTC/USDT", ob)

	// Header and spread should match data width
	if got, want := visibleLen(lines[0]), maxWidth; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Find spread line (after asks)
	spreadIdx := 1 + len(asks) // header + asks
	if got, want := visibleLen(lines[spreadIdx]), maxWidth; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNewBookDisplay_Books(t *testing.T) {
	symbol := domains.SymbolCoinExFuturesBTCUSDT
	ob := insights.NewOrderBook(symbol)

	entries := []BookEntry{
		{Title: "BTC/USDT", OB: ob},
		{Title: "SOL/USDT", OB: ob},
	}
	d := NewBookDisplay(5, entries)

	if got, want := len(d.Books()), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := d.Books()[0].Title, "BTC/USDT"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := d.Books()[1].Title, "SOL/USDT"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// containsVisible strips ANSI codes and checks if the substring is present.
func containsVisible(s, substr string) bool {
	plain := stripANSI(s)
	return len(plain) > 0 && len(substr) > 0 && contains(plain, substr)
}

func stripANSI(s string) string {
	var b []rune
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b = append(b, r)
	}
	return string(b)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

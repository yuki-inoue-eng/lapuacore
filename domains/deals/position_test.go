package deals

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func TestPosSide(t *testing.T) {
	tests := []struct {
		name string
		pos  string
		want domains.Side
	}{
		{
			name: "positive returns Buy",
			pos:  "1.5",
			want: domains.SideBuy,
		},
		{
			name: "negative returns Sell",
			pos:  "-2.0",
			want: domains.SideSell,
		},
		{
			name: "zero returns None",
			pos:  "0",
			want: domains.SideNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := PosSide(decimal.RequireFromString(tt.pos)), tt.want; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestPosition_GetQty(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		wantSide domains.Side
		wantQty  string
	}{
		{
			name:     "positive position",
			value:    "3.5",
			wantSide: domains.SideBuy,
			wantQty:  "3.5",
		},
		{
			name:     "negative position",
			value:    "-2.0",
			wantSide: domains.SideSell,
			wantQty:  "2.0",
		},
		{
			name:     "zero position",
			value:    "0",
			wantSide: domains.SideNone,
			wantQty:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPosition()
			p.value = decimal.RequireFromString(tt.value)

			side, qty := p.GetQty()
			if got, want := side, tt.wantSide; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if !qty.Equal(decimal.RequireFromString(tt.wantQty)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

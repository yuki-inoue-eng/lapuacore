package insights

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func makeTradeData(side domains.Side, price, volume string) *TradeData {
	return &TradeData{
		ExecAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ArrivedAt: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		Side:      side,
		Volume:    decimal.RequireFromString(volume),
		Price:     decimal.RequireFromString(price),
	}
}

func TestTradeData_GetSize(t *testing.T) {
	tests := []struct {
		name     string
		side     domains.Side
		volume   string
		wantSize string
	}{
		{
			name:     "buy returns positive size",
			side:     domains.SideBuy,
			volume:   "1.5",
			wantSize: "1.5",
		},
		{
			name:     "sell returns negative size",
			side:     domains.SideSell,
			volume:   "2.0",
			wantSize: "-2.0",
		},
		{
			name:     "none returns zero",
			side:     domains.SideNone,
			volume:   "3.0",
			wantSize: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := makeTradeData(tt.side, "100", tt.volume)
			if got := td.GetSize().Equal(mustDecimal(t, tt.wantSize)); !got {
				t.Errorf("got %v, want true", got)
			}
		})
	}
}

func TestTradeDataList_Aggregation(t *testing.T) {
	tests := []struct {
		name        string
		list        TradeDataList
		wantSumSize string
		wantSellVol string
		wantBuyVol  string
	}{
		{
			name: "mixed buy and sell",
			list: TradeDataList{
				makeTradeData(domains.SideBuy, "100", "1.0"),
				makeTradeData(domains.SideSell, "101", "0.5"),
				makeTradeData(domains.SideBuy, "99", "2.0"),
			},
			wantSumSize: "2.5", // 1.0 + (-0.5) + 2.0
			wantSellVol: "0.5",
			wantBuyVol:  "3.0",
		},
		{
			name: "all buy",
			list: TradeDataList{
				makeTradeData(domains.SideBuy, "100", "1.0"),
				makeTradeData(domains.SideBuy, "100", "2.0"),
			},
			wantSumSize: "3.0",
			wantSellVol: "0",
			wantBuyVol:  "3.0",
		},
		{
			name: "all sell",
			list: TradeDataList{
				makeTradeData(domains.SideSell, "100", "1.0"),
				makeTradeData(domains.SideSell, "100", "2.0"),
			},
			wantSumSize: "-3.0",
			wantSellVol: "3.0",
			wantBuyVol:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.list.GetSumSize().Equal(mustDecimal(t, tt.wantSumSize)) {
				t.Errorf("got %v, want true", false)
			}
			if !tt.list.GetSumSellVolume().Equal(mustDecimal(t, tt.wantSellVol)) {
				t.Errorf("got %v, want true", false)
			}
			if !tt.list.GetSumBuyVolume().Equal(mustDecimal(t, tt.wantBuyVol)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestTradeDataList_GetTimes(t *testing.T) {
	execAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	arrivedAt := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)

	list := TradeDataList{
		{ExecAt: execAt, ArrivedAt: arrivedAt, Side: domains.SideBuy, Volume: decimal.NewFromInt(1), Price: decimal.NewFromInt(100)},
		{ExecAt: execAt.Add(time.Second), ArrivedAt: arrivedAt.Add(time.Second), Side: domains.SideSell, Volume: decimal.NewFromInt(1), Price: decimal.NewFromInt(100)},
	}

	if got, want := list.GetExecAt(), execAt; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := list.GetArrivedAt(), arrivedAt; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTrade_IsReady(t *testing.T) {
	tests := []struct {
		name      string
		updates   int
		wantReady bool
	}{
		{
			name:      "not ready before any update",
			updates:   0,
			wantReady: false,
		},
		{
			name:      "ready after first update",
			updates:   1,
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTrade(domains.SymbolCoinExFuturesBTCUSDT)
			for i := 0; i < tt.updates; i++ {
				tr.Update(TradeDataList{makeTradeData(domains.SideBuy, "100", "1")})
			}
			if got, want := tr.IsReady(), tt.wantReady; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestTrade_SetUpdateCallback(t *testing.T) {
	tests := []struct {
		name         string
		handlerCount int
	}{
		{
			name:         "single handler is invoked",
			handlerCount: 1,
		},
		{
			name:         "multiple handlers are all invoked",
			handlerCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTrade(domains.SymbolCoinExFuturesBTCUSDT)
			counts := make([]int, tt.handlerCount)
			for i := 0; i < tt.handlerCount; i++ {
				idx := i
				tr.SetUpdateCallback(func(msg TradeDataList) {
					counts[idx]++
				})
			}

			tr.Update(TradeDataList{makeTradeData(domains.SideBuy, "100", "1")})

			for _, c := range counts {
				if got, want := c, 1; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
		})
	}
}

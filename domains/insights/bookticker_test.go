package insights

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func makeBookTickerData(seqID int64, execAt time.Time, askPrice, askVol, bidPrice, bidVol string) *BookTickerData {
	return &BookTickerData{
		SeqID:     seqID,
		ExecAt:    execAt,
		EventAt:   execAt,
		ArrivedAt: execAt,
		BestAsk: &OBRecord{
			SeqID:  seqID,
			Price:  decimal.RequireFromString(askPrice),
			Volume: decimal.RequireFromString(askVol),
		},
		BestBid: &OBRecord{
			SeqID:  seqID,
			Price:  decimal.RequireFromString(bidPrice),
			Volume: decimal.RequireFromString(bidVol),
		},
	}
}

func TestBookTicker_Update(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		messages       []*BookTickerData
		wantAskPrice   string
		wantBidPrice   string
		wantSeqID      int64
		wantExecAt     time.Time
		wantArrivedAt  time.Time
		wantBeforeAsk  string // empty means nil
		wantBeforeBid  string // empty means nil
	}{
		{
			name: "first update sets best prices",
			messages: []*BookTickerData{
				makeBookTickerData(1, baseTime, "100.50", "10", "100.40", "20"),
			},
			wantAskPrice:  "100.50",
			wantBidPrice:  "100.40",
			wantSeqID:     1,
			wantExecAt:    baseTime,
			wantArrivedAt: baseTime,
		},
		{
			name: "second update records before prices",
			messages: []*BookTickerData{
				makeBookTickerData(1, baseTime, "100.50", "10", "100.40", "20"),
				makeBookTickerData(2, baseTime.Add(time.Second), "100.60", "5", "100.45", "15"),
			},
			wantAskPrice:  "100.60",
			wantBidPrice:  "100.45",
			wantSeqID:     2,
			wantExecAt:    baseTime.Add(time.Second),
			wantArrivedAt: baseTime.Add(time.Second),
			wantBeforeAsk: "100.50",
			wantBeforeBid: "100.40",
		},
		{
			name: "old ExecAt is ignored",
			messages: []*BookTickerData{
				makeBookTickerData(1, baseTime.Add(time.Second), "100.50", "10", "100.40", "20"),
				makeBookTickerData(2, baseTime, "999", "1", "998", "1"), // older ExecAt
			},
			wantAskPrice:  "100.50",
			wantBidPrice:  "100.40",
			wantSeqID:     1,
			wantExecAt:    baseTime.Add(time.Second),
			wantArrivedAt: baseTime.Add(time.Second),
		},
		{
			name: "old SeqID is ignored",
			messages: []*BookTickerData{
				makeBookTickerData(5, baseTime, "100.50", "10", "100.40", "20"),
				makeBookTickerData(3, baseTime.Add(time.Second), "999", "1", "998", "1"), // older SeqID
			},
			wantAskPrice:  "100.50",
			wantBidPrice:  "100.40",
			wantSeqID:     5,
			wantExecAt:    baseTime,
			wantArrivedAt: baseTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT)
			for _, msg := range tt.messages {
				bt.Update(msg)
			}

			assert.Equal(t, true, bt.GetBestAsk().Price.Equal(mustDecimal(t, tt.wantAskPrice)))
			assert.Equal(t, true, bt.GetBestBid().Price.Equal(mustDecimal(t, tt.wantBidPrice)))
			assert.Equal(t, tt.wantSeqID, bt.GetSeqID())
			assert.Equal(t, tt.wantExecAt, *bt.GetLastExecAt())
			assert.Equal(t, tt.wantArrivedAt, *bt.GetLastArrivedAt())

			if tt.wantBeforeAsk != "" {
				bt.mu.RLock()
				assert.Equal(t, true, bt.beforeBestAsk.Price.Equal(mustDecimal(t, tt.wantBeforeAsk)))
				bt.mu.RUnlock()
			}
			if tt.wantBeforeBid != "" {
				bt.mu.RLock()
				assert.Equal(t, true, bt.beforeBestBid.Price.Equal(mustDecimal(t, tt.wantBeforeBid)))
				bt.mu.RUnlock()
			}
		})
	}
}

func TestBookTicker_IsReady(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		updates  int
		wantReady bool
	}{
		{
			name:      "not ready after zero updates",
			updates:   0,
			wantReady: false,
		},
		{
			name:      "not ready after one update",
			updates:   1,
			wantReady: false,
		},
		{
			name:      "ready after two updates",
			updates:   2,
			wantReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT)
			for i := 0; i < tt.updates; i++ {
				bt.Update(makeBookTickerData(
					int64(i+1),
					baseTime.Add(time.Duration(i)*time.Second),
					"100.50", "10", "100.40", "20",
				))
			}

			assert.Equal(t, tt.wantReady, bt.IsReady())
		})
	}
}

func TestBookTicker_GetDiff(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		first         *BookTickerData
		second        *BookTickerData
		wantDiffAsk   string
		wantDiffBid   string
		wantDiffAskVol string
		wantDiffBidVol string
	}{
		{
			name:           "diff best ask",
			first:          makeBookTickerData(1, baseTime, "100.50", "10", "100.40", "20"),
			second:         makeBookTickerData(2, baseTime.Add(time.Second), "100.60", "8", "100.45", "15"),
			wantDiffAsk:    "0.10",
			wantDiffBid:    "0.05",
			wantDiffAskVol: "-2",
			wantDiffBidVol: "-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT)
			bt.Update(tt.first)
			bt.Update(tt.second)

			diffAsk := bt.GetDiffBestAsk()
			diffBid := bt.GetDiffBestBid()

			assert.Equal(t, true, diffAsk.Price.Equal(mustDecimal(t, tt.wantDiffAsk)))
			assert.Equal(t, true, diffAsk.Volume.Equal(mustDecimal(t, tt.wantDiffAskVol)))
			assert.Equal(t, true, diffBid.Price.Equal(mustDecimal(t, tt.wantDiffBid)))
			assert.Equal(t, true, diffBid.Volume.Equal(mustDecimal(t, tt.wantDiffBidVol)))
		})
	}
}

func TestBookTicker_CalcBestPrice(t *testing.T) {
	bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT) // tickSize = 0.01

	tests := []struct {
		name        string
		midPrice    string
		wantBestAsk string
		wantBestBid string
	}{
		{
			name:        "midPrice is an exact tick multiple",
			midPrice:    "100.12",
			wantBestAsk: "100.13",
			wantBestBid: "100.12",
		},
		{
			name:        "midPrice is not a tick multiple",
			midPrice:    "100.129",
			wantBestAsk: "100.13",
			wantBestBid: "100.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAsk, gotBid := bt.CalcBestPrice(mustDecimal(t, tt.midPrice))

			assert.Equal(t, true, gotAsk.Equal(mustDecimal(t, tt.wantBestAsk)))
			assert.Equal(t, true, gotBid.Equal(mustDecimal(t, tt.wantBestBid)))
		})
	}
}

func TestBookTicker_RoundToTickSize(t *testing.T) {
	bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT) // tickSize = 0.01

	tests := []struct {
		name  string
		price string
		want  string
	}{
		{
			name:  "exact multiple unchanged",
			price: "100.12",
			want:  "100.12",
		},
		{
			name:  "not a multiple is truncated",
			price: "100.129",
			want:  "100.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bt.RoundToTickSize(mustDecimal(t, tt.price))

			assert.Equal(t, true, got.Equal(mustDecimal(t, tt.want)))
		})
	}
}

func TestBookTicker_Callback(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		callbackCount int
	}{
		{
			name:          "callback is invoked on update",
			callbackCount: 1,
		},
		{
			name:          "multiple callbacks are all invoked",
			callbackCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT)
			counts := make([]int, tt.callbackCount)
			for i := 0; i < tt.callbackCount; i++ {
				idx := i
				bt.SetUpdateCallback(func() {
					counts[idx]++
				})
			}

			bt.Update(makeBookTickerData(1, baseTime, "100", "1", "99", "1"))

			for _, c := range counts {
				assert.Equal(t, 1, c)
			}
		})
	}
}

func TestBookTicker_Getters(t *testing.T) {
	bt := NewBookTicker(domains.SymbolCoinExFuturesBTCUSDT)

	assert.Equal(t, true, bt.GetTickSize().Equal(mustDecimal(t, "0.01")))
	assert.Equal(t, true, bt.GetMinOrderQty().Equal(mustDecimal(t, "0.0001")))
}

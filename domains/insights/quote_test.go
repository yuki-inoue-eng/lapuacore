package insights

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func makeQuoteData(seqID int64, execAt time.Time, askPrice, askVol, bidPrice, bidVol string) *QuoteData {
	return &QuoteData{
		SeqID:     seqID,
		ExecAt:    execAt,
		EventAt:   execAt,
		ArrivedAt: execAt,
		BestAsk: &PriceLevel{
			SeqID:  seqID,
			Price:  decimal.RequireFromString(askPrice),
			Volume: decimal.RequireFromString(askVol),
		},
		BestBid: &PriceLevel{
			SeqID:  seqID,
			Price:  decimal.RequireFromString(bidPrice),
			Volume: decimal.RequireFromString(bidVol),
		},
	}
}

func TestQuote_Update(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		messages      []*QuoteData
		wantAskPrice  string
		wantBidPrice  string
		wantSeqID     int64
		wantExecAt    time.Time
		wantArrivedAt time.Time
		wantBeforeAsk string // empty means nil
		wantBeforeBid string // empty means nil
	}{
		{
			name: "first update sets best prices",
			messages: []*QuoteData{
				makeQuoteData(1, baseTime, "100.50", "10", "100.40", "20"),
			},
			wantAskPrice:  "100.50",
			wantBidPrice:  "100.40",
			wantSeqID:     1,
			wantExecAt:    baseTime,
			wantArrivedAt: baseTime,
		},
		{
			name: "second update records before prices",
			messages: []*QuoteData{
				makeQuoteData(1, baseTime, "100.50", "10", "100.40", "20"),
				makeQuoteData(2, baseTime.Add(time.Second), "100.60", "5", "100.45", "15"),
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
			messages: []*QuoteData{
				makeQuoteData(1, baseTime.Add(time.Second), "100.50", "10", "100.40", "20"),
				makeQuoteData(2, baseTime, "999", "1", "998", "1"), // older ExecAt
			},
			wantAskPrice:  "100.50",
			wantBidPrice:  "100.40",
			wantSeqID:     1,
			wantExecAt:    baseTime.Add(time.Second),
			wantArrivedAt: baseTime.Add(time.Second),
		},
		{
			name: "old SeqID is ignored",
			messages: []*QuoteData{
				makeQuoteData(5, baseTime, "100.50", "10", "100.40", "20"),
				makeQuoteData(3, baseTime.Add(time.Second), "999", "1", "998", "1"), // older SeqID
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
			q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT)
			for _, msg := range tt.messages {
				q.Update(msg)
			}

			assert.Equal(t, true, q.GetBestAsk().Price.Equal(mustDecimal(t, tt.wantAskPrice)))
			assert.Equal(t, true, q.GetBestBid().Price.Equal(mustDecimal(t, tt.wantBidPrice)))
			assert.Equal(t, tt.wantSeqID, q.GetSeqID())
			assert.Equal(t, tt.wantExecAt, *q.GetLastExecAt())
			assert.Equal(t, tt.wantArrivedAt, *q.GetLastArrivedAt())

			if tt.wantBeforeAsk != "" {
				q.mu.RLock()
				assert.Equal(t, true, q.beforeBestAsk.Price.Equal(mustDecimal(t, tt.wantBeforeAsk)))
				q.mu.RUnlock()
			}
			if tt.wantBeforeBid != "" {
				q.mu.RLock()
				assert.Equal(t, true, q.beforeBestBid.Price.Equal(mustDecimal(t, tt.wantBeforeBid)))
				q.mu.RUnlock()
			}
		})
	}
}

func TestQuote_IsReady(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		updates   int
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
			q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT)
			for i := 0; i < tt.updates; i++ {
				q.Update(makeQuoteData(
					int64(i+1),
					baseTime.Add(time.Duration(i)*time.Second),
					"100.50", "10", "100.40", "20",
				))
			}

			assert.Equal(t, tt.wantReady, q.IsReady())
		})
	}
}

func TestQuote_GetDiff(t *testing.T) {
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		first          *QuoteData
		second         *QuoteData
		wantDiffAsk    string
		wantDiffBid    string
		wantDiffAskVol string
		wantDiffBidVol string
	}{
		{
			name:           "diff best ask",
			first:          makeQuoteData(1, baseTime, "100.50", "10", "100.40", "20"),
			second:         makeQuoteData(2, baseTime.Add(time.Second), "100.60", "8", "100.45", "15"),
			wantDiffAsk:    "0.10",
			wantDiffBid:    "0.05",
			wantDiffAskVol: "-2",
			wantDiffBidVol: "-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT)
			q.Update(tt.first)
			q.Update(tt.second)

			diffAsk := q.GetDiffBestAsk()
			diffBid := q.GetDiffBestBid()

			assert.Equal(t, true, diffAsk.Price.Equal(mustDecimal(t, tt.wantDiffAsk)))
			assert.Equal(t, true, diffAsk.Volume.Equal(mustDecimal(t, tt.wantDiffAskVol)))
			assert.Equal(t, true, diffBid.Price.Equal(mustDecimal(t, tt.wantDiffBid)))
			assert.Equal(t, true, diffBid.Volume.Equal(mustDecimal(t, tt.wantDiffBidVol)))
		})
	}
}

func TestQuote_CalcBestPrice(t *testing.T) {
	q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT) // tickSize = 0.01

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
			gotAsk, gotBid := q.CalcBestPrice(mustDecimal(t, tt.midPrice))

			assert.Equal(t, true, gotAsk.Equal(mustDecimal(t, tt.wantBestAsk)))
			assert.Equal(t, true, gotBid.Equal(mustDecimal(t, tt.wantBestBid)))
		})
	}
}

func TestQuote_RoundToTickSize(t *testing.T) {
	q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT) // tickSize = 0.01

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
			got := q.RoundToTickSize(mustDecimal(t, tt.price))

			assert.Equal(t, true, got.Equal(mustDecimal(t, tt.want)))
		})
	}
}

func TestQuote_Callback(t *testing.T) {
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
			q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT)
			counts := make([]int, tt.callbackCount)
			for i := 0; i < tt.callbackCount; i++ {
				idx := i
				q.SetUpdateCallback(func() {
					counts[idx]++
				})
			}

			q.Update(makeQuoteData(1, baseTime, "100", "1", "99", "1"))

			for _, c := range counts {
				assert.Equal(t, 1, c)
			}
		})
	}
}

func TestQuote_Getters(t *testing.T) {
	q := NewQuote(domains.SymbolCoinExFuturesBTCUSDT)

	assert.Equal(t, true, q.GetTickSize().Equal(mustDecimal(t, "0.01")))
	assert.Equal(t, true, q.GetMinOrderQty().Equal(mustDecimal(t, "0.0001")))
}

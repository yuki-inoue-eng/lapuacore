package insights

import (
	"fmt"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

type orderBookHandlerLevel struct {
	seqID  int64
	price  string
	volume string
}

func orderBookHandlerMustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()

	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("invalid decimal %q: %v", s, err)
	}
	return d
}

func orderBookHandlerMakeRecords(t *testing.T, levels []orderBookHandlerLevel) []OBRecord {
	t.Helper()

	out := make([]OBRecord, 0, len(levels))
	for _, lv := range levels {
		out = append(out, OBRecord{
			SeqID:  lv.seqID,
			Price:  orderBookHandlerMustDecimal(t, lv.price),
			Volume: orderBookHandlerMustDecimal(t, lv.volume),
		})
	}
	return out
}

func orderBookHandlerSeedBook(t *testing.T, book *OrderBook, asks []orderBookHandlerLevel, bids []orderBookHandlerLevel) {
	t.Helper()

	for _, r := range orderBookHandlerMakeRecords(t, asks) {
		book.AsksMap.set(r)
	}
	for _, r := range orderBookHandlerMakeRecords(t, bids) {
		book.BidsMap.set(r)
	}
}

func orderBookHandlerCollectLevels(m *OBRecordMap) []string {
	var out []string
	m.SortedRange(func(price decimal.Decimal, record OBRecord) bool {
		out = append(out, fmt.Sprintf("%s:%s:%d", record.Price.String(), record.Volume.String(), record.SeqID))
		return true
	})
	return out
}

func orderBookHandlerFormatRecord(r *OBRecord) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s:%d", r.Price.String(), r.Volume.String(), r.SeqID)
}

func TestOrderBook_updateByDelta(t *testing.T) {
	tests := []struct {
		name        string
		initialAsks []orderBookHandlerLevel
		initialBids []orderBookHandlerLevel
		deltaAsks   []orderBookHandlerLevel
		deltaBids   []orderBookHandlerLevel
		wantAsks    []string
		wantBids    []string
	}{
		{
			name: "applies newer deltas and removes zero-volume levels",
			initialAsks: []orderBookHandlerLevel{
				{seqID: 1, price: "100", volume: "1"},
				{seqID: 1, price: "101", volume: "3"},
			},
			initialBids: []orderBookHandlerLevel{
				{seqID: 1, price: "99", volume: "2"},
				{seqID: 1, price: "98", volume: "4"},
			},
			deltaAsks: []orderBookHandlerLevel{
				{seqID: 2, price: "100", volume: "5"},
				{seqID: 2, price: "102", volume: "1"},
			},
			deltaBids: []orderBookHandlerLevel{
				{seqID: 2, price: "99", volume: "0"},
			},
			wantAsks: []string{
				"100:5:2",
				"101:3:1",
				"102:1:2",
			},
			wantBids: []string{
				"98:4:1",
			},
		},
		{
			name: "ignores stale or same-sequence deltas",
			initialAsks: []orderBookHandlerLevel{
				{seqID: 10, price: "100", volume: "1"},
			},
			initialBids: []orderBookHandlerLevel{
				{seqID: 10, price: "99", volume: "2"},
			},
			deltaAsks: []orderBookHandlerLevel{
				{seqID: 9, price: "100", volume: "5"},
			},
			deltaBids: []orderBookHandlerLevel{
				{seqID: 10, price: "99", volume: "0"},
			},
			wantAsks: []string{
				"100:1:10",
			},
			wantBids: []string{
				"99:2:10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookHandlerSeedBook(t, book, tt.initialAsks, tt.initialBids)

			book.updateByDelta(&OrderBookData{
				Type: DataTypeDelta,
				Asks: orderBookHandlerMakeRecords(t, tt.deltaAsks),
				Bids: orderBookHandlerMakeRecords(t, tt.deltaBids),
			})

			gotAsks := orderBookHandlerCollectLevels(book.AsksMap)
			gotBids := orderBookHandlerCollectLevels(book.BidsMap)

			assert.Equal(t, tt.wantAsks, gotAsks)
			assert.Equal(t, tt.wantBids, gotBids)
		})
	}
}

func TestOrderBook_resetBySnapshot(t *testing.T) {
	tests := []struct {
		name         string
		initialAsks  []orderBookHandlerLevel
		initialBids  []orderBookHandlerLevel
		snapshotAsks []orderBookHandlerLevel
		snapshotBids []orderBookHandlerLevel
		wantAsks     []string
		wantBids     []string
	}{
		{
			name: "replaces each side with snapshot data and keeps newer records on matching prices",
			initialAsks: []orderBookHandlerLevel{
				{seqID: 10, price: "100", volume: "9"},
				{seqID: 10, price: "101", volume: "8"},
			},
			initialBids: []orderBookHandlerLevel{
				{seqID: 10, price: "99", volume: "7"},
				{seqID: 10, price: "98", volume: "6"},
			},
			snapshotAsks: []orderBookHandlerLevel{
				{seqID: 5, price: "100", volume: "1"},
				{seqID: 5, price: "102", volume: "2"},
			},
			snapshotBids: []orderBookHandlerLevel{
				{seqID: 5, price: "99", volume: "3"},
				{seqID: 5, price: "97", volume: "4"},
			},
			wantAsks: []string{
				"100:9:10",
				"102:2:5",
			},
			wantBids: []string{
				"99:7:10",
				"97:4:5",
			},
		},
		{
			name: "leaves a side unchanged when the snapshot for that side is empty",
			initialAsks: []orderBookHandlerLevel{
				{seqID: 10, price: "100", volume: "9"},
			},
			initialBids: []orderBookHandlerLevel{
				{seqID: 10, price: "99", volume: "7"},
			},
			snapshotAsks: []orderBookHandlerLevel{
				{seqID: 5, price: "101", volume: "2"},
			},
			snapshotBids: nil,
			wantAsks: []string{
				"101:2:5",
			},
			wantBids: []string{
				"99:7:10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)
			orderBookHandlerSeedBook(t, book, tt.initialAsks, tt.initialBids)

			book.resetBySnapshot(&OrderBookData{
				Type: DataTypeSnapshot,
				Asks: orderBookHandlerMakeRecords(t, tt.snapshotAsks),
				Bids: orderBookHandlerMakeRecords(t, tt.snapshotBids),
			})

			gotAsks := orderBookHandlerCollectLevels(book.AsksMap)
			gotBids := orderBookHandlerCollectLevels(book.BidsMap)

			assert.Equal(t, tt.wantAsks, gotAsks)
			assert.Equal(t, tt.wantBids, gotBids)
		})
	}
}

func TestOrderBook_UpdateByOBData(t *testing.T) {
	baseExecAt := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)
	baseArrivedAt := time.Date(2026, 3, 22, 10, 0, 1, 0, time.UTC)

	tests := []struct {
		name                string
		updates             []*OrderBookData
		wantReady           bool
		wantBestAsk         string
		wantBestBid         string
		wantBeforeBestAsk   string
		wantBeforeBestBid   string
		wantDiffBestAsk     string
		wantDiffBestBid     string
		wantUpdateCallCount int
		wantDeferCallCount  int
		wantLastExecAt      time.Time
		wantLastArrivedAt   time.Time
	}{
		{
			name: "first update sets current best prices but does not make the book ready yet",
			updates: []*OrderBookData{
				{
					Type:      DataTypeDelta,
					ExecAt:    baseExecAt,
					ArrivedAt: baseArrivedAt,
					Asks: []OBRecord{
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "101"), Volume: orderBookHandlerMustDecimal(t, "1")},
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "102"), Volume: orderBookHandlerMustDecimal(t, "2")},
					},
					Bids: []OBRecord{
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "100"), Volume: orderBookHandlerMustDecimal(t, "1")},
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "99"), Volume: orderBookHandlerMustDecimal(t, "2")},
					},
				},
			},
			wantReady:           false,
			wantBestAsk:         "101:1:1",
			wantBestBid:         "100:1:1",
			wantBeforeBestAsk:   "",
			wantBeforeBestBid:   "",
			wantDiffBestAsk:     "",
			wantDiffBestBid:     "",
			wantUpdateCallCount: 1,
			wantDeferCallCount:  1,
			wantLastExecAt:      baseExecAt,
			wantLastArrivedAt:   baseArrivedAt,
		},
		{
			name: "second update stores previous best prices and makes the book ready",
			updates: []*OrderBookData{
				{
					Type:      DataTypeDelta,
					ExecAt:    baseExecAt,
					ArrivedAt: baseArrivedAt,
					Asks: []OBRecord{
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "101"), Volume: orderBookHandlerMustDecimal(t, "1")},
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "102"), Volume: orderBookHandlerMustDecimal(t, "2")},
					},
					Bids: []OBRecord{
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "100"), Volume: orderBookHandlerMustDecimal(t, "1")},
						{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "99"), Volume: orderBookHandlerMustDecimal(t, "2")},
					},
				},
				{
					Type:      DataTypeDelta,
					ExecAt:    baseExecAt.Add(2 * time.Second),
					ArrivedAt: baseArrivedAt.Add(2 * time.Second),
					Asks: []OBRecord{
						{SeqID: 2, Price: orderBookHandlerMustDecimal(t, "101"), Volume: orderBookHandlerMustDecimal(t, "3")},
					},
					Bids: []OBRecord{
						{SeqID: 2, Price: orderBookHandlerMustDecimal(t, "100"), Volume: orderBookHandlerMustDecimal(t, "0")},
						{SeqID: 2, Price: orderBookHandlerMustDecimal(t, "99"), Volume: orderBookHandlerMustDecimal(t, "5")},
					},
				},
			},
			wantReady:           true,
			wantBestAsk:         "101:3:2",
			wantBestBid:         "99:5:2",
			wantBeforeBestAsk:   "101:1:1",
			wantBeforeBestBid:   "100:1:1",
			wantDiffBestAsk:     "0:2:1",
			wantDiffBestBid:     "-1:4:1",
			wantUpdateCallCount: 2,
			wantDeferCallCount:  2,
			wantLastExecAt:      baseExecAt.Add(2 * time.Second),
			wantLastArrivedAt:   baseArrivedAt.Add(2 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)

			updateCallCount := 0
			deferCallCount := 0

			book.SetUpdateCallback(func() {
				updateCallCount++
			})
			book.SetDeferUpdateCallBack(func() {
				deferCallCount++
			})

			for _, update := range tt.updates {
				book.UpdateByOBData(update)
			}

			assert.Equal(t, tt.wantReady, book.IsReady())
			assert.Equal(t, tt.wantUpdateCallCount, updateCallCount)
			assert.Equal(t, tt.wantDeferCallCount, deferCallCount)

			assert.Equal(t, tt.wantBestAsk, orderBookHandlerFormatRecord(book.GetBestAsk()))
			assert.Equal(t, tt.wantBestBid, orderBookHandlerFormatRecord(book.GetBestBid()))
			assert.Equal(t, tt.wantBeforeBestAsk, orderBookHandlerFormatRecord(book.beforeBestAsk))
			assert.Equal(t, tt.wantBeforeBestBid, orderBookHandlerFormatRecord(book.beforeBestBid))

			if tt.wantDiffBestAsk != "" {
				diffAsk := book.GetDiffBestAsk()
				assert.Equal(t, tt.wantDiffBestAsk, fmt.Sprintf("%s:%s:%d", diffAsk.Price.String(), diffAsk.Volume.String(), diffAsk.SeqID))
			}
			if tt.wantDiffBestBid != "" {
				diffBid := book.GetDiffBestBid()
				assert.Equal(t, tt.wantDiffBestBid, fmt.Sprintf("%s:%s:%d", diffBid.Price.String(), diffBid.Volume.String(), diffBid.SeqID))
			}

			gotLastExecAt := book.GetLastExecAt()
			gotLastArrivedAt := book.GetLastArrivedAt()

			assert.Equal(t, true, gotLastExecAt != nil)
			assert.Equal(t, true, gotLastArrivedAt != nil)
			assert.Equal(t, true, gotLastExecAt.Equal(tt.wantLastExecAt))
			assert.Equal(t, true, gotLastArrivedAt.Equal(tt.wantLastArrivedAt))
		})
	}
}

func TestOrderBook_DropDeferUpdateCallBack(t *testing.T) {
	tests := []struct {
		name               string
		dropBeforeUpdate   bool
		wantDeferCallCount int
	}{
		{
			name:               "defer callback is called when it is registered",
			dropBeforeUpdate:   false,
			wantDeferCallCount: 1,
		},
		{
			name:               "defer callback becomes a no-op after DropDeferUpdateCallBack",
			dropBeforeUpdate:   true,
			wantDeferCallCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book := NewOrderBook(domains.SymbolCoinExFuturesBTCUSDT)

			deferCallCount := 0
			book.SetDeferUpdateCallBack(func() {
				deferCallCount++
			})

			if tt.dropBeforeUpdate {
				book.DropDeferUpdateCallBack()
			}

			book.UpdateByOBData(&OrderBookData{
				Type:      DataTypeDelta,
				ExecAt:    time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC),
				ArrivedAt: time.Date(2026, 3, 22, 12, 0, 1, 0, time.UTC),
				Asks: []OBRecord{
					{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "101"), Volume: orderBookHandlerMustDecimal(t, "1")},
				},
				Bids: []OBRecord{
					{SeqID: 1, Price: orderBookHandlerMustDecimal(t, "100"), Volume: orderBookHandlerMustDecimal(t, "1")},
				},
			})

			assert.Equal(t, tt.wantDeferCallCount, deferCallCount)
		})
	}
}

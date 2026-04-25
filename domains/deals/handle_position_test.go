package deals

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains"
)

func makePositionData(ts time.Time, mode PositionMode, side domains.Side, qty string) *PositionData {
	return &PositionData{
		Timestamp:    ts,
		PositionMode: mode,
		Side:         side,
		Qty:          decimal.RequireFromString(qty),
	}
}

func TestHandlePositionData_UpdatesPosition(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		datas   []*PositionData
		wantPos string
	}{
		{
			name: "buy sets positive position",
			datas: []*PositionData{
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideBuy, "1.5"),
			},
			wantPos: "1.5",
		},
		{
			name: "sell sets negative position",
			datas: []*PositionData{
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideSell, "2.0"),
			},
			wantPos: "-2.0",
		},
		{
			name: "zero qty sets zero position",
			datas: []*PositionData{
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideBuy, "0"),
			},
			wantPos: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dealer := newTestDealer(&mockAgent{})
			dealer.HandlePositionData(tt.datas)

			if !dealer.CurrentPosition.Get().Equal(decimal.RequireFromString(tt.wantPos)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestHandlePositionData_TimestampOrdering(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		first   []*PositionData
		second  []*PositionData
		wantPos string
	}{
		{
			name: "newer timestamp updates position",
			first: []*PositionData{
				makePositionData(now.Add(1*time.Second), PositionModeOneWay, domains.SideBuy, "1.0"),
			},
			second: []*PositionData{
				makePositionData(now.Add(2*time.Second), PositionModeOneWay, domains.SideSell, "2.0"),
			},
			wantPos: "-2.0",
		},
		{
			name: "older timestamp is ignored",
			first: []*PositionData{
				makePositionData(now.Add(2*time.Second), PositionModeOneWay, domains.SideBuy, "1.0"),
			},
			second: []*PositionData{
				makePositionData(now.Add(1*time.Second), PositionModeOneWay, domains.SideSell, "9.0"),
			},
			wantPos: "1.0",
		},
		{
			name: "equal timestamp still updates",
			first: []*PositionData{
				makePositionData(now.Add(1*time.Second), PositionModeOneWay, domains.SideBuy, "1.0"),
			},
			second: []*PositionData{
				makePositionData(now.Add(1*time.Second), PositionModeOneWay, domains.SideSell, "3.0"),
			},
			wantPos: "-3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dealer := newTestDealer(&mockAgent{})
			dealer.HandlePositionData(tt.first)
			dealer.HandlePositionData(tt.second)

			if !dealer.CurrentPosition.Get().Equal(decimal.RequireFromString(tt.wantPos)) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

func TestHandlePositionData_HandlersInvoked(t *testing.T) {
	now := time.Now()

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
			dealer := newTestDealer(&mockAgent{})
			counts := make([]int, tt.handlerCount)
			for i := 0; i < tt.handlerCount; i++ {
				idx := i
				dealer.SetPosUpdatedHandler(func(msg []*PositionData) {
					counts[idx]++
				})
			}

			dealer.HandlePositionData([]*PositionData{
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideBuy, "1.0"),
			})

			for _, c := range counts {
				if got, want := c, 1; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestHandlePositionData_Validation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		datas      []*PositionData
		wantUpdate bool
	}{
		{
			name: "hedge mode is rejected",
			datas: []*PositionData{
				makePositionData(now.Add(time.Second), PositionModeHedgeOfBuySide, domains.SideBuy, "1.0"),
			},
			wantUpdate: false,
		},
		{
			name: "multiple one-way entries rejected",
			datas: []*PositionData{
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideBuy, "1.0"),
				makePositionData(now.Add(time.Second), PositionModeOneWay, domains.SideSell, "2.0"),
			},
			wantUpdate: false,
		},
		{
			name:       "empty data rejected",
			datas:      []*PositionData{},
			wantUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dealer := newTestDealer(&mockAgent{})
			handlerCalled := false
			dealer.SetPosUpdatedHandler(func(msg []*PositionData) {
				handlerCalled = true
			})

			dealer.HandlePositionData(tt.datas)

			if got, want := handlerCalled, tt.wantUpdate; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if !dealer.CurrentPosition.Get().Equal(decimal.Zero) {
				t.Errorf("got %v, want true", false)
			}
		})
	}
}

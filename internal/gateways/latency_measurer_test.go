package gateways

import (
	"context"
	"testing"
	"time"
)

func dur(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

func TestAggLatency(t *testing.T) {
	tests := []struct {
		name    string
		adds    []time.Duration
		wantAvg *time.Duration
		wantMax *time.Duration
		wantMin *time.Duration
	}{
		{
			name:    "zero_count",
			adds:    nil,
			wantAvg: nil,
			wantMax: nil,
			wantMin: nil,
		},
		{
			name:    "single_add",
			adds:    []time.Duration{dur(100)},
			wantAvg: ptr(dur(100)),
			wantMax: ptr(dur(100)),
			wantMin: ptr(dur(100)),
		},
		{
			name:    "multiple_adds",
			adds:    []time.Duration{dur(100), dur(200), dur(300)},
			wantAvg: ptr(dur(200)),
			wantMax: ptr(dur(300)),
			wantMin: ptr(dur(100)),
		},
		{
			name:    "same_values",
			adds:    []time.Duration{dur(50), dur(50), dur(50)},
			wantAvg: ptr(dur(50)),
			wantMax: ptr(dur(50)),
			wantMin: ptr(dur(50)),
		},
		{
			name:    "descending_order",
			adds:    []time.Duration{dur(300), dur(200), dur(100)},
			wantAvg: ptr(dur(200)),
			wantMax: ptr(dur(300)),
			wantMin: ptr(dur(100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := NewAggLatency()
			for _, d := range tt.adds {
				agg.Add(d)
			}
			if got, want := agg.Avg(), tt.wantAvg; !durPtrEqual(got, want) {
				t.Errorf("Avg: got %v, want %v", got, want)
			}
			if got, want := agg.Max(), tt.wantMax; !durPtrEqual(got, want) {
				t.Errorf("Max: got %v, want %v", got, want)
			}
			if got, want := agg.Min(), tt.wantMin; !durPtrEqual(got, want) {
				t.Errorf("Min: got %v, want %v", got, want)
			}
		})
	}
}

func TestRecordLatency(t *testing.T) {
	tests := []struct {
		name    string
		records []struct {
			topic   string
			latency time.Duration
		}
		wantTopicCount int
		wantTopics     []string
	}{
		{
			name: "single_topic",
			records: []struct {
				topic   string
				latency time.Duration
			}{
				{"orderbook", dur(100)},
				{"orderbook", dur(200)},
				{"orderbook", dur(300)},
			},
			wantTopicCount: 1,
			wantTopics:     []string{"orderbook"},
		},
		{
			name: "multiple_topics",
			records: []struct {
				topic   string
				latency time.Duration
			}{
				{"orderbook", dur(100)},
				{"trade", dur(200)},
				{"order", dur(300)},
			},
			wantTopicCount: 3,
			wantTopics:     []string{"orderbook", "trade", "order"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewLatencyMeasurer(1 * time.Hour) // aggregate manually
			for _, r := range tt.records {
				m.RecordLatency(r.topic, r.latency)
			}

			// aggregate and Export
			m.aggregate()
			history := m.Export()

			if got, want := len(history), 1; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			aggMap := history[0].AggLatencyMap
			if got, want := len(aggMap), tt.wantTopicCount; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			for _, topic := range tt.wantTopics {
				_, ok := aggMap[topic]
				if !ok {
					t.Errorf("got %v, want true", ok)
				}
			}
		})
	}

	t.Run("single_topic_aggregated_values", func(t *testing.T) {
		m := NewLatencyMeasurer(1 * time.Hour)
		m.RecordLatency("orderbook", dur(100))
		m.RecordLatency("orderbook", dur(200))
		m.RecordLatency("orderbook", dur(300))

		m.aggregate()
		history := m.Export()

		agg := history[0].AggLatencyMap["orderbook"]
		if got, want := agg.Avg(), ptr(dur(200)); !durPtrEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := agg.Max(), ptr(dur(300)); !durPtrEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := agg.Min(), ptr(dur(100)); !durPtrEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("export_clears_history", func(t *testing.T) {
		m := NewLatencyMeasurer(1 * time.Hour)
		m.RecordLatency("orderbook", dur(100))
		m.aggregate()

		first := m.Export()
		if got, want := len(first), 1; got != want {
			t.Errorf("got %v, want %v", got, want)
		}

		second := m.Export()
		if got, want := len(second), 0; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestAggregate(t *testing.T) {
	tests := []struct {
		name          string
		recordBatches [][]struct {
			topic   string
			latency time.Duration
		}
		wantHistoryCount int
	}{
		{
			name: "rotates_current_map",
			recordBatches: [][]struct {
				topic   string
				latency time.Duration
			}{
				{{"orderbook", dur(100)}},
				{{"orderbook", dur(200)}},
			},
			wantHistoryCount: 2,
		},
		{
			name: "empty_interval",
			recordBatches: [][]struct {
				topic   string
				latency time.Duration
			}{{}},
			wantHistoryCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewLatencyMeasurer(1 * time.Hour)
			for _, batch := range tt.recordBatches {
				for _, r := range batch {
					m.RecordLatency(r.topic, r.latency)
				}
				m.aggregate()
			}

			history := m.Export()
			if got, want := len(history), tt.wantHistoryCount; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}

	t.Run("each_interval_independent", func(t *testing.T) {
		m := NewLatencyMeasurer(1 * time.Hour)

		// Interval 1: 100ms
		m.RecordLatency("orderbook", dur(100))
		m.aggregate()

		// Interval 2: 500ms
		m.RecordLatency("orderbook", dur(500))
		m.aggregate()

		history := m.Export()
		if got, want := len(history), 2; got != want {
			t.Errorf("got %v, want %v", got, want)
		}

		agg1 := history[0].AggLatencyMap["orderbook"]
		if got, want := agg1.Avg(), ptr(dur(100)); !durPtrEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		agg2 := history[1].AggLatencyMap["orderbook"]
		if got, want := agg2.Avg(), ptr(dur(500)); !durPtrEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestStart(t *testing.T) {
	t.Run("aggregates_on_interval", func(t *testing.T) {
		m := NewLatencyMeasurer(50 * time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			m.Start(ctx)
			close(done)
		}()

		m.RecordLatency("orderbook", dur(100))
		time.Sleep(120 * time.Millisecond) // wait for at least one tick
		cancel()
		<-done

		history := m.Export()
		if !(len(history) >= 1) {
			t.Error("expected at least 1 aggregated entry")
		}

		// Verify aggregated values
		found := false
		for _, h := range history {
			if agg, ok := h.AggLatencyMap["orderbook"]; ok {
				if got, want := agg.Avg(), ptr(dur(100)); !durPtrEqual(got, want) {
					t.Errorf("got %v, want %v", got, want)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("got %v, want true", found)
		}
	})

	t.Run("stops_on_ctx_cancel", func(t *testing.T) {
		m := NewLatencyMeasurer(50 * time.Millisecond)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			m.Start(ctx)
			close(done)
		}()

		cancel()
		select {
		case <-done:
			// Start returned normally
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not return after ctx cancel")
		}
	})
}

// durPtrEqual compares two *time.Duration values.
func durPtrEqual(a, b *time.Duration) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptr(d time.Duration) *time.Duration {
	return &d
}

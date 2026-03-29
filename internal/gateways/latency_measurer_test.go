package gateways

import (
	"context"
	"testing"
	"time"

	"github.com/bmizerany/assert"
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
			assert.Equal(t, tt.wantAvg, agg.Avg())
			assert.Equal(t, tt.wantMax, agg.Max())
			assert.Equal(t, tt.wantMin, agg.Min())
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
			m := NewLatencyMeasurer(1 * time.Hour) // aggregate は手動で行う
			for _, r := range tt.records {
				m.RecordLatency(r.topic, r.latency)
			}

			// aggregate して Export
			m.aggregate()
			history := m.Export()

			assert.Equal(t, 1, len(history))
			aggMap := history[0].AggLatencyMap
			assert.Equal(t, tt.wantTopicCount, len(aggMap))
			for _, topic := range tt.wantTopics {
				_, ok := aggMap[topic]
				assert.Equal(t, true, ok)
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
		assert.Equal(t, ptr(dur(200)), agg.Avg())
		assert.Equal(t, ptr(dur(300)), agg.Max())
		assert.Equal(t, ptr(dur(100)), agg.Min())
	})

	t.Run("export_clears_history", func(t *testing.T) {
		m := NewLatencyMeasurer(1 * time.Hour)
		m.RecordLatency("orderbook", dur(100))
		m.aggregate()

		first := m.Export()
		assert.Equal(t, 1, len(first))

		second := m.Export()
		assert.Equal(t, 0, len(second))
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
			assert.Equal(t, tt.wantHistoryCount, len(history))
		})
	}

	t.Run("each_interval_independent", func(t *testing.T) {
		m := NewLatencyMeasurer(1 * time.Hour)

		// 第1区間: 100ms
		m.RecordLatency("orderbook", dur(100))
		m.aggregate()

		// 第2区間: 500ms
		m.RecordLatency("orderbook", dur(500))
		m.aggregate()

		history := m.Export()
		assert.Equal(t, 2, len(history))

		agg1 := history[0].AggLatencyMap["orderbook"]
		assert.Equal(t, ptr(dur(100)), agg1.Avg())

		agg2 := history[1].AggLatencyMap["orderbook"]
		assert.Equal(t, ptr(dur(500)), agg2.Avg())
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
		time.Sleep(120 * time.Millisecond) // ticker が少なくとも1回発火するのを待つ
		cancel()
		<-done

		history := m.Export()
		assert.T(t, len(history) >= 1, "expected at least 1 aggregated entry")

		// 集約された値を検証
		found := false
		for _, h := range history {
			if agg, ok := h.AggLatencyMap["orderbook"]; ok {
				assert.Equal(t, ptr(dur(100)), agg.Avg())
				found = true
			}
		}
		assert.Equal(t, true, found)
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
			// Start が正常に終了した
		case <-time.After(1 * time.Second):
			t.Fatal("Start did not return after ctx cancel")
		}
	})
}

func ptr(d time.Duration) *time.Duration {
	return &d
}

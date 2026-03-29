package gateways

import (
	"context"
	"sync"
	"time"
)

type AggLatencyMap map[string]*AggLatency

type AggLatencyMapWithTime struct {
	Time time.Time
	AggLatencyMap
}

// AggLatency holds the sum and count of latencies.
// By incrementally adding latencies, it efficiently computes aggregate statistics such as average, max, and min.
type AggLatency struct {
	counter int
	sumVal  time.Duration
	maxVal  time.Duration
	minVal  time.Duration
}

func NewAggLatency() *AggLatency {
	return &AggLatency{}
}
func (l *AggLatency) Add(latency time.Duration) {
	l.sumVal += latency
	if l.maxVal < latency || l.counter == 0 {
		l.maxVal = latency
	}
	if l.minVal > latency || l.counter == 0 {
		l.minVal = latency
	}
	l.counter++
}
func (l *AggLatency) Avg() *time.Duration {
	if l.counter == 0 {
		return nil
	}
	result := time.Duration(int64(l.sumVal) / int64(l.counter))
	return &result
}

func (l *AggLatency) Max() *time.Duration {
	if l.counter == 0 {
		return nil
	}
	return &l.maxVal
}
func (l *AggLatency) Min() *time.Duration {
	if l.counter == 0 {
		return nil
	}
	return &l.minVal
}

type LatencyMeasurer struct {
	mu                         sync.RWMutex
	aggregateInterval          time.Duration
	currentMeasurementsByTopic AggLatencyMap // records latency per topic name

	history []*AggLatencyMapWithTime // holds aggregated measurements until exported
}

func NewLatencyMeasurer(aggInterval time.Duration) *LatencyMeasurer {
	return &LatencyMeasurer{
		aggregateInterval:          aggInterval,
		currentMeasurementsByTopic: AggLatencyMap{},
		history:                    []*AggLatencyMapWithTime{},
	}
}

// aggregate swaps m.currentMeasurementsByTopic with a new map and appends the completed map to history.
func (m *LatencyMeasurer) aggregate() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Swap with a new map.
	m.history = append(m.history, &AggLatencyMapWithTime{
		Time:          time.Now(), // record aggregation time
		AggLatencyMap: m.currentMeasurementsByTopic,
	})
	m.currentMeasurementsByTopic = AggLatencyMap{}
}

func (m *LatencyMeasurer) Start(ctx context.Context) {
	aggregateTicker := time.NewTicker(m.aggregateInterval)
	defer aggregateTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-aggregateTicker.C:
			m.aggregate()
		}
	}
}

func (m *LatencyMeasurer) Export() []*AggLatencyMapWithTime {
	m.mu.Lock()
	defer m.mu.Unlock()
	history := m.history
	m.history = []*AggLatencyMapWithTime{}
	return history
}

func (m *LatencyMeasurer) RecordLatency(topicName string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.currentMeasurementsByTopic[topicName]; ok {
		m.currentMeasurementsByTopic[topicName].Add(latency)
	} else {
		m.currentMeasurementsByTopic[topicName] = NewAggLatency()
		m.currentMeasurementsByTopic[topicName].Add(latency)
	}
}

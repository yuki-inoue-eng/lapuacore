package translators

import (
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/metrics/measurements"
)

type LatencyTranslator struct {
}

func NewLatencyTranslator() *LatencyTranslator {
	return &LatencyTranslator{}
}

func (t *LatencyTranslator) TranslateToLatencies(withTimes []*gateways.AggLatencyMapWithTime) []*measurements.Latency {
	var latencies []*measurements.Latency
	for _, withTime := range withTimes {
		latencies = append(latencies, t.translateToLatencies(withTime.Time, &withTime.AggLatencyMap)...)
	}
	return latencies
}

func (t *LatencyTranslator) translateToLatencies(time time.Time, aggMap *gateways.AggLatencyMap) []*measurements.Latency {
	var latencies []*measurements.Latency
	for topicName, aggLatency := range *aggMap {
		latencies = append(latencies, t.translateToLatency(time, topicName, aggLatency))
	}
	return latencies
}

func (t *LatencyTranslator) translateToLatency(time time.Time, topicName string, aggLatency *gateways.AggLatency) *measurements.Latency {
	latency := &measurements.Latency{
		Time:  time,
		Topic: topicName,
	}
	if avg := aggLatency.Avg(); avg != nil {
		s := aggLatency.Avg().Milliseconds()
		latency.AvgMilliSec = &s
	}
	if mx := aggLatency.Max(); mx != nil {
		s := aggLatency.Max().Milliseconds()
		latency.MaxMilliSec = &s
	}
	if mn := aggLatency.Min(); mn != nil {
		s := aggLatency.Min().Milliseconds()
		latency.MinMilliSec = &s
	}
	return latency
}

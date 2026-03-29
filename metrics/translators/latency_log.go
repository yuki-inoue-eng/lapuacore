package translators

import (
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/metrics/measurements"
)

type LatencyLogTranslator struct {
}

func NewLatencyLogTranslator() *LatencyLogTranslator {
	return &LatencyLogTranslator{}
}

func (t *LatencyLogTranslator) TranslateToLatencyLogs(withTimes []*gateways.AggLatencyMapWithTime) []*measurements.LatencyLog {
	var logs []*measurements.LatencyLog
	for _, withTime := range withTimes {
		logs = append(logs, t.translateToLatencyLogs(withTime.Time, &withTime.AggLatencyMap)...)
	}
	return logs
}

func (t *LatencyLogTranslator) translateToLatencyLogs(time time.Time, aggMap *gateways.AggLatencyMap) []*measurements.LatencyLog {
	var logs []*measurements.LatencyLog
	for topicName, aggLatency := range *aggMap {
		logs = append(logs, t.translateToLatencyLog(time, topicName, aggLatency))
	}
	return logs
}

func (t *LatencyLogTranslator) translateToLatencyLog(time time.Time, topicName string, aggLatency *gateways.AggLatency) *measurements.LatencyLog {
	log := &measurements.LatencyLog{
		Time:  time,
		Topic: topicName,
	}
	if avg := aggLatency.Avg(); avg != nil {
		s := aggLatency.Avg().Milliseconds()
		log.AvgMilliSec = &s
	}
	if mx := aggLatency.Max(); mx != nil {
		s := aggLatency.Max().Milliseconds()
		log.MaxMilliSec = &s
	}
	if mn := aggLatency.Min(); mn != nil {
		s := aggLatency.Min().Milliseconds()
		log.MinMilliSec = &s
	}
	return log
}

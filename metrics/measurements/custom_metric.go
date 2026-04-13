package measurements

import (
	"time"

	"github.com/InfluxCommunity/influxdb3-go/influxdb3"
)

// CustomMetric represents a user-defined measurement entry with arbitrary tags and fields.
type CustomMetric struct {
	measurement string
	timestamp   time.Time
	tags        map[string]string
	fields      map[string]float64
}

func (m *CustomMetric) validate() bool {
	return len(m.fields) > 0
}

func (m *CustomMetric) ToPoint(strategyName string) *influxdb3.Point {
	if !m.validate() {
		return nil
	}
	point := influxdb3.NewPointWithMeasurement(m.measurement).
		SetTimestamp(m.timestamp)
	if strategyName != "" {
		point.SetTag("strategy", strategyName)
	}
	for k, v := range m.tags {
		point.SetTag(k, v)
	}
	for k, v := range m.fields {
		point.SetField(k, v)
	}
	return point
}

func NewCustomMetric(
	timestamp time.Time,
	measurement string,
	tags map[string]string,
	fields map[string]float64,
) *CustomMetric {
	return &CustomMetric{
		measurement: measurement,
		timestamp:   timestamp,
		tags:        tags,
		fields:      fields,
	}
}

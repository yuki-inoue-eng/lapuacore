package measurements

import (
	"time"

	"github.com/InfluxCommunity/influxdb3-go/influxdb3"
)

// Latency represents a WebSocket latency measurement entry.
type Latency struct {
	Time        time.Time
	Topic       string
	AvgMilliSec *int64
	MaxMilliSec *int64
	MinMilliSec *int64
}

func (l *Latency) ToPoint(strategyName string) *influxdb3.Point {
	point := influxdb3.NewPointWithMeasurement(wsLatencyMeasurementName).
		SetTag("topic", l.Topic).
		SetTimestamp(l.Time)
	if strategyName != "" {
		point.SetTag("strategy", strategyName)
	}
	setPointerValue(point, "avg_ms", l.AvgMilliSec)
	setPointerValue(point, "max_ms", l.MaxMilliSec)
	setPointerValue(point, "min_ms", l.MinMilliSec)
	return point
}

type Latencies []*Latency

func (ls Latencies) ToPoints(strategyName string) []*influxdb3.Point {
	var points []*influxdb3.Point
	for _, l := range ls {
		points = append(points, l.ToPoint(strategyName))
	}
	return points
}

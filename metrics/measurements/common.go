package measurements

import "github.com/InfluxCommunity/influxdb3-go/influxdb3"

// defines measurement names
const (
	wsLatencyLogsMeasurementName = "ws_latency_logs"
)

// setPointerValue sets a field on an influxdb3.Point if the pointer value is non-nil.
func setPointerValue[T any](point *influxdb3.Point, name string, ptrValue *T) {
	if ptrValue != nil {
		point.SetField(name, *ptrValue)
	}
}

type DataPoint interface {
	ToPoint(strategyName string) *influxdb3.Point
}

package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/influxdb3"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/metrics/measurements"
	"github.com/yuki-inoue-eng/lapuacore/metrics/translators"
)

const (
	exportInterval = 5 * time.Second
)

type Exporter struct {
	noopMode     bool
	strategyName string
	bucketClient *influxdb3.Client

	// translators
	latencyTranslator *translators.LatencyTranslator

	// components for collection
	latencyMeasurer []*gateways.LatencyMeasurer
}

// NewExporter initializes a metrics exporter.
// If url or token is empty, it runs in noop mode (metrics are discarded).
func NewExporter(bucketName, strategyName, url, token string) *Exporter {
	if url == "" || token == "" {
		return newExporter("", nil, true)
	}
	client, err := NewInfluxDBClient(url, bucketName, token)
	if err != nil {
		panic(err)
	}
	return newExporter(strategyName, client, false)
}

func newExporter(strategyName string, client *influxdb3.Client, noopMode bool) *Exporter {
	return &Exporter{
		noopMode:             noopMode,
		strategyName:         strategyName,
		bucketClient:         client,
		latencyTranslator: translators.NewLatencyTranslator(),
	}
}

func (c *Exporter) SetLatencyMeasurer(measurer *gateways.LatencyMeasurer) {
	c.latencyMeasurer = append(c.latencyMeasurer, measurer)
}

func (c *Exporter) Start(ctx context.Context) {
	exportTicker := time.NewTicker(exportInterval)
	defer exportTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-exportTicker.C:
			if c.noopMode {
				c.discardWSLatencies()
				continue
			}
			c.collectAndExportLatencies(ctx)
		}
	}
}

func (c *Exporter) discardWSLatencies() {
	for _, measurer := range c.latencyMeasurer {
		measurer.Export()
	}
}

func (c *Exporter) collectAndExportLatencies(ctx context.Context) {
	go func() {
		latencies := c.collectLatencies()
		if err := c.exportLatencies(ctx, latencies); err != nil {
			slog.Error(err.Error())
		}
	}()
}

func (c *Exporter) collectLatencies() []*measurements.Latency {
	var latencies measurements.Latencies
	for _, measurer := range c.latencyMeasurer {
		latencies = append(latencies, c.latencyTranslator.TranslateToLatencies(measurer.Export())...)
	}
	return latencies
}

func (c *Exporter) exportLatencies(ctx context.Context, latencies measurements.Latencies) error {
	if len(latencies) == 0 {
		return nil
	}
	if err := c.bucketClient.WritePoints(ctx, latencies.ToPoints(c.strategyName)...); err != nil {
		return err
	}
	return nil
}

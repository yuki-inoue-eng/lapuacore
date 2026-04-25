package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	mu           sync.Mutex
	noopMode     bool
	strategyName string
	bucketClient *influxdb3.Client

	// translators
	latencyTranslator *translators.LatencyTranslator

	// components for collection
	latencyMeasurer   []*gateways.LatencyMeasurer
	customMetricQueue []*measurements.CustomMetric
}

// NewExporter initializes a metrics exporter.
// If url or token is empty, it runs in noop mode (metrics are discarded).
func NewExporter(bucketName, strategyName, url, token string) (*Exporter, error) {
	if url == "" || token == "" {
		return newExporter("", nil, true), nil
	}
	client, err := NewInfluxDBClient(url, bucketName, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create InfluxDB client: %w", err)
	}
	return newExporter(strategyName, client, false), nil
}

func newExporter(strategyName string, client *influxdb3.Client, noopMode bool) *Exporter {
	return &Exporter{
		noopMode:          noopMode,
		strategyName:      strategyName,
		bucketClient:      client,
		latencyTranslator: translators.NewLatencyTranslator(),
	}
}

func (c *Exporter) SetLatencyMeasurer(measurer *gateways.LatencyMeasurer) {
	c.latencyMeasurer = append(c.latencyMeasurer, measurer)
}

// WriteCustomMetrics enqueues user-defined metrics for periodic export to InfluxDB.
func (c *Exporter) WriteCustomMetrics(metrics []*measurements.CustomMetric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.customMetricQueue = append(c.customMetricQueue, metrics...)
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
				c.discardCustomMetrics()
				continue
			}
			c.collectAndExportLatencies(ctx)
			c.collectAndExportCustomMetrics(ctx)
		}
	}
}

func (c *Exporter) discardWSLatencies() {
	for _, measurer := range c.latencyMeasurer {
		measurer.Export()
	}
}

func (c *Exporter) discardCustomMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.customMetricQueue = nil
}

func (c *Exporter) collectAndExportLatencies(ctx context.Context) {
	go func() {
		latencies := c.collectLatencies()
		if err := c.exportLatencies(ctx, latencies); err != nil {
			slog.Error("failed to export latencies", "error", err)
		}
	}()
}

func (c *Exporter) collectAndExportCustomMetrics(ctx context.Context) {
	go func() {
		c.mu.Lock()
		metrics := c.customMetricQueue
		c.customMetricQueue = nil
		c.mu.Unlock()
		if err := c.exportCustomMetrics(ctx, metrics); err != nil {
			slog.Error("failed to export custom metrics", "error", err)
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

func (c *Exporter) exportCustomMetrics(ctx context.Context, metrics []*measurements.CustomMetric) error {
	if len(metrics) == 0 {
		return nil
	}
	var points []*influxdb3.Point
	for _, m := range metrics {
		if point := m.ToPoint(c.strategyName); point != nil {
			points = append(points, point)
		}
	}
	if err := c.bucketClient.WritePoints(ctx, points...); err != nil {
		return err
	}
	return nil
}

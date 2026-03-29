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
	latencyLogTranslator *translators.LatencyLogTranslator

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
		latencyLogTranslator: translators.NewLatencyLogTranslator(),
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
				c.discardWSLatencyLogs()
				continue
			}
			c.collectAndExportLatencyLogs(ctx)
		}
	}
}

func (c *Exporter) discardWSLatencyLogs() {
	for _, measurer := range c.latencyMeasurer {
		measurer.Export()
	}
}

func (c *Exporter) collectAndExportLatencyLogs(ctx context.Context) {
	go func() {
		logs := c.collectLatencyLogs()
		if err := c.exportLatencyLogs(ctx, logs); err != nil {
			slog.Error(err.Error())
		}
	}()
}

func (c *Exporter) collectLatencyLogs() []*measurements.LatencyLog {
	var logs measurements.LatencyLogs
	for _, measurer := range c.latencyMeasurer {
		logs = append(logs, c.latencyLogTranslator.TranslateToLatencyLogs(measurer.Export())...)
	}
	return logs
}

func (c *Exporter) exportLatencyLogs(ctx context.Context, logs measurements.LatencyLogs) error {
	if len(logs) == 0 {
		return nil
	}
	if err := c.bucketClient.WritePoints(ctx, logs.ToPoints(c.strategyName)...); err != nil {
		return err
	}
	return nil
}

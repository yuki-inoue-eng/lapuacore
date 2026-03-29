package lapua

import (
	"log/slog"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/initializers"
	"github.com/yuki-inoue-eng/lapuacore/initializers/discord"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/initializers/logger"
	"github.com/yuki-inoue-eng/lapuacore/metrics"

	"context"
)

// Public components
var (
	Ctx      context.Context
	Cancel   context.CancelFunc
	Exporter *metrics.Exporter
	Discord  *discord.Client
)

func IsInitialized() bool {
	return Ctx != nil && Exporter != nil
}

// InitAndStart initializes and starts all core components.
// All configuration values are passed as arguments (configs package is not yet ported).
func InitAndStart(
	strategyName string,
	bucketName string,
	influxUrl string,
	influxToken string,
	discordInfoUrl string,
	discordWarnUrl string,
	discordEmergencyUrl string,
	logFilePath string,
) {
	Ctx, Cancel = initializers.NewCancellableContext()
	logger.InitLogger(Ctx, logFilePath)
	Exporter = metrics.NewExporter(bucketName, strategyName, influxUrl, influxToken)
	Discord = discord.NewClient(strategyName, discordInfoUrl, discordWarnUrl, discordEmergencyUrl)
	go Exporter.Start(Ctx)
}

// InitAndStartNoopMode starts with noop exporter and noop discord client (for testing).
func InitAndStartNoopMode() {
	Ctx, Cancel = initializers.NewCancellableContext()
	Exporter = metrics.NewExporter("", "", "", "")
	Discord = discord.NewClient("", "", "", "")
	go Exporter.Start(Ctx)
}

// WaitForInsightsToBeReady blocks until all registered Insights report ready.
func WaitForInsightsToBeReady() {
	insReadyStates := genInsReadyStates()

	areAllInsReady := func() bool {
		for _, isReady := range insReadyStates {
			if !isReady {
				return false
			}
		}
		return true
	}

	updateInsReadyStates := func(ins exchanges.Insight) {
		before := insReadyStates[ins]
		current := ins.IsEverythingReady()
		insReadyStates[ins] = current
		if !before && current {
			slog.Info(ins.EXName() + " insights is ready")
		}
	}

	for {
		time.Sleep(1 * time.Second)
		for ins := range insReadyStates {
			updateInsReadyStates(ins)
		}
		if areAllInsReady() {
			slog.Info("All insights are ready")
			return
		}
	}
}

// WaitForCtxDone blocks until the context is cancelled, then waits a grace period.
func WaitForCtxDone() {
	const extensionSec = 3
	for {
		select {
		case <-Ctx.Done():
			time.Sleep(extensionSec * time.Second)
			return
		}
	}
}

// StartHeartBeat sends a periodic heartbeat message via Discord.
func StartHeartBeat() {
	go func() {
		heartBeatTicker := time.NewTicker(5 * time.Minute)
		for {
			select {
			case <-heartBeatTicker.C:
				Discord.SendInfo("heart beat")
			case <-Ctx.Done():
				return
			}
		}
	}()
}

func genInsReadyStates() map[exchanges.Insight]bool {
	states := map[exchanges.Insight]bool{}
	for i := range exchanges.Insights {
		states[exchanges.Insights[i]] = false
	}
	return states
}

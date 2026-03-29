package lapua

import (
	"errors"
	"log/slog"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/configs"
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
	Params   *configs.ParamMap
	Secrets  *configs.Secret
	Exporter *metrics.Exporter
	Discord  *discord.Client
)

// File paths set by InitFilePaths.
var (
	ConfigFilePath string
	SecretFilePath string
	LogFilePath    string
)

func IsInitialized() bool {
	return Ctx != nil && Exporter != nil
}

// InitFilePaths sets the file paths used by InitAndStart and InitAndStartDCMode.
func InitFilePaths(configFilePath, secretFilePath, logFilePath string) {
	ConfigFilePath = configFilePath
	SecretFilePath = secretFilePath
	LogFilePath = logFilePath
}

// InitAndStart initializes and starts all core components using config and secret files.
func InitAndStart(watcherOpts ...configs.Option) {
	validateFilePaths()
	Ctx, Cancel = initializers.NewCancellableContext()
	logger.InitLogger(Ctx, LogFilePath)
	watcher := configs.NewWatcher(ConfigFilePath, SecretFilePath, watcherOpts...)
	config := watcher.GetConfig()
	Secrets = watcher.GetSecret()
	Params = config.Params
	Exporter = metrics.NewExporter(
		config.Strategy.Name,
		config.Strategy.Name,
		Secrets.InfluxDB.GetUrl(),
		Secrets.InfluxDB.GetToken(),
	)
	Discord = discord.NewClient(config.Strategy.Name,
		Secrets.Discord.GetInfoUrl(),
		Secrets.Discord.GetWarnUrl(),
		Secrets.Discord.GetEmergencyUrl(),
	)
	go watcher.Start(Ctx)
	go Exporter.Start(Ctx)
}

// InitAndStartDCMode initializes and starts in data curator mode.
// The bucketName from config params is used as the Discord username.
func InitAndStartDCMode(watcherOpts ...configs.Option) {
	validateFilePaths()
	Ctx, Cancel = initializers.NewCancellableContext()
	logger.InitLogger(Ctx, LogFilePath)
	watcher := configs.NewWatcher(ConfigFilePath, SecretFilePath, watcherOpts...)
	Secrets = watcher.GetSecret()
	Params = watcher.GetConfig().Params

	bucketName := Params.Get("bucketName")
	if bucketName == "" {
		panic(errors.New("influx db bucket name must be specified in config file"))
	}
	Exporter = metrics.NewExporter(
		bucketName,
		"",
		Secrets.InfluxDB.GetUrl(),
		Secrets.InfluxDB.GetToken(),
	)
	Discord = discord.NewClient(bucketName,
		Secrets.Discord.GetInfoUrl(),
		Secrets.Discord.GetWarnUrl(),
		Secrets.Discord.GetEmergencyUrl(),
	)
	go watcher.Start(Ctx)
	go Exporter.Start(Ctx)
}

// InitAndStartNoopMode starts with noop exporter and noop discord client (for testing).
func InitAndStartNoopMode() {
	Ctx, Cancel = initializers.NewCancellableContext()
	Exporter = metrics.NewExporter("", "", "", "")
	Discord = discord.NewClient("", "", "", "")
	go Exporter.Start(Ctx)
}

func validateFilePaths() {
	if ConfigFilePath == "" {
		panic("config file path is not set")
	}
	if SecretFilePath == "" {
		panic("secret file path is not set")
	}
	if LogFilePath == "" {
		panic("log file path is not set")
	}
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

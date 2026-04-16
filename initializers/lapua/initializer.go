package lapua

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/configs"
	"github.com/yuki-inoue-eng/lapuacore/initializers"
	"github.com/yuki-inoue-eng/lapuacore/initializers/discord"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges"
	"github.com/yuki-inoue-eng/lapuacore/initializers/logger"
	"github.com/yuki-inoue-eng/lapuacore/metrics"
)

// Public components
var (
	Ctx      context.Context
	Cancel   context.CancelCauseFunc
	Params   *configs.ParamMap
	Secrets  *configs.Secret
	Exporter *metrics.Exporter
	Discord  *discord.Client
)

// File paths read from environment variables by initFilePaths.
var (
	configFilePath string
	secretFilePath string
	logFilePath    string
)

func IsInitialized() bool {
	return Ctx != nil && Exporter != nil
}

// initFilePaths reads file paths from environment variables and validates them.
func initFilePaths() {
	configFilePath = os.Getenv("LAPUA_CONFIG_PATH")
	secretFilePath = os.Getenv("LAPUA_SECRET_PATH")
	logFilePath = os.Getenv("LAPUA_LOG_PATH")

	if configFilePath == "" {
		panic("LAPUA_CONFIG_PATH environment variable is not set")
	}
	if secretFilePath == "" {
		panic("LAPUA_SECRET_PATH environment variable is not set")
	}
	if logFilePath == "" {
		panic("LAPUA_LOG_PATH environment variable is not set")
	}
}

// InitAndStart initializes and starts all core components using config and secret files.
// File paths are read from environment variables: LAPUA_CONFIG_PATH, LAPUA_SECRET_PATH, LAPUA_LOG_PATH.
func InitAndStart() {
	initFilePaths()
	Ctx, Cancel = initializers.NewCancellableContext()
	logger.InitLogger(Ctx, logFilePath)
	watcher, err := configs.NewWatcher(Cancel, configFilePath, secretFilePath)
	if err != nil {
		panic(err)
	}
	config := watcher.GetConfig()
	Secrets = watcher.GetSecret()
	Params = config.Params
	exporter, err := metrics.NewExporter(
		config.Strategy.Name,
		config.Strategy.Name,
		Secrets.InfluxDB.GetUrl(),
		Secrets.InfluxDB.GetToken(),
	)
	if err != nil {
		panic(err)
	}
	Exporter = exporter
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
// File paths are read from environment variables: LAPUA_CONFIG_PATH, LAPUA_SECRET_PATH, LAPUA_LOG_PATH.
func InitAndStartDCMode() {
	initFilePaths()
	Ctx, Cancel = initializers.NewCancellableContext()
	logger.InitLogger(Ctx, logFilePath)
	watcher, err := configs.NewWatcher(Cancel, configFilePath, secretFilePath)
	if err != nil {
		panic(err)
	}
	Secrets = watcher.GetSecret()
	Params = watcher.GetConfig().Params

	bucketName := Params.Get("bucketName")
	if bucketName == "" {
		panic("influx db bucket name must be specified in config file")
	}
	exporter, err := metrics.NewExporter(
		bucketName,
		"",
		Secrets.InfluxDB.GetUrl(),
		Secrets.InfluxDB.GetToken(),
	)
	if err != nil {
		panic(err)
	}
	Exporter = exporter
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
	Exporter, _ = metrics.NewExporter("", "", "", "")
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

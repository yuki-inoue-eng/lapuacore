package coinex

import (
	"context"
	"fmt"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/agent"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws"
	"github.com/yuki-inoue-eng/lapuacore/metrics"
)

var gatewayManager *GatewayManager

// GatewayManager orchestrates CoinEx exchange connectivity.
type GatewayManager struct {
	cred            gateways.Credential
	exporter        *metrics.Exporter
	latencyMeasurer *gateways.LatencyMeasurer

	insights *coinexInsights
	deals    *coinexDeals

	publicChannel  *ws.Channel
	privateChannel *ws.Channel

	privateAPIAgent *agent.PrivateAPIAgent
}

// InitGatewayManager initializes the CoinEx gateway manager.
// cred must not be nil (simulator mode is not yet supported).
func InitGatewayManager(cred gateways.Credential, exporter *metrics.Exporter) {
	const aggInterval = 5 * time.Second

	if cred == nil {
		panic(fmt.Errorf("credential must not be nil (simulator mode is not yet supported)"))
	}
	if exporter == nil {
		panic(fmt.Errorf("exporter must not be nil"))
	}

	latencyMeasurer := gateways.NewLatencyMeasurer(aggInterval)
	exporter.SetLatencyMeasurer(latencyMeasurer)

	privateAPIAgent := agent.NewPrivateAPIAgent(cred)
	privateChannel := ws.NewPrivateChannel(cred, latencyMeasurer)
	publicChannel := ws.NewPublicChannel(latencyMeasurer)

	gatewayManager = &GatewayManager{
		cred:            cred,
		exporter:        exporter,
		latencyMeasurer: latencyMeasurer,

		privateChannel:  privateChannel,
		publicChannel:   publicChannel,
		privateAPIAgent: privateAPIAgent,
	}
}

func (m *GatewayManager) setDeals(deals *coinexDeals) {
	m.deals = deals
}

func (m *GatewayManager) setInsights(insights *coinexInsights) {
	m.insights = insights
}

// StartGateway launches all gateway goroutines.
func StartGateway(ctx context.Context) {
	if gatewayManager == nil {
		panic("gatewayManager is not initialized")
	}

	go gatewayManager.latencyMeasurer.Start(ctx)

	gatewayManager.privateAPIAgent.Start(ctx)

	go func() {
		if err := gatewayManager.privateChannel.Start(ctx); err != nil {
			panic(err)
		}
	}()

	go func() {
		if err := gatewayManager.publicChannel.Start(ctx); err != nil {
			panic(err)
		}
	}()
}

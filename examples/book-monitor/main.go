package main

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/initializers/exchanges/coinex"
	"github.com/yuki-inoue-eng/lapuacore/initializers/lapua"
	"github.com/yuki-inoue-eng/lapuacore/pkg/display"
)

func main() {
	s := strategy{}
	s.init()
	s.start()
}

type strategy struct {
	display *display.BookDisplay
}

func (s *strategy) init() {
	lapua.InitAndStartNoopMode()

	symbolBTC := domains.SymbolCoinExFuturesBTCUSDT
	symbolSOL := domains.SymbolCoinExFuturesSOLUSDT
	symbolList := []*domains.Symbol{symbolBTC, symbolSOL}
	coinex.InitGatewayManager(nil, 3)
	coinex.InitInsights(nil, symbolList, nil)
	coinex.StartGateway()

	s.display = display.NewBookDisplay(5, []display.BookEntry{
		{symbolBTC.Name(), coinex.Insights.GetOrderBook(symbolBTC)},
		{symbolSOL.Name(), coinex.Insights.GetOrderBook(symbolSOL)},
	})

	lapua.WaitForInsightsToBeReady()
}

func (s *strategy) start() {
	for _, entry := range s.display.Books() {
		entry.OB.SetUpdateCallback(s.display.Render)
	}
	lapua.WaitForCtxDone()
}

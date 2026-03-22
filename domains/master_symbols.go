package domains

const (
	ExchangeUnknown = "Unknown"
	ExchangeCoinEx  = "CoinEx"

	ProductUnknown = "Unknown"
	ProductFutures = "futures"
	ProductLinear  = "linear"
)

var defaultFeeRatePercent = map[string]float64{
	ExchangeUnknown: 0.00,
	ExchangeCoinEx:  0.03,
}

// ここに各取引所のシンボルを追加していく。

var (
	SymbolUnknown = newSymbol(ExchangeUnknown, ProductUnknown, "Unknown", 0, 0, AssetUnknown, nil)

	// ------------------ CoinEx ------------------

	SymbolCoinExFuturesBTCUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "BTCUSDT", 0.01, 0.0001, AssetUSDT, nil)
	SymbolCoinExFuturesETHUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "ETHUSDT", 0.01, 0.005, AssetUSDT, nil)
	SymbolCoinExFuturesXRPUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "XRPUSDT", 0.0001, 5, AssetUSDT, nil)
	SymbolCoinExFuturesSOLUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "SOLUSDT", 0.01, 0.05, AssetUSDT, nil)
)

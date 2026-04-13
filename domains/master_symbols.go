package domains

const (
	ExchangeUnknown = "Unknown"
	ExchangeCoinEx  = "CoinEx"
	ExchangeBybit   = "Bybit"

	ProductUnknown = "Unknown"
	ProductFutures = "futures"
	ProductLinear  = "linear"
)

var defaultFeeRatePercent = map[string]float64{
	ExchangeUnknown: 0.00,
	ExchangeCoinEx:  0.03,
	ExchangeBybit:   0.055,
}

// ここに各取引所のシンボルを追加していく。

var (
	SymbolUnknown = newSymbol(ExchangeUnknown, ProductUnknown, "Unknown", 0, 0, AssetUnknown, nil)

	// ------------------ CoinEx ------------------

	SymbolCoinExFuturesBTCUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "BTCUSDT", 0.01, 0.0001, AssetUSDT, nil)
	SymbolCoinExFuturesETHUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "ETHUSDT", 0.01, 0.005, AssetUSDT, nil)
	SymbolCoinExFuturesXRPUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "XRPUSDT", 0.0001, 5, AssetUSDT, nil)
	SymbolCoinExFuturesSOLUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "SOLUSDT", 0.01, 0.05, AssetUSDT, nil)

	// ------------------ Bybit ------------------

	SymbolBybitLinearBTCUSDT = newSymbol(ExchangeBybit, ProductLinear, "BTCUSDT", 0.1, 0.001, AssetUSDT, nil)
	SymbolBybitLinearETHUSDT = newSymbol(ExchangeBybit, ProductLinear, "ETHUSDT", 0.01, 0.01, AssetUSDT, nil)
	SymbolBybitLinearSOLUSDT = newSymbol(ExchangeBybit, ProductLinear, "SOLUSDT", 0.001, 0.1, AssetUSDT, nil)
	SymbolBybitLinearXRPUSDT = newSymbol(ExchangeBybit, ProductLinear, "XRPUSDT", 0.0001, 10, AssetUSDT, nil)
)

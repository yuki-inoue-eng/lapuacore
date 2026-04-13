package domains

const (
	ExchangeUnknown = "Unknown"
	ExchangeCoinEx  = "CoinEx"
	ExchangeBybit   = "Bybit"

	ProductUnknown = "Unknown"
	ProductFutures = "futures"
	ProductLinear  = "linear"
)

// ここに各取引所のシンボルを追加していく。

var (
	SymbolUnknown = newSymbol(ExchangeUnknown, ProductUnknown, "Unknown", 0, 0, AssetUnknown)

	// ------------------ CoinEx ------------------

	SymbolCoinExFuturesBTCUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "BTCUSDT", 0.01, 0.0001, AssetUSDT)
	SymbolCoinExFuturesETHUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "ETHUSDT", 0.01, 0.005, AssetUSDT)
	SymbolCoinExFuturesXRPUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "XRPUSDT", 0.0001, 5, AssetUSDT)
	SymbolCoinExFuturesSOLUSDT = newSymbol(ExchangeCoinEx, ProductFutures, "SOLUSDT", 0.01, 0.05, AssetUSDT)

	// ------------------ Bybit ------------------

	SymbolBybitLinearBTCUSDT = newSymbol(ExchangeBybit, ProductLinear, "BTCUSDT", 0.1, 0.001, AssetUSDT)
	SymbolBybitLinearETHUSDT = newSymbol(ExchangeBybit, ProductLinear, "ETHUSDT", 0.01, 0.01, AssetUSDT)
	SymbolBybitLinearSOLUSDT = newSymbol(ExchangeBybit, ProductLinear, "SOLUSDT", 0.001, 0.1, AssetUSDT)
	SymbolBybitLinearXRPUSDT = newSymbol(ExchangeBybit, ProductLinear, "XRPUSDT", 0.0001, 10, AssetUSDT)
)

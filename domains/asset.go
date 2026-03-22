package domains

type Asset int

const (
	AssetUnknown Asset = iota
	AssetBTC
	AssetUSDT
	AssetUSDC
)

func (c Asset) String() string {
	switch c {
	case AssetBTC:
		return "BTC"
	case AssetUSDT:
		return "USDT"
	case AssetUSDC:
		return "USDC"
	default:
		return "Unknown"
	}
}

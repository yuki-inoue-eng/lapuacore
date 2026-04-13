package ws

const (
	HostName = "stream.bybit.com"

	ScopePathPublic  = "/public"
	ScopePathPrivate = "/private"

	ProductPathSpot    = "/spot"
	ProductPathLinear  = "/linear"
	ProductPathInverse = "/inverse"
	ProductPathOption  = "/option"

	APIv5 = "/v5"
)

type Product int

const (
	ProductUnknown Product = iota
	ProductSpot
	ProductLinear
	ProductInverse
	ProductOption
)

func (t Product) Path() string {
	switch t {
	case ProductSpot:
		return ProductPathSpot
	case ProductLinear:
		return ProductPathLinear
	case ProductInverse:
		return ProductPathInverse
	case ProductOption:
		return ProductPathOption
	default:
		return ""
	}
}

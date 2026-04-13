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

type ScopeType int

const (
	scopeTypeUnknown ScopeType = iota
	scopeTypePublic
	scopeTypePrivate
)

func (t ScopeType) String() string {
	switch t {
	case scopeTypePublic:
		return "public"
	case scopeTypePrivate:
		return "private"
	default:
		return ""
	}
}

func (t ScopeType) Path() string {
	switch t {
	case scopeTypePublic:
		return ScopePathPublic
	case scopeTypePrivate:
		return ScopePathPrivate
	default:
		return ""
	}
}

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

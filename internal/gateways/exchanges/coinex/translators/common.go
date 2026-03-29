package translators

import "github.com/yuki-inoue-eng/lapuacore/domains"

type sideTranslator struct{}

func newSideTranslator() *sideTranslator {
	return &sideTranslator{}
}

func (t *sideTranslator) Translate(msg string) domains.Side {
	switch msg {
	case "buy":
		return domains.SideBuy
	case "sell":
		return domains.SideSell
	default:
		return domains.SideNone
	}
}

func (t *sideTranslator) TranslateToDto(side domains.Side) string {
	switch side {
	case domains.SideBuy:
		return "buy"
	case domains.SideSell:
		return "sell"
	default:
		return "none"
	}
}

type orderTypeTranslator struct{}

func newOrderTypeTranslator() *orderTypeTranslator {
	return &orderTypeTranslator{}
}

func (t *orderTypeTranslator) translateOrderTypeToDto(oType domains.OrderType) string {
	switch oType {
	case domains.OrderTypeMarket:
		return "market"
	case domains.OrderTypeLimit:
		return "limit"
	case domains.OrderTypeLimitIOC:
		return "ioc"
	case domains.OrderTypeLimitFOK:
		return "fok"
	case domains.OrderTypeLimitMaker:
		return "maker_only"
	default:
		return ""
	}
}

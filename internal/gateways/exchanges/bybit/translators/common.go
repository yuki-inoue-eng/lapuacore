package translators

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/insights"
)

type messageType int

const (
	MessageTypeUnknown messageType = iota
	MessageTypeSnapshot
	MessageTypeDelta
)

func toMessageType(str string) messageType {
	switch str {
	case "snapshot":
		return MessageTypeSnapshot
	case "delta":
		return MessageTypeDelta
	default:
		return MessageTypeUnknown
	}
}

func (t messageType) translate() insights.DataType {
	switch t {
	case MessageTypeSnapshot:
		return insights.DataTypeSnapshot
	case MessageTypeDelta:
		return insights.DataTypeDelta
	default:
		return insights.DataTypeUnknown
	}
}

type SideTranslator struct{}

func NewSideTranslator() *SideTranslator {
	return &SideTranslator{}
}

func (t *SideTranslator) Translate(msg string) domains.Side {
	switch msg {
	case "Buy":
		return domains.SideBuy
	case "Sell":
		return domains.SideSell
	default:
		return domains.SideNone
	}
}

func (t *SideTranslator) TranslateSideToDto(side domains.Side) string {
	switch side {
	case domains.SideBuy:
		return "Buy"
	case domains.SideSell:
		return "Sell"
	default:
		return "None"
	}
}

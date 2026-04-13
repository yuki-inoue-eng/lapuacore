package translators

import (
	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
)

type OrderTranslator struct {
	sideTranslator *SideTranslator
}

func NewOrderTranslator() *OrderTranslator {
	return &OrderTranslator{
		sideTranslator: NewSideTranslator(),
	}
}

func (t *OrderTranslator) TranslateToDto(symbol *domains.Symbol, order *deals.Order) *dtos.OrderDto {
	rawDto := &dtos.RawOrderDto{
		OrderLinkID: order.GetID(),
		Symbol:      symbol.Name(),
		OrderType:   "Market",
		Price:       nil,
		Qty:         order.GetQty().String(),
		Side:        t.sideTranslator.TranslateSideToDto(order.GetSide()),
	}

	if order.GetPrice().IsPositive() {
		rawDto.OrderType = "Limit"
		price := order.GetPrice().String()
		rawDto.Price = &price
	}

	return &dtos.OrderDto{
		Category:    symbol.Product(),
		RawOrderDto: rawDto,
	}
}

func (t *OrderTranslator) TranslateToCancelDto(symbol *domains.Symbol, orderID string) *dtos.CancelDto {
	return &dtos.CancelDto{
		Category:    symbol.Product(),
		OrderLinkID: orderID,
		Symbol:      symbol.Name(),
	}
}

func (t *OrderTranslator) TranslateToAmendDetailDto(symbol *domains.Symbol, orderID string, detail *deals.AmendDetail) *dtos.AmendDetailDto {
	return &dtos.AmendDetailDto{
		OrderLinkID: orderID,
		Category:    symbol.Product(),
		Symbol:      symbol.Name(),
		Price:       detail.Price.String(),
		Qty:         detail.Qty.String(),
	}
}

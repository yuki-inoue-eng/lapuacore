package translators

import (
	"strconv"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type OrderTranslator struct {
	sideTranslator      *sideTranslator
	orderTypeTranslator *orderTypeTranslator
}

func NewOrderTranslator() *OrderTranslator {
	return &OrderTranslator{
		sideTranslator:      newSideTranslator(),
		orderTypeTranslator: newOrderTypeTranslator(),
	}
}

func (t *OrderTranslator) TranslateToListDto(symbol *domains.Symbol, orders []*deals.Order) dtos.OrderListDto {
	var orderDtos []dtos.OrderDto
	for i := range orders {
		orderDtos = append(orderDtos, t.TranslateToDto(symbol, orders[i]))
	}
	return dtos.OrderListDto{Orders: orderDtos}
}

func (t *OrderTranslator) TranslateToDto(symbol *domains.Symbol, order *deals.Order) dtos.OrderDto {
	var price *string
	if order.GetOrderType().IsPricable() {
		p := order.GetPrice().String()
		price = &p
	}
	return dtos.OrderDto{
		Symbol:     symbol.Name(),
		MarketType: "FUTURES",
		Side:       t.sideTranslator.TranslateToDto(order.GetSide()),
		Type:       t.orderTypeTranslator.translateOrderTypeToDto(order.GetOrderType()),
		Amount:     order.GetQty().String(),
		Price:      price,
		ClientID:   order.GetID(),
		IsHide:     order.IsHide(),
	}
}

func (t *OrderTranslator) TranslateToAmendDetailDto(symbol *domains.Symbol, publicOrderID string, detail *deals.AmendDetail) (*dtos.AmendDetailDto, error) {
	pOrderID, err := strconv.ParseInt(publicOrderID, 10, 64)
	if err != nil {
		return nil, err
	}
	return &dtos.AmendDetailDto{
		PublicOrderID: pOrderID,
		Symbol:        symbol.Name(),
		MarketType:    "FUTURES",
		Amount:        detail.Qty.String(),
		Price:         detail.Price.String(),
	}, nil
}

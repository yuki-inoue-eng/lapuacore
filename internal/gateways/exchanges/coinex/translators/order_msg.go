package translators

import (
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

type OrderMsgTranslator struct {
	orderStatusTranslator *orderStatusTranslator
	sideTranslator        *sideTranslator
}

func NewOrderMsgTranslator() *OrderMsgTranslator {
	return &OrderMsgTranslator{
		orderStatusTranslator: newOrderStatusTranslator(),
		sideTranslator:        newSideTranslator(),
	}
}

func (t *OrderMsgTranslator) TranslateToData(confirmedAt *time.Time, data *dtos.OrderDataDto) (*deals.OrderData, error) {
	arrivedAt := time.UnixMilli(data.Order.UpdatedAt)

	price, err := decimal.NewFromString(data.Order.Price)
	if err != nil {
		return nil, err
	}
	qty, err := decimal.NewFromString(data.Order.Amount)
	if err != nil {
		return nil, err
	}
	fee, err := decimal.NewFromString(data.Order.Fee)
	if err != nil {
		return nil, err
	}
	cumExecQty, err := decimal.NewFromString(data.Order.LastFilledAmount)
	if err != nil {
		return nil, err
	}
	// LastFilledPrice is likely not the average execution price in CoinEx,
	// but it is the closest available field.
	avgPrice, err := decimal.NewFromString(data.Order.LastFilledPrice)
	if err != nil {
		return nil, err
	}
	filledQty, err := decimal.NewFromString(data.Order.FilledAmount)
	if err != nil {
		return nil, err
	}
	unFilledQty, err := decimal.NewFromString(data.Order.UnfilledAmount)
	if err != nil {
		return nil, err
	}

	return &deals.OrderData{
		ID:           data.Order.ClientID,
		PublicID:     strconv.FormatInt(data.Order.OrderID, 10),
		Side:         t.sideTranslator.Translate(data.Order.Side),
		Price:        price,
		Qty:          qty,
		AvgExecPrice: avgPrice,
		CumFee:       fee,
		CumExecQty:   cumExecQty,
		Status:       t.orderStatusTranslator.Translate(data.Event, filledQty, unFilledQty),
		ArrivedAt:    &arrivedAt,
		ConfirmedAt:  confirmedAt,
	}, nil
}

type orderStatusTranslator struct{}

func newOrderStatusTranslator() *orderStatusTranslator {
	return &orderStatusTranslator{}
}

// Translate maps CoinEx order events to OrderDataStatus.
// see: https://docs.coinex.com/api/v2/enum#order_event
func (t *orderStatusTranslator) Translate(event string, filledQty, unFilledQty decimal.Decimal) deals.OrderDataStatus {
	switch event {
	case "put":
		if filledQty.IsZero() {
			return deals.OrderDataStatusOpened
		}
		return deals.OrderDataStatusPartiallyFilled
	case "update":
		return deals.OrderDataStatusPartiallyFilled
	case "modify":
		return deals.OrderDataStatusOpened
	case "finish":
		if unFilledQty.IsZero() {
			return deals.OrderDataStatusFilled
		}
		return deals.OrderDataStatusCanceled
	default:
		return deals.OrderDataStatusUnknown
	}
}

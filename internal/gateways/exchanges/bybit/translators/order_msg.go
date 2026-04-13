package translators

import (
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
)

type OrderMsgTranslator struct {
	orderStatusTranslator *orderStatusTranslator
	sideTranslator        *SideTranslator
}

func NewOrderMsgTranslator() *OrderMsgTranslator {
	return &OrderMsgTranslator{
		orderStatusTranslator: newOrderStatusTranslator(),
		sideTranslator:        NewSideTranslator(),
	}
}

func (t *OrderMsgTranslator) TranslateToData(confirmedAt *time.Time, data *dtos.OrderDataDto) (*deals.OrderData, error) {
	arrivedAtMsStr, err := strconv.ParseInt(data.UpdatedTimeMs, 10, 64)
	if err != nil {
		return nil, err
	}
	arrivedAtMs := time.UnixMilli(arrivedAtMsStr)

	price, err := decimal.NewFromString(data.Price)
	if err != nil {
		return nil, err
	}

	qty, err := decimal.NewFromString(data.Qty)
	if err != nil {
		return nil, err
	}

	fee, err := decimal.NewFromString(data.Fee)
	if err != nil {
		return nil, err
	}

	avgPrice := decimal.Zero
	if data.AvgPrice != "" {
		avgPrice, err = decimal.NewFromString(data.AvgPrice)
		if err != nil {
			return nil, err
		}
	}

	cumExecQty, err := decimal.NewFromString(data.CumExecQty)
	if err != nil {
		return nil, err
	}

	return &deals.OrderData{
		ID:           data.OrderLinkID,
		PublicID:     data.OrderID,
		Side:         t.sideTranslator.Translate(data.Side),
		Price:        price,
		Qty:          qty,
		AvgExecPrice: avgPrice,
		CumFee:       fee,
		CumExecQty:   cumExecQty,
		Status:       t.orderStatusTranslator.Translate(data.OrderStatus),
		ArrivedAt:    &arrivedAtMs,
		ConfirmedAt:  confirmedAt,
	}, nil
}

type orderStatusTranslator struct{}

func newOrderStatusTranslator() *orderStatusTranslator {
	return &orderStatusTranslator{}
}

func (t *orderStatusTranslator) Translate(status string) deals.OrderDataStatus {
	switch status {
	case "New":
		return deals.OrderDataStatusOpened
	case "Rejected":
		return deals.OrderDataStatusRejected
	case "PartiallyFilled":
		return deals.OrderDataStatusPartiallyFilled
	case "Filled":
		return deals.OrderDataStatusFilled
	case "Cancelled":
		return deals.OrderDataStatusCanceled
	default:
		return deals.OrderDataStatusUnknown
	}
}

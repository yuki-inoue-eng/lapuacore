package dtos

type OrderMsgDto struct {
	Topic        string          `json:"topic"`
	ID           string          `json:"id"`
	CreationTime int64           `json:"creationTime"`
	DataDto      []*OrderDataDto `json:"data"`
}

type OrderDataDto struct {
	OrderLinkID string `json:"orderLinkId"`
	OrderID     string `json:"orderId"`

	Side  string `json:"side"`
	Price string `json:"price"`
	Qty   string `json:"qty"`
	Fee   string `json:"cumExecFee"`

	AvgPrice   string `json:"avgPrice"`
	CumExecQty string `json:"cumExecQty"`

	Category      string `json:"category"`
	Symbol        string `json:"symbol"`
	UpdatedTimeMs string `json:"updatedTime"`
	OrderStatus   string `json:"orderStatus"`
}

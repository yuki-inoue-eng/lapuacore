package dtos

type OrderDto struct {
	Category string `json:"category"`
	*RawOrderDto
}

type RawOrderDto struct {
	OrderLinkID string  `json:"orderLinkId"`
	Symbol      string  `json:"symbol"`
	OrderType   string  `json:"orderType"`
	Price       *string `json:"price"`
	Qty         string  `json:"qty"`
	Side        string  `json:"side"`
}

type CancelDto struct {
	Category    string `json:"category"`
	OrderLinkID string `json:"orderLinkId"`
	Symbol      string `json:"symbol"`
}

type AmendDetailDto struct {
	OrderLinkID string `json:"orderLinkId"`
	Category    string `json:"category"`
	Symbol      string `json:"symbol"`
	Price       string `json:"price"`
	Qty         string `json:"qty"`
}

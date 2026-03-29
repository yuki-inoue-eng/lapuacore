package dtos

type OrderListDto struct {
	Orders []OrderDto `json:"orders"`
}

type OrderDto struct {
	Symbol     string  `json:"market"`
	MarketType string  `json:"market_type"`
	Side       string  `json:"side"`
	Type       string  `json:"type"`
	Amount     string  `json:"amount"`
	Price      *string `json:"price,omitempty"` // omit for market orders
	ClientID   string  `json:"client_id"`
	IsHide     bool    `json:"is_hide"`
}

type AmendDetailDto struct {
	PublicOrderID int64  `json:"order_id"`
	Symbol        string `json:"market"`
	MarketType    string `json:"market_type"`
	Amount        string `json:"amount,omitempty"`
	Price         string `json:"price,omitempty"`
}

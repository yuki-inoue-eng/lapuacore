package dtos

type OrderMsgDto struct {
	Data *OrderDataDto `json:"data"`
}

type OrderDataDto struct {
	Event string `json:"event"`
	Order struct {
		OrderID          int64  `json:"order_id"`
		Market           string `json:"market"`
		Side             string `json:"side"`
		Type             string `json:"type"`
		Amount           string `json:"amount"`
		Price            string `json:"price"`
		UnfilledAmount   string `json:"unfilled_amount"`
		FilledAmount     string `json:"filled_amount"`
		FilledValue      string `json:"filled_value"`
		Fee              string `json:"fee"`
		FeeCcy           string `json:"fee_ccy"`
		TakerFeeRate     string `json:"taker_fee_rate"`
		MakerFeeRate     string `json:"maker_fee_rate"`
		ClientID         string `json:"client_id"`
		LastFilledAmount string `json:"last_filled_amount"`
		LastFilledPrice  string `json:"last_filled_price"`
		CreatedAt        int64  `json:"created_at"`
		UpdatedAt        int64  `json:"updated_at"`
	} `json:"order"`
}

package dtos

type TradeMsgDto struct {
	Data struct {
		Dtos []*TradeDataDto `json:"deal_list"`
	} `json:"data"`
}

type TradeDataDto struct {
	DealID          int64  `json:"deal_id"`
	TransactionTime int64  `json:"created_at"` // execution timestamp in milliseconds
	Side            string `json:"side"`
	Price           string `json:"price"`
	Volume          string `json:"amount"`
}

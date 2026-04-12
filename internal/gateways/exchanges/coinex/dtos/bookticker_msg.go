package dtos

type BookTickerMsgDto struct {
	Data BTDataDto `json:"data"`
}

type BTDataDto struct {
	Symbol   string `json:"market"`
	Ts       int64  `json:"updated_at"`
	BidPrice string `json:"best_bid_price"`
	BidSize  string `json:"best_bid_size"`
	AskPrice string `json:"best_ask_price"`
	AskSize  string `json:"best_ask_size"`
}

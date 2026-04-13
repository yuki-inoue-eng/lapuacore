package dtos

type TradeMsgDto struct {
	Topic   string          `json:"topic"`
	Type    string          `json:"type"`
	Ts      int64           `json:"ts"`
	DataDto []*TradeDataDto `json:"data"`
}

type TradeDataDto struct {
	Timestamp int64  `json:"T"`
	TradeID   string `json:"i"`
	Symbol    string `json:"s"`
	Side      string `json:"S"`
	Volume    string `json:"v"`
	Price     string `json:"p"`
}

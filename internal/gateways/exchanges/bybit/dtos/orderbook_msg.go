package dtos

type OrderBookMsgDto struct {
	Topic string    `json:"topic"`
	Type  string    `json:"type"`
	Ts    int64     `json:"ts"`
	Data  OBDataDto `json:"data"`
}

type OBDataDto struct {
	Bids     [][]string `json:"b"`
	Asks     [][]string `json:"a"`
	UpdateID int64      `json:"u"`
}

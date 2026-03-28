package dtos

type OrderBookMsgDto struct {
	Data OBDeltaDataDto `json:"data"`
}

type OBDeltaDataDto struct {
	Symbol     string    `json:"market"`
	IsSnapshot bool      `json:"is_full"`
	Depth      OBRawData `json:"depth"`
}

type OBRawData struct {
	UpdatedAt int64      `json:"updated_at"`
	Bids      [][]string `json:"bids"`
	Asks      [][]string `json:"asks"`
}

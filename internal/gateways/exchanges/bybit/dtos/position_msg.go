package dtos

type PositionMsgDto struct {
	Topic        string             `json:"topic"`
	ID           string             `json:"id"`
	CreationTime int64              `json:"creationTime"`
	DataDto      []*PositionDataDto `json:"data"`
}

type PositionDataDto struct {
	PositionIdx   int    `json:"positionIdx"`
	PositionValue string `json:"positionValue"`
	Side          string `json:"side"`
	Qty           string `json:"size"`

	Category      string `json:"category"`
	Symbol        string `json:"symbol"`
	UpdatedTimeMs string `json:"updatedTime"`
}

package dtos

type PositionMsgDto struct {
	Method string           `json:"method"`
	Data   *PositionDataDto `json:"data"`
	ID     any              `json:"id"`
}

type PositionDataDto struct {
	Event    string `json:"event"`
	Position struct {
		Market       string `json:"market"`
		Side         string `json:"side"` // long or short
		OpenInterest string `json:"open_interest"`
		CreatedAt    int64  `json:"created_at"`
		UpdatedAt    int64  `json:"updated_at"`
	} `json:"position"`
}

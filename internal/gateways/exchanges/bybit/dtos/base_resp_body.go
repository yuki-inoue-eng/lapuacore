package dtos

type BaseRespBody struct {
	RetCode     int    `json:"retCode"`
	RetMsg      string `json:"retMsg"`
	TimestampMs int64  `json:"time"`
}

func (b *BaseRespBody) IsRetCodeOK() bool {
	const retCodeOK = 0
	return b.RetCode == retCodeOK
}

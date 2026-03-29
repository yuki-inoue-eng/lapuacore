package dtos

import "encoding/json"

// OrdersRespBody is the response body for batch order operations.
// see: https://docs.coinex.com/api/v2/futures/order/http/put-multi-order#response-example
type OrdersRespBody struct {
	BaseRespBody
	Data []OrderRespBody `json:"data"`
}

type OrderRespBody struct {
	BaseRespBody
	Data *OrderRespData `json:"data"`
}

type OrderRespData struct {
	PublicOrderID int64  `json:"order_id"`
	ClientID      string `json:"client_id"`
	UpdatedAt     int64  `json:"updated_at"`
}

type BaseRespBody struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

func (b *BaseRespBody) IsRetCodeOK() bool {
	const retCodeOK = 0
	return b.Code == retCodeOK
}

func (o *OrdersRespBody) UnmarshalJSON(data []byte) error {
	type Alias OrdersRespBody
	aux := &struct {
		Data json.RawMessage `json:"data"`
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var arrayCheck []json.RawMessage
	if err := json.Unmarshal(aux.Data, &arrayCheck); err != nil {
		// data is not an array (CoinEx returns "data":{} on error)
		o.Data = nil
		return nil
	}
	if err := json.Unmarshal(aux.Data, &o.Data); err != nil {
		return err
	}
	return nil
}

package agent

const (
	hostName     = "api.coinex.com"
	pingEndpoint = "/v2/ping"

	createOrderEndpoint      = "/v2/futures/order"
	cancelOrderEndpoint      = "/v2/futures/cancel-order"
	amendOrderEndpoint       = "/v2/futures/modify-order"
	batchCreateOrderEndpoint = "/v2/futures/batch-order"
	batchCancelOrderEndpoint = "/v2/futures/cancel-batch-order"
)
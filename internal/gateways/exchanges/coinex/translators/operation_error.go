package translators

import (
	"fmt"

	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
)

// OperationErrorTranslator translates CoinEx error codes to domain errors.
// see: https://docs.coinex.com/api/v2/error
type OperationErrorTranslator struct{}

func NewOperationErrorTranslator() *OperationErrorTranslator {
	return &OperationErrorTranslator{}
}

func (t *OperationErrorTranslator) Translate(respErr *dtos.BaseRespBody) error {
	errMsg := fmt.Sprintf("coinex error (%d): %s", respErr.Code, respErr.Msg)
	switch respErr.Code {
	case 4008, 4010, 4017, 4512:
		return deals.Error(deals.AuthError, errMsg)
	case 3109:
		return deals.Error(deals.InsufficientBalanceError, errMsg)
	case 3103:
		return deals.Error(deals.OrderIsNotExists, errMsg)
	case 3132:
		// 3132: position is closing
		return deals.Error(deals.WarnError, errMsg)
	case 3129, 3008, 3007, 227:
		// 3129: price setting error
		// 3008: server overloaded
		// 3007: service unavailable
		// 227:  tonce check error
		return deals.Error(deals.InfoError, errMsg)
	case 4123, 4213:
		return deals.Error(deals.ExecLateLimitError, errMsg)
	default:
		return deals.Error(deals.OtherError, errMsg)
	}
}

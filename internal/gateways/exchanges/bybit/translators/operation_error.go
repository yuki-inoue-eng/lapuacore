package translators

import (
	"fmt"

	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
)

type OperationErrorTranslator struct{}

func NewOperationErrorTranslator() *OperationErrorTranslator {
	return &OperationErrorTranslator{}
}

func (t *OperationErrorTranslator) errMsg(code int, msg string) string {
	return fmt.Sprintf("bybit error (%d): %s", code, msg)
}

func (t *OperationErrorTranslator) Translate(respErr *dtos.BaseRespBody) error {
	errMsg := t.errMsg(respErr.RetCode, respErr.RetMsg)
	switch respErr.RetCode {
	case 10000:
		return deals.Error(deals.ServerTimeoutError, errMsg)
	case 10001:
		return deals.Error(deals.ParameterError, errMsg)
	case 10003, 10004, 10005, 10007:
		return deals.Error(deals.AuthError, errMsg)
	case 10006:
		return deals.Error(deals.ExecLateLimitError, errMsg)
	case 110001:
		return deals.Error(deals.OrderIsNotExists, errMsg)
	default:
		return deals.Error(deals.OtherError, errMsg)
	}
}

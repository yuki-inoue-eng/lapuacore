package deals

import (
	"errors"
	"fmt"
)

var (
	ServerTimeoutError       = errors.New("server timeout")
	ParameterError           = errors.New("parameter error")
	PriceSettingError        = errors.New("price setting error")
	AuthError                = errors.New("auth error")
	ExecLateLimitError       = errors.New("exec late limit error")
	LateLimitError           = errors.New("late limit error")
	InsufficientBalanceError = errors.New("insufficient balance error")
	AccountLockError         = errors.New("account locked")
	HttpRequestError         = errors.New("http request error")

	InfoError  = errors.New("info error")
	WarnError  = errors.New("warn error")
	OtherError = errors.New("other error")

	OrderIsNotExists = errors.New("order is not exists in exchange")
)

var (
	DealingErrorOrderIsAlreadyExists      = errors.New("order is already exists")
	DealingErrorOrderNotReadyForOperation = errors.New("order is not ready for operation")
	DealingErrorOrderNotFound             = errors.New("order not found")
	DealingErrorOrderIsNotAmendable       = errors.New("order is not amendable")
)

func Error(err error, msg string) error {
	return fmt.Errorf("%w: %v", err, errors.New(msg))
}
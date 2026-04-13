package gateways

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	AuthErrFailedToSendAuthRequest = errors.New("failed to send auth request")
	AuthErrRetryLimitReached       = errors.New("failed to auth: retry limit reached")
)

// AuthRequestSender sends an exchange-specific auth request over the connection.
type AuthRequestSender func(conn *websocket.Conn, credential Credential) error

// AuthResponseChecker parses a raw WebSocket message and determines if it is
// an auth response. Returns isAuthResp=false for unrelated messages.
// On auth response: success indicates whether authentication succeeded,
// errDetail provides a human-readable failure reason.
type AuthResponseChecker func(rawMsg []byte) (isAuthResp bool, success bool, errDetail string, err error)

// authResult is the internal signal sent from HandleAuthMessage to Start.
type authResult struct {
	success   bool
	errDetail string
}

type author struct {
	exName     string
	conn       *websocket.Conn
	credential Credential

	sendRequest   AuthRequestSender
	checkResponse AuthResponseChecker

	numOfRetry    int
	retryCount    int
	retryInterval time.Duration

	chanCloseOnce  sync.Once
	authResultChan chan authResult
	authErrChan    chan error
}

// NewAuthor creates a shared Author with exchange-specific auth behavior
// injected via sendRequest and checkResponse.
func NewAuthor(
	exName string,
	conn *websocket.Conn,
	credential Credential,
	numOfRetry int,
	retryInterval time.Duration,
	sendRequest AuthRequestSender,
	checkResponse AuthResponseChecker,
) Author {
	return &author{
		exName:         exName,
		conn:           conn,
		credential:     credential,
		numOfRetry:     numOfRetry,
		retryInterval:  retryInterval,
		sendRequest:    sendRequest,
		checkResponse:  checkResponse,
		authResultChan: make(chan authResult),
		authErrChan:    make(chan error),
	}
}

func (a *author) GetAuthErrChan() <-chan error {
	return a.authErrChan
}

func (a *author) chanClose() {
	a.chanCloseOnce.Do(func() {
		close(a.authResultChan)
		close(a.authErrChan)
	})
}

// HandleAuthMessage checks if the message is an auth response and signals the result.
func (a *author) HandleAuthMessage(rawMsg []byte) error {
	isAuthResp, success, errDetail, err := a.checkResponse(rawMsg)
	if err != nil {
		return err
	}
	if !isAuthResp {
		return nil
	}
	a.authResultChan <- authResult{success: success, errDetail: errDetail}
	return nil
}

// Start sends the auth request and retries on failure.
func (a *author) Start(ctx context.Context) {
	defer a.chanClose()
	if err := a.sendRequest(a.conn, a.credential); err != nil {
		a.authErrChan <- AuthErrFailedToSendAuthRequest
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case result := <-a.authResultChan:
			if result.success {
				slog.Info(fmt.Sprintf("%s private channel authenticated", a.exName))
				continue
			}
			slog.Error(fmt.Sprintf("%s auth failed: %s", a.exName, result.errDetail))
			if a.retryCount >= a.numOfRetry {
				a.authErrChan <- AuthErrRetryLimitReached
				return
			}
			time.Sleep(a.retryInterval)
			a.retryCount++
			slog.Info(fmt.Sprintf("retry %s auth: %d", a.exName, a.retryCount))
			if err := a.sendRequest(a.conn, a.credential); err != nil {
				a.authErrChan <- AuthErrFailedToSendAuthRequest
				return
			}
		}
	}
}

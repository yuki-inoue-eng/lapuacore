package ws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

var (
	authErrFailedToSendAuthRequest = errors.New("failed to send auth request")
	authErrRetryLimitReached       = errors.New("failed to auth: retry limit reached")
)

type authRespMsg struct {
	ID      int64  `json:"id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type author struct {
	conn       *websocket.Conn
	credential gateways.Credential
	msgID      int64

	numOfRetry    int
	retryCount    int
	retryInterval time.Duration

	chanCloseOnce sync.Once
	authRespChan  chan authRespMsg
	authErrChan   chan error
}

func newAuthor(conn *websocket.Conn, credential gateways.Credential, numOfRetry int, retryInterval time.Duration) gateways.Author {
	return &author{
		conn:          conn,
		credential:    credential,
		msgID:         genAuthMsgID(),
		numOfRetry:    numOfRetry,
		retryInterval: retryInterval,
		authRespChan:  make(chan authRespMsg),
		authErrChan:   make(chan error),
	}
}

func (a *author) GetAuthErrChan() <-chan error {
	return a.authErrChan
}

func (a *author) chanClose() {
	a.chanCloseOnce.Do(func() {
		close(a.authRespChan)
		close(a.authErrChan)
	})
}

// HandleAuthMessage checks if the message is an auth response and signals the result.
func (a *author) HandleAuthMessage(rawMsg []byte) error {
	msg := authRespMsg{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil
	}
	if msg.ID != a.msgID {
		return nil
	}
	a.authRespChan <- msg
	return nil
}

// Start sends the auth request and retries on failure.
func (a *author) Start(ctx context.Context) {
	defer a.chanClose()
	if err := a.sendAuthRequest(); err != nil {
		a.authErrChan <- authErrFailedToSendAuthRequest
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-a.authRespChan:
			if msg.Code == 0 {
				slog.Info("coinex private channel authenticated")
				continue
			}
			slog.Error(fmt.Sprintf("coinex auth failed (code=%d): %s", msg.Code, msg.Message))
			if a.retryCount >= a.numOfRetry {
				a.authErrChan <- authErrRetryLimitReached
				return
			}
			time.Sleep(a.retryInterval)
			a.retryCount++
			slog.Info(fmt.Sprintf("retry coinex auth: %d", a.retryCount))
			if err := a.sendAuthRequest(); err != nil {
				a.authErrChan <- authErrFailedToSendAuthRequest
				return
			}
		}
	}
}

func (a *author) sendAuthRequest() error {
	return a.conn.WriteMessage(websocket.TextMessage, a.buildAuthMsg())
}

func (a *author) buildAuthMsg() []byte {
	apiKey := a.credential.GetApiKey()
	secret := a.credential.GetSecret()
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp))
	sign := hex.EncodeToString(h.Sum(nil))

	return []byte(fmt.Sprintf(`{
		"id": %d,
		"method": "server.sign",
		"params": {
			"access_id": "%s",
			"signed_str": "%s",
			"timestamp": %s
		}
	}`, a.msgID, apiKey, sign, timestamp))
}

func genAuthMsgID() int64 {
	u := uuid.New()
	i := new(big.Int)
	i.SetBytes(u[:8])
	v := i.Int64()
	if v < 0 {
		v = ^v
	}
	return v
}

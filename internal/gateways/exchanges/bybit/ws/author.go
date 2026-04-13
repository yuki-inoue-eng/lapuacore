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
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

var (
	authErrFailedToSendAuthRequest = errors.New("failed to send auth request")
	authErrRetryLimitReached       = errors.New("failed to auth: retry limit reached")
)

type authResponseMsg struct {
	Op      string `json:"op"`
	Success bool   `json:"success"`
	ConnID  string `json:"conn_id"`
}

func (m *authResponseMsg) isAuthenticated() bool {
	return m.Success
}

type author struct {
	conn       *websocket.Conn
	credential gateways.Credential

	numOfRetry    int
	retryCount    int
	retryInterval time.Duration

	chanCloseOnce sync.Once
	authRespChan  chan authResponseMsg
	authErrChan   chan error
}

func newAuthor(conn *websocket.Conn, credential gateways.Credential, numOfRetry int, retryInterval time.Duration) *author {
	return &author{
		conn:          conn,
		credential:    credential,
		numOfRetry:    numOfRetry,
		retryInterval: retryInterval,
		authRespChan:  make(chan authResponseMsg),
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

func (a *author) start(ctx context.Context) {
	defer a.chanClose()
	if err := a.sendAuthRequest(); err != nil {
		a.authErrChan <- authErrFailedToSendAuthRequest
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-a.authRespChan:
			if msg.isAuthenticated() {
				slog.Info("bybit private channel authenticated")
				continue
			} else {
				slog.Error(fmt.Sprintf("failed to auth: %v", msg))
			}
			if a.retryCount >= a.numOfRetry {
				a.authErrChan <- authErrRetryLimitReached
			}
			time.Sleep(a.retryInterval)
			a.retryCount++
			slog.Info(fmt.Sprintf("retry auth: %d", a.retryCount))
			if err := a.sendAuthRequest(); err != nil {
				a.authErrChan <- authErrFailedToSendAuthRequest
			}
		}
	}
}

func (a *author) sendAuthRequest() error {
	if err := a.conn.WriteMessage(websocket.TextMessage, a.genAuthMessage()); err != nil {
		return err
	}
	return nil
}

func (a *author) genAuthMessage() []byte {
	signature, expire := a.signWss("GET/realtime", 3*time.Second)
	return []byte(fmt.Sprintf(`{"op": "auth", "args": ["%s", %d,"%s"]}`, a.credential.GetApiKey(), expire, signature))
}

func (a *author) isAuthRespMsg(rawMsg []byte) (bool, error) {
	msg := struct {
		Op string `json:"op"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return false, err
	}
	return msg.Op == "auth", nil
}

func (a *author) handleAuthMessage(rawMsg []byte) error {
	if isAuthMsg, err := a.isAuthRespMsg(rawMsg); err != nil {
		return err
	} else if !isAuthMsg {
		return nil
	}
	var msg authResponseMsg
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return err
	}
	a.authRespChan <- msg
	return nil
}

func (a *author) signWss(endpoint string, lifetime time.Duration) (string, int64) {
	expire := time.Now().Add(lifetime).UnixMilli()
	data := endpoint + strconv.FormatInt(expire, 10)
	hmac256 := hmac.New(sha256.New, []byte(a.credential.GetSecret()))
	hmac256.Write([]byte(data))
	return hex.EncodeToString(hmac256.Sum(nil)), expire
}

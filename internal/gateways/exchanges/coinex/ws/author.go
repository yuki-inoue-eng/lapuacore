package ws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
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

	once     sync.Once
	authDone chan struct{}
	authFail chan error
}

func newAuthor(conn *websocket.Conn, credential gateways.Credential) *author {
	return &author{
		conn:       conn,
		credential: credential,
		msgID:      genAuthMsgID(),
		authDone:   make(chan struct{}),
		authFail:   make(chan error, 1),
	}
}

func (a *author) sendAuthRequest() error {
	return a.conn.WriteMessage(websocket.TextMessage, a.buildAuthMsg())
}

// handleMessage is registered as a message handler on the receiver.
// It signals authDone or authFail when the auth response arrives.
func (a *author) handleMessage(rawMsg []byte) error {
	msg := authRespMsg{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil
	}
	if msg.ID != a.msgID {
		return nil
	}
	a.once.Do(func() {
		if msg.Code == 0 {
			close(a.authDone)
		} else {
			a.authFail <- fmt.Errorf("auth failed (code=%d): %s", msg.Code, msg.Message)
		}
	})
	return nil
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
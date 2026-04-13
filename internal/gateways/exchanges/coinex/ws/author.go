package ws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

// coinexAuthState holds per-connection state needed for CoinEx auth.
type coinexAuthState struct {
	msgID int64
}

func newAuthor(conn *websocket.Conn, credential gateways.Credential, numOfRetry int, retryInterval time.Duration) gateways.Author {
	state := &coinexAuthState{msgID: genAuthMsgID()}
	return gateways.NewAuthor(
		"coinex",
		conn,
		credential,
		numOfRetry,
		retryInterval,
		state.sendRequest,
		state.checkResponse,
	)
}

func (s *coinexAuthState) sendRequest(conn *websocket.Conn, credential gateways.Credential) error {
	apiKey := credential.GetApiKey()
	secret := credential.GetSecret()
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(timestamp))
	sign := hex.EncodeToString(h.Sum(nil))

	msg := []byte(fmt.Sprintf(`{
		"id": %d,
		"method": "server.sign",
		"params": {
			"access_id": "%s",
			"signed_str": "%s",
			"timestamp": %s
		}
	}`, s.msgID, apiKey, sign, timestamp))
	return conn.WriteMessage(websocket.TextMessage, msg)
}

func (s *coinexAuthState) checkResponse(rawMsg []byte) (isAuthResp bool, success bool, errDetail string, err error) {
	msg := struct {
		ID      int64  `json:"id"`
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return false, false, "", nil
	}
	if msg.ID != s.msgID {
		return false, false, "", nil
	}
	if msg.Code == 0 {
		return true, true, "", nil
	}
	return true, false, fmt.Sprintf("code=%d: %s", msg.Code, msg.Message), nil
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

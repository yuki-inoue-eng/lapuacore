package ws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

func newAuthor(conn *websocket.Conn, credential gateways.Credential, numOfRetry int, retryInterval time.Duration) gateways.Author {
	return gateways.NewAuthor(
		"bybit",
		conn,
		credential,
		numOfRetry,
		retryInterval,
		bybitSendAuthRequest,
		bybitCheckAuthResponse,
	)
}

// bybitSendAuthRequest sends a Bybit WebSocket auth message with HMAC signature.
func bybitSendAuthRequest(conn *websocket.Conn, credential gateways.Credential) error {
	expire := time.Now().Add(3 * time.Second).UnixMilli()
	data := "GET/realtime" + strconv.FormatInt(expire, 10)
	hmac256 := hmac.New(sha256.New, []byte(credential.GetSecret()))
	hmac256.Write([]byte(data))
	signature := hex.EncodeToString(hmac256.Sum(nil))

	msg := []byte(fmt.Sprintf(`{"op": "auth", "args": ["%s", %d,"%s"]}`, credential.GetApiKey(), expire, signature))
	return conn.WriteMessage(websocket.TextMessage, msg)
}

// bybitCheckAuthResponse parses a Bybit auth response message.
func bybitCheckAuthResponse(rawMsg []byte) (isAuthResp bool, success bool, errDetail string, err error) {
	msg := struct {
		Op      string `json:"op"`
		Success bool   `json:"success"`
		RetMsg  string `json:"ret_msg"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return false, false, "", err
	}
	if msg.Op != "auth" {
		return false, false, "", nil
	}
	if msg.Success {
		return true, true, "", nil
	}
	return true, false, msg.RetMsg, nil
}

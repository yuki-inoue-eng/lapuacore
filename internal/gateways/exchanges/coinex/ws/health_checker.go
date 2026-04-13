package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

func newHealthChecker(conn *websocket.Conn, pingInterval, timeoutDuration time.Duration) gateways.HealthChecker {
	return gateways.NewHealthChecker(conn, pingInterval, timeoutDuration, coinexPingSender, coinexPongMatcher)
}

// coinexPingSender sends a CoinEx server.ping TextMessage.
func coinexPingSender(conn *websocket.Conn) error {
	now := time.Now().Unix()
	pingMsg := fmt.Sprintf(`{"method":"server.ping","params":{},"id":%d}`, now)
	return conn.WriteMessage(websocket.TextMessage, []byte(pingMsg))
}

// coinexPongMatcher detects CoinEx pong responses returned as TextMessage.
func coinexPongMatcher(rawMsg []byte) bool {
	msg := struct {
		Data struct {
			Result string `json:"result"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return false
	}
	return msg.Data.Result == "pong"
}

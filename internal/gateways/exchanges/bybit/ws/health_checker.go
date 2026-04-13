package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

func newHealthChecker(conn *websocket.Conn, pingInterval, timeoutDuration time.Duration) gateways.HealthChecker {
	// Bybit uses WebSocket-level ping/pong, so matchPong is nil.
	return gateways.NewHealthChecker(conn, pingInterval, timeoutDuration, bybitPingSender, nil)
}

// bybitPingSender sends a WebSocket PingMessage with a JSON req_id.
func bybitPingSender(conn *websocket.Conn) error {
	msg, err := json.Marshal(struct {
		ReqID int64 `json:"req_id"`
	}{ReqID: time.Now().Unix()})
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.PingMessage, msg)
}

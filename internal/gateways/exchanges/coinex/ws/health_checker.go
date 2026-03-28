package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	healthErrFailedToSendPing  = errors.New("failed to send ping msg")
	healthErrConnectionTimeout = errors.New("connection is timeout")
)

type healthChecker struct {
	conn            *websocket.Conn
	pingInterval    time.Duration
	timeoutDuration time.Duration
	lastSendPingAt  int64
	lastReceivedAt  int64

	chanCloseOnce   sync.Once
	pongChan        chan []byte
	healthAlertChan chan error
}

func newHealthChecker(conn *websocket.Conn, pingInterval, timeoutDuration time.Duration) *healthChecker {
	pongChan := make(chan []byte)
	conn.SetPongHandler(func(msg string) error {
		pongChan <- []byte(msg)
		return nil
	})
	return &healthChecker{
		conn:            conn,
		pongChan:        pongChan,
		healthAlertChan: make(chan error),
		pingInterval:    pingInterval,
		timeoutDuration: timeoutDuration,
	}
}

// pongReceiveHandleFunc は CoinEx が TextMessage で返す pong を検出する
func (c *healthChecker) pongReceiveHandleFunc(rawMsg []byte) error {
	msg := struct {
		Data struct {
			Result string `json:"result"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil
	}
	if msg.Data.Result == "pong" {
		c.pongChan <- rawMsg
	}
	return nil
}

func (c *healthChecker) getHealthAlertChan() <-chan error {
	return c.healthAlertChan
}

func (c *healthChecker) sendPing() error {
	now := time.Now().Unix()
	c.lastSendPingAt = now
	pingMsg := fmt.Sprintf(`{"method":"server.ping","params":{},"id":%d}`, now)
	return c.conn.WriteMessage(websocket.TextMessage, []byte(pingMsg))
}

func (c *healthChecker) isTimeout() bool {
	if c.lastSendPingAt == 0 {
		return false
	}
	if c.lastReceivedAt == 0 {
		return time.Since(time.Unix(c.lastSendPingAt, 0)) > c.timeoutDuration
	}
	return time.Since(time.Unix(c.lastReceivedAt, 0)) > c.timeoutDuration
}

func (c *healthChecker) chanClose() {
	c.chanCloseOnce.Do(func() {
		close(c.healthAlertChan)
		close(c.pongChan)
	})
}

func (c *healthChecker) start(ctx context.Context) {
	pingTicker := time.NewTicker(c.pingInterval)
	defer pingTicker.Stop()
	defer c.chanClose()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := c.sendPing(); err != nil {
				c.healthAlertChan <- healthErrFailedToSendPing
			}
			if c.isTimeout() {
				c.healthAlertChan <- healthErrConnectionTimeout
			}
		case <-c.pongChan:
			c.lastReceivedAt = time.Now().Unix()
		}
	}
}
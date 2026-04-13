package ws

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	healthErrFailedToSendPing  = errors.New("failed to send ping msg")
	healthErrConnectionTimeout = errors.New("connection is timeout")
)

type pingPongMessage struct {
	ReqID int64 `json:"req_id"`
}

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
	conn.SetPongHandler(func(pongMsgStr string) error {
		pongChan <- []byte(pongMsgStr)
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

func (c *healthChecker) GetHealthAlertChan() <-chan error {
	return c.healthAlertChan
}

func (c *healthChecker) sendPing() error {
	now := time.Now().Unix()
	pingPongMsg := pingPongMessage{ReqID: now}
	msg, err := json.Marshal(&pingPongMsg)
	if err != nil {
		return err
	}
	if err := c.conn.WriteMessage(websocket.PingMessage, msg); err != nil {
		return err
	}
	c.lastSendPingAt = now
	return nil
}

func (c *healthChecker) isTimeout() bool {
	if c.lastSendPingAt == 0 {
		return false
	}
	if c.lastReceivedAt == 0 {
		return time.Duration(time.Now().Unix()-c.lastSendPingAt) > c.timeoutDuration
	}
	return time.Duration(time.Now().Unix()-c.lastReceivedAt) > c.timeoutDuration
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

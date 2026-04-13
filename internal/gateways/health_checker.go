package gateways

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	HealthErrFailedToSendPing  = errors.New("failed to send ping msg")
	HealthErrConnectionTimeout = errors.New("connection is timeout")
)

// PingSender sends a ping message over the WebSocket connection.
type PingSender func(conn *websocket.Conn) error

// PongMatcher checks if a raw message is a pong response.
// Returns true if the message is a pong.
type PongMatcher func(rawMsg []byte) bool

type healthChecker struct {
	conn            *websocket.Conn
	pingInterval    time.Duration
	timeoutDuration time.Duration
	lastSendPingAt  time.Time
	lastReceivedAt  time.Time

	sendPing  PingSender
	matchPong PongMatcher

	chanCloseOnce   sync.Once
	pongChan        chan []byte
	healthAlertChan chan error
}

// NewHealthChecker creates a shared HealthChecker with exchange-specific
// ping/pong behavior injected via sendPing and matchPong.
// If matchPong is nil, pong detection relies solely on conn.SetPongHandler.
func NewHealthChecker(
	conn *websocket.Conn,
	pingInterval, timeoutDuration time.Duration,
	sendPing PingSender,
	matchPong PongMatcher,
) HealthChecker {
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
		sendPing:        sendPing,
		matchPong:       matchPong,
	}
}

func (c *healthChecker) GetHealthAlertChan() <-chan error {
	return c.healthAlertChan
}

// PongReceiveHandleFunc detects pong responses delivered as regular messages
// (e.g. CoinEx returns pong as TextMessage). If matchPong is nil this is a no-op.
func (c *healthChecker) PongReceiveHandleFunc(rawMsg []byte) error {
	if c.matchPong == nil {
		return nil
	}
	if c.matchPong(rawMsg) {
		c.pongChan <- rawMsg
	}
	return nil
}

func (c *healthChecker) isTimeout() bool {
	if c.lastSendPingAt.IsZero() {
		return false
	}
	if c.lastReceivedAt.IsZero() {
		return time.Since(c.lastSendPingAt) > c.timeoutDuration
	}
	return time.Since(c.lastReceivedAt) > c.timeoutDuration
}

func (c *healthChecker) chanClose() {
	c.chanCloseOnce.Do(func() {
		close(c.healthAlertChan)
		close(c.pongChan)
	})
}

func (c *healthChecker) Start(ctx context.Context) {
	pingTicker := time.NewTicker(c.pingInterval)
	defer pingTicker.Stop()
	defer c.chanClose()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := c.sendPing(c.conn); err != nil {
				c.healthAlertChan <- HealthErrFailedToSendPing
			}
			c.lastSendPingAt = time.Now()
			if c.isTimeout() {
				c.healthAlertChan <- HealthErrConnectionTimeout
			}
		case <-c.pongChan:
			c.lastReceivedAt = time.Now()
		}
	}
}

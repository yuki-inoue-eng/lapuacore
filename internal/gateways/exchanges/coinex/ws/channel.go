package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/ws/topics"
)

const wsEndpoint = "wss://socket.coinex.com/v2/futures"

type ChannelStatus int

const (
	ChannelStatusUnknown ChannelStatus = iota
	ChannelStatusConnected
	ChannelStatusDisconnected
	ChannelStatusReconnecting
)

type Channel struct {
	mu        sync.RWMutex
	isRunning bool
	status    ChannelStatus

	conn              *websocket.Conn
	topicMg           *topics.Manager
	reconnectInterval time.Duration

	healthChecker *healthChecker
	msgReceiver   *messageReceiver

	reconnectCount   int
	reconnectSigChan chan struct{}

	msgChan   <-chan []byte
	errChan   <-chan error
	alertChan <-chan error
}

func NewPublicChannel() *Channel {
	return &Channel{
		topicMg:          topics.NewManager(),
		reconnectInterval: 1 * time.Second,
		reconnectSigChan: make(chan struct{}, 1000),
	}
}

func (c *Channel) SetTopics(ts []topics.Topic) {
	c.topicMg.SetTopics(ts)
}

func (c *Channel) GetStatus() ChannelStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Channel) setStatus(status ChannelStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func (c *Channel) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *Channel) resetReconnectSigChan() {
	close(c.reconnectSigChan)
	c.reconnectSigChan = make(chan struct{}, 1000)
}

func (c *Channel) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(wsEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}
	c.conn = conn

	const (
		pingInterval    = 1 * time.Second
		timeoutDuration = 1 * time.Second
	)

	c.msgReceiver = newMessageReceiver(c.conn)
	c.msgChan = c.msgReceiver.getMsgChan()
	c.errChan = c.msgReceiver.getErrChan()

	c.healthChecker = newHealthChecker(c.conn, pingInterval, timeoutDuration)
	c.alertChan = c.healthChecker.getHealthAlertChan()
	c.msgReceiver.setHandler(c.healthChecker.pongReceiveHandleFunc)
	c.msgReceiver.setHandler(c.topicMg.HandleSubscribeResp)

	return nil
}

func (c *Channel) initAndListen(ctx context.Context) error {
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize channel: %w", err)
	}

	go c.msgReceiver.start()
	time.Sleep(200 * time.Millisecond)
	go c.healthChecker.start(ctx)
	time.Sleep(200 * time.Millisecond)

	for _, msg := range c.topicMg.SubscribeRequests() {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return fmt.Errorf("failed to send subscribe request: %w", err)
		}
	}
	return nil
}

// Start initializes the channel, starts subscriptions and health checks.
// It automatically reconnects on disconnection. Cancel ctx to stop.
func (c *Channel) Start(ctx context.Context) error {
	c.mu.Lock()
	c.isRunning = true
	c.mu.Unlock()

	wsCtx, wsCancel := context.WithCancel(context.Background())
	defer wsCancel()

	if err := c.initAndListen(wsCtx); err != nil {
		return err
	}

	reconnect := func() error {
		wsCancel()
		wsCtx, wsCancel = context.WithCancel(context.Background())
		return c.initAndListen(wsCtx)
	}

	c.setStatus(ChannelStatusConnected)
	for {
		select {
		case <-ctx.Done():
			if err := c.conn.Close(); err != nil {
				slog.Error(fmt.Sprintf("failed to close websocket: %v", err))
			}
			wsCancel()
			c.mu.Lock()
			c.isRunning = false
			c.mu.Unlock()
			return nil

		case err := <-c.alertChan:
			slog.Error(fmt.Sprintf("receive health alert (coinex): %v", err))
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case err := <-c.errChan:
			slog.Error(fmt.Sprintf("receive error (coinex): %v", err))
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case <-c.reconnectSigChan:
			time.Sleep(c.reconnectInterval)
			c.setStatus(ChannelStatusReconnecting)
			c.resetReconnectSigChan()
			slog.Info("try to reconnect (coinex)")
			if err := reconnect(); err != nil {
				c.setStatus(ChannelStatusDisconnected)
				c.reconnectCount++
				slog.Error(fmt.Sprintf("failed to reconnect (coinex count: %d): %v", c.reconnectCount, err))
				c.reconnectSigChan <- struct{}{}
			} else {
				c.setStatus(ChannelStatusConnected)
				c.reconnectCount = 0
				slog.Info("succeeded to reconnect (coinex)")
			}

		case rawMsg := <-c.msgChan:
			timestamp := time.Now()
			if err := c.topicMg.HandleTopicMessage(&timestamp, rawMsg); err != nil {
				slog.Error(fmt.Sprintf("failed to handle message: %v", err), "message", string(rawMsg))
			}
		}
	}
}
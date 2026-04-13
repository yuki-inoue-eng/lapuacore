package gateways

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// AuthorConstructor creates an Author for private channel authentication.
type AuthorConstructor func(conn *websocket.Conn, credential Credential, numOfRetry int, retryInterval time.Duration) Author

// Author handles private channel authentication.
type Author interface {
	GetAuthErrChan() <-chan error
	HandleAuthMessage(rawMsg []byte) error
	Start(ctx context.Context)
}

// HealthCheckerConstructor creates a HealthChecker for connection monitoring.
type HealthCheckerConstructor func(conn *websocket.Conn, pingInterval, timeoutDuration time.Duration) HealthChecker

// HealthChecker monitors WebSocket connection health via ping/pong.
type HealthChecker interface {
	GetHealthAlertChan() <-chan error
	PongReceiveHandleFunc(rawMsg []byte) error
	Start(ctx context.Context)
}

// ScopeType distinguishes public and private channels.
type ScopeType int

const (
	ScopeTypeUnknown ScopeType = iota
	ScopeTypePublic
	ScopeTypePrivate
)

func (t ScopeType) String() string {
	switch t {
	case ScopeTypePublic:
		return "public"
	case ScopeTypePrivate:
		return "private"
	default:
		return ""
	}
}

// ChannelStatus represents the current state of a WebSocket channel.
type ChannelStatus int

const (
	ChannelStatusUnknown ChannelStatus = iota
	ChannelStatusConnected
	ChannelStatusDisconnected
	ChannelStatusReconnecting
)

// Channel is a unified WebSocket channel that handles connection lifecycle,
// message routing, health checking, and authentication.
type Channel struct {
	exName           string
	endpoint         string
	newAuthor        AuthorConstructor
	newHealthChecker HealthCheckerConstructor

	mu        sync.RWMutex
	isRunning bool
	status    ChannelStatus

	conn              *websocket.Conn
	topicMg           TopicManager
	reconnectInterval time.Duration
	scopeType         ScopeType

	credential Credential

	author          Author
	healthChecker   HealthChecker
	msgReceiver     *MessageReceiver
	latencyMeasurer *LatencyMeasurer

	reconnectCount   int
	reconnectSigChan chan struct{}

	msgChan     <-chan []byte
	errChan     <-chan error
	authErrChan <-chan error
	alertChan   <-chan error
}

// NewChannel creates a new WebSocket channel.
func NewChannel(
	exName string,
	endpoint string,
	scopeType ScopeType,
	credential Credential,
	measurer *LatencyMeasurer,
	newAuthor AuthorConstructor,
	newHealthChecker HealthCheckerConstructor,
) *Channel {
	return &Channel{
		exName:            exName,
		endpoint:          endpoint,
		reconnectInterval: 1 * time.Second,
		scopeType:         scopeType,
		credential:        credential,
		latencyMeasurer:   measurer,

		newAuthor:        newAuthor,
		newHealthChecker: newHealthChecker,

		reconnectSigChan: make(chan struct{}, 1000),
	}
}

// SetTopicMg sets the topic manager for this channel.
func (c *Channel) SetTopicMg(mg TopicManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.topicMg = mg
}

// GetTopicMg returns the topic manager.
func (c *Channel) GetTopicMg() TopicManager {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.topicMg
}

// IsRunning returns whether Start has been called and the channel is active.
func (c *Channel) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *Channel) isPrivateChan() bool {
	return c.scopeType == ScopeTypePrivate
}

func (c *Channel) sendSubscribeRequest() error {
	for _, msg := range c.topicMg.SubscribeRequests() {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Channel) resetReconnectSigChan() {
	close(c.reconnectSigChan)
	c.reconnectSigChan = make(chan struct{}, 1000)
}

func (c *Channel) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %v", err)
	}
	c.conn = conn

	const (
		authNumOfRetry    = 3
		authRetryInterval = 1 * time.Second

		pingInterval    = 3 * time.Second
		timeoutDuration = 3 * time.Second
	)

	// setup message receiver
	c.msgReceiver = NewMessageReceiver(c.conn)
	c.msgChan = c.msgReceiver.GetMsgChan()
	c.errChan = c.msgReceiver.GetErrChan()

	// setup health checker
	c.healthChecker = c.newHealthChecker(c.conn, pingInterval, timeoutDuration)
	c.alertChan = c.healthChecker.GetHealthAlertChan()
	c.msgReceiver.SetHandler(c.healthChecker.PongReceiveHandleFunc)

	// setup latency measurer
	c.msgReceiver.SetHandler(c.recordLatencyHandleFunc)

	// setup subscribe response handler
	c.msgReceiver.SetHandler(c.topicMg.HandleSubscribeResp)

	// setup author
	if c.isPrivateChan() {
		c.author = c.newAuthor(conn, c.credential, authNumOfRetry, authRetryInterval)
		c.msgReceiver.SetHandler(c.author.HandleAuthMessage)
		c.authErrChan = c.author.GetAuthErrChan()
	}

	return nil
}

func (c *Channel) initAndListen(ctx context.Context) error {
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize channel: %v", err)
	}

	go c.msgReceiver.Start()
	time.Sleep(200 * time.Millisecond)
	go c.healthChecker.Start(ctx)
	if c.isPrivateChan() {
		go c.author.Start(ctx)
	}

	time.Sleep(200 * time.Millisecond)

	if err := c.sendSubscribeRequest(); err != nil {
		return fmt.Errorf("failed to send subscribe request: %v", err)
	}
	return nil
}

func (c *Channel) handleTopicMessage(timestamp *time.Time, rawMsg []byte) error {
	if string(rawMsg) == "" {
		return nil
	}
	return c.topicMg.HandleTopicMessage(timestamp, rawMsg)
}

// GetLatencyMeasurer returns the latency measurer.
func (c *Channel) GetLatencyMeasurer() *LatencyMeasurer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latencyMeasurer
}

// GetStatus returns the current channel status.
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

// Start initializes the channel and begins listening.
// Automatically reconnects on disconnection. Cancel ctx to stop.
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
				slog.Error(fmt.Errorf("failed to close websocket: %v", err).Error())
			}
			wsCancel()
			c.mu.Lock()
			c.isRunning = false
			c.mu.Unlock()
			return nil

		case err := <-c.alertChan:
			slog.Error(fmt.Errorf("receive health alert (%s): %v", c.exName, err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case err := <-c.errChan:
			slog.Error(fmt.Errorf("receive error (%s %s channel): %v", c.exName, c.scopeType.String(), err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case err := <-c.authErrChan:
			slog.Error(fmt.Errorf("receive auth error (%s %s channel): %v", c.exName, c.scopeType.String(), err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case <-c.reconnectSigChan:
			time.Sleep(c.reconnectInterval)
			c.setStatus(ChannelStatusReconnecting)
			c.resetReconnectSigChan()
			slog.Info(fmt.Sprintf("try to reconnect (%s %s channel)", c.exName, c.scopeType.String()))
			if err := reconnect(); err != nil {
				c.setStatus(ChannelStatusDisconnected)
				c.reconnectCount++
				slog.Error(fmt.Errorf("failed to reconnect (%s %s channel count: %d): %v", c.exName, c.scopeType.String(), c.reconnectCount, err).Error())
				c.reconnectSigChan <- struct{}{}
			} else {
				c.setStatus(ChannelStatusConnected)
				c.reconnectCount = 0
				slog.Info(fmt.Sprintf("succeeded to reconnect (%s %s channel)", c.exName, c.scopeType.String()))
			}

		case rawMsg := <-c.msgChan:
			timestamp := time.Now()
			if err := c.handleTopicMessage(&timestamp, rawMsg); err != nil {
				slog.Error(
					fmt.Errorf("failed to handle message: %v", err).Error(),
					"message", string(rawMsg),
				)
			}
		}
	}
}

func (c *Channel) recordLatencyHandleFunc(rawMsg []byte) error {
	if c.latencyMeasurer == nil {
		return nil
	}
	tName, latency, err := c.topicMg.MeasureLatency(rawMsg)
	if err != nil {
		return err
	}
	if tName == "" {
		return nil
	}
	c.latencyMeasurer.RecordLatency(tName, latency)
	return nil
}

package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/ws/topics"
)

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
	topicMap          map[string]topics.Topic
	apiVersion        string
	reconnectInterval time.Duration
	scopeType         ScopeType
	product           Product

	// credential
	credential gateways.Credential

	healthChecker   *healthChecker
	msgReceiver     *messageReceiver
	author          *author
	latencyMeasurer *gateways.LatencyMeasurer

	// reconnect chan
	reconnectCount   int
	reconnectSigChan chan struct{}

	// receive channels
	msgChan     <-chan []byte
	errChan     <-chan error
	authErrChan <-chan error
	alertChan   <-chan error
}

// NewPublicChannel creates a public WebSocket channel for market data.
func NewPublicChannel(
	product Product,
	measurer *gateways.LatencyMeasurer,
) *Channel {
	return newChannel(
		scopeTypePublic,
		product,
		nil,
		measurer,
	)
}

// NewPrivateChannel creates a private WebSocket channel for account data.
func NewPrivateChannel(
	credential gateways.Credential,
	measurer *gateways.LatencyMeasurer,
) *Channel {
	return newChannel(
		scopeTypePrivate,
		ProductUnknown,
		credential,
		measurer,
	)
}

func newChannel(
	scopeType ScopeType,
	product Product,
	credential gateways.Credential,
	measurer *gateways.LatencyMeasurer,
) *Channel {
	return &Channel{
		topicMap:          map[string]topics.Topic{},
		apiVersion:        APIv5,
		reconnectInterval: 1 * time.Second,
		scopeType:         scopeType,
		product:           product,
		credential:        credential,
		latencyMeasurer:   measurer,

		reconnectSigChan: make(chan struct{}, 1000),
	}
}

func (c *Channel) SetTopics(topics []topics.Topic) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, topic := range topics {
		c.topicMap[topic.TopicName()] = topic
	}
}

func (c *Channel) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isRunning
}

func (c *Channel) isPrivateChan() bool {
	return c.scopeType == scopeTypePrivate
}

func (c *Channel) path() string {
	switch c.scopeType {
	case scopeTypePublic:
		return HostName + c.apiVersion + c.scopeType.Path() + c.product.Path()
	default:
		return HostName + c.apiVersion + c.scopeType.Path()
	}
}

func (c *Channel) topicName(rawMsg []byte) (string, error) {
	msg := struct {
		TopicName string `json:"topic"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return "", err
	}
	return msg.TopicName, nil
}

func (c *Channel) genSubscribeMessage() ([]byte, error) {
	var topicNames []string
	for _, topic := range c.topicMap {
		topicNames = append(topicNames, string(topic.TopicName()))
	}

	message, err := json.Marshal(struct {
		Op   string   `json:"op"`
		Args []string `json:"args"`
	}{
		Op:   "subscribe",
		Args: topicNames,
	})
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (c *Channel) sendSubscribeRequest() error {
	subscribeMessage, err := c.genSubscribeMessage()
	if err != nil {
		return err
	}
	if err = c.conn.WriteMessage(websocket.TextMessage, subscribeMessage); err != nil {
		return err
	}
	return nil
}

func (c *Channel) ResetReconnectSigChan() {
	defer close(c.reconnectSigChan)
	c.reconnectSigChan = make(chan struct{}, 1000)
}

func (c *Channel) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{Proxy: http.ProxyFromEnvironment, HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial("wss://"+c.path(), nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %v", err)
	}
	c.conn = conn

	const (
		authNumOfRetry    = 3
		authRetryInterval = 1 * time.Second

		pingInterval    = 1 * time.Second
		timeoutDuration = 1 * time.Second
	)

	// setup message receiver
	c.msgReceiver = newMessageReceiver(c.conn)
	c.msgChan = c.msgReceiver.GetMsgChan()
	c.errChan = c.msgReceiver.GetErrChan()

	// setup health checker
	c.healthChecker = newHealthChecker(c.conn, pingInterval, timeoutDuration)
	c.alertChan = c.healthChecker.GetHealthAlertChan()

	// setup latency measurer
	c.msgReceiver.SetHandler(c.recordLatencyHandleFunc)

	// setup author
	if c.isPrivateChan() {
		c.author = newAuthor(conn, c.credential, authNumOfRetry, authRetryInterval)
		c.msgReceiver.SetHandler(c.author.handleAuthMessage)
		c.authErrChan = c.author.GetAuthErrChan()
	}
	return nil
}

func (c *Channel) initAndListen(ctx context.Context) error {
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize channel: %v", err)
	}

	go c.msgReceiver.start()
	time.Sleep(200 * time.Millisecond)
	go c.healthChecker.start(ctx)
	if c.isPrivateChan() {
		go c.author.start(ctx)
	}

	time.Sleep(200 * time.Millisecond)

	if err := c.sendSubscribeRequest(); err != nil {
		return fmt.Errorf("failed to send subscribe request: %v", err)
	}
	return nil
}

func (c *Channel) handleTopicMessage(timestamp *time.Time, rawMsg []byte) error {
	topicName, err := c.topicName(rawMsg)
	if err != nil {
		return fmt.Errorf("failed to identify topic name: %v", err)
	}
	if t, ok := c.topicMap[topicName]; ok {
		if err = t.MsgHandler(timestamp, rawMsg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Channel) GetLatencyMeasurer() *gateways.LatencyMeasurer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latencyMeasurer
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

// Start initializes the channel and begins listening.
// Automatically reconnects on disconnection. Cancel ctx to stop.
func (c *Channel) Start(ctx context.Context) error {
	c.isRunning = true
	wsCtx, wsCancel := context.WithCancel(context.Background())
	defer wsCancel()
	if err := c.initAndListen(wsCtx); err != nil {
		wsCancel()
		return err
	}

	reconnect := func() error {
		wsCtx, wsCancel = context.WithCancel(context.Background())
		if err := c.initAndListen(wsCtx); err != nil {
			wsCancel()
			return fmt.Errorf("failed to reinitialize channel: %v", err)
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			if err := c.conn.Close(); err != nil {
				slog.Error(fmt.Errorf("failed to close websocket: %v", err).Error())
			}
			wsCancel()
			c.isRunning = false
			return nil

		case err := <-c.alertChan:
			slog.Error(fmt.Errorf("receive health alert (bybit): %v", err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case err := <-c.errChan:
			slog.Error(fmt.Errorf("receive error message (bybit %s channel): %v", c.scopeType.String(), err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case err := <-c.authErrChan:
			slog.Error(fmt.Errorf("receive auth error (bybit %s channel): %v", c.scopeType.String(), err).Error())
			c.reconnectSigChan <- struct{}{}
			c.setStatus(ChannelStatusDisconnected)

		case <-c.reconnectSigChan:
			time.Sleep(c.reconnectInterval)
			c.setStatus(ChannelStatusReconnecting)
			c.ResetReconnectSigChan()
			slog.Info(fmt.Sprintf("try to reconnect (bybit %s channel)", c.scopeType.String()))
			if err := reconnect(); err != nil {
				c.setStatus(ChannelStatusDisconnected)
				c.reconnectCount++
				slog.Error(fmt.Errorf("failed to reconnect (bybit %s channel count: %d): %v", c.scopeType.String(), c.reconnectCount, err).Error())
				c.reconnectSigChan <- struct{}{}
			} else {
				c.setStatus(ChannelStatusConnected)
				c.reconnectCount = 0
				slog.Info(fmt.Sprintf("succeeded to reconnect (bybit %s channel)", c.scopeType.String()))
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
	topicName, latency, err := measureLatency(rawMsg)
	if err != nil {
		return err
	}
	if topicName == "" {
		return nil
	}
	c.latencyMeasurer.RecordLatency(topicName, latency)
	return nil
}

func measureLatency(rawMsg []byte) (string, time.Duration, error) {
	now := time.Now()
	msg := struct {
		CreationTime int64  `json:"creationTime"`
		Ts           int64  `json:"ts"`
		Topic        string `json:"topic"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return "", 0, err
	}

	t := msg.Ts
	if t == 0 {
		t = msg.CreationTime
	}

	return msg.Topic, now.Sub(time.UnixMilli(t)), nil
}

package topics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
)

// Manager implements gateways.TopicManager for Bybit.
type Manager struct {
	mu       sync.RWMutex
	topicMap map[string]gateways.Topic
}

func NewManager() *Manager {
	return &Manager{
		topicMap: map[string]gateways.Topic{},
	}
}

func (mg *Manager) SetTopics(ts []gateways.Topic) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	for _, t := range ts {
		mg.topicMap[t.TopicName()] = t
	}
}

// SubscribeRequests returns a single batch subscribe message containing all topic names.
func (mg *Manager) SubscribeRequests() [][]byte {
	mg.mu.RLock()
	defer mg.mu.RUnlock()

	var topicNames []string
	for name := range mg.topicMap {
		topicNames = append(topicNames, name)
	}
	if len(topicNames) == 0 {
		return nil
	}

	msg, err := json.Marshal(struct {
		Op   string   `json:"op"`
		Args []string `json:"args"`
	}{
		Op:   "subscribe",
		Args: topicNames,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("failed to marshal subscribe message: %v", err))
		return nil
	}
	return [][]byte{msg}
}

func (mg *Manager) HandleTopicMessage(timestamp *time.Time, rawMsg []byte) error {
	name, err := mg.getTopicName(rawMsg)
	if err != nil {
		return fmt.Errorf("failed to identify topic name: %v", err)
	}
	if name == "" {
		return nil
	}
	mg.mu.RLock()
	t, ok := mg.topicMap[name]
	mg.mu.RUnlock()
	if ok {
		if err = t.MsgHandler(timestamp, rawMsg); err != nil {
			return err
		}
	}
	return nil
}

func (mg *Manager) HandleSubscribeResp(rawMsg []byte) error {
	msg := struct {
		Op      string `json:"op"`
		Success bool   `json:"success"`
		ConnID  string `json:"conn_id"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return err
	}
	if msg.Op != "subscribe" {
		return nil
	}
	if msg.Success {
		slog.Info("succeeded to subscribe bybit topics")
	} else {
		slog.Warn("failed to subscribe bybit topics")
	}
	return nil
}

func (mg *Manager) getTopicName(rawMsg []byte) (string, error) {
	msg := struct {
		Topic string `json:"topic"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return "", err
	}
	return msg.Topic, nil
}

func (mg *Manager) MeasureLatency(rawMsg []byte) (string, time.Duration, error) {
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
	if t == 0 {
		return "", 0, nil
	}

	return msg.Topic, now.Sub(time.UnixMilli(t)), nil
}

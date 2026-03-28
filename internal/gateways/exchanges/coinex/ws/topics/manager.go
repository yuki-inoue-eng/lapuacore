package topics

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	chanOB = "depth.update"
)

type Manager struct {
	mu       sync.RWMutex
	topicMap map[string]Topic
	msgIDMap map[string]string // maps subscribe message ID to topic name
}

func NewManager() *Manager {
	return &Manager{
		topicMap: map[string]Topic{},
		msgIDMap: map[string]string{},
	}
}

func (mg *Manager) SetTopics(ts []Topic) {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	for _, t := range ts {
		mg.topicMap[t.TopicName()] = t
	}
}

func (mg *Manager) SubscribeRequests() [][]byte {
	mg.mu.Lock()
	defer mg.mu.Unlock()
	var result [][]byte
	for _, t := range mg.topicMap {
		msg := t.SubscribeMsg()
		if msg == nil {
			continue
		}
		result = append(result, msg)
		mg.msgIDMap[t.SubscribeMsgID()] = t.TopicName()
	}
	return result
}

func (mg *Manager) HandleTopicMessage(timestamp *time.Time, rawMsg []byte) error {
	name, err := mg.getTopicName(rawMsg)
	if err != nil {
		return fmt.Errorf("failed to identify topic name: %w", err)
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
		ID      int64  `json:"id"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return err
	}
	msgID := strconv.FormatInt(msg.ID, 10)
	mg.mu.RLock()
	channelName, ok := mg.msgIDMap[msgID]
	mg.mu.RUnlock()
	if !ok {
		return nil
	}
	if msg.Message == "OK" {
		slog.Info(fmt.Sprintf("succeeded to subscribe coinex %s channel", channelName))
	} else {
		slog.Warn(fmt.Sprintf("failed to subscribe coinex %s channel", channelName))
	}
	return nil
}

func (mg *Manager) getTopicName(rawMsg []byte) (string, error) {
	msg := struct {
		Method string `json:"method"`
		Data   struct {
			Symbol string `json:"market"`
		} `json:"data"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return "", err
	}
	if msg.Method != chanOB {
		return "", nil
	}
	return fmt.Sprintf("orderBook@%s", msg.Data.Symbol), nil
}

func genMsgID() int64 {
	u := uuid.New()
	i := new(big.Int)
	i.SetBytes(u[:8])
	intVal := i.Int64()
	if intVal < 0 {
		intVal = ^intVal
	}
	return intVal
}

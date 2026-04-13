package topics

import (
	"encoding/json"
	"fmt"
)

// topicBase provides default implementations for methods shared by all Bybit topics.
// SubscribeMsgID and SubscribeMsg are no-ops because subscription is handled
// as a batch by the Manager.
type topicBase struct{}

func (topicBase) SubscribeMsgID() string { return "" }
func (topicBase) SubscribeMsg() []byte   { return nil }

// MessageID extracts topic:ts as a unique identifier for Bybit messages.
func (topicBase) MessageID(rawMsg []byte) string {
	msg := struct {
		Topic string `json:"topic"`
		Ts    int64  `json:"ts"`
	}{}
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return ""
	}
	if msg.Topic == "" || msg.Ts == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", msg.Topic, msg.Ts)
}

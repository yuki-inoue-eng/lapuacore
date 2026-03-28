package topics

import "time"

// Topic is the interface for a WebSocket channel topic.
type Topic interface {
	TopicName() string
	SubscribeMsgID() string
	SubscribeMsg() []byte
	MsgHandler(timestamp *time.Time, rawMsg []byte) error
}
